package router

import "net/http"

type Route struct {
	// isGlobal=True == no prefix route
	IsGlobal bool
	Method   string
	Path     string
	Endpoint interface{}
	Params   []string
}

type RouteOption func(*Route)

func NewRoute(endpoint interface{}, opts ...RouteOption) Route {
	route := Route{
		IsGlobal: false,
		Method:   http.MethodGet,
		Path:     "/",
		Endpoint: endpoint,
		Params:   []string{},
	}

	for _, o := range opts {
		o(&route)
	}

	return route
}

func RouteIsGlobal(n bool) RouteOption {
	return func(o *Route) {
		o.IsGlobal = n
	}
}

func RouteMethod(n string) RouteOption {
	return func(o *Route) {
		o.Method = n
	}
}

func RoutePath(n string) RouteOption {
	return func(o *Route) {
		o.Path = n
	}
}

func RouteEndpoint(n interface{}) RouteOption {
	return func(o *Route) {
		o.Endpoint = n
	}
}

func RouteParams(n []string) RouteOption {
	return func(o *Route) {
		o.Params = n
	}
}
