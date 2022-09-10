package router

import (
	"context"

	"git.jochum.dev/jo-micro/router/internal/proto/routerclientpb"
	"git.jochum.dev/jo-micro/router/internal/util"
	"go-micro.dev/v4/server"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Handler is the handler for git.jochum.dev/jo-micro/router/proto/routerpb.RrouterService
type Handler struct {
	routerURI string
	routes    []*routerclientpb.RoutesReply_Route
}

// NewHandler returns a new dynrouterpb Handler
func NewHandler(routerURI string, routes ...Route) *Handler {
	pbRoutes := []*routerclientpb.RoutesReply_Route{}
	for _, r := range routes {
		pbRoutes = append(pbRoutes, &routerclientpb.RoutesReply_Route{
			IsGlobal: r.IsGlobal,
			Method:   r.Method,
			Path:     r.Path,
			Endpoint: util.ReflectFunctionName(r.Endpoint),
			Params:   r.Params,
		})
	}

	return &Handler{routerURI, pbRoutes}
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
