package handlers

import (
	"context"
	"net/http"
)

type HealthChecker interface {
	PingContext(context.Context) error
}

func Live() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "miem-3d-lab-print", "status": "alive"})
	})
}

// Ready returns a readiness endpoint backed by a database connectivity check.
func Ready(checker HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"status": "method_not_allowed"})
			return
		}

		if err := checker.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"service": "miem-3d-lab-print", "status": "not_ready",
			})
			return
		}

		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"service": "miem-3d-lab-print", "status": "ready"})
	})
}
