package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Mimist-Illusionard/obslab/internal/handlers"
	"github.com/Mimist-Illusionard/obslab/internal/jwt"
	"github.com/Mimist-Illusionard/obslab/internal/logs"
	"github.com/Mimist-Illusionard/obslab/internal/metrics"
	"github.com/Mimist-Illusionard/obslab/internal/middleware"
	"github.com/Mimist-Illusionard/obslab/internal/repository"
	"github.com/Mimist-Illusionard/obslab/internal/trace"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var port = flag.Int("port", 8080, "The port app exposed on")

func main() {
	flag.Parse()

	r := mux.NewRouter()
	metric := metrics.New()

	otel.SetErrorHandler(
		otel.ErrorHandlerFunc(func(err error) {
			log.Printf("OTEL ERROR: %v", err)
		}),
	)

	tr, err := trace.New(context.Background())
	if err != nil {
		log.Fatalf("tracer create: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := tr.Shutdown(ctx); err != nil {
			log.Printf("shutdown tracing: %v", err)
		}
	}()

	prometheus.MustRegister(metric.RequestsTotal, metric.RequestDuration)
	r.Use(otelmux.Middleware("redditclone"))

	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/html/index.html")
	}).Methods(http.MethodGet)

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))),
	)

	r.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/html/manifest.json")
	}).Methods(http.MethodGet)

	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/html/favicon.ico")
	}).Methods(http.MethodGet)

	zapLogger, _ := logs.NewLogger(logs.Zap)

	logger := middleware.NewLogger(10,
		"X-Request-ID",
		"abcdefghijklmnopqrstuvwxyz0123456789",
		zapLogger, metric)

	jwtService := jwt.NewJwtGenerator("6sKqRZtGhIOAWIzO6yvPTBxUkD7P3CTIuvFexVtv5Rz")

	auth := middleware.NewAuth(jwtService, zapLogger)

	r.Use(logger.LogMiddleware)

	um := repository.UserMemoryRepository{}
	uh := handlers.NewUserHandler(&um, jwtService)

	pr := repository.NewPostMemoryRepository()
	ph := handlers.NewPostHandler(pr, jwtService)

	api := r.PathPrefix("/api").Subrouter()
	uh.Initialize(api)

	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(auth.AuthMiddleware)
	ph.Initialize(protected)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	zapLogger.Info(
		"starting server",
		zap.Int("port", *port),
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zapLogger.Fatal("server failed", zap.Error(err))
	}
}
