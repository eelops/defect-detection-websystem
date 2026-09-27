package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProductionBootstrapDoesNotSeedDemoData(t *testing.T) {
	app := newTestApp(t)
	for table, want := range map[string]int{"users": 1, "defects": 0, "production_lines": 0} {
		var count int
		if err := app.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
	t.Setenv("ADMIN_PASSWORD", "")
	root := t.TempDir()
	if other, err := newApp(filepath.Join(root, "test.db"), filepath.Join(root, "uploads")); err == nil {
		other.db.Close()
		t.Fatal("bootstrap accepted missing password")
	}
}

func TestPasswordChangesRevokeAllSessions(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(map[bool]string{false: "self", true: "admin"}[reset], func(t *testing.T) {
			app := newTestApp(t)
			user := createTestUser(t, app, "password_test", "测试员", "inspector")
			tokens := []string{bearerForUser(t, app, user.Username), bearerForUser(t, app, user.Username)}
			if reset {
				if _, err := app.updateUser(user.ID, updateUserRequest{Password: stringPtr("NewPass@123456")}); err != nil {
					t.Fatal(err)
				}
			} else {
				resp := performRequest(t, app, http.MethodPut, "/api/auth/password", `{"current_password":"Pass@123456","new_password":"NewPass@123456"}`, map[string]string{"Authorization": tokens[0]})
				if resp.Code != http.StatusOK {
					t.Fatalf("change: %d %s", resp.Code, resp.Body.String())
				}
			}
			for _, token := range tokens {
				resp := performRequest(t, app, http.MethodGet, "/api/auth/me", "", map[string]string{"Authorization": token})
				if resp.Code != http.StatusUnauthorized {
					t.Fatalf("old session still works: %d", resp.Code)
				}
			}
			resp := performRequest(t, app, http.MethodPost, "/api/auth/login", `{"username":"password_test","password":"NewPass@123456"}`, nil)
			if resp.Code != http.StatusOK {
				t.Fatalf("new password: %d", resp.Code)
			}
		})
	}
}

func TestImageValidationAndCanonicalExtension(t *testing.T) {
	app := newTestApp(t)
	valid := testPNG(t)
	for name, data := range map[string][]byte{"html": []byte("<html>not an image</html>"), "truncated": valid[:len(valid)/2], "oversize": make([]byte, maxImageBytes+1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := app.saveUploadedFile(bytes.NewReader(data), "image.png"); err == nil {
				t.Fatal("invalid image accepted")
			}
		})
	}
	files, err := os.ReadDir(app.uploadDir)
	if err != nil || len(files) != 0 {
		t.Fatalf("invalid upload left files: %v, %d", err, len(files))
	}
	path, err := app.saveUploadedFile(bytes.NewReader(valid), "malicious.html")
	if err != nil || filepath.Ext(path) != ".png" {
		t.Fatalf("canonical image: %q %v", path, err)
	}
	if _, err := app.saveBase64Image(base64.StdEncoding.EncodeToString([]byte("fake image"))); err == nil {
		t.Fatal("base64 bypassed validation")
	}
}

func TestUploadEndpointValidation(t *testing.T) {
	app := newTestApp(t)
	auth := bearerForUser(t, app, "admin")
	for _, valid := range []bool{false, true} {
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		for key, value := range map[string]string{"name": "测试上传", "defect_type": "裂纹", "severity": "low"} {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatal(err)
			}
		}
		file, err := writer.CreateFormFile("image", "photo.html")
		if err != nil {
			t.Fatal(err)
		}
		data := []byte("<html>bad</html>")
		if valid {
			data = testPNG(t)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/defects", &b)
		req.Header.Set("Authorization", auth)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp := httptest.NewRecorder()
		app.routes().ServeHTTP(resp, req)
		want := http.StatusBadRequest
		if valid {
			want = http.StatusOK
		}
		if resp.Code != want {
			t.Fatalf("upload valid=%v: %d %s", valid, resp.Code, resp.Body.String())
		}
	}
	t.Setenv("EXTERNAL_API_KEY", "test-key")
	resp := performRequest(t, app, http.MethodPost, "/api/external/report", `{"image_base64":"`+strings.Repeat("A", maxRequestBytes)+`"}`, map[string]string{"X-API-Key": "test-key"})
	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body: %d", resp.Code)
	}
}

func TestWorkflowVersionRejectsStaleOperation(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	tx, err := app.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := advanceDefectVersion(tx, defect.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = app.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := advanceDefectVersion(tx, defect.ID, 0); !errors.Is(err, errDefectConflict) {
		t.Fatalf("stale write: %v", err)
	}
}

func TestWorkflowFailureRollsBackStateAndVersion(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	admin, _, err := app.findUserByUsername("admin")
	if err != nil {
		t.Fatal(err)
	}
	inspector := createTestUser(t, app, "rollback_test", "处理员", "inspector")
	if _, err := app.db.Exec(`CREATE TRIGGER fail_flow BEFORE INSERT ON defect_flow_logs BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := app.applyDefectAction(admin, defect.ID, DefectActionRequest{Action: "assign", AssigneeID: &inspector.ID, AcceptorID: &inspector.ID}); err == nil {
		t.Fatal("expected failure")
	}
	var status string
	var version int
	if err := app.db.QueryRow(`SELECT status, workflow_version FROM defects WHERE id = ?`, defect.ID).Scan(&status, &version); err != nil {
		t.Fatal(err)
	}
	if status != "open" || version != 0 {
		t.Fatalf("failed flow persisted: %s, %d", status, version)
	}
}

func TestPendingAcceptanceCanBeReassignedAfterAcceptorDisabled(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	admin, _, err := app.findUserByUsername("admin")
	if err != nil {
		t.Fatal(err)
	}
	worker := createTestUser(t, app, "reassign_worker", "处理员", "inspector")
	acceptor := createTestUser(t, app, "reassign_acceptor", "验收员", "inspector")
	for _, action := range []DefectActionRequest{{Action: "assign", AssigneeID: &worker.ID, AcceptorID: &acceptor.ID}, {Action: "start_process"}, {Action: "submit_for_acceptance", ProcessResult: stringPtr("完成")}} {
		if _, err := app.applyDefectAction(admin, defect.ID, action); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.updateUser(acceptor.ID, updateUserRequest{IsActive: boolPtr(false)}); err != nil {
		t.Fatal(err)
	}
	updated, err := app.applyDefectAction(admin, defect.ID, DefectActionRequest{Action: "reassign", AssigneeID: &worker.ID, AcceptorID: &worker.ID})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "assigned" || *updated.AcceptorID != worker.ID {
		t.Fatalf("recovery failed: %+v", updated)
	}
}

func TestEditDefectPreservesWorkflowAndRejectsBlankTitle(t *testing.T) {
	app := newTestApp(t)
	defect := seedDefect(t, app)
	user := createTestUser(t, app, "edit_test", "质检员", "inspector")
	headers := map[string]string{"Authorization": bearerForUser(t, app, user.Username)}
	target := "/api/defects/" + strconv.FormatInt(defect.ID, 10)
	resp := performRequest(t, app, http.MethodPut, target, `{"name":"新标题","severity":"low","description":"新描述"}`, headers)
	if resp.Code != http.StatusOK {
		t.Fatalf("edit: %d %s", resp.Code, resp.Body.String())
	}
	updated, err := app.findDefect(defect.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "新标题" || updated.Status != "open" {
		t.Fatalf("bad edit: %+v", updated)
	}
	resp = performRequest(t, app, http.MethodPut, target, `{"name":"  "}`, headers)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("blank title accepted: %d", resp.Code)
	}
}
