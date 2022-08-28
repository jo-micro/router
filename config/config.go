package config

const (
	Name      = "go.micro.router"
	Version   = "0.0.1-dev0"
	RouterURI = "router"
	PkgPath   = "github.com/go-micro/router"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	Env            string
	Address        string
	RefreshSeconds int
}

func GetConfig() Config {
	return *_cfg
}

func GetServerConfig() ServerConfig {
	return _cfg.Server
}
