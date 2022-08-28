package router

import (
	"context"

	"github.com/go-micro/router/proto/routerclientpb"
	"github.com/go-micro/router/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Handler is the handler for github.com/go-micro/router/proto/routerpb.RrouterService
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

// Routes returns the registered routes
func (h *Handler) Routes(ctx context.Context, req *emptypb.Empty, rsp *routerclientpb.RoutesReply) error {
	rsp.RouterURI = h.routerURI
	rsp.Routes = h.routes

	return nil
}
