package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	libredis "github.com/go-redis/redis/v8"

	limiter "github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"

	"github.com/gin-gonic/gin"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"jochum.dev/jo-micro/auth2"
	"jochum.dev/jo-micro/router/internal/ilogger"
	"jochum.dev/jo-micro/router/internal/proto/routerclientpb"
	"jochum.dev/jo-micro/router/internal/proto/routerserverpb"
	"jochum.dev/jo-micro/router/internal/util"
)

type JSONRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Handler is the handler for the proxy
type Handler struct {
	service    micro.Service
	engine     *gin.Engine
	routerAuth auth2.RouterPlugin
	routes     map[string]*routerclientpb.RoutesReply_Route
	rlStore    limiter.Store
}

func NewHandler() (*Handler, error) {
	return &Handler{
		routes: make(map[string]*routerclientpb.RoutesReply_Route),
	}, nil
}

func (h *Handler) Init(service micro.Service, engine *gin.Engine, routerAuth auth2.RouterPlugin, refreshSeconds int, rlStoreURL string) error {
	h.service = service
	h.engine = engine
	h.routerAuth = routerAuth
	globalGroup := h.engine.Group("")

	if strings.HasPrefix(rlStoreURL, "redis://") {
		// Create a redis client.
		option, err := libredis.ParseURL(rlStoreURL)
		if err != nil {
			return err
		}
		client := libredis.NewClient(option)

		// Create a store with the redis client.
		store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
			Prefix:   "rl",
			MaxRetry: 10,
		})
		if err != nil {
			return err
		}
		h.rlStore = store
	} else if rlStoreURL == "memory://" {
		h.rlStore = memory.NewStore()
	}

	// Refresh routes for the proxy every 10 seconds
	go func() {
		for {
			ctx := context.Background()

			services, err := util.FindByEndpoint(h.service, "RouterClientService.Routes")
			if err != nil {
				ilogger.Logrus().Error(err)
				continue
			}

			for _, s := range services {
				ilogger.Logrus().WithField("service", s.Name).Tracef("Found service")
				client := routerclientpb.NewRouterClientService(s.Name, h.service.Client())
				sCtx, err := auth2.ClientAuthRegistry().Plugin().ServiceContext(ctx)
				if err != nil {
					ilogger.Logrus().Error(err)
					continue
				}
				resp, err := client.Routes(sCtx, &emptypb.Empty{})
				if err != nil {
					ilogger.Logrus().Error(err)
					// failure in getting routes, silently ignore
					continue
				}

				serviceGroup := globalGroup.Group(fmt.Sprintf("/%s", resp.GetRouterURI()))

				for _, route := range resp.Routes {
					var g *gin.RouterGroup = nil

					if route.IsGlobal {
						g = globalGroup
					} else {
						g = serviceGroup
					}

					// Calculate the pathMethod of the route and register it if it's not registered yet
					pathMethod := fmt.Sprintf("%s:%s%s", route.Method, g.BasePath(), route.Path)
					path := fmt.Sprintf("%s%s", g.BasePath(), route.Path)
					if _, ok := h.routes[pathMethod]; !ok {
						ilogger.Logrus().
							WithField("service", s.Name).
							WithField("endpoint", route.Endpoint).
							WithField("method", route.Method).
							WithField("path", path).
							WithField("ratelimitClientIP", route.RatelimitClientIP).
							Debug("found route")

						clientIPRatelimiter := make([]*limiter.Limiter, len(route.RatelimitClientIP))
						if len(route.RatelimitClientIP) > 0 {
							if h.rlStore == nil {
								ilogger.Logrus().
									WithField("service", s.Name).
									WithField("endpoint", route.Endpoint).
									WithField("method", route.Method).
									WithField("path", path).
									WithField("ratelimitClientIP", route.RatelimitClientIP).
									Error("found a route with a limiter but there is no limiter store")
								continue
							}

							haveError := false
							for idx, rate := range route.RatelimitClientIP {
								rate, err := limiter.NewRateFromFormatted(rate)
								if err != nil {
									ilogger.Logrus().
										WithField("service", s.Name).
										WithField("endpoint", route.Endpoint).
										WithField("method", route.Method).
										WithField("path", path).
										WithField("ratelimitClientIP", route.RatelimitClientIP).
										Error(err)
									haveError = true
									break
								}

								clientIPRatelimiter[idx] = limiter.New(h.rlStore, rate)
							}

							if haveError {
								continue
							}
						}

						g.Handle(route.Method, route.Path, h.proxy(s.Name, route, route.AuthRequired, path, clientIPRatelimiter))
						h.routes[pathMethod] = route
						h.routes[pathMethod].Path = path
					}
				}
			}

			time.Sleep(time.Duration(refreshSeconds) * time.Second)
		}
	}()

	return nil
}

func (h *Handler) Stop() error {
	return nil
}

func (h *Handler) proxy(serviceName string, route *routerclientpb.RoutesReply_Route, authRequired bool, path string, clientIPRatelimiter []*limiter.Limiter) func(*gin.Context) {
	return func(c *gin.Context) {

		if len(clientIPRatelimiter) > 0 {
			for idx, l := range clientIPRatelimiter {
				context, err := l.Get(c, fmt.Sprintf("%s-%s-%s", path, l.Rate.Formatted, c.ClientIP()))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"status":  http.StatusInternalServerError,
						"message": "Internal server error",
					})
					c.Abort()
					return
				}

				if idx == 0 {
					c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
					c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
					c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))
				}

				if context.Reached {
					c.JSON(http.StatusTooManyRequests, gin.H{
						"status":  http.StatusTooManyRequests,
						"message": "To many requests",
					})
					c.Abort()
					return
				}
			}
		}

		// Map query/path params
		params := make(map[string]string)
		for _, p := range route.Params {
			if len(c.Query(p)) > 0 {
				params[p] = c.Query(p)
			}
		}
		for _, p := range route.Params {
			if len(c.Param(p)) > 0 {
				params[p] = c.Param(p)
			}
		}

		// Bind the request if POST/PATCH/PUT
		request := gin.H{}
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodPut {
			mf, err := c.MultipartForm()
			if err == nil {
				for k, files := range mf.File {
					for _, file := range files {
						fp, err := file.Open()
						if err != nil {
							continue
						}
						data, err := io.ReadAll(fp)
						if err != nil {
							continue
						}

						if len(files) > 1 {
							if _, ok := request[k]; !ok {
								request[k] = []string{base64.StdEncoding.EncodeToString(data)}
							} else {
								request[k] = append(request[k].([]string), base64.StdEncoding.EncodeToString(data))
							}
						} else {
							request[k] = base64.StdEncoding.EncodeToString(data)
						}
					}
				}

				for k, v := range mf.Value {
					if len(v) > 1 {
						request[k] = v
					} else {
						request[k] = v[0]
					}

				}
			} else {
				if c.ContentType() == "" {
					c.JSON(http.StatusUnsupportedMediaType, gin.H{
						"status":  http.StatusUnsupportedMediaType,
						"message": "provide a content-type header",
					})
					return
				}
				c.ShouldBind(&request)
			}
		}

		// Set query/route params to the request
		for pn, p := range params {
			request[pn] = p
		}

		req := h.service.Client().NewRequest(serviceName, route.Endpoint, request, client.WithContentType("application/json"))

		// Auth
		ctx, err := h.routerAuth.ForwardContext(c.Request, c)
		if err != nil && authRequired {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": err,
			})
			return
		}

		// remote call
		var response json.RawMessage
		err = h.service.Client().Call(ctx, req, &response)
		if err != nil {
			ilogger.Logrus().Error(err)

			pErr := errors.FromError(err)
			code := int(http.StatusInternalServerError)
			if pErr.Code != 0 {
				code = int(pErr.Code)
			}
			c.JSON(code, gin.H{
				"status":  code,
				"message": pErr.Detail,
			})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func (h *Handler) Routes(ctx context.Context, in *emptypb.Empty, out *routerserverpb.RoutesReply) error {
	for _, route := range h.routes {
		out.Routes = append(out.Routes, &routerserverpb.RoutesReply_Route{
			Method:            route.Method,
			Path:              route.Path,
			Params:            route.Params,
			Endpoint:          route.Endpoint,
			AuthRequired:      route.AuthRequired,
			RatelimitClientIP: route.RatelimitClientIP,
		})
	}

	return nil
}
