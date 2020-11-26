package api

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
	"sync/atomic"
	"time"
)

type Config struct {
	Hostname                  string        `mapstrcucture: "hostname"`
	Port                      string        `mapstructure: "port"`
	PortMetrics               int           `mapstructure:"port-metrics"`
	// lifecycle timings
	HttpServerShutdownTimeout time.Duration `mapstructure:"http-server-shutdown-timeout"`
	// k8s markers
	Unhealthy bool `mapstructure:"unhealthy"`
	Unready   bool `mapstructure:"unready"`
}

type Server struct {
	router *mux.Router
	logger *zap.Logger
	config *Config
}

func NewServer(config *Config, logger *zap.Logger) (*Server, error) {
	srv := &Server{
		router: mux.NewRouter(),
		logger: logger,
		config: config,
	}
	return srv, nil
}

func (s *Server) ListenAndServe(stopCh <-chan struct{}) {
	go s.startMetricsServer()

	s.registerHandlers()
	s.registerMiddlewares()

	// create the http server
	srv := s.startServer()

	// signal Kubernetes the server is ready to receive traffic
	if !s.config.Unhealthy {
		atomic.StoreInt32(&healthy, 1)
	}
	if !s.config.Unready {
		atomic.StoreInt32(&ready, 1)
	}

	// wait for SIGTERM or SIGINT
	<-stopCh
	ctx, cancel := context.WithTimeout(context.Background(), s.config.HttpServerShutdownTimeout)
	defer cancel()

	// all calls to /healthz and /readyz will fail from now on
	atomic.StoreInt32(&healthy, 0)
	atomic.StoreInt32(&ready, 0)

	s.logger.Info("Shutting down HTTP/HTTPS server", zap.Duration("timeout", s.config.HttpServerShutdownTimeout))

	// wait for Kubernetes readiness probe to remove this instance from the load balancer
	// the readiness check interval must be lower than the timeout
	if viper.GetString("level") != "debug" {
		time.Sleep(3 * time.Second)
	}

	// determine if the http server was started
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Warn("HTTP server graceful shutdown failed", zap.Error(err))
		}
	}
}

func (s *Server) startMetricsServer() {
	if s.config.PortMetrics > 0 {
		mux := http.DefaultServeMux
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		srv := &http.Server{
			Addr:    fmt.Sprintf(":%v", s.config.PortMetrics),
			Handler: mux,
		}

		srv.ListenAndServe()
	}
}

func (s *Server) registerHandlers() {
	s.router.Handle("/metrics", promhttp.Handler())
	s.router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	s.router.HandleFunc("/", s.indexHandler).HeadersRegexp("User-Agent", "^Mozilla.*").Methods("GET")
	s.router.HandleFunc("/", s.infoHandler).Methods("GET")
	s.router.HandleFunc("/version", s.versionHandler).Methods("GET")
	s.router.HandleFunc("/echo", s.echoHandler).Methods("POST")
	s.router.HandleFunc("/env", s.envHandler).Methods("GET", "POST")
	s.router.HandleFunc("/headers", s.echoHeadersHandler).Methods("GET", "POST")
	s.router.HandleFunc("/delay/{wait:[0-9]+}", s.delayHandler).Methods("GET").Name("delay")
	s.router.HandleFunc("/healthz", s.healthzHandler).Methods("GET")
	s.router.HandleFunc("/readyz", s.readyzHandler).Methods("GET")
	s.router.HandleFunc("/readyz/enable", s.enableReadyHandler).Methods("POST")
	s.router.HandleFunc("/readyz/disable", s.disableReadyHandler).Methods("POST")
	s.router.HandleFunc("/panic", s.panicHandler).Methods("GET")
	s.router.HandleFunc("/status/{code:[0-9]+}", s.statusHandler).Methods("GET", "POST", "PUT").Name("status")
	s.router.HandleFunc("/store", s.storeWriteHandler).Methods("POST", "PUT")
	s.router.HandleFunc("/store/{hash}", s.storeReadHandler).Methods("GET").Name("store")
	s.router.HandleFunc("/cache/{key}", s.cacheWriteHandler).Methods("POST", "PUT")
	s.router.HandleFunc("/cache/{key}", s.cacheDeleteHandler).Methods("DELETE")
	s.router.HandleFunc("/cache/{key}", s.cacheReadHandler).Methods("GET").Name("cache")
	s.router.HandleFunc("/configs", s.configReadHandler).Methods("GET")
	s.router.HandleFunc("/token", s.tokenGenerateHandler).Methods("POST")
	s.router.HandleFunc("/token/validate", s.tokenValidateHandler).Methods("GET")
	s.router.HandleFunc("/api/info", s.infoHandler).Methods("GET")
	s.router.HandleFunc("/api/echo", s.echoHandler).Methods("POST")
	s.router.HandleFunc("/ws/echo", s.echoWsHandler)
	s.router.HandleFunc("/chunked", s.chunkedHandler)
	s.router.HandleFunc("/chunked/{wait:[0-9]+}", s.chunkedHandler)
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	s.router.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		doc, err := swag.ReadDoc()
		if err != nil {
			s.logger.Error("swagger error", zap.Error(err), zap.String("path", "/swagger.json"))
		}
		w.Write([]byte(doc))
	})
}

func (s *Server) registerMiddlewares() {
	prom := NewPrometheusMiddleware()
	s.router.Use(prom.Handler)
	httpLogger := NewLoggingMiddleware(s.logger)
	s.router.Use(httpLogger.Handler)
	s.router.Use(versionMiddleware)
}
