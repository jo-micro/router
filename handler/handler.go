package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-micro/router/config"
	"github.com/go-micro/router/proto/routerclientpb"
	"github.com/go-micro/router/util"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/errors"
	"go-micro.dev/v4/logger"
	"google.golang.org/protobuf/types/known/emptypb"
)

type JSONRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Handler is the handler for the proxy
type Handler struct {
	service micro.Service
	engine  *gin.Engine
	routes  map[string]bool
}

func NewHandler(service micro.Service, engine *gin.Engine) (*Handler, error) {
	return &Handler{
		service: service,
		engine:  engine,
		routes:  make(map[string]bool),
	}, nil
}

func (h *Handler) Start() error {
	globalGroup := h.engine.Group("")
	globalGroup.Handle("GET", fmt.Sprintf("/%s/routes", config.RouterURI), h.ginRoutes)

	// Refresh routes for the proxy every 10 seconds
	go func() {
		ctx := context.Background()

		for {
			services, err := util.FindByEndpoint(h.service, routerclientpb.RouterClientService.Routes)
			if err != nil {
				logger.Error(err)
				continue
			}

			for _, s := range services {
				client := routerclientpb.NewRouterClientService(s.Name, h.service.Client())
				resp, err := client.Routes(ctx, &emptypb.Empty{})
				if err != nil {
					// failure in getting routes, silently ignore
					continue
				}

				serviceGroup := globalGroup.Group(fmt.Sprintf("/%s", resp.GetRouterURI()))

				for _, route := range resp.Routes {
					logger.Info("Found route for Endpoint %s", route.Endpoint)

					var g *gin.RouterGroup = nil

					if route.IsGlobal {
						g = globalGroup
					} else {
						g = serviceGroup
					}

					// Calculate the path of the route and register it if it's not registered yet
					path := fmt.Sprintf("%s: %s/%s", route.Method, g.BasePath(), route.Path)
					if _, ok := h.routes[path]; !ok {
						g.Handle(route.GetMethod(), route.GetPath(), h.proxy(s.Name, route))
						h.routes[path] = true
					}
				}
			}

			time.Sleep(time.Duration(config.GetServerConfig().RefreshSeconds) * time.Second)
		}
	}()

	return nil
}

func (h *Handler) Stop() error {
	return nil
}

func (h *Handler) proxy(serviceName string, route *routerclientpb.RoutesReply_Route) func(*gin.Context) {
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
						data, err := ioutil.ReadAll(fp)
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

		ctx := util.CtxFromRequest(c, c.Request)

		// remote call
		var response json.RawMessage
		err := h.service.Client().Call(ctx, req, &response)
		if err != nil {
			logger.Error(err)

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

func (h *Handler) ginRoutes(c *gin.Context) {
	ginRoutes := h.engine.Routes()
	rRoutes := []JSONRoute{}
	for _, route := range ginRoutes {
		rRoutes = append(rRoutes, JSONRoute{Method: route.Method, Path: route.Path})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  200,
		"message": "Dumping the routes",
		"data":    rRoutes,
	})
}
