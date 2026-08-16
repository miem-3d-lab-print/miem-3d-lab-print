package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubHealthChecker struct {
	err error
}

func (s stubHealthChecker) PingContext(context.Context) error {
	return s.err
}

func TestReady(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		checkerErr error
		wantStatus int
	}{
		{name: "ready", method: http.MethodGet, wantStatus: http.StatusOK},
		{name: "ready head", method: http.MethodHead, wantStatus: http.StatusOK},
		{name: "database unavailable", method: http.MethodGet, checkerErr: errors.New("offline"), wantStatus: http.StatusServiceUnavailable},
		{name: "method not allowed", method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, "/api/health", nil)
			response := httptest.NewRecorder()

			Ready(stubHealthChecker{err: tt.checkerErr}).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
		})
	}
}
