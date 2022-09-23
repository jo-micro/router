package main

import (
	"log"
	"net/http"

	ginlogrus "github.com/toorop/gin-logrus"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"

	"github.com/gin-gonic/gin"
	httpServer "github.com/go-micro/plugins/v4/server/http"
	"jochum.dev/jo-micro/auth2"
	"jochum.dev/jo-micro/auth2/plugins/verifier/endpointroles"
	"jochum.dev/jo-micro/router"

	"jochum.dev/jo-micro/router/cmd/microrouterd/config"
	"jochum.dev/jo-micro/router/cmd/microrouterd/handler"
	"jochum.dev/jo-micro/router/internal/ilogger"
	"jochum.dev/jo-micro/router/internal/proto/routerserverpb"
	"jochum.dev/jo-micro/router/internal/util"
)

func internalService(routerHandler *handler.Handler) {
	srv := micro.NewService()

	opts := []micro.Option{
		micro.Name(config.Name + "-internal"),
		micro.Version(config.Version),
		micro.WrapHandler(auth2.ClientAuthRegistry().Wrapper()),
		micro.Action(func(c *cli.Context) error {
			if err := auth2.ClientAuthRegistry().Init(auth2.CliContext(c), auth2.Service(srv), auth2.Logrus(ilogger.Logrus())); err != nil {
				ilogger.Logrus().Fatal(err)
			}

			routerserverpb.RegisterRouterServerServiceHandler(srv.Server(), routerHandler)

			authVerifier := endpointroles.NewVerifier(
				endpointroles.WithLogrus(ilogger.Logrus()),
			)
			authVerifier.AddRules(
				endpointroles.RouterRule,
				endpointroles.NewRule(
					endpointroles.Endpoint(routerserverpb.RouterServerService.Routes),
					endpointroles.RolesAllow(auth2.RolesServiceAndAdmin),
				),
			)
			auth2.ClientAuthRegistry().Plugin().SetVerifier(authVerifier)

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
		ilogger.Logrus().Fatal(err)
	}

	// Stop the handler
	if err := routerHandler.Stop(); err != nil {
		ilogger.Logrus().Fatal(err)
	}

	// Stop the client/service auth plugin
	if err := auth2.ClientAuthRegistry().Stop(); err != nil {
		ilogger.Logrus().Fatal(err)
	}
}

func main() {
	srv := micro.NewService(
		micro.Server(httpServer.NewServer()),
	)

	routerAuthReg := auth2.RouterAuthRegistry()

	flags := ilogger.MergeFlags(routerAuthReg.MergeFlags(auth2.ClientAuthRegistry().MergeFlags([]cli.Flag{
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
	})))

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
			if err := ilogger.Start(c); err != nil {
				log.Fatal(err)
				return err
			}

			// Initialize the Auth Plugin over RouterAuthRegistry
			if err := routerAuthReg.Init(auth2.CliContext(c), auth2.Service(srv), auth2.Logrus(ilogger.Logrus())); err != nil {
				ilogger.Logrus().Fatal(err)
			}

			// Initialize GIN
			if c.Bool("router_debugmode") {
				gin.SetMode(gin.DebugMode)
			} else {
				gin.SetMode(gin.ReleaseMode)
			}
			r := gin.New()

			// Initalize the Handler
			if err := routerHandler.Init(srv, r, routerAuthReg.Plugin(), c.Int("router_refresh")); err != nil {
				ilogger.Logrus().Fatal(err)
			}

			// Add middlewares to gin
			r.Use(ginlogrus.Logger(ilogger.Logrus()), gin.Recovery())

			r.NoRoute(func(c *gin.Context) {
				c.JSON(404, gin.H{"code": http.StatusNotFound, "message": "page not found"})
			})

			// Register gin with micro
			if err := micro.RegisterHandler(srv.Server(), r); err != nil {
				ilogger.Logrus().Fatal(err)
			}

			return nil
		}),
	}
	srv.Init(opts...)

	go internalService(routerHandler)

	// Run server
	if err := srv.Run(); err != nil {
		ilogger.Logrus().Fatal(err)
	}

	// Stop the plugin in RouterAuthRegistry
	if err := routerAuthReg.Stop(); err != nil {
		ilogger.Logrus().Fatal(err)
	}

	// Stop the logger
	if err := ilogger.Stop(); err != nil {
		ilogger.Logrus().Fatal(err)
	}
}
