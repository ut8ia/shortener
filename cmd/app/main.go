package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/ut8ia/shortener/pkg/api"
	"github.com/ut8ia/shortener/pkg/signals"
	"github.com/ut8ia/shortener/pkg/version"
	"time"

	//"github.com/ut8ia/shortener/pkg/api"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func main() {
	initConfig()

	//configure logging
	logger, _ := initZap(viper.GetString("level"))
	defer logger.Sync()
	stdLog := zap.RedirectStdLog(logger)
	defer stdLog()

	// configure http server
	var srvCfg api.Config
	if err := viper.Unmarshal(&srvCfg); err != nil {
		logger.Panic("failed too config http api")
	}

	logger.Info("Starting http API",
		zap.String("version", viper.GetString("version")),
		zap.String("revision", viper.GetString("revision")),
		zap.String("port", srvCfg.Port))

	srv, _ := api.NewServer(&srvCfg, logger)
	stopCh := signals.SetupSignalHandler()
	srv.ListenAndServe(stopCh)

}

func initConfig() {
	// flags definition
	fs := pflag.NewFlagSet("default", pflag.ContinueOnError)
	fs.Int("port", 8000, "HTTP port")
	fs.Int("port-metrics", 0, "metrics port")
	fs.String("config-path", "", "config dir path")
	fs.String("config", "config.yaml", "config file name")
	// timeouts
	fs.Duration("http-server-shutdown-timeout", 5*time.Second, "server graceful shutdown timeout duration")
	//logging
	fs.String("level", "info", "log level debug, info, warn, error, flat or panic")

	err := fs.Parse(os.Args[1:])

	switch {

	case err == pflag.ErrHelp:
		os.Exit(0)
	case err != nil:
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
		fs.PrintDefaults()
		os.Exit(2)
	}

	fs.PrintDefaults()
	viper.BindPFlags(fs)
	hostname, _ := os.Hostname()

	viper.Set("hostname", hostname)
	viper.Set("version", version.VERSION)
	viper.Set("revision", version.REVISION)
	viper.AutomaticEnv()

}

func initZap(loglevel string) (*zap.Logger, error) {

	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)

	switch loglevel {
	case "debug":
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	case "panic":
		level = zap.NewAtomicLevelAt(zap.PanicLevel)
	}

	zapEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	zapConfig := zap.Config{
		Level:       level,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zapEncoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return zapConfig.Build()

}
