package config

var (
	Version = "0.0.1-dev0"
)

const (
	Name    = "go.micro.router"
	PkgPath = "github.com/go-micro/router"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

type Config struct {
	Router RouterConfig
}

type RouterConfig struct {
	Env            string
	Address        string
	RouterURI      string
	RefreshSeconds int
}

func GetConfig() Config {
	return *_cfg
}

func GetRouterConfig() RouterConfig {
	return _cfg.Router
}
