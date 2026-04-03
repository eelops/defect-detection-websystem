package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const dbPath = "./defects.db"
const uploadDir = "./uploads"

var severityMap = map[string]string{
	"low":      "low",
	"低":         "low",
	"medium":   "medium",
	"中":         "medium",
	"high":     "high",
	"高":         "high",
	"critical": "critical",
	"严重":        "critical",
}

type App struct {
	db        *sql.DB
	uploadDir string
}

type Defect struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DefectType  string    `json:"defect_type"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	Location    *string   `json:"location,omitempty"`
	Description *string   `json:"description,omitempty"`
	ImagePath   *string   `json:"image_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DefectUpdate struct {
	Name        *string `json:"name"`
	DefectType  *string `json:"defect_type"`
	Severity    *string `json:"severity"`
	Status      *string `json:"status"`
	Location    *string `json:"location"`
	Description *string `json:"description"`
}

type ExternalDefectReport struct {
	ImageBase64 string   `json:"image_base64"`
	DefectType  string   `json:"defect_type"`
	Severity    string   `json:"severity"`
	Location    *string  `json:"location"`
	Confidence  *float64 `json:"confidence"`
	Description *string  `json:"description"`
	Source      *string  `json:"source"`
}

func main() {
	if err := loadEnvFile(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	app, err := newApp(dbPath, uploadDir)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	defer app.db.Close()

	log.Println("Go backend listening on :8000")
	if err := http.ListenAndServe(":8000", app.routes()); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func newApp(databasePath, uploadPath string) (*App, error) {
	if err := os.MkdirAll(uploadPath, 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &App{db: db, uploadDir: uploadPath}, nil
}

func (app *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.withCORS(app.handleRoot))
	mux.Handle("/uploads/", app.withCORSHandler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(app.uploadDir)))))
	mux.HandleFunc("/api/defects", app.withCORS(app.handleDefects))
	mux.HandleFunc("/api/defects/", app.withCORS(app.handleDefectByID))
	mux.HandleFunc("/api/alerts", app.withCORS(app.handleAlerts))
	mux.HandleFunc("/api/external/report", app.withCORS(app.handleExternalReport))
	return mux
}

func (app *App) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (app *App) withCORSHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "缺陷检测管理系统 API (Go)"})
}

func (app *App) handleDefects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.listDefects(w, r)
	case http.MethodPost:
		app.createDefect(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleDefectByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/defects/")
	if idStr == "" || strings.Contains(idStr, "/") {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺陷ID无效")
		return
	}

	switch r.Method {
	case http.MethodGet:
		app.getDefect(w, id)
	case http.MethodPut:
		app.updateDefect(w, r, id)
	case http.MethodDelete:
		app.deleteDefect(w, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	rows, err := app.db.Query(`
		SELECT id, name, defect_type, severity, status, location, description, image_path, created_at, updated_at
		FROM defects
		WHERE status IN ('open', 'fixing') AND severity IN ('high', 'critical')
		ORDER BY datetime(created_at) DESC, id DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询报警失败")
		return
	}
	defer rows.Close()

	defects, err := scanDefects(rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取报警失败")
		return
	}
	writeJSON(w, http.StatusOK, defects)
}

func (app *App) handleExternalReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	if err := verifyExternalAPIKey(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var payload ExternalDefectReport
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	if strings.TrimSpace(payload.ImageBase64) == "" || strings.TrimSpace(payload.DefectType) == "" || strings.TrimSpace(payload.Severity) == "" {
		writeError(w, http.StatusBadRequest, "缺少必要字段")
		return
	}

	severity, err := normalizeSeverity(payload.Severity)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	imagePath, err := app.saveBase64Image(payload.ImageBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	source := "外部系统"
	if payload.Source != nil && strings.TrimSpace(*payload.Source) != "" {
		source = strings.TrimSpace(*payload.Source)
	}
	name := fmt.Sprintf("[%s] %s 缺陷", source, strings.TrimSpace(payload.DefectType))

	descriptionParts := make([]string, 0, 3)
	if payload.Description != nil && strings.TrimSpace(*payload.Description) != "" {
		descriptionParts = append(descriptionParts, strings.TrimSpace(*payload.Description))
	}
	descriptionParts = append(descriptionParts, fmt.Sprintf("来源：%s", source))
	if payload.Confidence != nil {
		descriptionParts = append(descriptionParts, fmt.Sprintf("置信度：%v", *payload.Confidence))
	}
	description := strings.Join(descriptionParts, "\n")

	defect, err := app.insertDefect(name, payload.DefectType, severity, payload.Location, stringPtr(description), stringPtr(imagePath))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建缺陷失败")
		return
	}
	writeJSON(w, http.StatusOK, defect)
}

func (app *App) createDefect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "表单解析失败")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	defectType := strings.TrimSpace(r.FormValue("defect_type"))
	severityRaw := strings.TrimSpace(r.FormValue("severity"))
	location := nullableString(strings.TrimSpace(r.FormValue("location")))
	description := nullableString(strings.TrimSpace(r.FormValue("description")))
	if name == "" || defectType == "" || severityRaw == "" {
		writeError(w, http.StatusBadRequest, "缺少必要字段")
		return
	}

	severity, err := normalizeSeverity(severityRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var imagePath *string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		savedPath, saveErr := app.saveUploadedFile(file, header.Filename)
		if saveErr != nil {
			writeError(w, http.StatusInternalServerError, "图片保存失败")
			return
		}
		imagePath = &savedPath
	} else if !errors.Is(err, http.ErrMissingFile) {
		writeError(w, http.StatusBadRequest, "图片读取失败")
		return
	}

	defect, err := app.insertDefect(name, defectType, severity, location, description, imagePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建缺陷失败")
		return
	}
	writeJSON(w, http.StatusOK, defect)
}

func (app *App) listDefects(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, defect_type, severity, status, location, description, image_path, created_at, updated_at
		FROM defects
		WHERE 1=1
	`
	args := []any{}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if defectType := strings.TrimSpace(r.URL.Query().Get("defect_type")); defectType != "" {
		query += " AND defect_type = ?"
		args = append(args, defectType)
	}
	if severityRaw := strings.TrimSpace(r.URL.Query().Get("severity")); severityRaw != "" {
		severity, err := normalizeSeverity(severityRaw)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		query += " AND severity = ?"
		args = append(args, severity)
	}
	query += " ORDER BY datetime(created_at) DESC, id DESC"

	rows, err := app.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询缺陷失败")
		return
	}
	defer rows.Close()

	defects, err := scanDefects(rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取缺陷失败")
		return
	}
	writeJSON(w, http.StatusOK, defects)
}

func (app *App) getDefect(w http.ResponseWriter, id int64) {
	defect, err := app.findDefect(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询缺陷失败")
		return
	}
	writeJSON(w, http.StatusOK, defect)
}

func (app *App) updateDefect(w http.ResponseWriter, r *http.Request, id int64) {
	var payload DefectUpdate
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}

	current, err := app.findDefect(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询缺陷失败")
		return
	}

	name := current.Name
	defectType := current.DefectType
	severity := current.Severity
	status := current.Status
	location := current.Location
	description := current.Description

	if payload.Name != nil {
		name = *payload.Name
	}
	if payload.DefectType != nil {
		defectType = *payload.DefectType
	}
	if payload.Severity != nil {
		normalized, err := normalizeSeverity(*payload.Severity)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		severity = normalized
	}
	if payload.Status != nil {
		status = *payload.Status
	}
	if payload.Location != nil {
		location = payload.Location
	}
	if payload.Description != nil {
		description = payload.Description
	}

	_, err = app.db.Exec(`
		UPDATE defects
		SET name = ?, defect_type = ?, severity = ?, status = ?, location = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, name, defectType, severity, status, ptrValue(location), ptrValue(description), nowString(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "更新缺陷失败")
		return
	}

	updated, err := app.findDefect(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取更新结果失败")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (app *App) deleteDefect(w http.ResponseWriter, id int64) {
	defect, err := app.findDefect(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询缺陷失败")
		return
	}

	if defect.ImagePath != nil && *defect.ImagePath != "" {
		_ = os.Remove(filepath.Join(".", strings.TrimPrefix(*defect.ImagePath, "/")))
	}
	if _, err := app.db.Exec(`DELETE FROM defects WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "删除成功"})
}

func (app *App) insertDefect(name, defectType, severity string, location, description, imagePath *string) (*Defect, error) {
	result, err := app.db.Exec(`
		INSERT INTO defects (name, defect_type, severity, status, location, description, image_path, created_at, updated_at)
		VALUES (?, ?, ?, 'open', ?, ?, ?, ?, ?)
	`, name, defectType, severity, ptrValue(location), ptrValue(description), ptrValue(imagePath), nowString(), nowString())
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	defect, err := app.findDefect(id)
	if err != nil {
		return nil, err
	}
	if defect.Severity == "critical" {
		sendCriticalDefectEmail(defect)
	}
	return defect, nil
}

func (app *App) findDefect(id int64) (*Defect, error) {
	row := app.db.QueryRow(`
		SELECT id, name, defect_type, severity, status, location, description, image_path, created_at, updated_at
		FROM defects WHERE id = ?
	`, id)
	return scanDefect(row)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS defects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			defect_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			location TEXT,
			description TEXT,
			image_path TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	return ensureColumn(db, "defects", "location", "ALTER TABLE defects ADD COLUMN location TEXT")
}

func ensureColumn(db *sql.DB, tableName, columnName, alterSQL string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == columnName {
			return nil
		}
	}
	_, err = db.Exec(alterSQL)
	return err
}

func scanDefects(rows *sql.Rows) ([]Defect, error) {
	items := []Defect{}
	for rows.Next() {
		item, err := scanDefect(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDefect(s scanner) (*Defect, error) {
	var item Defect
	var location, description, imagePath sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(
		&item.ID,
		&item.Name,
		&item.DefectType,
		&item.Severity,
		&item.Status,
		&location,
		&description,
		&imagePath,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	item.Location = nullStringPtr(location)
	item.Description = nullStringPtr(description)
	item.ImagePath = nullStringPtr(imagePath)
	item.CreatedAt = mustParseTime(createdAt)
	item.UpdatedAt = mustParseTime(updatedAt)
	return &item, nil
}

func verifyExternalAPIKey(r *http.Request) error {
	expectedKey := os.Getenv("EXTERNAL_API_KEY")
	if expectedKey == "" {
		return errors.New("服务端未配置 EXTERNAL_API_KEY")
	}
	if r.Header.Get("X-API-Key") != expectedKey {
		return errors.New("无效的 API Key")
	}
	return nil
}

func normalizeSeverity(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("不支持的缺陷等级")
	}
	if value, ok := severityMap[strings.ToLower(trimmed)]; ok {
		return value, nil
	}
	if value, ok := severityMap[trimmed]; ok {
		return value, nil
	}
	return "", errors.New("不支持的缺陷等级")
}

func (app *App) saveUploadedFile(file io.Reader, originalName string) (string, error) {
	ext := filepath.Ext(originalName)
	if ext == "" {
		ext = ".png"
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	targetPath := filepath.Join(app.uploadDir, filename)
	out, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}
	return "/uploads/" + filename, nil
}

func (app *App) saveBase64Image(imageBase64 string) (string, error) {
	data := strings.TrimSpace(imageBase64)
	ext := ".png"
	if strings.HasPrefix(data, "data:") {
		parts := strings.SplitN(data, ",", 2)
		if len(parts) != 2 {
			return "", errors.New("图片 base64 格式不正确")
		}
		meta := parts[0]
		decodedMime := strings.TrimPrefix(strings.Split(meta, ";")[0], "data:")
		if extensions, err := mime.ExtensionsByType(decodedMime); err == nil && len(extensions) > 0 {
			ext = extensions[0]
		}
		data = parts[1]
	}
	binary, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		binary, err = base64.RawStdEncoding.DecodeString(data)
		if err != nil {
			return "", errors.New("图片 base64 解码失败")
		}
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	targetPath := filepath.Join(app.uploadDir, filename)
	if err := os.WriteFile(targetPath, binary, 0o644); err != nil {
		return "", err
	}
	return "/uploads/" + filename, nil
}

func sendCriticalDefectEmail(defect *Defect) {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	to := os.Getenv("ALERT_EMAIL_TO")
	if host == "" || port == "" || user == "" || password == "" || from == "" || to == "" {
		log.Println("邮件通知配置不完整，已跳过严重缺陷邮件发送")
		return
	}

	subject := fmt.Sprintf("[严重缺陷告警] %s", defect.Name)
	body := fmt.Sprintf("系统检测到新的严重缺陷，请及时处理。\n\n缺陷标题：%s\n缺陷类型：%s\n缺陷位置：%s\n严重等级：%s\n处理状态：%s\n缺陷描述：%s\n创建时间：%s\n",
		defect.Name,
		defect.DefectType,
		ptrValue(defect.Location),
		defect.Severity,
		defect.Status,
		ptrValue(defect.Description),
		defect.CreatedAt.Format(time.RFC3339),
	)
	message := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body)
	auth := smtp.PlainAuth("", user, password, host)
	if err := smtp.SendMail(host+":"+port, auth, from, []string{to}, message); err != nil {
		log.Printf("严重缺陷邮件发送失败: %v", err)
	}
}

func loadEnvFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}

func nowString() string {
	return time.Now().Format(time.RFC3339)
}

func mustParseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return t
	}
	t, err = time.Parse("2006-01-02 15:04:05.999999", value)
	if err == nil {
		return t
	}
	t, err = time.Parse("2006-01-02 15:04:05", value)
	if err == nil {
		return t
	}
	return time.Now()
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func stringPtr(value string) *string {
	return &value
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
