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

type Option func(*Route)

func NewRoute(endpoint interface{}, opts ...Option) Route {
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

func IsGlobal(n bool) Option {
	return func(o *Route) {
		o.IsGlobal = n
	}
}

func Method(n string) Option {
	return func(o *Route) {
		o.Method = n
	}
}

func Path(n string) Option {
	return func(o *Route) {
		o.Path = n
	}
}

func Endpoint(n interface{}) Option {
	return func(o *Route) {
		o.Endpoint = n
	}
}

func Params(n ...string) Option {
	return func(o *Route) {
		o.Params = n
	}
}
