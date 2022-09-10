package main

import (
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"

	"github.com/gin-gonic/gin"
	httpServer "github.com/go-micro/plugins/v4/server/http"
	"jochum.dev/jo-micro/router"

	"jochum.dev/jo-micro/router/internal/config"
	"jochum.dev/jo-micro/router/internal/handler"
	"jochum.dev/jo-micro/router/internal/proto/routerserverpb"
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

			r := router.NewHandler(
				config.GetRouterConfig().RouterURI,
				router.NewRoute(
					router.Method(router.MethodGet),
					router.Path("/routes"),
					router.Endpoint(routerserverpb.RouterServerService.Routes),
				),
			)
			r.RegisterWithServer(srv.Server())

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

	if config.GetRouterConfig().Env == config.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Version(config.Version),
		micro.Address(config.GetRouterConfig().Address),
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
