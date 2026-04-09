package management

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const publicAuthUploadPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Public Auth Upload</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f4f7fb;
      --panel: #ffffff;
      --text: #16202a;
      --muted: #5f6b76;
      --accent: #0f766e;
      --accent-hover: #115e59;
      --border: #d7e0e8;
      --danger: #b42318;
      --success: #067647;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.12), transparent 28rem),
        linear-gradient(180deg, #eef4f8 0%, var(--bg) 100%);
      color: var(--text);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .panel {
      width: min(100%, 560px);
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 18px;
      box-shadow: 0 24px 60px rgba(15, 23, 42, 0.10);
      padding: 28px;
    }
    h1 {
      margin: 0 0 10px;
      font-size: 28px;
      line-height: 1.15;
    }
    p {
      margin: 0 0 18px;
      color: var(--muted);
      line-height: 1.6;
    }
    form {
      display: grid;
      gap: 14px;
    }
    label {
      font-size: 14px;
      font-weight: 600;
    }
    input {
      width: 100%;
      margin-top: 8px;
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 12px 14px;
      font: inherit;
      background: #fff;
      color: var(--text);
    }
    button {
      border: 0;
      border-radius: 12px;
      padding: 12px 16px;
      font: inherit;
      font-weight: 700;
      color: #fff;
      background: var(--accent);
      cursor: pointer;
    }
    button:hover { background: var(--accent-hover); }
    button:disabled {
      cursor: wait;
      opacity: 0.75;
    }
    .hint {
      font-size: 13px;
      color: var(--muted);
      margin-top: -6px;
    }
    .status {
      min-height: 24px;
      font-size: 14px;
      font-weight: 600;
    }
    .status.error { color: var(--danger); }
    .status.success { color: var(--success); }
  </style>
</head>
<body>
  <main class="panel">
    <h1>Public Auth Upload</h1>
    <p>Upload auth JSON files without exposing the full management key. This page accepts only the dedicated upload key.</p>
    <form id="upload-form">
      <label>
        Upload key
        <input id="upload-key" type="password" autocomplete="current-password" placeholder="Enter upload-only key" required>
      </label>
      <label>
        Auth JSON files
        <input id="auth-files" type="file" accept=".json,application/json" multiple required>
      </label>
      <div class="hint">This page only submits to POST /v0/management/auth-files and cannot access cleanup or other management actions.</div>
      <button id="submit-button" type="submit">Upload</button>
      <div id="status" class="status" aria-live="polite"></div>
    </form>
  </main>
  <script>
    const form = document.getElementById("upload-form");
    const button = document.getElementById("submit-button");
    const statusBox = document.getElementById("status");
    const keyInput = document.getElementById("upload-key");
    const filesInput = document.getElementById("auth-files");

    function setStatus(message, type) {
      statusBox.textContent = message || "";
      statusBox.className = "status" + (type ? " " + type : "");
    }

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const files = filesInput.files;
      if (!files || files.length === 0) {
        setStatus("Choose at least one JSON file.", "error");
        return;
      }

      button.disabled = true;
      setStatus("Uploading...", "");

      const formData = new FormData();
      for (const file of files) {
        formData.append("file", file, file.name);
      }

      try {
        const response = await fetch("/v0/management/auth-files", {
          method: "POST",
          headers: {
            "X-Public-Upload-Key": keyInput.value
          },
          body: formData
        });
        const payload = await response.json().catch(() => ({}));
        if (!response.ok && response.status !== 207) {
          const message = payload.error || "Upload failed.";
          setStatus(message, "error");
          return;
        }
        if (response.status === 207) {
          const failed = Array.isArray(payload.failed) ? payload.failed.length : 0;
          const uploaded = typeof payload.uploaded === "number" ? payload.uploaded : 0;
          setStatus("Uploaded " + uploaded + " file(s), " + failed + " failed.", "error");
          return;
        }
        const uploaded = typeof payload.uploaded === "number" ? payload.uploaded : files.length;
        setStatus("Uploaded " + uploaded + " file(s) successfully.", "success");
        form.reset();
      } catch (_error) {
        setStatus("Network error while uploading.", "error");
      } finally {
        button.disabled = false;
      }
    });
  </script>
</body>
</html>
`

func compareProtectedSecret(provided, configured string) bool {
	provided = strings.TrimSpace(provided)
	configured = strings.TrimSpace(configured)
	if provided == "" || configured == "" {
		return false
	}
	if strings.HasPrefix(configured, "$2a$") || strings.HasPrefix(configured, "$2b$") || strings.HasPrefix(configured, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(configured), []byte(provided)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) == 1
}

func managementKeyFromRequest(c *gin.Context) string {
	if c == nil {
		return ""
	}
	var provided string
	if ah := strings.TrimSpace(c.GetHeader("Authorization")); ah != "" {
		parts := strings.SplitN(ah, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			provided = parts[1]
		} else {
			provided = ah
		}
	}
	if provided == "" {
		provided = strings.TrimSpace(c.GetHeader("X-Management-Key"))
	}
	if provided == "" {
		provided = strings.TrimSpace(c.GetHeader("X-Public-Upload-Key"))
	}
	return provided
}

func (h *Handler) publicAuthUploadSecret() string {
	if h == nil || h.cfg == nil {
		return ""
	}
	return strings.TrimSpace(h.cfg.RemoteManagement.PublicAuthUpload.SecretKey)
}

func (h *Handler) publicAuthUploadEnabled() bool {
	if h == nil || h.cfg == nil {
		return false
	}
	cfg := h.cfg.RemoteManagement.PublicAuthUpload
	return cfg.Enabled && strings.TrimSpace(cfg.SecretKey) != ""
}

// ServePublicAuthUploadPage renders the upload-only public auth page.
func (h *Handler) ServePublicAuthUploadPage(c *gin.Context) {
	if !h.publicAuthUploadEnabled() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(publicAuthUploadPageHTML))
}
