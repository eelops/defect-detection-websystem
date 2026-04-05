package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeSeverity(t *testing.T) {
	cases := map[string]string{
		"low":        "low",
		"高":          "high",
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
	resp := performRequest(t, app, http.MethodGet, "/", "", nil)

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

func TestListDefectsRequiresAuth(t *testing.T) {
	app := newTestApp(t)
	resp := performRequest(t, app, http.MethodGet, "/api/defects", "", nil)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/defects status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestLoginAndMeSuccess(t *testing.T) {
	app := newTestApp(t)
	loginBody := `{"username":"admin","password":"Admin@123456"}`
	resp := performRequest(t, app, http.MethodPost, "/api/auth/login", loginBody, map[string]string{"Content-Type": "application/json"})
	if resp.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Token string `json:"token"`
		User  User   `json:"user"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload.Token == "" || payload.User.Role != "admin" {
		t.Fatalf("unexpected login response: %+v", payload)
	}

	me := performRequest(t, app, http.MethodGet, "/api/auth/me", "", map[string]string{
		"Authorization": "Bearer " + payload.Token,
	})
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", me.Code, me.Body.String())
	}
}

func TestViewerCannotCreateOrDeleteDefects(t *testing.T) {
	app := newTestApp(t)
	viewer := createTestUser(t, app, "viewer_seed_test", "查看员", "viewer")
	seedDefect(t, app)

	createResp := performRequest(t, app, http.MethodPost, "/api/defects", "", map[string]string{
		"Authorization": bearerForUser(t, app, viewer.Username),
		"Content-Type":  "multipart/form-data",
	})
	if createResp.Code != http.StatusForbidden {
		t.Fatalf("viewer create status = %d, want %d", createResp.Code, http.StatusForbidden)
	}

	deleteResp := performRequest(t, app, http.MethodDelete, "/api/defects/1", "", map[string]string{
		"Authorization": bearerForUser(t, app, viewer.Username),
	})
	if deleteResp.Code != http.StatusForbidden {
		t.Fatalf("viewer delete status = %d, want %d", deleteResp.Code, http.StatusForbidden)
	}
}

func TestInspectorCanUpdateButCannotDeleteDefect(t *testing.T) {
	app := newTestApp(t)
	inspector := createTestUser(t, app, "inspector_seed_test", "质检员", "inspector")
	defect := seedDefect(t, app)

	updateResp := performRequest(t, app, http.MethodPut, fmt.Sprintf("/api/defects/%d", defect.ID), `{"description":"已补充处理说明"}`, map[string]string{
		"Authorization": bearerForUser(t, app, inspector.Username),
		"Content-Type":  "application/json",
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("inspector update status = %d, body = %s", updateResp.Code, updateResp.Body.String())
	}

	deleteResp := performRequest(t, app, http.MethodDelete, fmt.Sprintf("/api/defects/%d", defect.ID), "", map[string]string{
		"Authorization": bearerForUser(t, app, inspector.Username),
	})
	if deleteResp.Code != http.StatusForbidden {
		t.Fatalf("inspector delete status = %d, want %d", deleteResp.Code, http.StatusForbidden)
	}
}

func TestAdminCanManageUsers(t *testing.T) {
	app := newTestApp(t)
	adminAuth := map[string]string{
		"Authorization": bearerForUser(t, app, "admin"),
		"Content-Type":  "application/json",
	}
	createResp := performRequest(t, app, http.MethodPost, "/api/users", `{"username":"u1","password":"Pass@123456","display_name":"用户1","role":"viewer"}`, adminAuth)
	if createResp.Code != http.StatusOK {
		t.Fatalf("create user status = %d, body = %s", createResp.Code, createResp.Body.String())
	}

	listResp := performRequest(t, app, http.MethodGet, "/api/users", "", map[string]string{
		"Authorization": adminAuth["Authorization"],
	})
	if listResp.Code != http.StatusOK {
		t.Fatalf("list users status = %d", listResp.Code)
	}

	updateResp := performRequest(t, app, http.MethodPut, "/api/users/2", `{"role":"inspector","is_active":false}`, adminAuth)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update user status = %d, body = %s", updateResp.Code, updateResp.Body.String())
	}
}

func TestViewerCannotManageUsers(t *testing.T) {
	app := newTestApp(t)
	viewer := createTestUser(t, app, "viewer_manage_test", "查看员2", "viewer")
	resp := performRequest(t, app, http.MethodGet, "/api/users", "", map[string]string{
		"Authorization": bearerForUser(t, app, viewer.Username),
	})
	if resp.Code != http.StatusForbidden {
		t.Fatalf("viewer manage users status = %d, want %d", resp.Code, http.StatusForbidden)
	}
}

func TestLogoutInvalidatesToken(t *testing.T) {
	app := newTestApp(t)
	authHeader := bearerForUser(t, app, "admin")
	logoutResp := performRequest(t, app, http.MethodPost, "/api/auth/logout", "", map[string]string{
		"Authorization": authHeader,
	})
	if logoutResp.Code != http.StatusOK {
		t.Fatalf("logout status = %d", logoutResp.Code)
	}
	meResp := performRequest(t, app, http.MethodGet, "/api/auth/me", "", map[string]string{
		"Authorization": authHeader,
	})
	if meResp.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want %d", meResp.Code, http.StatusUnauthorized)
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
		"X-API-Key":    "secret-key",
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
		"X-API-Key":    "secret-key",
	})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("bad payload status = %d, want %d, body = %s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
}

func TestOptionsIncludesAuthorizationHeader(t *testing.T) {
	app := newTestApp(t)
	resp := performRequest(t, app, http.MethodOptions, "/api/auth/me", "", nil)
	if got := resp.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("allow headers = %q, expected Authorization", got)
	}
}

func TestExportExcelRequiresAuth(t *testing.T) {
	app := newTestApp(t)
	resp := performRequest(t, app, http.MethodGet, "/api/defects/export.xlsx", "", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/defects/export.xlsx status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestExportExcelSuccess(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	_, err := app.db.Exec(`UPDATE defects SET status = ?, assignee_name = ?, updated_at = ? WHERE id = ?`, "assigned", "张三", time.Now().Format(time.RFC3339), defect.ID)
	if err != nil {
		t.Fatalf("update defect: %v", err)
	}

	resp := performRequest(t, app, http.MethodGet, "/api/defects/export.xlsx?severity=high", "", map[string]string{
		"Authorization": bearerForUser(t, app, "admin"),
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/defects/export.xlsx status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "spreadsheetml") {
		t.Fatalf("content-type = %q", got)
	}
	if got := resp.Header().Get("Content-Disposition"); !strings.Contains(got, ".xlsx") {
		t.Fatalf("content-disposition = %q", got)
	}
	if resp.Body.Len() == 0 {
		t.Fatal("expected non-empty excel body")
	}
	if !strings.HasPrefix(resp.Body.String()[:2], "PK") {
		t.Fatalf("expected xlsx zip signature, got %q", resp.Body.String()[:2])
	}
}

func TestExportDefectPDFSuccess(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	resp := performRequest(t, app, http.MethodGet, fmt.Sprintf("/api/defects/%d/export.pdf", defect.ID), "", map[string]string{
		"Authorization": bearerForUser(t, app, "admin"),
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/defects/{id}/export.pdf status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "application/pdf") {
		t.Fatalf("content-type = %q", got)
	}
	if got := resp.Header().Get("Content-Disposition"); !strings.Contains(got, ".pdf") {
		t.Fatalf("content-disposition = %q", got)
	}
	if resp.Body.Len() == 0 {
		t.Fatal("expected non-empty pdf body")
	}
	if !strings.HasPrefix(resp.Body.String()[:4], "%PDF") {
		t.Fatalf("expected pdf signature, got %q", resp.Body.String()[:4])
	}
}

func TestExportDefectPDFNotFound(t *testing.T) {
	app := newTestApp(t)
	resp := performRequest(t, app, http.MethodGet, "/api/defects/999/export.pdf", "", map[string]string{
		"Authorization": bearerForUser(t, app, "admin"),
	})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("GET /api/defects/999/export.pdf status = %d, want %d", resp.Code, http.StatusNotFound)
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
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "Admin@123456")
	t.Setenv("ADMIN_DISPLAY_NAME", "系统管理员")

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

func bearerForUser(t *testing.T, app *App, username string) string {
	t.Helper()
	resp := performRequest(t, app, http.MethodPost, "/api/auth/login", fmt.Sprintf(`{"username":"%s","password":"Pass@123456"}`, username), map[string]string{"Content-Type": "application/json"})
	if username == "admin" {
		resp = performRequest(t, app, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"Admin@123456"}`, map[string]string{"Content-Type": "application/json"})
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("login for %s failed: %d %s", username, resp.Code, resp.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login token: %v", err)
	}
	return "Bearer " + payload.Token
}

func createTestUser(t *testing.T, app *App, username, displayName, role string) *User {
	t.Helper()
	user, err := app.createUser(createUserRequest{
		Username:    username,
		Password:    "Pass@123456",
		DisplayName: displayName,
		Role:        role,
	})
	if err != nil {
		t.Fatalf("createTestUser failed: %v", err)
	}
	return user
}

func seedDefect(t *testing.T, app *App) *Defect {
	t.Helper()
	defect, err := app.insertDefect("测试缺陷", "裂纹", "high", stringPtr("A产线"), stringPtr("待处理"), nil)
	if err != nil {
		t.Fatalf("seedDefect failed: %v", err)
	}
	return defect
}

func TestChangePasswordSuccess(t *testing.T) {
	app := newTestApp(t)
	auth := bearerForUser(t, app, "admin")
	resp := performRequest(t, app, http.MethodPut, "/api/auth/password", `{"current_password":"Admin@123456","new_password":"NewPass@123456"}`, map[string]string{
		"Authorization": auth,
		"Content-Type":  "application/json",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("change password status = %d, body = %s", resp.Code, resp.Body.String())
	}

	loginResp := performRequest(t, app, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"NewPass@123456"}`, map[string]string{"Content-Type": "application/json"})
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login with new password status = %d, body = %s", loginResp.Code, loginResp.Body.String())
	}
}

func TestDisabledUserCannotUseOldSession(t *testing.T) {
	app := newTestApp(t)
	user := createTestUser(t, app, "viewer_session_test", "查看员3", "viewer")
	auth := bearerForUser(t, app, user.Username)
	_, err := app.db.Exec(`UPDATE users SET is_active = 0, updated_at = ? WHERE id = ?`, time.Now().Format(time.RFC3339), user.ID)
	if err != nil {
		t.Fatalf("disable user: %v", err)
	}
	resp := performRequest(t, app, http.MethodGet, "/api/auth/me", "", map[string]string{"Authorization": auth})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("disabled user me status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}
