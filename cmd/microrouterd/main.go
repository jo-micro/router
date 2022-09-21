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
	iLogger "jochum.dev/jo-micro/router/internal/logger"
	"jochum.dev/jo-micro/router/internal/proto/routerserverpb"
	"jochum.dev/jo-micro/router/internal/util"
)

func internalService(routerHandler *handler.Handler) {
	srv := micro.NewService()

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "router_basepath",
			Usage:   "Router basepath",
			EnvVars: []string{"MICRO_ROUTER_BASEPATH"},
			Value:   "router",
		},
	}

	opts := []micro.Option{
		micro.Name(config.Name + "-internal"),
		micro.Version(config.Version),
		micro.Flags(flags...),
		micro.Action(func(c *cli.Context) error {
			routerserverpb.RegisterRouterServerServiceHandler(srv.Server(), routerHandler)

			r := router.NewHandler(
				c.String("router_basepath"),
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

	authReg := auth2.RouterAuthRegistry()

	flags := []cli.Flag{
		// General
		&cli.BoolFlag{
			Name:    "router_debugmode",
			Usage:   "Run gin in debugmode?",
			EnvVars: []string{"MICRO_ROUTER_DEBUGMODE"},
			Value:   false,
		},
		&cli.StringFlag{
			Name:    "router_basepath",
			Usage:   "Router basepath",
			EnvVars: []string{"MICRO_ROUTER_BASEPATH"},
			Value:   "router",
		},
		&cli.IntFlag{
			Name:    "router_refresh",
			Usage:   "Router refresh routes every x seconds",
			EnvVars: []string{"MICRO_ROUTER_REFRESH"},
			Value:   10,
		},
		&cli.StringFlag{
			Name:    "router_listen",
			Usage:   "Router listen on",
			EnvVars: []string{"MICRO_ROUTER_LISTEN"},
			Value:   ":8080",
		},
	}
	flags = append(flags, iLogger.Flags()...)
	flags = append(flags, authReg.Flags()...)

	routerHandler, err := handler.NewHandler()
	if err != nil {
		logger.Fatal(err)
	}

	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Version(config.Version),
		micro.Address(util.GetEnvDefault("MICRO_ROUTER_LISTEN", ":8080")),
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

			// Initialize GIN
			if c.Bool("router_debugmode") {
				gin.SetMode(gin.DebugMode)
			} else {
				gin.SetMode(gin.ReleaseMode)
			}
			r := gin.New()

			// Initalize the Handler
			if err := routerHandler.Init(srv, r, authReg.Plugin(), c.Int("router_refresh")); err != nil {
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
