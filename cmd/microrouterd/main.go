package main

import (
	"net/http"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"

	"github.com/gin-gonic/gin"
	httpServer "github.com/go-micro/plugins/v4/server/http"
	"github.com/go-micro/router"

	"github.com/go-micro/router/config"
	"github.com/go-micro/router/handler"
	"github.com/go-micro/router/proto/routerclientpb"
	"github.com/go-micro/router/proto/routerserverpb"
)

func internalService(engine *gin.Engine) {
	srv := micro.NewService()

	routerHandler, err := handler.NewHandler(srv, engine)
	if err != nil {
		logger.Fatal(err)
	}

	opts := []micro.Option{
		micro.Name(config.Name + "-internal"),
		micro.Version(config.Version),
		micro.Action(func(c *cli.Context) error {
			if err := routerHandler.Start(); err != nil {
				logger.Fatal(err)
			}

			routerserverpb.RegisterRouterServerServiceHandler(srv.Server(), routerHandler)

			routerHandler := router.NewHandler(
				config.GetServerConfig().RouterURI,
				router.NewRoute(
					router.RouteMethod(http.MethodGet),
					router.RoutePath("/routes"),
					router.RouteEndpoint(routerserverpb.RouterServerService.Routes),
				),
			)
			routerclientpb.RegisterRouterClientServiceHandler(srv.Server(), routerHandler)

			return nil
		}),
	}

	srv.Init(opts...)

	// Run server
	if err := srv.Run(); err != nil {
		logger.Fatal(err)
	}

	if err := routerHandler.Stop(); err != nil {
		logger.Fatal(err)
	}

}

func main() {
	srv := micro.NewService(
		micro.Server(httpServer.NewServer()),
	)

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}

	if config.GetServerConfig().Env == config.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Version(config.Version),
		micro.Address(config.GetServerConfig().Address),
		micro.Action(func(c *cli.Context) error {
			r.Use(gin.Logger(), gin.Recovery())

			if err := micro.RegisterHandler(srv.Server(), r); err != nil {
				logger.Fatal(err)
			}

			return nil
		}),
	}
	srv.Init(opts...)

	go internalService(r)

	// Run server
	if err := srv.Run(); err != nil {
		logger.Fatal(err)
	}
}
