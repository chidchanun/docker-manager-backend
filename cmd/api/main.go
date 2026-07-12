package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"docker-manager-backend/internal/audit"
	"docker-manager-backend/internal/auth"
	"docker-manager-backend/internal/config"
	"docker-manager-backend/internal/dockerclient"
	"docker-manager-backend/internal/handler"
	appmetrics "docker-manager-backend/internal/metrics"
	"docker-manager-backend/internal/middleware"
	"docker-manager-backend/internal/response"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	dockerClient, err := dockerclient.New()
	if err != nil {
		log.Fatalf("Docker connection error: %v", err)
	}
	defer dockerClient.Close()

	sessionManager := auth.NewSessionManager(
		appConfig.SessionTTL,
		appConfig.CookieSecure,
		appConfig.RedisAddress,
	)
	defer sessionManager.Close()
	auditStore := audit.New(appConfig.RedisAddress)
	defer auditStore.Close()

	authHandler := handler.NewAuthHandler(
		appConfig.AdminEmail,
		appConfig.AdminPasswordHash,
		sessionManager,
	)

	dockerHandler := handler.NewDockerHandler(
		dockerClient,
	)

	authMiddleware := middleware.NewAuthMiddleware(
		sessionManager,
	)
	auditMiddleware := middleware.NewAuditMiddleware(auditStore)
	auditHandler := handler.NewAuditHandler(auditStore)
	loginLimiter := middleware.NewLoginRateLimiter(10, 15*time.Minute)

	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc(
		"GET /api/health",
		healthHandler,
	)

	mux.Handle(
		"POST /api/auth/login",
		loginLimiter.Limit(http.HandlerFunc(authHandler.Login)),
	)

	// Helper สำหรับ protected routes
	protected := func(
		pattern string,
		handlerFunc func(
			http.ResponseWriter,
			*http.Request,
		),
	) {
		mux.Handle(
			pattern,
			authMiddleware.Require(auditMiddleware.Record(
				http.HandlerFunc(handlerFunc),
			)),
		)
	}

	// Authenticated user routes
	protected(
		"GET /api/auth/me",
		authHandler.Me,
	)
	protected("GET /api/audit", auditHandler.List)

	protected(
		"POST /api/auth/logout",
		authHandler.Logout,
	)

	// Docker routes
	protected(
		"GET /api/docker/info",
		dockerHandler.Info,
	)

	protected(
		"GET /api/containers",
		dockerHandler.ListContainers,
	)

	protected(
		"GET /api/containers/stats",
		dockerHandler.ContainerStats,
	)

	protected(
		"GET /api/containers/{id}",
		dockerHandler.ContainerDetail,
	)

	protected(
		"GET /api/containers/{id}/logs",
		dockerHandler.ContainerLogs,
	)
	protected("GET /api/containers/{id}/logs/stream", dockerHandler.ContainerLogsStream)

	protected(
		"POST /api/containers/{id}/start",
		dockerHandler.StartContainer,
	)

	protected(
		"POST /api/containers/{id}/stop",
		dockerHandler.StopContainer,
	)

	protected(
		"POST /api/containers/{id}/restart",
		dockerHandler.RestartContainer,
	)
	protected("POST /api/containers/{id}/pause", dockerHandler.PauseContainer)
	protected("POST /api/containers/{id}/unpause", dockerHandler.UnpauseContainer)
	protected("POST /api/containers/{id}/kill", dockerHandler.KillContainer)
	protected("DELETE /api/containers/{id}", dockerHandler.RemoveContainer)
	protected("PATCH /api/containers/{id}/policy", dockerHandler.UpdateContainerPolicy)

	server := &http.Server{
		Addr:              appConfig.Address,
		Handler:           appmetrics.HTTP(middleware.SecurityHeaders(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      75 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	metricsServer := &http.Server{Addr: appConfig.MetricsAddress, Handler: promhttp.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("metrics server error: %v", err)
		}
	}()

	go func() {
		log.Printf(
			"Docker Manager API running at http://%s",
			appConfig.Address,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)

	signal.Notify(
		shutdownSignal,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-shutdownSignal

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	_ = metricsServer.Shutdown(ctx)

	log.Println("Server stopped")
}

func healthHandler(
	w http.ResponseWriter,
	_ *http.Request,
) {
	response.JSON(
		w,
		http.StatusOK,
		HealthResponse{
			Status:    "ok",
			Service:   "docker-manager-backend",
			Timestamp: time.Now().UTC(),
		},
	)
}
