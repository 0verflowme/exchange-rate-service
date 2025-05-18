package http

import (
	"net/http"
	"time"

	"exchange-rate-service/internal/metrics"
	"exchange-rate-service/pkg/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"fmt"
)

type Router struct {
	handler *Handler
	log     *logger.Logger
	metrics *metrics.Metrics
}

func NewRouter(handler *Handler, log *logger.Logger, metrics *metrics.Metrics) *Router {
	return &Router{
		handler: handler,
		log:     log,
		metrics: metrics,
	}
}

func (r *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		crw := &customResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(crw, req)

		if req.URL.Path != "/metrics" {
			duration := time.Since(start).Seconds()
			r.metrics.HTTPRequestDuration.WithLabelValues(req.URL.Path, req.Method).Observe(duration)
			r.metrics.HTTPRequestsTotal.WithLabelValues(req.URL.Path, req.Method, fmt.Sprint('0'+crw.statusCode/100)+"xx").Inc()
		}

		duration := time.Since(start)
		r.log.Info("HTTP request",
			"method", req.Method,
			"path", req.URL.Path,
			"query", req.URL.RawQuery,
			"status", crw.statusCode,
			"duration", duration,
			"remote_addr", req.RemoteAddr,
			"user_agent", req.UserAgent(),
		)
	})
}

type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (crw *customResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

func (r *Router) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/rates", r.handler.GetLatestRateHandler)
	mux.HandleFunc("/api/v1/convert", r.handler.ConvertCurrencyHandler)
	mux.HandleFunc("/api/v1/historical", r.handler.GetHistoricalRateHandler)
	mux.HandleFunc("/api/v1/historical/range", r.handler.GetHistoricalRatesHandler)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	apiWithMiddleware := r.loggingMiddleware(mux)

	rootMux := http.NewServeMux()

	rootMux.Handle("/", apiWithMiddleware)
	rootMux.Handle("/api/", apiWithMiddleware)

	rootMux.Handle("/metrics", promhttp.Handler())

	return rootMux
}
