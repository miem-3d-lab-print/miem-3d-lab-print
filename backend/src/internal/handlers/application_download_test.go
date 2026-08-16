package handlers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

func TestWriteDownloadStreamsFile(t *testing.T) {
	t.Parallel()

	handler := &ApplicationHandler{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	recorder := httptest.NewRecorder()
	handler.writeDownload(recorder, &services.FileDownload{
		Reader:   io.NopCloser(strings.NewReader("3d-model")),
		Filename: "модель.stl",
		Size:     len("3d-model"),
	})

	response := recorder.Result()
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, "filename*") {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if got := recorder.Body.String(); got != "3d-model" {
		t.Fatalf("body = %q", got)
	}
}
