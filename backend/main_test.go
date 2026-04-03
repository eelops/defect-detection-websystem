package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeSeverity(t *testing.T) {
	cases := map[string]string{
		"low":      "low",
		"高":         "high",
		" critical ": "critical",
	}

	for input, want := range cases {
		got, err := normalizeSeverity(input)
		if err != nil {
			t.Fatalf("normalizeSeverity(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("normalizeSeverity(%q) = %q, want %q", input, got, want)
		}
	}

	if _, err := normalizeSeverity("unknown"); err == nil {
		t.Fatal("expected error for unsupported severity")
	}
}

func TestSaveBase64ImageRejectsInvalidInput(t *testing.T) {
	app := newTestApp(t)

	if _, err := app.saveBase64Image("not-base64"); err == nil {
		t.Fatal("expected invalid base64 to fail")
	}
}

func TestVerifyExternalAPIKey(t *testing.T) {
	t.Setenv("EXTERNAL_API_KEY", "secret-key")

	req := httptest.NewRequest(http.MethodPost, "/api/external/report", nil)
	req.Header.Set("X-API-Key", "secret-key")
	if err := verifyExternalAPIKey(req); err != nil {
		t.Fatalf("expected key to pass, got error: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/external/report", nil)
	req.Header.Set("X-API-Key", "wrong")
	if err := verifyExternalAPIKey(req); err == nil {
		t.Fatal("expected invalid key to fail")
	}
}

func TestRootHandler(t *testing.T) {
	app := newTestApp(t)
	resp := performRequest(t, app, http.MethodGet, "/", "", map[string]string{})

	if resp.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", resp.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["message"] == "" {
		t.Fatal("expected message in root response")
	}
}

func TestListDefectsEmpty(t *testing.T) {
	app := newTestApp(t)
	resp := performRequest(t, app, http.MethodGet, "/api/defects", "", map[string]string{})

	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/defects status = %d, want %d", resp.Code, http.StatusOK)
	}
	if strings.TrimSpace(resp.Body.String()) != "[]" {
		t.Fatalf("GET /api/defects body = %s, want []", resp.Body.String())
	}
}

func TestExternalReportSuccess(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("EXTERNAL_API_KEY", "secret-key")

	payload := `{
		"image_base64": "` + base64.StdEncoding.EncodeToString([]byte("fake image")) + `",
		"defect_type": "裂纹",
		"severity": "严重",
		"location": "产线A",
		"confidence": 0.98,
		"description": "自动检测上报",
		"source": "模型服务"
	}`

	resp := performRequest(t, app, http.MethodPost, "/api/external/report", payload, map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":   "secret-key",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/external/report status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var defect Defect
	if err := json.Unmarshal(resp.Body.Bytes(), &defect); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if defect.ID == 0 {
		t.Fatal("expected created defect id")
	}
	if defect.Severity != "critical" {
		t.Fatalf("expected normalized severity critical, got %q", defect.Severity)
	}
	if defect.ImagePath == nil || *defect.ImagePath == "" {
		t.Fatal("expected image path to be saved")
	}
	if _, err := os.Stat(filepath.Join(app.uploadDir, filepath.Base(*defect.ImagePath))); err != nil {
		t.Fatalf("expected saved image file: %v", err)
	}
}

func TestExternalReportRejectsBadPayload(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("EXTERNAL_API_KEY", "secret-key")

	resp := performRequest(t, app, http.MethodPost, "/api/external/report", `{"image_base64":"%%%not-base64%%%","defect_type":"裂纹","severity":"high"}`, map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":   "secret-key",
	})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("bad payload status = %d, want %d, body = %s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("ALERT_EMAIL_TO", "")

	root := t.TempDir()
	app, err := newApp(filepath.Join(root, "test.db"), filepath.Join(root, "uploads"))
	if err != nil {
		t.Fatalf("newApp failed: %v", err)
	}
	t.Cleanup(func() {
		_ = app.db.Close()
	})
	return app
}

func performRequest(t *testing.T, app *App, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp := httptest.NewRecorder()
	app.routes().ServeHTTP(resp, req)
	return resp
}
