package logger

import (
	"fmt"
	"os"
	"runtime"

	microLogrus "github.com/go-micro/plugins/v4/logger/logrus"
	microLogger "go-micro.dev/v4/logger"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var myLogger *logrus.Logger = nil
var initialized = false

func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "loglevel",
			Value:   "info",
			Usage:   "Logrus log level default 'info', {panic,fatal,error,warn,info,debug,trace} available",
			EnvVars: []string{"LOG_LEVEL"},
		},
	}
}

func Intialized() bool {
	return initialized
}

func Start(cli *cli.Context) error {
	if initialized {
		return nil
	}

	lvl, err := logrus.ParseLevel(cli.String("loglevel"))
	if err != nil {
		return err
	}

	myLogger = logrus.New()
	myLogger.Out = os.Stdout
	myLogger.Level = lvl

	microLogger.DefaultLogger = microLogrus.NewLogger(microLogrus.WithLogger(myLogger))

	initialized = true
	return nil
}

func Stop() error {
	initialized = false
	myLogger = nil

	return nil
}

func Logrus() *logrus.Logger {
	return myLogger
}

func WithCaller() *logrus.Entry {
	e := logrus.NewEntry(myLogger)
	_, file, no, ok := runtime.Caller(1)
	if ok {
		e.WithField("caller", fmt.Sprintf("%s:%d", file, no))
	}

	return e
}
