package services

import (
	"archive/zip"
	"bytes"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func fileHeader(t *testing.T, filename string, data []byte) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest("POST", "/", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = request.MultipartForm.RemoveAll() })
	return request.MultipartForm.File["file"][0]
}

func zipFile(t *testing.T, filename string, data []byte) []byte {
	t.Helper()

	body := &bytes.Buffer{}
	writer := zip.NewWriter(body)
	entry, err := writer.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestValidateZIPWithModel(t *testing.T) {
	stl := make([]byte, 84)
	header := fileHeader(t, "models.zip", zipFile(t, "part.stl", stl))

	format, err := validateFile(header)
	if err != nil {
		t.Fatalf("validateFile() error = %v", err)
	}
	if format != "ZIP" {
		t.Fatalf("format = %q, want ZIP", format)
	}
}

func TestValidateZIPRejectsArchiveWithoutModel(t *testing.T) {
	header := fileHeader(t, "documents.zip", zipFile(t, "readme.txt", []byte("text")))

	_, err := validateFile(header)
	var formatError *ErrInvalidFileFormat
	if !errors.As(err, &formatError) {
		t.Fatalf("error = %v, want ErrInvalidFileFormat", err)
	}
}

func TestValidateFilesRejectsTooManyFiles(t *testing.T) {
	files := make([]*multipart.FileHeader, maxFilesPerApp+1)
	_, err := (&ApplicationService{}).validateFiles(files)
	var limitError *ErrFilesLimitReached
	if !errors.As(err, &limitError) {
		t.Fatalf("error = %v, want ErrFilesLimitReached", err)
	}
}
