package management

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func newPublicUploadHandler(t *testing.T) *Handler {
	t.Helper()
	t.Setenv("MANAGEMENT_PASSWORD", "admin-secret")
	gin.SetMode(gin.TestMode)

	authDir := t.TempDir()
	manager := coreauth.NewManager(nil, nil, nil)
	return NewHandlerWithoutConfigFilePath(&config.Config{
		AuthDir: authDir,
		RemoteManagement: config.RemoteManagement{
			PublicAuthUpload: config.PublicAuthUpload{
				Enabled:   true,
				SecretKey: "upload-secret",
			},
		},
	}, manager)
}

func TestServePublicAuthUploadPage(t *testing.T) {
	h := newPublicUploadHandler(t)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/auth-upload.html", nil)

	h.ServePublicAuthUploadPage(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Public Auth Upload") {
		t.Fatalf("expected public upload page content, got %s", rec.Body.String())
	}
}

func TestPublicAuthUploadKeyCanUploadViaManagementEndpoint(t *testing.T) {
	h := newPublicUploadHandler(t)

	engine := gin.New()
	engine.GET("/auth-upload.html", h.ServePublicAuthUploadPage)
	group := engine.Group("/v0/management")
	group.Use(h.Middleware())
	group.POST("/auth-files", h.UploadAuthFile)

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
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if _, err = os.Stat(filepath.Join(h.cfg.AuthDir, "alpha.json")); err != nil {
		t.Fatalf("expected uploaded file to exist: %v", err)
	}
}

func TestPublicAuthUploadKeyCannotDeleteViaManagementEndpoint(t *testing.T) {
	h := newPublicUploadHandler(t)

	engine := gin.New()
	group := engine.Group("/v0/management")
	group.Use(h.Middleware())
	group.DELETE("/auth-files", h.DeleteAuthFile)

	req := httptest.NewRequest(http.MethodDelete, "/v0/management/auth-files?name=alpha.json", nil)
	req.Header.Set("Authorization", "Bearer upload-secret")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestPublicAuthUploadRequiresKey(t *testing.T) {
	h := newPublicUploadHandler(t)

	engine := gin.New()
	group := engine.Group("/v0/management")
	group.Use(h.Middleware())
	group.POST("/auth-files", h.UploadAuthFile)

	req := httptest.NewRequest(http.MethodPost, "/v0/management/auth-files", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}
