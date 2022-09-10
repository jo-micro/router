package config

var (
	Version = "0.0.1-dev0"
)

const (
	Name    = "go.micro.router"
	PkgPath = "jochum.dev/jo-micro/router"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

type Config struct {
	Router RouterConfig
	Auth   AuthConfig
}

type RouterConfig struct {
	Env            string
	Address        string
	RouterURI      string
	RefreshSeconds int
}

type TokenKeys struct {
	PubKey string
}

type AuthConfig struct {
	AccessToken  TokenKeys
	RefreshToken TokenKeys
}

func GetConfig() *Config {
	return &_cfg
}

func GetRouterConfig() RouterConfig {
	return _cfg.Router
}

func GetAuthConfig() AuthConfig {
	return _cfg.Auth
}

// internal instance of Config
var _cfg = Config{
	Router: RouterConfig{
		Env:            EnvProd,
		Address:        ":8080",
		RouterURI:      "router",
		RefreshSeconds: 10,
	},
	Auth: AuthConfig{
		AccessToken: TokenKeys{
			PubKey: "",
		},
		RefreshToken: TokenKeys{
			PubKey: "",
		},
	},
}
