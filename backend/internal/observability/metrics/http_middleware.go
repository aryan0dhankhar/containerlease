package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// HTTPMetricsMiddleware instruments requests with Prometheus metrics
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		dur := time.Since(start)
		ObserveHTTPRequest(r.Method, r.URL.Path, strconv.Itoa(ww.status), dur)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
