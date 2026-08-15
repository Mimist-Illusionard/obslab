package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Mimist-Illusionard/obslab/internal/handlers"
	"github.com/Mimist-Illusionard/obslab/internal/jwt"
	"github.com/Mimist-Illusionard/obslab/internal/middleware"
	"github.com/Mimist-Illusionard/obslab/internal/repository"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {
	r := mux.NewRouter()

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

	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()

	logger := middleware.NewLogger(10,
		"X-Request-ID",
		"abcdefghijklmnopqrstuvwxyz0123456789",
		zapLogger)

	jwtService := jwt.NewJwtGenerator("6sKqRZtGhIOAWIzO6yvPTBxUkD7P3CTIuvFexVtv5Rz")

	auth := middleware.NewAuth(jwtService, zapLogger)

	r.Use(logger.LogMiddleware)

	um := repository.UserMemoryRepository{}
	uh := handlers.NewUserHandler(&um, zapLogger, jwtService)

	pr := repository.NewPostMemoryRepository()
	ph := handlers.NewPostHandler(pr, zapLogger, jwtService)

	api := r.PathPrefix("/api").Subrouter()
	uh.Initialize(api)

	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(auth.AuthMiddleware)
	ph.Initialize(protected)

	port := 8083

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	zapLogger.Info(
		"starting server",
		zap.Int("port", port),
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zapLogger.Fatal("server failed", zap.Error(err))
	}
}
