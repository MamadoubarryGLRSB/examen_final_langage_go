package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type logCtxKey struct{}

// pour récupérer le status code dans les logs
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func withLogBatchID(r *http.Request) (*http.Request, *string) {
	var id string
	ctx := context.WithValue(r.Context(), logCtxKey{}, &id)
	return r.WithContext(ctx), &id
}

func setLogBatchID(r *http.Request, batchID string) {
	if p, ok := r.Context().Value(logCtxKey{}).(*string); ok {
		*p = batchID
	}
}

func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// pas de log sur healthz
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		r, batchID := withLogBatchID(r)
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rw, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if *batchID != "" {
			attrs = append(attrs, "batch_id", *batchID)
		}

		logger.Info("request", attrs...)
	})
}

func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"path", r.URL.Path,
					"panic", fmt.Sprintf("%v", rec),
				)
				writeError(w, http.StatusInternalServerError, "internal", "erreur interne du serveur")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
