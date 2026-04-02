package middleware

import (
	"LickLib/cmd/internal/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())

		// zählen
		metrics.HTTPRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			status,
		).Inc()

		// Latenz
		metrics.HTTPRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
			status,
		).Observe(duration)
	})
}
