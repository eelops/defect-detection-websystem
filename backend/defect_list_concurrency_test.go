package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefectListReadableDuringWrite(t *testing.T) {
	app := newTestApp(t)
	seedDefect(t, app)
	tx, err := app.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var oldName string
	if err := tx.QueryRow("SELECT name FROM defects ORDER BY id LIMIT 1").Scan(&oldName); err != nil {
		t.Fatal(err)
	}
	// A long-running reader must not block another connection's write commit.
	if _, err := app.db.Exec("UPDATE defects SET name = 'committed' WHERE id = (SELECT MIN(id) FROM defects)"); err != nil {
		t.Fatal(err)
	}
	var snapshotName string
	if err := tx.QueryRow("SELECT name FROM defects ORDER BY id LIMIT 1").Scan(&snapshotName); err != nil {
		t.Fatal(err)
	}
	if snapshotName != oldName {
		t.Fatal("reader snapshot changed during concurrent write")
	}
	items, err := app.queryDefects(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || items[0].Name != "committed" {
		t.Fatalf("unexpected snapshot: %+v", items)
	}
}

func TestConcurrentDefectPageRequests(t *testing.T) {
	app := newTestApp(t)
	seedDefect(t, app)
	token := bearerForUser(t, app, "admin")
	handler := app.routes()
	paths := []string{"/api/defects", "/api/production-lines", "/api/equipment", "/api/defect-types", "/api/alerts"}
	results := make(chan *httptest.ResponseRecorder, 40)
	start := make(chan struct{})
	for i := 0; i < 40; i++ {
		go func(path string) {
			<-start
			r := httptest.NewRequest(http.MethodGet, path, nil)
			r.Header.Set("Authorization", token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			results <- w
		}(paths[i%len(paths)])
	}
	close(start)
	for i := 0; i < 40; i++ {
		w := <-results
		if w.Code != http.StatusOK {
			t.Errorf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
}
