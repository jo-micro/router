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
	jwtClient "jochum.dev/jo-micro/auth2/plugins/client/jwt"
	jwtRouter "jochum.dev/jo-micro/auth2/plugins/router/jwt"
	"jochum.dev/jo-micro/auth2/plugins/verifier/endpointroles"
	"jochum.dev/jo-micro/components"
	"jochum.dev/jo-micro/logruscomponent"
	"jochum.dev/jo-micro/router"

	"jochum.dev/jo-micro/router/cmd/microrouterd/config"
	"jochum.dev/jo-micro/router/cmd/microrouterd/handler"
	"jochum.dev/jo-micro/router/internal/proto/routerserverpb"
	"jochum.dev/jo-micro/router/internal/util"
)

func internalService(cReg *components.Registry, r *gin.Engine) {
	routerHandler := handler.New()

	auth2ClientReg := auth2.ClientAuthMustReg(cReg)

	opts := []micro.Option{
		micro.Name(config.Name + "-internal"),
		micro.Version(config.Version),
		micro.WrapHandler(auth2ClientReg.WrapHandler()),
		micro.Action(func(c *cli.Context) error {
			// Start the components
			if err := cReg.Init(c); err != nil {
				log.Fatal(err)
				return err
			}

			// Initalize the Handler
			if err := routerHandler.Init(cReg, r, c.Int("router_refresh"), c.String("router_ratelimiter_store_url")); err != nil {
				logger.Fatal(err)
				return err
			}

			routerserverpb.RegisterRouterServerServiceHandler(cReg.Service().Server(), routerHandler)

			authVerifier := endpointroles.NewVerifier(
				endpointroles.WithLogrus(logruscomponent.MustReg(cReg).Logger()),
			)
			authVerifier.AddRules(
				endpointroles.RouterRule,
				endpointroles.NewRule(
					endpointroles.Endpoint(routerserverpb.RouterServerService.Routes),
					endpointroles.RolesAllow(auth2.RolesServiceAndAdmin),
				),
			)
			auth2.ClientAuthMustReg(cReg).Plugin().SetVerifier(authVerifier)

			return nil
		}),
	}

	cReg.Service().Init(opts...)

	// Run server
	if err := cReg.Service().Run(); err != nil {
		logger.Fatal(err)
		return
	}

	// Stop the handler
	if err := routerHandler.Stop(); err != nil {
		logger.Fatal(err)
		return
	}

	// Stop the client/service auth plugin
	if err := cReg.Stop(); err != nil {
		logger.Fatal(err)
		return
	}
}

func main() {
	service := micro.NewService(
		micro.Server(httpServer.NewServer()),
	)

	cReg := components.New(service, "router", logruscomponent.New(), auth2.RouterAuthComponent())
	auth2RouterReg := auth2.RouterAuthMustReg(cReg)
	auth2RouterReg.Register(jwtRouter.New())

	iService := micro.NewService()
	iCReg := components.New(iService, "router", logruscomponent.New(), auth2.ClientAuthComponent(), router.New())

	auth2ClientReg := auth2.ClientAuthMustReg(iCReg)
	auth2ClientReg.Register(jwtClient.New())

	var r *gin.Engine

	flags := components.FilterDuplicateFlags(iCReg.AppendFlags(cReg.AppendFlags([]cli.Flag{
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
		&cli.StringFlag{
			Name:    "router_ratelimiter_store_url",
			Usage:   "Ratelimiter store URL, for example redis://localhost:6379/0",
			EnvVars: []string{"MICRO_ROUTER_RATELIMITER_STORE_URL"},
			Value:   "memory://",
		},
	})))

	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Version(config.Version),
		micro.Address(util.GetEnvDefault("MICRO_ROUTER_LISTEN", ":8080")),
		micro.Flags(flags...),
		micro.Action(func(c *cli.Context) error {
			// Start the components
			if err := cReg.Init(c); err != nil {
				log.Fatal(err)
				return err
			}

			// Initialize GIN
			if c.Bool("router_debugmode") {
				gin.SetMode(gin.DebugMode)
			} else {
				gin.SetMode(gin.ReleaseMode)
			}
			r = gin.New()
			r.ForwardedByClientIP = true

			// Add middlewares to gin
			r.Use(ginlogrus.Logger(logruscomponent.MustReg(cReg).Logger()), gin.Recovery())

			r.NoRoute(func(c *gin.Context) {
				c.JSON(http.StatusNotFound, gin.H{"errors": []gin.H{{"id": "NOT_FOUND", "message": "page not found"}}})
			})

			// Register gin with micro
			if err := micro.RegisterHandler(service.Server(), r); err != nil {
				logger.Fatal(err)
				return err
			}

			return nil
		}),
	}
	service.Init(opts...)

	go internalService(iCReg, r)

	// Run server
	if err := service.Run(); err != nil {
		logger.Fatal(err)
		return
	}

	// Stop the plugin in RouterAuthRegistry
	if err := cReg.Stop(); err != nil {
		logger.Fatal(err)
		return
	}
}
