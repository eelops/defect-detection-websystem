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

func TestAssignableUsersEndpointOnlyReturnsActiveInspectors(t *testing.T) {
	app := newTestApp(t)
	activeInspector := createTestUser(t, app, "assignable_active_test", "可指派质检员", "inspector")
	disabledInspector := createTestUser(t, app, "assignable_disabled_test", "停用质检员", "inspector")
	viewer := createTestUser(t, app, "assignable_viewer_test", "查看员", "viewer")
	if _, err := app.updateUser(disabledInspector.ID, updateUserRequest{IsActive: boolPtr(false)}); err != nil {
		t.Fatalf("disable inspector: %v", err)
	}

	resp := performRequest(t, app, http.MethodGet, "/api/users/assignable", "", map[string]string{
		"Authorization": bearerForUser(t, app, activeInspector.Username),
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("assignable users status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var users []User
	if err := json.Unmarshal(resp.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode users: %v", err)
	}
	foundActive := false
	for _, user := range users {
		if user.Role != "inspector" || !user.IsActive {
			t.Fatalf("unexpected assignable user: %+v", user)
		}
		if user.ID == activeInspector.ID {
			foundActive = true
		}
		if user.ID == disabledInspector.ID || user.ID == viewer.ID {
			t.Fatalf("non-assignable user leaked: %+v", user)
		}
	}
	if !foundActive {
		t.Fatal("expected active inspector to be returned")
	}
}

func TestAssignRequiresActiveInspectorParticipants(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	activeInspector := createTestUser(t, app, "participant_active_test", "有效质检员", "inspector")
	disabledInspector := createTestUser(t, app, "participant_disabled_test", "停用质检员", "inspector")
	viewer := createTestUser(t, app, "participant_viewer_test", "查看员", "viewer")
	if _, err := app.updateUser(disabledInspector.ID, updateUserRequest{IsActive: boolPtr(false)}); err != nil {
		t.Fatalf("disable inspector: %v", err)
	}
	adminAuth := bearerForUser(t, app, "admin")

	cases := []struct {
		name string
		body string
	}{
		{
			name: "viewer assignee",
			body: fmt.Sprintf(`{"action":"assign","assignee_id":%d,"acceptor_id":%d}`, viewer.ID, activeInspector.ID),
		},
		{
			name: "missing acceptor",
			body: fmt.Sprintf(`{"action":"assign","assignee_id":%d}`, activeInspector.ID),
		},
		{
			name: "disabled acceptor",
			body: fmt.Sprintf(`{"action":"assign","assignee_id":%d,"acceptor_id":%d}`, activeInspector.ID, disabledInspector.ID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequest(t, app, http.MethodPost, fmt.Sprintf("/api/defects/%d/actions", defect.ID), tc.body, map[string]string{
				"Authorization": adminAuth,
				"Content-Type":  "application/json",
			})
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("assign status = %d, want %d, body = %s", resp.Code, http.StatusBadRequest, resp.Body.String())
			}
		})
	}
}

func TestAssignedInspectorsCanCompleteDefectFlow(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	assignee := createTestUser(t, app, "flow_assignee_test", "处理质检员", "inspector")
	acceptor := createTestUser(t, app, "flow_acceptor_test", "验收质检员", "inspector")
	adminAuth := bearerForUser(t, app, "admin")
	assigneeAuth := bearerForUser(t, app, assignee.Username)
	acceptorAuth := bearerForUser(t, app, acceptor.Username)

	assignResp := performRequest(t, app, http.MethodPost, fmt.Sprintf("/api/defects/%d/actions", defect.ID), fmt.Sprintf(`{"action":"assign","assignee_id":%d,"acceptor_id":%d}`, assignee.ID, acceptor.ID), map[string]string{
		"Authorization": adminAuth,
		"Content-Type":  "application/json",
	})
	if assignResp.Code != http.StatusOK {
		t.Fatalf("assign status = %d, body = %s", assignResp.Code, assignResp.Body.String())
	}

	startResp := performRequest(t, app, http.MethodPost, fmt.Sprintf("/api/defects/%d/actions", defect.ID), `{"action":"start_process"}`, map[string]string{
		"Authorization": assigneeAuth,
		"Content-Type":  "application/json",
	})
	if startResp.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", startResp.Code, startResp.Body.String())
	}

	submitResp := performRequest(t, app, http.MethodPost, fmt.Sprintf("/api/defects/%d/actions", defect.ID), `{"action":"submit_for_acceptance","process_result":"已完成返工处理"}`, map[string]string{
		"Authorization": assigneeAuth,
		"Content-Type":  "application/json",
	})
	if submitResp.Code != http.StatusOK {
		t.Fatalf("submit status = %d, body = %s", submitResp.Code, submitResp.Body.String())
	}

	acceptResp := performRequest(t, app, http.MethodPost, fmt.Sprintf("/api/defects/%d/actions", defect.ID), `{"action":"accept","acceptance_comment":"验收通过"}`, map[string]string{
		"Authorization": acceptorAuth,
		"Content-Type":  "application/json",
	})
	if acceptResp.Code != http.StatusOK {
		t.Fatalf("accept status = %d, body = %s", acceptResp.Code, acceptResp.Body.String())
	}

	closeResp := performRequest(t, app, http.MethodPost, fmt.Sprintf("/api/defects/%d/actions", defect.ID), `{"action":"close"}`, map[string]string{
		"Authorization": adminAuth,
		"Content-Type":  "application/json",
	})
	if closeResp.Code != http.StatusOK {
		t.Fatalf("close status = %d, body = %s", closeResp.Code, closeResp.Body.String())
	}

	updated, err := app.findDefect(defect.ID)
	if err != nil {
		t.Fatalf("find defect: %v", err)
	}
	if updated.Status != "closed" {
		t.Fatalf("status = %q, want closed", updated.Status)
	}
	var flowLogCount int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM defect_flow_logs WHERE defect_id = ?`, defect.ID).Scan(&flowLogCount); err != nil {
		t.Fatalf("count flow logs: %v", err)
	}
	if flowLogCount != 5 {
		t.Fatalf("flow log count = %d, want 5", flowLogCount)
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

func TestAlertsReturnUnclosedCriticalDefects(t *testing.T) {
	app := newTestApp(t)
	critical, err := app.insertDefect("严重告警缺陷", "裂纹", "critical", stringPtr("A产线-工位3"), nil, nil)
	if err != nil {
		t.Fatalf("insert critical defect: %v", err)
	}
	closed, err := app.insertDefect("已关闭严重缺陷", "裂纹", "critical", stringPtr("A产线-工位4"), nil, nil)
	if err != nil {
		t.Fatalf("insert closed critical defect: %v", err)
	}
	_, err = app.db.Exec(`UPDATE defects SET status = 'closed', updated_at = ? WHERE id = ?`, time.Now().Format(time.RFC3339), closed.ID)
	if err != nil {
		t.Fatalf("close defect: %v", err)
	}
	if _, err := app.insertDefect("高等级但非严重", "裂纹", "high", stringPtr("A产线-工位5"), nil, nil); err != nil {
		t.Fatalf("insert high defect: %v", err)
	}

	resp := performRequest(t, app, http.MethodGet, "/api/alerts", "", map[string]string{
		"Authorization": bearerForUser(t, app, "admin"),
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/alerts status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var alerts []Defect
	if err := json.Unmarshal(resp.Body.Bytes(), &alerts); err != nil {
		t.Fatalf("decode alerts: %v", err)
	}
	foundCritical := false
	for _, alert := range alerts {
		if alert.ID == critical.ID {
			foundCritical = true
		}
		if alert.ID == closed.ID {
			t.Fatalf("closed defect appeared in alerts: %+v", alert)
		}
		if alert.Severity != "critical" {
			t.Fatalf("non-critical defect appeared in alerts: %+v", alert)
		}
		if alert.Status == "closed" {
			t.Fatalf("closed status appeared in alerts: %+v", alert)
		}
	}
	if !foundCritical {
		t.Fatalf("inserted critical defect missing from alerts: %+v", alerts)
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
