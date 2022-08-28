package config

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/go-micro/plugins/v4/config/encoder/toml"
	"github.com/go-micro/plugins/v4/config/encoder/yaml"
	"github.com/pkg/errors"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/reader"
	"go-micro.dev/v4/config/reader/json"
	"go-micro.dev/v4/config/source/env"
	"go-micro.dev/v4/config/source/file"
	"go-micro.dev/v4/logger"
)

// internal instance of Config
var _cfg *Config = &Config{
	Server: ServerConfig{
		Env:            EnvProd,
		Address:        ":8080",
		RefreshSeconds: 10,
	},
}

// goSafe will run func in goroutine safely, avoid crash from unexpected panic
func goSafe(fn func()) {
	if fn == nil {
		return
	}
	go func() {
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("[panic]%v\n%s", e, debug.Stack())
			}
		}()
		fn()
	}()
}

// Load will load configurations and update it when changed
func Load() error {
	var configor config.Config
	var err error
	switch strings.ToLower(os.Getenv("CONFIG_TYPE")) {
	case "toml":
		filename := "config.toml"
		if name := os.Getenv("CONFIG_FILE"); len(name) > 0 {
			filename = name
		}
		configor, err = config.NewConfig(
			config.WithSource(file.NewSource(file.WithPath(filename))),
			config.WithReader(json.NewReader(reader.WithEncoder(toml.NewEncoder()))),
		)
	case "yaml":
		filename := "config.yaml"
		if name := os.Getenv("CONFIG_FILE"); len(name) > 0 {
			filename = name
		}
		configor, err = config.NewConfig(
			config.WithSource(file.NewSource(file.WithPath(filename))),
			config.WithReader(json.NewReader(reader.WithEncoder(yaml.NewEncoder()))),
		)
	default:
		configor, err = config.NewConfig(
			config.WithSource(env.NewSource()),
		)
	}
	if err != nil {
		return errors.Wrap(err, "configor.New")
	}
	if err := configor.Load(); err != nil {
		return errors.Wrap(err, "configor.Load")
	}
	if err := configor.Scan(_cfg); err != nil {
		return errors.Wrap(err, "configor.Scan")
	}
	w, err := configor.Watch()
	if err != nil {
		return errors.Wrap(err, "configor.Watch")
	}
	goSafe(func() {
		for {
			v, err := w.Next()
			if err != nil {
				logger.Error(err)
				return
			}
			if err := v.Scan(_cfg); err != nil {
				logger.Error(err)
				return
			}
		}
	})
	return nil
}
