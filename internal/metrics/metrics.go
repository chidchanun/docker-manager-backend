package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	requests         = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "docker_manager_http_requests_total", Help: "HTTP requests"}, []string{"method", "route", "status"})
	duration         = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "docker_manager_http_request_duration_seconds", Help: "HTTP latency"}, []string{"method", "route"})
	LoginFailures    = prometheus.NewCounter(prometheus.CounterOpts{Name: "docker_manager_login_failures_total", Help: "Failed logins"})
	ContainerActions = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "docker_manager_container_actions_total", Help: "Container actions"}, []string{"action", "result"})
	DockerErrors     = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "docker_manager_docker_errors_total", Help: "Docker API errors"}, []string{"operation"})
	CPU              = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "docker_manager_container_cpu_percent", Help: "Container CPU"}, []string{"id", "name"})
	Memory           = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "docker_manager_container_memory_bytes", Help: "Container memory"}, []string{"id", "name"})
	NetworkRX        = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "docker_manager_container_network_receive_bytes", Help: "Container RX"}, []string{"id", "name"})
	NetworkTX        = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "docker_manager_container_network_transmit_bytes", Help: "Container TX"}, []string{"id", "name"})
)

func init() {
	prometheus.MustRegister(requests, duration, LoginFailures, ContainerActions, DockerErrors, CPU, Memory, NetworkRX, NetworkTX)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		route := r.Pattern
		if route == "" {
			route = "unknown"
		}
		requests.WithLabelValues(r.Method, route, strconv.Itoa(wrapped.status)).Inc()
		duration.WithLabelValues(r.Method, route).Observe(time.Since(started).Seconds())
	})
}
