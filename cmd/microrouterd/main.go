// -lang=go1.19

package main

import (
	"log"

	ginlogrus "github.com/toorop/gin-logrus"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"

	"github.com/gin-gonic/gin"
	httpServer "github.com/go-micro/plugins/v4/server/http"
	"jochum.dev/jo-micro/auth2"
	"jochum.dev/jo-micro/router"

	"jochum.dev/jo-micro/router/cmd/microrouterd/config"
	"jochum.dev/jo-micro/router/cmd/microrouterd/handler"
	iConfig "jochum.dev/jo-micro/router/internal/config"
	iLogger "jochum.dev/jo-micro/router/internal/logger"
	"jochum.dev/jo-micro/router/internal/proto/routerserverpb"
)

func internalService(routerHandler *handler.Handler) {
	srv := micro.NewService()

	opts := []micro.Option{
		micro.Name(config.Name + "-internal"),
		micro.Version(config.Version),
		micro.Action(func(c *cli.Context) error {
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
		iLogger.Logrus().Fatal(err)
	}

	if err := routerHandler.Stop(); err != nil {
		iLogger.Logrus().Fatal(err)
	}
}

func main() {
	srv := micro.NewService(
		micro.Server(httpServer.NewServer()),
	)

	if err := iConfig.Load(config.GetConfig()); err != nil {
		logger.Fatal(err)
	}

	if config.GetRouterConfig().Env == config.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	routerHandler, err := handler.NewHandler()
	if err != nil {
		logger.Fatal(err)
	}

	authReg := auth.RouterAuthRegistry()

	flags := []cli.Flag{}
	flags = append(flags, iLogger.Flags()...)
	flags = append(flags, authReg.Flags()...)

	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Version(config.Version),
		micro.Address(config.GetRouterConfig().Address),
		micro.Flags(flags...),
		micro.Action(func(c *cli.Context) error {
			// Start the logger
			if err := iLogger.Start(c); err != nil {
				log.Fatal(err)
				return err
			}

			// Initialize the Auth Plugin over RouterAuthRegistry
			if err := authReg.Init(c, srv); err != nil {
				iLogger.Logrus().Fatal(err)
			}

			// Initalize the Handler
			if err := routerHandler.Init(srv, r, authReg.Plugin()); err != nil {
				iLogger.Logrus().Fatal(err)
			}

			// Add middlewares to gin
			r.Use(ginlogrus.Logger(iLogger.Logrus()), gin.Recovery())

			// Register gin with micro
			if err := micro.RegisterHandler(srv.Server(), r); err != nil {
				iLogger.Logrus().Fatal(err)
			}

			return nil
		}),
	}
	srv.Init(opts...)

	go internalService(routerHandler)

	// Run server
	if err := srv.Run(); err != nil {
		iLogger.Logrus().Fatal(err)
	}

	// Stop the plugin in RouterAuthRegistry
	if err := authReg.Stop(); err != nil {
		iLogger.Logrus().Fatal(err)
	}

	// Stop the logger
	if err := iLogger.Stop(); err != nil {
		iLogger.Logrus().Fatal(err)
	}
}
