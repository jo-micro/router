package router

import (
	"log"
)

type Route struct {
	IsGlobal     bool // isGlobal=True == no prefix route
	Method       string
	Path         string
	Endpoint     interface{}
	Params       []string
	AuthRequired bool // Default false
	// https://github.com/ulule/limiter - default is no rate Limiter at all, put the strictes limit first
	RatelimitClientIP []string
}

type Option func(*Route)

func NewRoute(opts ...Option) *Route {
	route := &Route{
		IsGlobal:          false,
		Method:            MethodGet,
		Path:              "/",
		Endpoint:          nil,
		Params:            []string{},
		AuthRequired:      false,
		RatelimitClientIP: []string{},
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

func AuthRequired() Option {
	return func(o *Route) {
		o.AuthRequired = true
	}
}

func RatelimitClientIP(n ...string) Option {
	return func(o *Route) {
		o.RatelimitClientIP = n
	}
}
