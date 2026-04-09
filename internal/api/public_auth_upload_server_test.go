package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	proxyconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestPublicAuthUploadPageRouteDisabledByDefault(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth-upload.html", nil)
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: got %d want %d; body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestPublicAuthUploadPageRouteEnabled(t *testing.T) {
	server := newTestServer(t)
	server.cfg.RemoteManagement.PublicAuthUpload = proxyconfig.PublicAuthUpload{
		Enabled:   true,
		SecretKey: "upload-secret",
	}
	server.mgmt.SetConfig(server.cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth-upload.html", nil)
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestManagementAuthUploadRouteWorksWithPublicUploadKeyOnly(t *testing.T) {
	server := newTestServer(t)
	server.cfg.RemoteManagement.PublicAuthUpload = proxyconfig.PublicAuthUpload{
		Enabled:   true,
		SecretKey: "upload-secret",
	}
	server.mgmt.SetConfig(server.cfg)
	server.managementRoutesEnabled.Store(true)
	server.registerManagementRoutes()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "alpha.json")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err = part.Write([]byte(`{"type":"codex","email":"alpha@example.com"}`)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v0/management/auth-files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Public-Upload-Key", "upload-secret")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if _, err = os.Stat(filepath.Join(server.cfg.AuthDir, "alpha.json")); err != nil {
		t.Fatalf("expected uploaded file to exist: %v", err)
	}
}
