package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS("https://allowed.example")(next)

	t.Run("allowed origin", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Origin", "https://allowed.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "https://allowed.example" {
			t.Errorf("Access-Control-Allow-Origin = %q", origin)
		}
	})

	t.Run("unknown origin", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Origin", "https://unknown.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty", origin)
		}
	})

	t.Run("preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set("Origin", "https://allowed.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", response.Code, http.StatusNoContent)
		}
	})
}
