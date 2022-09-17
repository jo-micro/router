package router

import (
	"log"
)

type Route struct {
	// isGlobal=True == no prefix route
	IsGlobal bool
	Method   string
	Path     string
	Endpoint interface{}
	Params   []string
}

type Option func(*Route)

func NewRoute(opts ...Option) *Route {
	route := &Route{
		IsGlobal: false,
		Method:   MethodGet,
		Path:     "/",
		Endpoint: nil,
		Params:   []string{},
	}

	for _, o := range opts {
		o(route)
	}

	if route.Endpoint == nil {
		log.Println("router.Endpoint() is a required argument")
		return nil
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
