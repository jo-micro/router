package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4/server"
	"google.golang.org/protobuf/types/known/emptypb"
	"jochum.dev/jo-micro/auth2"
	"jochum.dev/jo-micro/auth2/plugins/verifier/endpointroles"
	"jochum.dev/jo-micro/components"
	"jochum.dev/jo-micro/logruscomponent"
	"jochum.dev/jo-micro/router/internal/proto/routerclientpb"
	"jochum.dev/jo-micro/router/internal/util"
)

const Name = "router"

// Handler is the handler for jochum.dev/jo-micro/router/proto/routerpb.RrouterService
type Handler struct {
	initialized bool
	routerURI   string
	routes      []*routerclientpb.RoutesReply_Route
}

// NewHandler returns a new dynrouterpb Handler
func New() *Handler {
	return &Handler{initialized: false, routes: []*routerclientpb.RoutesReply_Route{}}
}

func MustReg(cReg *components.Registry) *Handler {
	return cReg.Must(Name).(*Handler)
}

func (h *Handler) Name() string {
	return Name
}

func (h *Handler) Priority() int {
	return 1000
}

func (h *Handler) Initialized() bool {
	return h.initialized
}

func (h *Handler) Init(r *components.Registry, cli *cli.Context) error {
	if h.initialized {
		return nil
	}

	h.routerURI = cli.String(fmt.Sprintf("%s_router_basepath", strings.ToLower(r.FlagPrefix())))

	if _, err := r.Get(auth2.ClientAuthName); err != nil {
		authVerifier := endpointroles.NewVerifier(
			endpointroles.WithLogrus(logruscomponent.MustReg(r).Logger()),
		)
		authVerifier.AddRules(
			endpointroles.NewRule(
				endpointroles.Endpoint(routerclientpb.RouterClientService.Routes),
				endpointroles.RolesAllow([]string{auth2.ROLE_SERVICE}),
			),
		)
		auth2.ClientAuthMustReg(r).Plugin().AddVerifier(authVerifier)
	}

	h.RegisterWithServer(r.Service().Server())

	h.initialized = true
	return nil
}

func (h *Handler) Stop() error {
	return nil
}

func (h *Handler) Flags(r *components.Registry) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    fmt.Sprintf("%s_router_basepath", strings.ToLower(r.FlagPrefix())),
			Usage:   "Router basepath",
			EnvVars: []string{fmt.Sprintf("%s_ROUTER_BASEPATH", strings.ToUpper(r.FlagPrefix()))},
			Value:   fmt.Sprintf("api/v1/%s", strings.ToLower(r.FlagPrefix())),
		},
	}
}

func (h *Handler) Health(context context.Context) error {
	return nil
}

func (h *Handler) WrapHandlerFunc(ctx context.Context, req server.Request, rsp interface{}) error {
	return nil
}

func (h *Handler) Add(routes ...*Route) {
	for _, r := range routes {
		// NewRoute returns nil if no Endpoint has been specified, ignore these here
		if r == nil {
			continue
		}

		h.routes = append(h.routes, &routerclientpb.RoutesReply_Route{
			IsGlobal:          r.IsGlobal,
			Method:            r.Method,
			Path:              r.Path,
			Endpoint:          util.ReflectFunctionName(r.Endpoint),
			Params:            r.Params,
			AuthRequired:      r.AuthRequired,
			RatelimitClientIP: r.RatelimitClientIP,
			RatelimitUser:     r.RatelimitUser,
		})
	}
}

// RegisterWithServer registers this Handler with a server
func (h *Handler) RegisterWithServer(s server.Server) {
	routerclientpb.RegisterRouterClientServiceHandler(s, h)
}

// Routes returns the registered routes
func (h *Handler) Routes(ctx context.Context, req *emptypb.Empty, rsp *routerclientpb.RoutesReply) error {
	rsp.RouterURI = h.routerURI
	rsp.Routes = h.routes

	return nil
}
