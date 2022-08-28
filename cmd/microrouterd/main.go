package main

import (
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/logger"

	"github.com/gin-gonic/gin"
	httpServer "github.com/go-micro/plugins/v4/server/http"

	"github.com/go-micro/router/config"
	"github.com/go-micro/router/handler"
)

func main() {
	srv := micro.NewService(
		micro.Server(httpServer.NewServer()),
	)

	if config.GetServerConfig().Env == config.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	routerHandler, err := handler.NewHandler(srv, router)
	if err != nil {
		logger.Fatal(err)
	}

	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Version(config.Version),
		micro.Address(config.GetServerConfig().Address),
		micro.Client(client.NewClient(client.ContentType("application/json"))),
		micro.Action(func(c *cli.Context) error {
			router.Use(gin.Logger(), gin.Recovery())

			if err := micro.RegisterHandler(srv.Server(), router); err != nil {
				logger.Fatal(err)
			}

			if err := routerHandler.Start(); err != nil {
				logger.Fatal(err)
			}

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
