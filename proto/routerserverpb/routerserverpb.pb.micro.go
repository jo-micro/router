// Code generated by protoc-gen-micro. DO NOT EDIT.
// source: routerserverpb.proto

package routerserverpb

import (
	fmt "fmt"
	proto "google.golang.org/protobuf/proto"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	math "math"
)

import (
	context "context"
	api "go-micro.dev/v4/api"
	client "go-micro.dev/v4/client"
	server "go-micro.dev/v4/server"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Reference imports to suppress errors if they are not otherwise used.
var _ api.Endpoint
var _ context.Context
var _ client.Option
var _ server.Option

// Api Endpoints for RouterServerService service

func NewRouterServerServiceEndpoints() []*api.Endpoint {
	return []*api.Endpoint{}
}

// Client API for RouterServerService service

type RouterServerService interface {
	Routes(ctx context.Context, in *emptypb.Empty, opts ...client.CallOption) (*RoutesReply, error)
}

type routerServerService struct {
	c    client.Client
	name string
}

func NewRouterServerService(name string, c client.Client) RouterServerService {
	return &routerServerService{
		c:    c,
		name: name,
	}
}

func (c *routerServerService) Routes(ctx context.Context, in *emptypb.Empty, opts ...client.CallOption) (*RoutesReply, error) {
	req := c.c.NewRequest(c.name, "RouterServerService.Routes", in)
	out := new(RoutesReply)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for RouterServerService service

type RouterServerServiceHandler interface {
	Routes(context.Context, *emptypb.Empty, *RoutesReply) error
}

func RegisterRouterServerServiceHandler(s server.Server, hdlr RouterServerServiceHandler, opts ...server.HandlerOption) error {
	type routerServerService interface {
		Routes(ctx context.Context, in *emptypb.Empty, out *RoutesReply) error
	}
	type RouterServerService struct {
		routerServerService
	}
	h := &routerServerServiceHandler{hdlr}
	return s.Handle(s.NewHandler(&RouterServerService{h}, opts...))
}

type routerServerServiceHandler struct {
	RouterServerServiceHandler
}

func (h *routerServerServiceHandler) Routes(ctx context.Context, in *emptypb.Empty, out *RoutesReply) error {
	return h.RouterServerServiceHandler.Routes(ctx, in, out)
}
