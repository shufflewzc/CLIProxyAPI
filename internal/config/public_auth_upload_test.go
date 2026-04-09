package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigOptional_HashesPublicAuthUploadSecret(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := `
remote-management:
  public-auth-upload:
    enabled: true
    secret-key: plain-upload-secret
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigOptional(configPath, false)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}
	if !cfg.RemoteManagement.PublicAuthUpload.Enabled {
		t.Fatalf("expected public auth upload to stay enabled")
	}
	if got := cfg.RemoteManagement.PublicAuthUpload.SecretKey; got == "" || got == "plain-upload-secret" {
		t.Fatalf("expected hashed public auth upload key, got %q", got)
	}
	if !strings.HasPrefix(cfg.RemoteManagement.PublicAuthUpload.SecretKey, "$2") {
		t.Fatalf("expected bcrypt hash, got %q", cfg.RemoteManagement.PublicAuthUpload.SecretKey)
	}
}
