package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	auth "jochum.dev/jo-micro/auth2"
	iLogger "jochum.dev/jo-micro/router/internal/logger"
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
	routerAuth auth.RouterPlugin
	routes     map[string]*routerclientpb.RoutesReply_Route
}

func NewHandler() (*Handler, error) {
	return &Handler{
		routes: make(map[string]*routerclientpb.RoutesReply_Route),
	}, nil
}

func (h *Handler) Init(service micro.Service, engine *gin.Engine, routerAuth auth.RouterPlugin, refreshSeconds int) error {
	h.service = service
	h.engine = engine
	h.routerAuth = routerAuth
	globalGroup := h.engine.Group("")

	// Refresh routes for the proxy every 10 seconds
	go func() {
		ctx := context.Background()

		for {
			services, err := util.FindByEndpoint(h.service, "RouterClientService.Routes")
			if err != nil {
				iLogger.Logrus().Error(err)
				continue
			}

			for _, s := range services {
				iLogger.Logrus().WithField("service", s.Name).Tracef("Found service")
				client := routerclientpb.NewRouterClientService(s.Name, h.service.Client())
				resp, err := client.Routes(ctx, &emptypb.Empty{})
				if err != nil {
					iLogger.Logrus().Error(err)
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
					pathMethod := fmt.Sprintf("%s:%s%s", route.GetMethod(), g.BasePath(), route.GetPath())
					path := fmt.Sprintf("%s%s", g.BasePath(), route.GetPath())
					if _, ok := h.routes[pathMethod]; !ok {
						iLogger.Logrus().
							WithField("service", s.Name).
							WithField("endpoint", route.GetEndpoint()).
							WithField("method", route.GetMethod()).
							WithField("path", path).
							Debugf("Found route")

						g.Handle(route.GetMethod(), route.GetPath(), h.proxy(s.Name, route, route.AuthRequired))
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

func (h *Handler) proxy(serviceName string, route *routerclientpb.RoutesReply_Route, authRequired bool) func(*gin.Context) {
	return func(c *gin.Context) {
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

		req := h.service.Client().NewRequest(serviceName, route.GetEndpoint(), request, client.WithContentType("application/json"))

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
			iLogger.Logrus().Error(err)

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
			Method: route.Method,
			Path:   route.Path,
		})
	}

	return nil
}
