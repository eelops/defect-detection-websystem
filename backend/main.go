package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const dbPath = "./defects.db"
const uploadDir = "./uploads"
const sessionDuration = 24 * time.Hour
const reportSystemName = "缺陷检测管理系统"
const pdfChineseFontName = "zh-cn"
const pdfChineseFontPath = "C:/Windows/Fonts/simhei.ttf"

var severityMap = map[string]string{
	"low":      "low",
	"低":        "low",
	"medium":   "medium",
	"中":        "medium",
	"high":     "high",
	"高":        "high",
	"critical": "critical",
	"严重":       "critical",
}

var roleLabels = map[string]string{
	"admin":     "管理员",
	"inspector": "质检员",
	"viewer":    "查看员",
}

var defectStatusLabels = map[string]string{
	"open":               "待分派",
	"assigned":           "已指派",
	"processing":         "处理中",
	"pending_acceptance": "待验收",
	"accepted":           "验收通过",
	"rejected":           "验收退回",
	"closed":             "已关闭",
}

var defectActionTransitions = map[string]map[string]string{
	"assign": {
		"open":     "assigned",
		"rejected": "assigned",
	},
	"reassign": {
		"assigned":   "assigned",
		"processing": "assigned",
		"rejected":   "assigned",
	},
	"start_process": {
		"assigned": "processing",
	},
	"submit_for_acceptance": {
		"processing": "pending_acceptance",
	},
	"accept": {
		"pending_acceptance": "accepted",
	},
	"reject": {
		"pending_acceptance": "rejected",
	},
	"close": {
		"accepted": "closed",
	},
}

type contextKey string

const currentUserKey contextKey = "currentUser"
const currentSessionTokenKey contextKey = "currentSessionToken"

type App struct {
	db        *sql.DB
	uploadDir string
}

type Defect struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	DefectType        string     `json:"defect_type"`
	DefectTypeID      *int64     `json:"defect_type_id"`
	Severity          string     `json:"severity"`
	Status            string     `json:"status"`
	Location          *string    `json:"location"`
	ProductionLineID  *int64     `json:"production_line_id"`
	ProductionLine    *string    `json:"production_line_name"`
	EquipmentID       *int64     `json:"equipment_id"`
	EquipmentName     *string    `json:"equipment_name"`
	Description       *string    `json:"description,omitempty"`
	ImagePath         *string    `json:"image_path,omitempty"`
	AssigneeID        *int64     `json:"assignee_id,omitempty"`
	AssigneeName      *string    `json:"assignee_name,omitempty"`
	AcceptorID        *int64     `json:"acceptor_id,omitempty"`
	AcceptorName      *string    `json:"acceptor_name,omitempty"`
	AssignedAt        *time.Time `json:"assigned_at,omitempty"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	AcceptedAt        *time.Time `json:"accepted_at,omitempty"`
	ClosedAt          *time.Time `json:"closed_at,omitempty"`
	ProcessResult     *string    `json:"process_result,omitempty"`
	AcceptanceComment *string    `json:"acceptance_comment,omitempty"`
	AvailableActions  []string   `json:"available_actions,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type DefectUpdate struct {
	Name             *string `json:"name"`
	DefectType       *string `json:"defect_type"`
	DefectTypeID     *int64  `json:"defect_type_id"`
	Severity         *string `json:"severity"`
	Status           *string `json:"status"`
	Location         *string `json:"location"`
	ProductionLineID *int64  `json:"production_line_id"`
	EquipmentID      *int64  `json:"equipment_id"`
	Description      *string `json:"description"`
}

type DefectActionRequest struct {
	Action            string `json:"action"`
	AssigneeID        *int64 `json:"assignee_id"`
	AcceptorID        *int64 `json:"acceptor_id"`
	ProcessResult     *string `json:"process_result"`
	AcceptanceComment *string `json:"acceptance_comment"`
	Comment           *string `json:"comment"`
}

type DefectFlowLog struct {
	ID           int64     `json:"id"`
	DefectID     int64     `json:"defect_id"`
	Action       string    `json:"action"`
	FromStatus   string    `json:"from_status"`
	ToStatus     string    `json:"to_status"`
	OperatorID   int64     `json:"operator_id"`
	OperatorName string    `json:"operator_name"`
	OperatorRole string    `json:"operator_role"`
	Comment      *string   `json:"comment,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type MasterDataItem struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ProductionLine struct {
	MasterDataItem
}

type Equipment struct {
	MasterDataItem
	ProductionLineID   int64   `json:"production_line_id"`
	ProductionLineName *string `json:"production_line_name,omitempty"`
	Model              *string `json:"model,omitempty"`
}

type DefectType struct {
	MasterDataItem
	Severity *string `json:"severity,omitempty"`
}

type masterDataPayload struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type equipmentPayload struct {
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Description      *string `json:"description"`
	Status           *string `json:"status"`
	ProductionLineID int64   `json:"production_line_id"`
	Model            *string `json:"model"`
}

type defectTypePayload struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Severity    *string `json:"severity"`
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

type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	RoleLabel   string     `json:"role_label"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsActive    *bool  `json:"is_active"`
}

type updateUserRequest struct {
	DisplayName *string `json:"display_name"`
	Role        *string `json:"role"`
	IsActive    *bool   `json:"is_active"`
	Password    *string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type sessionRecord struct {
	User      *User
	ExpiresAt time.Time
}

type OperationLog struct {
	ID               int64          `json:"id"`
	ActorUserID      *int64         `json:"actor_user_id,omitempty"`
	ActorUsername    string         `json:"actor_username,omitempty"`
	ActorDisplayName string         `json:"actor_display_name,omitempty"`
	ActorRole        string         `json:"actor_role,omitempty"`
	ActorType        string         `json:"actor_type"`
	Module           string         `json:"module"`
	Action           string         `json:"action"`
	TargetType       string         `json:"target_type,omitempty"`
	TargetID         *int64         `json:"target_id,omitempty"`
	TargetName       string         `json:"target_name,omitempty"`
	Result           string         `json:"result"`
	Summary          string         `json:"summary"`
	Details          map[string]any `json:"details,omitempty"`
	IPAddress        string         `json:"ip_address,omitempty"`
	UserAgent        string         `json:"user_agent,omitempty"`
	RequestMethod    string         `json:"request_method,omitempty"`
	RequestPath      string         `json:"request_path,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

type operationLogEntry struct {
	ActorType  string
	Module     string
	Action     string
	TargetType string
	TargetID   *int64
	TargetName string
	Result     string
	Summary    string
	Details    map[string]any
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

	app := &App{db: db, uploadDir: uploadPath}
	if err := app.ensureBootstrapAdmin(); err != nil {
		db.Close()
		return nil, err
	}
	if err := app.ensureDemoUsers(); err != nil {
		db.Close()
		return nil, err
	}
	if err := app.ensureDemoMasterData(); err != nil {
		db.Close()
		return nil, err
	}
	if err := app.ensureDemoDefects(); err != nil {
		db.Close()
		return nil, err
	}
	return app, nil
}

func (app *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.withCORS(app.handleRoot))
	mux.Handle("/uploads/", app.withCORSHandler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(app.uploadDir)))))
	mux.HandleFunc("/api/auth/login", app.withCORS(app.handleLogin))
	mux.HandleFunc("/api/auth/logout", app.withCORS(app.withAuth(app.handleLogout)))
	mux.HandleFunc("/api/auth/me", app.withCORS(app.withAuth(app.handleMe)))
	mux.HandleFunc("/api/auth/password", app.withCORS(app.withAuth(app.handleChangePassword)))
	mux.HandleFunc("/api/users", app.withCORS(app.withAuth(app.requireRoles(app.handleUsers, "admin"))))
	mux.HandleFunc("/api/users/", app.withCORS(app.withAuth(app.requireRoles(app.handleUserByID, "admin"))))
	mux.HandleFunc("/api/operation-logs", app.withCORS(app.withAuth(app.requireRoles(app.handleOperationLogs, "admin"))))
	mux.HandleFunc("/api/production-lines", app.withCORS(app.withAuth(app.requireRoles(app.handleProductionLines, "admin", "inspector", "viewer"))))
	mux.HandleFunc("/api/production-lines/", app.withCORS(app.withAuth(app.requireRoles(app.handleProductionLineByID, "admin"))))
	mux.HandleFunc("/api/equipment", app.withCORS(app.withAuth(app.requireRoles(app.handleEquipment, "admin", "inspector", "viewer"))))
	mux.HandleFunc("/api/equipment/", app.withCORS(app.withAuth(app.requireRoles(app.handleEquipmentByID, "admin"))))
	mux.HandleFunc("/api/defect-types", app.withCORS(app.withAuth(app.requireRoles(app.handleDefectTypes, "admin", "inspector", "viewer"))))
	mux.HandleFunc("/api/defect-types/", app.withCORS(app.withAuth(app.requireRoles(app.handleDefectTypeByID, "admin"))))
	mux.HandleFunc("/api/defects/export.xlsx", app.withCORS(app.withAuth(app.requireRoles(app.handleDefectExportExcel, "admin", "inspector", "viewer"))))
	mux.HandleFunc("/api/defects", app.withCORS(app.withAuth(app.handleDefects)))
	mux.HandleFunc("/api/defects/", app.withCORS(app.withAuth(app.handleDefectByID)))
	mux.HandleFunc("/api/alerts", app.withCORS(app.withAuth(app.requireRoles(app.handleAlerts, "admin", "inspector", "viewer"))))
	mux.HandleFunc("/api/external/report", app.withCORS(app.handleExternalReport))
	return mux
}

func (app *App) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *App) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		session, err := app.findSession(token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, "登录状态无效或已过期")
				return
			}
			writeError(w, http.StatusInternalServerError, "校验登录状态失败")
			return
		}
		if session.ExpiresAt.Before(time.Now()) {
			_ = app.deleteSession(token)
			writeError(w, http.StatusUnauthorized, "登录状态无效或已过期")
			return
		}
		ctx := context.WithValue(r.Context(), currentUserKey, session.User)
		ctx = context.WithValue(ctx, currentSessionTokenKey, token)
		next(w, r.WithContext(ctx))
	}
}

func (app *App) requireRoles(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r.Context())
		if user == nil || !allowed[user.Role] {
			app.writeOperationLog(r, operationLogEntry{
				ActorType:  "user",
				Module:     moduleFromPath(r.URL.Path),
				Action:     "forbidden",
				TargetType: targetTypeFromPath(r.URL.Path),
				Result:     "failure",
				Summary:    fmt.Sprintf("账号 %s 无权限访问 %s", firstNonEmpty(userDisplayName(user), "未知账号"), r.URL.Path),
				Details: map[string]any{
					"required_roles": roles,
					"actual_role":    roleOfUser(user),
				},
			})
			writeError(w, http.StatusForbidden, "当前账号没有权限执行该操作")
			return
		}
		next(w, r)
	}
}

func (app *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "缺陷检测管理系统 API (Go)"})
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	var payload loginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	username := strings.TrimSpace(payload.Username)
	password := strings.TrimSpace(payload.Password)
	if username == "" || password == "" {
		writeError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	user, passwordHash, err := app.findUserByUsername(username)
	if err != nil {
		app.writeOperationLog(r, operationLogEntry{
			ActorType:  "user",
			Module:     "auth",
			Action:     "login",
			TargetType: "session",
			TargetName: username,
			Result:     "failure",
			Summary:    fmt.Sprintf("账号 %s 登录失败", username),
		})
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	if !user.IsActive {
		app.writeOperationLog(r, operationLogEntry{
			ActorType:  "user",
			Module:     "auth",
			Action:     "login",
			TargetType: "session",
			TargetID:   int64Ptr(user.ID),
			TargetName: user.Username,
			Result:     "failure",
			Summary:    fmt.Sprintf("账号 %s 登录失败：账号已禁用", user.Username),
		})
		writeError(w, http.StatusForbidden, "账号已被禁用")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		app.writeOperationLog(r, operationLogEntry{
			ActorType:  "user",
			Module:     "auth",
			Action:     "login",
			TargetType: "session",
			TargetID:   int64Ptr(user.ID),
			TargetName: user.Username,
			Result:     "failure",
			Summary:    fmt.Sprintf("账号 %s 登录失败", user.Username),
		})
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建登录状态失败")
		return
	}
	if err := app.createSession(user.ID, token); err != nil {
		writeError(w, http.StatusInternalServerError, "创建登录状态失败")
		return
	}
	_, _ = app.db.Exec(`UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`, nowString(), nowString(), user.ID)
	freshUser, err := app.findUser(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取用户信息失败")
		return
	}
	app.writeOperationLog(withUserContext(r, freshUser), operationLogEntry{
		ActorType:  "user",
		Module:     "auth",
		Action:     "login",
		TargetType: "session",
		TargetID:   int64Ptr(freshUser.ID),
		TargetName: freshUser.Username,
		Result:     "success",
		Summary:    fmt.Sprintf("账号 %s 登录成功", freshUser.Username),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  freshUser,
	})
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	if err := app.deleteSession(currentSessionToken(r.Context())); err != nil {
		writeError(w, http.StatusInternalServerError, "退出登录失败")
		return
	}
	app.writeOperationLog(r, operationLogEntry{
		ActorType:  "user",
		Module:     "auth",
		Action:     "logout",
		TargetType: "session",
		TargetID:   int64Ptr(currentUser(r.Context()).ID),
		TargetName: currentUser(r.Context()).Username,
		Result:     "success",
		Summary:    fmt.Sprintf("账号 %s 退出登录", currentUser(r.Context()).Username),
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "退出登录成功"})
}

func (app *App) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	writeJSON(w, http.StatusOK, currentUser(r.Context()))
}

func (app *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	user := currentUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "缺少登录凭证")
		return
	}

	var payload changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	currentPassword := strings.TrimSpace(payload.CurrentPassword)
	newPassword := strings.TrimSpace(payload.NewPassword)
	if currentPassword == "" || newPassword == "" {
		writeError(w, http.StatusBadRequest, "原密码和新密码不能为空")
		return
	}

	_, passwordHash, err := app.findUserByIDWithHash(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取用户信息失败")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		app.writeOperationLog(r, operationLogEntry{
			ActorType:  "user",
			Module:     "auth",
			Action:     "change_password",
			TargetType: "user",
			TargetID:   int64Ptr(user.ID),
			TargetName: user.Username,
			Result:     "failure",
			Summary:    fmt.Sprintf("账号 %s 修改密码失败：原密码不正确", user.Username),
		})
		writeError(w, http.StatusBadRequest, "原密码不正确")
		return
	}
	newPasswordHash, err := hashPassword(newPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "更新密码失败")
		return
	}
	if _, err := app.db.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, newPasswordHash, nowString(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "更新密码失败")
		return
	}
	app.writeOperationLog(r, operationLogEntry{
		ActorType:  "user",
		Module:     "auth",
		Action:     "change_password",
		TargetType: "user",
		TargetID:   int64Ptr(user.ID),
		TargetName: user.Username,
		Result:     "success",
		Summary:    fmt.Sprintf("账号 %s 修改本人密码", user.Username),
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "密码修改成功"})
}

func (app *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := app.listUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "查询用户失败")
			return
		}
		writeJSON(w, http.StatusOK, users)
	case http.MethodPost:
		var payload createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "JSON 格式不正确")
			return
		}
		user, err := app.createUser(payload)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNIQUE") {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		app.writeOperationLog(r, operationLogEntry{
			ActorType:  "user",
			Module:     "user",
			Action:     "create",
			TargetType: "user",
			TargetID:   int64Ptr(user.ID),
			TargetName: user.Username,
			Result:     "success",
			Summary:    fmt.Sprintf("创建用户 %s", user.Username),
			Details: map[string]any{
				"display_name": user.DisplayName,
				"role":         user.Role,
				"is_active":    user.IsActive,
			},
		})
		writeJSON(w, http.StatusOK, user)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if idStr == "" || strings.Contains(idStr, "/") {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户ID无效")
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	var payload updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	before, err := app.findUser(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "用户不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "读取用户信息失败")
		return
	}
	updated, err := app.updateUser(id, payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "用户不存在")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	action := "update"
	if payload.Password != nil {
		action = "reset_password"
	}
	if payload.IsActive != nil && before.IsActive != updated.IsActive {
		if updated.IsActive {
			action = "enable"
		} else {
			action = "disable"
		}
	}
	app.writeOperationLog(r, operationLogEntry{
		ActorType:  "user",
		Module:     "user",
		Action:     action,
		TargetType: "user",
		TargetID:   int64Ptr(updated.ID),
		TargetName: updated.Username,
		Result:     "success",
		Summary:    summarizeUserAction(action, updated),
		Details:    diffUsers(before, updated, payload.Password != nil),
	})
	writeJSON(w, http.StatusOK, updated)
}

func (app *App) handleDefects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.requireRoles(app.listDefects, "admin", "inspector", "viewer")(w, r)
	case http.MethodPost:
		app.requireRoles(app.createDefect, "admin", "inspector")(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleDefectByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/defects/")
	if idStr == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.Trim(idStr, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺陷ID无效")
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			app.requireRoles(func(w http.ResponseWriter, r *http.Request) { app.getDefect(w, r, id) }, "admin", "inspector", "viewer")(w, r)
		case http.MethodPut:
			app.requireRoles(func(w http.ResponseWriter, r *http.Request) { app.updateDefect(w, r, id) }, "admin", "inspector")(w, r)
		case http.MethodDelete:
			app.requireRoles(func(w http.ResponseWriter, r *http.Request) { app.deleteDefect(w, r, id) }, "admin")(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		}
		return
	}

	switch parts[1] {
	case "actions":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
			return
		}
		app.requireRoles(func(w http.ResponseWriter, r *http.Request) { app.handleDefectAction(w, r, id) }, "admin", "inspector")(w, r)
	case "flow-logs":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
			return
		}
		app.requireRoles(func(w http.ResponseWriter, r *http.Request) { app.handleDefectFlowLogs(w, r, id) }, "admin", "inspector", "viewer")(w, r)
	case "export.pdf":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
			return
		}
		app.requireRoles(func(w http.ResponseWriter, r *http.Request) { app.handleDefectExportPDF(w, r, id) }, "admin", "inspector", "viewer")(w, r)
	default:
		http.NotFound(w, r)
	}
}


func (app *App) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	rows, err := app.db.Query(`
		SELECT id, name, defect_type, severity, status, location, description, image_path,
			assignee_id, assignee_name, acceptor_id, acceptor_name, assigned_at, processed_at, accepted_at, closed_at,
			process_result, acceptance_comment, created_at, updated_at
		FROM defects
		WHERE status IN ('open', 'assigned', 'processing', 'pending_acceptance', 'rejected') AND severity IN ('high', 'critical')
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
		app.writeOperationLog(r, operationLogEntry{
			ActorType:  "external",
			Module:     "external",
			Action:     "external_report",
			TargetType: "api",
			TargetName: "/api/external/report",
			Result:     "failure",
			Summary:    "外部接口上报失败：API Key 校验未通过",
		})
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
	defectTypeID, err := parseOptionalInt64(r.FormValue("defect_type_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺陷类型ID无效")
		return
	}
	severityRaw := strings.TrimSpace(r.FormValue("severity"))
	productionLineID, err := parseOptionalInt64(r.FormValue("production_line_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "产线ID无效")
		return
	}
	equipmentID, err := parseOptionalInt64(r.FormValue("equipment_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "设备ID无效")
		return
	}
	location := nullableString(strings.TrimSpace(r.FormValue("location")))
	description := nullableString(strings.TrimSpace(r.FormValue("description")))

	masterDefectType, masterProductionLine, masterEquipment, err := app.validateDefectMasterData(defectTypeID, productionLineID, equipmentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if masterDefectType != nil {
		defectType = masterDefectType.Name
	}
	if masterProductionLine != nil && location == nil {
		location = stringPtr(masterProductionLine.Name)
	}
	if masterEquipment != nil {
		location = stringPtr(firstNonEmpty(ptrValue(masterEquipment.ProductionLineName), ptrValue(masterEquipment.Model), masterEquipment.Name))
	}
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
	if defectTypeID != nil || productionLineID != nil || equipmentID != nil {
		_, err = app.db.Exec(`UPDATE defects SET defect_type_id = ?, production_line_id = ?, equipment_id = ?, updated_at = ? WHERE id = ?`, nullInt64Value(defectTypeID), nullInt64Value(productionLineID), nullInt64Value(equipmentID), nowString(), defect.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "保存主数据关联失败")
			return
		}
		defect, err = app.findDefect(defect.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "读取缺陷失败")
			return
		}
	}
	writeJSON(w, http.StatusOK, defect)
}

func (app *App) listDefects(w http.ResponseWriter, r *http.Request) {
	defects, err := app.queryDefects(r.URL.Query())
	if err != nil {
		if errors.Is(err, errInvalidSeverity) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "查询缺陷失败")
		return
	}
	writeJSON(w, http.StatusOK, defects)
}

func (app *App) handleDefectExportExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	defects, err := app.queryDefects(r.URL.Query())
	if err != nil {
		if errors.Is(err, errInvalidSeverity) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "生成 Excel 报表失败")
		return
	}

	buffer, err := buildDefectsExcelReport(defects, r.URL.Query())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成 Excel 报表失败")
		return
	}

	filename := fmt.Sprintf("defects_report_%s.xlsx", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buffer.Bytes())
}

func (app *App) queryDefects(values map[string][]string) ([]Defect, error) {
	query := `
		SELECT d.id, d.name, COALESCE(dt.name, d.defect_type) AS defect_type, d.defect_type_id, d.severity, d.status,
			d.location, d.production_line_id, pl.name AS production_line_name, d.equipment_id, eq.name AS equipment_name,
			d.description, d.image_path,
			d.assignee_id, d.assignee_name, d.acceptor_id, d.acceptor_name, d.assigned_at, d.processed_at, d.accepted_at, d.closed_at,
			d.process_result, d.acceptance_comment, d.created_at, d.updated_at
		FROM defects d
		LEFT JOIN defect_types dt ON dt.id = d.defect_type_id
		LEFT JOIN production_lines pl ON pl.id = d.production_line_id
		LEFT JOIN equipment eq ON eq.id = d.equipment_id
		WHERE 1=1
	`
	args := []any{}
	get := func(key string) string {
		return strings.TrimSpace(firstValue(values, key))
	}
	if status := get("status"); status != "" {
		query += " AND d.status = ?"
		args = append(args, status)
	}
	if defectType := get("defect_type"); defectType != "" {
		query += " AND COALESCE(dt.name, d.defect_type) = ?"
		args = append(args, defectType)
	}
	if defectTypeID := get("defect_type_id"); defectTypeID != "" {
		query += " AND d.defect_type_id = ?"
		args = append(args, defectTypeID)
	}
	if productionLineID := get("production_line_id"); productionLineID != "" {
		query += " AND d.production_line_id = ?"
		args = append(args, productionLineID)
	}
	if equipmentID := get("equipment_id"); equipmentID != "" {
		query += " AND d.equipment_id = ?"
		args = append(args, equipmentID)
	}
	if severityRaw := get("severity"); severityRaw != "" {
		severity, err := normalizeSeverity(severityRaw)
		if err != nil {
			return nil, errInvalidSeverity
		}
		query += " AND d.severity = ?"
		args = append(args, severity)
	}
	if assigneeID := get("assignee_id"); assigneeID != "" {
		query += " AND d.assignee_id = ?"
		args = append(args, assigneeID)
	}
	if acceptorID := get("acceptor_id"); acceptorID != "" {
		query += " AND d.acceptor_id = ?"
		args = append(args, acceptorID)
	}
	query += " ORDER BY datetime(d.updated_at) DESC, d.id DESC"

	rows, err := app.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDefects(rows)
}

func buildDefectsExcelReport(defects []Defect, filters map[string][]string) (*bytes.Buffer, error) {
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()

	sheet := file.GetSheetName(0)
	file.SetSheetName(sheet, "缺陷台账")
	sheet = "缺陷台账"

	titleStyle, _ := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 16}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	headerStyle, _ := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "#FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"#1F4E78"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	bodyStyle, _ := file.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}})
	metaStyle, _ := file.NewStyle(&excelize.Style{Font: &excelize.Font{Italic: true, Color: "#4F81BD"}})

	_ = file.MergeCell(sheet, "A1", "J1")
	_ = file.SetCellValue(sheet, "A1", reportSystemName+" - 缺陷台账报表")
	_ = file.SetCellStyle(sheet, "A1", "A1", titleStyle)
	_ = file.SetRowHeight(sheet, 1, 28)

	_ = file.MergeCell(sheet, "A2", "J2")
	_ = file.SetCellValue(sheet, "A2", fmt.Sprintf("导出时间：%s", time.Now().Format("2006-01-02 15:04:05")))
	_ = file.SetCellStyle(sheet, "A2", "A2", metaStyle)

	_ = file.MergeCell(sheet, "A3", "J3")
	_ = file.SetCellValue(sheet, "A3", fmt.Sprintf("筛选条件：%s", formatFilterSummary(filters)))
	_ = file.SetCellStyle(sheet, "A3", "A3", metaStyle)

	headers := []string{"ID", "标题", "类型", "等级", "状态", "位置", "描述", "处理人", "创建时间", "更新时间"}
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 5)
		_ = file.SetCellValue(sheet, cell, header)
	}
	_ = file.SetCellStyle(sheet, "A5", "J5", headerStyle)

	for index, defect := range defects {
		row := index + 6
		values := []any{
			defect.ID,
			defect.Name,
			defect.DefectType,
			severityLabelMapForExport(defect.Severity),
			statusLabelForExport(defect.Status),
			ptrValue(defect.Location),
			ptrValue(defect.Description),
			firstNonEmpty(ptrValue(defect.AssigneeName), ptrValue(defect.AcceptorName)),
			defect.CreatedAt.Format("2006-01-02 15:04:05"),
			defect.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		for column, value := range values {
			cell, _ := excelize.CoordinatesToCellName(column+1, row)
			_ = file.SetCellValue(sheet, cell, value)
		}
	}
	_ = file.SetCellStyle(sheet, "A6", fmt.Sprintf("J%d", len(defects)+5), bodyStyle)

	_ = file.SetColWidth(sheet, "A", "A", 10)
	_ = file.SetColWidth(sheet, "B", "B", 28)
	_ = file.SetColWidth(sheet, "C", "C", 18)
	_ = file.SetColWidth(sheet, "D", "E", 14)
	_ = file.SetColWidth(sheet, "F", "F", 18)
	_ = file.SetColWidth(sheet, "G", "G", 36)
	_ = file.SetColWidth(sheet, "H", "H", 16)
	_ = file.SetColWidth(sheet, "I", "J", 22)

	buffer := &bytes.Buffer{}
	if err := file.Write(buffer); err != nil {
		return nil, err
	}
	return buffer, nil
}

var errInvalidSeverity = errors.New("不支持的缺陷等级")

func firstValue(values map[string][]string, key string) string {
	if len(values[key]) == 0 {
		return ""
	}
	return values[key][0]
}

func formatFilterSummary(filters map[string][]string) string {
	parts := []string{}
	if value := strings.TrimSpace(firstValue(filters, "defect_type")); value != "" {
		parts = append(parts, "类型="+value)
	}
	if value := strings.TrimSpace(firstValue(filters, "production_line_id")); value != "" {
		parts = append(parts, "产线ID="+value)
	}
	if value := strings.TrimSpace(firstValue(filters, "equipment_id")); value != "" {
		parts = append(parts, "设备ID="+value)
	}
	if value := strings.TrimSpace(firstValue(filters, "severity")); value != "" {
		parts = append(parts, "等级="+value)
	}
	if value := strings.TrimSpace(firstValue(filters, "status")); value != "" {
		parts = append(parts, "状态="+value)
	}
	if len(parts) == 0 {
		return "全部缺陷"
	}
	return strings.Join(parts, "；")
}

func severityLabelMapForExport(value string) string {
	switch value {
	case "low":
		return "低"
	case "medium":
		return "中"
	case "high":
		return "高"
	case "critical":
		return "严重"
	default:
		return value
	}
}

func statusLabelForExport(value string) string {
	if label := defectStatusLabels[value]; label != "" {
		return label
	}
	return value
}


func (app *App) getDefect(w http.ResponseWriter, r *http.Request, id int64) {
	defect, err := app.findDefect(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询缺陷失败")
		return
	}
	defect.AvailableActions = app.availableDefectActions(currentUser(r.Context()), defect)
	writeJSON(w, http.StatusOK, defect)
}

func (app *App) updateDefect(w http.ResponseWriter, r *http.Request, id int64) {
	var payload DefectUpdate
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	if payload.Status != nil {
		writeError(w, http.StatusBadRequest, "状态流转请使用流程动作接口")
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
	defectTypeID := current.DefectTypeID
	severity := current.Severity
	location := current.Location
	productionLineID := current.ProductionLineID
	equipmentID := current.EquipmentID
	description := current.Description

	if payload.Name != nil {
		name = strings.TrimSpace(*payload.Name)
	}
	if payload.DefectType != nil {
		defectType = strings.TrimSpace(*payload.DefectType)
		if defectType != "" {
			defectTypeID = nil
		}
	}
	if payload.DefectTypeID != nil {
		defectTypeID = payload.DefectTypeID
	}
	if payload.Severity != nil {
		normalized, err := normalizeSeverity(*payload.Severity)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		severity = normalized
	}
	if payload.Location != nil {
		location = payload.Location
	}
	if payload.ProductionLineID != nil {
		productionLineID = payload.ProductionLineID
	}
	if payload.EquipmentID != nil {
		equipmentID = payload.EquipmentID
	}
	if payload.Description != nil {
		description = payload.Description
	}

	masterDefectType, masterProductionLine, masterEquipment, err := app.validateDefectMasterData(defectTypeID, productionLineID, equipmentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if masterDefectType != nil {
		defectType = masterDefectType.Name
	}
	if masterProductionLine != nil && location == nil {
		location = stringPtr(masterProductionLine.Name)
	}
	if masterEquipment != nil {
		location = stringPtr(firstNonEmpty(ptrValue(masterEquipment.ProductionLineName), ptrValue(masterEquipment.Model), masterEquipment.Name))
	}

	_, err = app.db.Exec(`
		UPDATE defects
		SET name = ?, defect_type = ?, defect_type_id = ?, severity = ?, location = ?, production_line_id = ?, equipment_id = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, name, defectType, nullInt64Value(defectTypeID), severity, ptrValue(location), nullInt64Value(productionLineID), nullInt64Value(equipmentID), ptrValue(description), nowString(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "更新缺陷失败")
		return
	}

	updated, err := app.findDefect(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取更新结果失败")
		return
	}
	updated.AvailableActions = app.availableDefectActions(currentUser(r.Context()), updated)
	writeJSON(w, http.StatusOK, updated)
}

func (app *App) deleteDefect(w http.ResponseWriter, r *http.Request, id int64) {
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
	app.writeOperationLog(r, operationLogEntry{
		ActorType:  "user",
		Module:     "defect",
		Action:     "delete",
		TargetType: "defect",
		TargetID:   int64Ptr(id),
		TargetName: defect.Name,
		Result:     "success",
		Summary:    fmt.Sprintf("删除缺陷 #%d：%s", id, defect.Name),
	})
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
		SELECT d.id, d.name, COALESCE(dt.name, d.defect_type) AS defect_type, d.defect_type_id, d.severity, d.status,
			d.location, d.production_line_id, pl.name AS production_line_name, d.equipment_id, eq.name AS equipment_name,
			d.description, d.image_path,
			d.assignee_id, d.assignee_name, d.acceptor_id, d.acceptor_name, d.assigned_at, d.processed_at, d.accepted_at, d.closed_at,
			d.process_result, d.acceptance_comment, d.created_at, d.updated_at
		FROM defects d
		LEFT JOIN defect_types dt ON dt.id = d.defect_type_id
		LEFT JOIN production_lines pl ON pl.id = d.production_line_id
		LEFT JOIN equipment eq ON eq.id = d.equipment_id
		WHERE d.id = ?
	`, id)
	return scanDefect(row)
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS defects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			defect_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			location TEXT,
			description TEXT,
			image_path TEXT,
			assignee_id INTEGER,
			assignee_name TEXT,
			acceptor_id INTEGER,
			acceptor_name TEXT,
			assigned_at TEXT,
			processed_at TEXT,
			accepted_at TEXT,
			closed_at TEXT,
			process_result TEXT,
			acceptance_comment TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			display_name TEXT NOT NULL,
			role TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			last_login_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS production_lines (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				code TEXT NOT NULL UNIQUE,
				name TEXT NOT NULL,
				description TEXT,
				status TEXT NOT NULL DEFAULT 'active',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS equipment (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				code TEXT NOT NULL UNIQUE,
				name TEXT NOT NULL,
				description TEXT,
				status TEXT NOT NULL DEFAULT 'active',
				production_line_id INTEGER NOT NULL,
				model TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				FOREIGN KEY (production_line_id) REFERENCES production_lines(id)
			)`,
			`CREATE TABLE IF NOT EXISTS defect_types (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				code TEXT NOT NULL UNIQUE,
				name TEXT NOT NULL,
				description TEXT,
				severity TEXT,
				status TEXT NOT NULL DEFAULT 'active',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS operation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			actor_user_id INTEGER,
			actor_username TEXT,
			actor_display_name TEXT,
			actor_role TEXT,
			actor_type TEXT NOT NULL,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			target_type TEXT,
			target_id INTEGER,
			target_name TEXT,
			result TEXT NOT NULL,
			summary TEXT NOT NULL,
			details_json TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_method TEXT,
			request_path TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS defect_flow_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			defect_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			from_status TEXT NOT NULL,
			to_status TEXT NOT NULL,
			operator_id INTEGER NOT NULL,
			operator_name TEXT NOT NULL,
			operator_role TEXT NOT NULL,
			comment TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (defect_id) REFERENCES defects(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at ON operation_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_logs_module ON operation_logs(module)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_logs_actor_user_id ON operation_logs(actor_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_logs_target ON operation_logs(target_type, target_id)`,
		`CREATE INDEX IF NOT EXISTS idx_defect_flow_logs_defect_id ON defect_flow_logs(defect_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	for _, column := range []struct{ name, alter string }{
		{"location", "ALTER TABLE defects ADD COLUMN location TEXT"},
		{"assignee_id", "ALTER TABLE defects ADD COLUMN assignee_id INTEGER"},
		{"assignee_name", "ALTER TABLE defects ADD COLUMN assignee_name TEXT"},
		{"acceptor_id", "ALTER TABLE defects ADD COLUMN acceptor_id INTEGER"},
		{"acceptor_name", "ALTER TABLE defects ADD COLUMN acceptor_name TEXT"},
		{"assigned_at", "ALTER TABLE defects ADD COLUMN assigned_at TEXT"},
		{"processed_at", "ALTER TABLE defects ADD COLUMN processed_at TEXT"},
		{"accepted_at", "ALTER TABLE defects ADD COLUMN accepted_at TEXT"},
		{"closed_at", "ALTER TABLE defects ADD COLUMN closed_at TEXT"},
		{"process_result", "ALTER TABLE defects ADD COLUMN process_result TEXT"},
		{"acceptance_comment", "ALTER TABLE defects ADD COLUMN acceptance_comment TEXT"},
			{"defect_type_id", "ALTER TABLE defects ADD COLUMN defect_type_id INTEGER"},
			{"production_line_id", "ALTER TABLE defects ADD COLUMN production_line_id INTEGER"},
			{"equipment_id", "ALTER TABLE defects ADD COLUMN equipment_id INTEGER"},
	} {
		if err := ensureColumn(db, "defects", column.name, column.alter); err != nil {
			return err
		}
	}
	return nil
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
	var defectTypeID, productionLineID, equipmentID sql.NullInt64
	var location, productionLineName, equipmentName, description, imagePath sql.NullString
	var assigneeID, acceptorID sql.NullInt64
	var assigneeName, acceptorName sql.NullString
	var assignedAt, processedAt, acceptedAt, closedAt sql.NullString
	var processResult, acceptanceComment sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(
		&item.ID,
		&item.Name,
		&item.DefectType,
		&defectTypeID,
		&item.Severity,
		&item.Status,
		&location,
		&productionLineID,
		&productionLineName,
		&equipmentID,
		&equipmentName,
		&description,
		&imagePath,
		&assigneeID,
		&assigneeName,
		&acceptorID,
		&acceptorName,
		&assignedAt,
		&processedAt,
		&acceptedAt,
		&closedAt,
		&processResult,
		&acceptanceComment,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	item.DefectTypeID = nullInt64Ptr(defectTypeID)
	item.Location = nullStringPtr(location)
	item.ProductionLineID = nullInt64Ptr(productionLineID)
	item.ProductionLine = nullStringPtr(productionLineName)
	item.EquipmentID = nullInt64Ptr(equipmentID)
	item.EquipmentName = nullStringPtr(equipmentName)
	item.Description = nullStringPtr(description)
	item.ImagePath = nullStringPtr(imagePath)
	item.AssigneeID = nullInt64Ptr(assigneeID)
	item.AssigneeName = nullStringPtr(assigneeName)
	item.AcceptorID = nullInt64Ptr(acceptorID)
	item.AcceptorName = nullStringPtr(acceptorName)
	item.AssignedAt = nullTimePtr(assignedAt)
	item.ProcessedAt = nullTimePtr(processedAt)
	item.AcceptedAt = nullTimePtr(acceptedAt)
	item.ClosedAt = nullTimePtr(closedAt)
	item.ProcessResult = nullStringPtr(processResult)
	item.AcceptanceComment = nullStringPtr(acceptanceComment)
	item.CreatedAt = mustParseTime(createdAt)
	item.UpdatedAt = mustParseTime(updatedAt)
	return &item, nil
}

func scanUser(s scanner) (*User, string, error) {
	var user User
	var passwordHash string
	var isActive int
	var lastLoginAt sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(
		&user.ID,
		&user.Username,
		&passwordHash,
		&user.DisplayName,
		&user.Role,
		&isActive,
		&lastLoginAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, "", err
	}
	user.RoleLabel = roleLabels[user.Role]
	user.IsActive = isActive == 1
	if lastLoginAt.Valid {
		t := mustParseTime(lastLoginAt.String)
		user.LastLoginAt = &t
	}
	user.CreatedAt = mustParseTime(createdAt)
	user.UpdatedAt = mustParseTime(updatedAt)
	return &user, passwordHash, nil
}

func (app *App) findUser(id int64) (*User, error) {
	row := app.db.QueryRow(`
		SELECT id, username, password_hash, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	user, _, err := scanUser(row)
	return user, err
}

func (app *App) findUserByUsername(username string) (*User, string, error) {
	row := app.db.QueryRow(`
		SELECT id, username, password_hash, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE username = ?
	`, username)
	return scanUser(row)
}

func (app *App) listUsers() ([]User, error) {
	rows, err := app.db.Query(`
		SELECT id, username, password_hash, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		user, _, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	return users, rows.Err()
}

func (app *App) createUser(payload createUserRequest) (*User, error) {
	username := strings.TrimSpace(payload.Username)
	displayName := strings.TrimSpace(payload.DisplayName)
	password := strings.TrimSpace(payload.Password)
	role := strings.TrimSpace(payload.Role)
	if username == "" || displayName == "" || password == "" || role == "" {
		return nil, errors.New("用户名、显示名称、密码、角色不能为空")
	}
	if err := validateRole(role); err != nil {
		return nil, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	isActive := 1
	if payload.IsActive != nil && !*payload.IsActive {
		isActive = 0
	}
	result, err := app.db.Exec(`
		INSERT INTO users (username, password_hash, display_name, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, username, hash, displayName, role, isActive, nowString(), nowString())
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return app.findUser(id)
}

func (app *App) updateUser(id int64, payload updateUserRequest) (*User, error) {
	user, passwordHash, err := app.findUserByIDWithHash(id)
	if err != nil {
		return nil, err
	}
	displayName := user.DisplayName
	role := user.Role
	isActive := user.IsActive
	if payload.DisplayName != nil {
		displayName = strings.TrimSpace(*payload.DisplayName)
		if displayName == "" {
			return nil, errors.New("显示名称不能为空")
		}
	}
	if payload.Role != nil {
		role = strings.TrimSpace(*payload.Role)
		if err := validateRole(role); err != nil {
			return nil, err
		}
	}
	if payload.IsActive != nil {
		isActive = *payload.IsActive
	}
	if payload.Password != nil {
		if strings.TrimSpace(*payload.Password) == "" {
			return nil, errors.New("密码不能为空")
		}
		passwordHash, err = hashPassword(strings.TrimSpace(*payload.Password))
		if err != nil {
			return nil, err
		}
	}
	isActiveInt := 0
	if isActive {
		isActiveInt = 1
	}
	_, err = app.db.Exec(`
		UPDATE users
		SET password_hash = ?, display_name = ?, role = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, passwordHash, displayName, role, isActiveInt, nowString(), id)
	if err != nil {
		return nil, err
	}
	if !isActive {
		_, _ = app.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, id)
	}
	return app.findUser(id)
}

func (app *App) findUserByIDWithHash(id int64) (*User, string, error) {
	row := app.db.QueryRow(`
		SELECT id, username, password_hash, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	return scanUser(row)
}

func (app *App) ensureBootstrapAdmin() error {
	var count int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	username := firstNonEmpty(strings.TrimSpace(os.Getenv("ADMIN_USERNAME")), "admin")
	password := firstNonEmpty(strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")), "Admin@123456")
	displayName := firstNonEmpty(strings.TrimSpace(os.Getenv("ADMIN_DISPLAY_NAME")), "系统管理员")
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = app.db.Exec(`
		INSERT INTO users (username, password_hash, display_name, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 'admin', 1, ?, ?)
	`, username, hash, displayName, nowString(), nowString())
	if err == nil {
		log.Printf("已初始化默认管理员账号: %s", username)
	}
	return err
}

func (app *App) ensureDemoUsers() error {
	demoUsers := []createUserRequest{
		{Username: "inspector1", Password: "Inspector@123456", DisplayName: "质检员示例", Role: "inspector", IsActive: boolPtr(true)},
		{Username: "viewer1", Password: "Viewer@123456", DisplayName: "工人查看账号", Role: "viewer", IsActive: boolPtr(true)},
	}
	for _, item := range demoUsers {
		if _, _, err := app.findUserByUsername(item.Username); err == nil {
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := app.createUser(item); err != nil {
			return err
		}
		log.Printf("已初始化演示账号: %s", item.Username)
	}
	return nil
}

func (app *App) ensureDemoMasterData() error {
	productionLineIDs := map[string]int64{}
	for _, item := range []masterDataPayload{
		{Code: "PL-A", Name: "产线A", Description: stringPtr("冲压主线"), Status: stringPtr("active")},
		{Code: "PL-B", Name: "产线B", Description: stringPtr("焊接主线"), Status: stringPtr("active")},
		{Code: "PL-C", Name: "产线C", Description: stringPtr("喷涂主线"), Status: stringPtr("active")},
	} {
		var id int64
		err := app.db.QueryRow(`SELECT id FROM production_lines WHERE code = ? LIMIT 1`, item.Code).Scan(&id)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			created, err := app.createProductionLine(item)
			if err != nil {
				return err
			}
			id = created.ID
		}
		productionLineIDs[item.Code] = id
	}

	defectTypeIDs := map[string]int64{}
	for _, item := range []defectTypePayload{
		{Code: "DT-CRACK", Name: "裂纹", Description: stringPtr("金属或外壳裂纹类缺陷"), Severity: stringPtr("critical"), Status: stringPtr("active")},
		{Code: "DT-WELD", Name: "焊接异常", Description: stringPtr("虚焊、偏焊等焊接类异常"), Severity: stringPtr("medium"), Status: stringPtr("active")},
		{Code: "DT-SURFACE", Name: "表面瑕疵", Description: stringPtr("喷涂、气泡、划痕等外观异常"), Severity: stringPtr("low"), Status: stringPtr("active")},
	} {
		var id int64
		err := app.db.QueryRow(`SELECT id FROM defect_types WHERE code = ? LIMIT 1`, item.Code).Scan(&id)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			created, err := app.createDefectType(item)
			if err != nil {
				return err
			}
			id = created.ID
		}
		defectTypeIDs[item.Code] = id
	}

	for _, item := range []equipmentPayload{
		{Code: "EQ-A-01", Name: "冲压机1号", Description: stringPtr("产线A 冲压设备"), Status: stringPtr("active"), ProductionLineID: productionLineIDs["PL-A"], Model: stringPtr("JH21-160")},
		{Code: "EQ-B-01", Name: "焊接机器人1号", Description: stringPtr("产线B 焊接设备"), Status: stringPtr("active"), ProductionLineID: productionLineIDs["PL-B"], Model: stringPtr("KR10 R1100")},
		{Code: "EQ-C-01", Name: "喷涂机1号", Description: stringPtr("产线C 喷涂设备"), Status: stringPtr("active"), ProductionLineID: productionLineIDs["PL-C"], Model: stringPtr("PT-300")},
	} {
		var id int64
		err := app.db.QueryRow(`SELECT id FROM equipment WHERE code = ? LIMIT 1`, item.Code).Scan(&id)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if _, err := app.createEquipment(item); err != nil {
				return err
			}
		}
	}

	return nil
}

func (app *App) ensureDemoDefects() error {
	lookupID := func(query string, arg string) (*int64, error) {
		var id int64
		if err := app.db.QueryRow(query, arg).Scan(&id); err != nil {
			return nil, err
		}
		return &id, nil
	}

	crackTypeID, err := lookupID(`SELECT id FROM defect_types WHERE code = ? LIMIT 1`, "DT-CRACK")
	if err != nil {
		return err
	}
	weldTypeID, err := lookupID(`SELECT id FROM defect_types WHERE code = ? LIMIT 1`, "DT-WELD")
	if err != nil {
		return err
	}
	surfaceTypeID, err := lookupID(`SELECT id FROM defect_types WHERE code = ? LIMIT 1`, "DT-SURFACE")
	if err != nil {
		return err
	}
	lineAID, err := lookupID(`SELECT id FROM production_lines WHERE code = ? LIMIT 1`, "PL-A")
	if err != nil {
		return err
	}
	lineBID, err := lookupID(`SELECT id FROM production_lines WHERE code = ? LIMIT 1`, "PL-B")
	if err != nil {
		return err
	}
	lineCID, err := lookupID(`SELECT id FROM production_lines WHERE code = ? LIMIT 1`, "PL-C")
	if err != nil {
		return err
	}
	eqAID, err := lookupID(`SELECT id FROM equipment WHERE code = ? LIMIT 1`, "EQ-A-01")
	if err != nil {
		return err
	}
	eqBID, err := lookupID(`SELECT id FROM equipment WHERE code = ? LIMIT 1`, "EQ-B-01")
	if err != nil {
		return err
	}
	eqCID, err := lookupID(`SELECT id FROM equipment WHERE code = ? LIMIT 1`, "EQ-C-01")
	if err != nil {
		return err
	}

	demoDefects := []struct {
		name             string
		defectType       string
		defectTypeID     *int64
		severity         string
		location         string
		description      string
		productionLineID *int64
		equipmentID      *int64
	}{
		{name: "冲压面板边缘裂纹", defectType: "裂纹", defectTypeID: crackTypeID, severity: "critical", location: "产线A-工位3", description: "冲压后边缘区域发现连续裂纹，需立即停线复核。", productionLineID: lineAID, equipmentID: eqAID},
		{name: "焊点虚焊偏移", defectType: "焊接异常", defectTypeID: weldTypeID, severity: "medium", location: "产线B-焊接区", description: "焊点存在虚焊和轻微偏移，建议复检并返工。", productionLineID: lineBID, equipmentID: eqBID},
		{name: "涂层气泡不均", defectType: "表面瑕疵", defectTypeID: surfaceTypeID, severity: "low", location: "产线C-喷涂段", description: "表面出现零星气泡，影响外观一致性。", productionLineID: lineCID, equipmentID: eqCID},
	}
	createdCount := 0
	for _, item := range demoDefects {
		var id int64
		err := app.db.QueryRow(`SELECT id FROM defects WHERE name = ? LIMIT 1`, item.name).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			created, err := app.insertDefect(item.name, item.defectType, item.severity, stringPtr(item.location), stringPtr(item.description), nil)
			if err != nil {
				return err
			}
			id = created.ID
			createdCount++
		} else if err != nil {
			return err
		}
		_, err = app.db.Exec(`
			UPDATE defects
			SET defect_type = ?, defect_type_id = ?, production_line_id = ?, equipment_id = ?, severity = ?, status = 'open', location = ?, description = ?, updated_at = ?
			WHERE id = ?
		`, item.defectType, nullInt64Value(item.defectTypeID), nullInt64Value(item.productionLineID), nullInt64Value(item.equipmentID), item.severity, item.location, item.description, nowString(), id)
		if err != nil {
			return err
		}
	}
	if createdCount > 0 {
		log.Printf("已初始化 %d 条示例缺陷", createdCount)
	}
	return nil
}

func (app *App) createSession(userID int64, token string) error {
	_, err := app.db.Exec(`
		INSERT INTO sessions (user_id, token_hash, expires_at, created_at, last_used_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, hashToken(token), time.Now().Add(sessionDuration).Format(time.RFC3339), nowString(), nowString())
	return err
}

func (app *App) findSession(token string) (*sessionRecord, error) {
	row := app.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.display_name, u.role, u.is_active, u.last_login_at, u.created_at, u.updated_at, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ?
	`, hashToken(token))
	var expiresAt string
	var user User
	var passwordHash string
	var isActive int
	var lastLoginAt sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&passwordHash,
		&user.DisplayName,
		&user.Role,
		&isActive,
		&lastLoginAt,
		&createdAt,
		&updatedAt,
		&expiresAt,
	); err != nil {
		return nil, err
	}
	user.RoleLabel = roleLabels[user.Role]
	user.IsActive = isActive == 1
	if !user.IsActive {
		return nil, sql.ErrNoRows
	}
	if lastLoginAt.Valid {
		t := mustParseTime(lastLoginAt.String)
		user.LastLoginAt = &t
	}
	user.CreatedAt = mustParseTime(createdAt)
	user.UpdatedAt = mustParseTime(updatedAt)
	_, _ = app.db.Exec(`UPDATE sessions SET last_used_at = ? WHERE token_hash = ?`, nowString(), hashToken(token))
	return &sessionRecord{User: &user, ExpiresAt: mustParseTime(expiresAt)}, nil
}

func (app *App) deleteSession(token string) error {
	if token == "" {
		return nil
	}
	_, err := app.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

func currentUser(ctx context.Context) *User {
	user, _ := ctx.Value(currentUserKey).(*User)
	return user
}

func currentSessionToken(ctx context.Context) string {
	token, _ := ctx.Value(currentSessionTokenKey).(string)
	return token
}

func bearerTokenFromHeader(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", errors.New("缺少登录凭证")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("登录凭证格式无效")
	}
	return strings.TrimSpace(parts[1]), nil
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

func validateRole(role string) error {
	if _, ok := roleLabels[role]; !ok {
		return errors.New("角色不合法")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullTimePtr(value sql.NullString) *time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	t := mustParseTime(value.String)
	return &t
}

func boolPtr(value bool) *bool {
	return &value
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func int64Ptr(value int64) *int64 {
	return &value
}

func userDisplayName(user *User) string {
	if user == nil {
		return ""
	}
	return firstNonEmpty(strings.TrimSpace(user.DisplayName), strings.TrimSpace(user.Username))
}

func roleOfUser(user *User) string {
	if user == nil {
		return ""
	}
	return user.Role
}

func moduleFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/auth"):
		return "auth"
	case strings.HasPrefix(path, "/api/users"):
		return "users"
	case strings.HasPrefix(path, "/api/operation-logs"):
		return "operation_logs"
	case strings.HasPrefix(path, "/api/production-lines"):
		return "production_lines"
	case strings.HasPrefix(path, "/api/equipment"):
		return "equipment"
	case strings.HasPrefix(path, "/api/defect-types"):
		return "defect_types"
	case strings.HasPrefix(path, "/api/defects"):
		return "defects"
	default:
		return "system"
	}
}

func targetTypeFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/users"):
		return "user"
	case strings.HasPrefix(path, "/api/production-lines"):
		return "production_line"
	case strings.HasPrefix(path, "/api/equipment"):
		return "equipment"
	case strings.HasPrefix(path, "/api/defect-types"):
		return "defect_type"
	case strings.HasPrefix(path, "/api/defects"):
		return "defect"
	default:
		return "resource"
	}
}

func withUserContext(r *http.Request, user *User) *http.Request {
	if user == nil {
		return r
	}
	ctx := context.WithValue(r.Context(), currentUserKey, user)
	return r.WithContext(ctx)
}

func (app *App) writeOperationLog(r *http.Request, entry operationLogEntry) {
	if r == nil {
		return
	}

	actorType := firstNonEmpty(strings.TrimSpace(entry.ActorType), "user")
	var actorUserID any
	actorUsername := ""
	actorDisplayName := ""
	actorRole := ""
	if user := currentUser(r.Context()); user != nil {
		actorUserID = user.ID
		actorUsername = user.Username
		actorDisplayName = user.DisplayName
		actorRole = user.Role
	}

	detailsJSON := ""
	if len(entry.Details) > 0 {
		if encoded, err := json.Marshal(entry.Details); err == nil {
			detailsJSON = string(encoded)
		}
	}

	_, err := app.db.Exec(`
		INSERT INTO operation_logs (
			actor_user_id, actor_username, actor_display_name, actor_role, actor_type,
			module, action, target_type, target_id, target_name, result, summary,
			details_json, ip_address, user_agent, request_method, request_path, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, actorUserID, actorUsername, actorDisplayName, actorRole, actorType, entry.Module, entry.Action, entry.TargetType, entry.TargetID, entry.TargetName, entry.Result, entry.Summary, detailsJSON, clientIP(r), r.UserAgent(), r.Method, r.URL.Path, nowString())
	if err != nil {
		log.Printf("写入操作日志失败: %v", err)
	}
}

func (app *App) handleOperationLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	items, total, err := app.listOperationLogs(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询操作日志失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (app *App) listOperationLogs(r *http.Request) ([]OperationLog, int, error) {
	where := []string{"1=1"}
	args := []any{}
	queryValues := r.URL.Query()

	if module := strings.TrimSpace(queryValues.Get("module")); module != "" {
		where = append(where, "module = ?")
		args = append(args, module)
	}
	if action := strings.TrimSpace(queryValues.Get("action")); action != "" {
		where = append(where, "action = ?")
		args = append(args, action)
	}
	if result := strings.TrimSpace(queryValues.Get("result")); result != "" {
		where = append(where, "result = ?")
		args = append(args, result)
	}
	if actorUsername := strings.TrimSpace(queryValues.Get("actor_username")); actorUsername != "" {
		where = append(where, "actor_username LIKE ?")
		args = append(args, "%"+actorUsername+"%")
	}
	if targetType := strings.TrimSpace(queryValues.Get("target_type")); targetType != "" {
		where = append(where, "target_type = ?")
		args = append(args, targetType)
	}
	if targetIDRaw := strings.TrimSpace(queryValues.Get("target_id")); targetIDRaw != "" {
		targetID, err := strconv.ParseInt(targetIDRaw, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		where = append(where, "target_id = ?")
		args = append(args, targetID)
	}
	if keyword := strings.TrimSpace(queryValues.Get("keyword")); keyword != "" {
		where = append(where, "(summary LIKE ? OR target_name LIKE ? OR details_json LIKE ?)")
		keywordPattern := "%" + keyword + "%"
		args = append(args, keywordPattern, keywordPattern, keywordPattern)
	}
	if startTime := strings.TrimSpace(queryValues.Get("start_time")); startTime != "" {
		where = append(where, "datetime(created_at) >= datetime(?)")
		args = append(args, startTime)
	}
	if endTime := strings.TrimSpace(queryValues.Get("end_time")); endTime != "" {
		where = append(where, "datetime(created_at) <= datetime(?)")
		args = append(args, endTime)
	}

	page := parsePositiveInt(queryValues.Get("page"), 1)
	pageSize := parsePositiveInt(queryValues.Get("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := app.db.QueryRow("SELECT COUNT(*) FROM operation_logs WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, pageSize, (page-1)*pageSize)
	rows, err := app.db.Query(`
		SELECT id, actor_user_id, actor_username, actor_display_name, actor_role, actor_type,
			module, action, target_type, target_id, target_name, result, summary,
			details_json, ip_address, user_agent, request_method, request_path, created_at
		FROM operation_logs
		WHERE `+whereSQL+`
		ORDER BY datetime(created_at) DESC, id DESC
		LIMIT ? OFFSET ?
	`, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []OperationLog{}
	for rows.Next() {
		item, err := scanOperationLog(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func scanOperationLog(s scanner) (*OperationLog, error) {
	var item OperationLog
	var actorUserID, targetID sql.NullInt64
	var actorUsername, actorDisplayName, actorRole, targetType, targetName sql.NullString
	var detailsJSON, ipAddress, userAgent, requestMethod, requestPath sql.NullString
	var createdAt string

	if err := s.Scan(
		&item.ID,
		&actorUserID,
		&actorUsername,
		&actorDisplayName,
		&actorRole,
		&item.ActorType,
		&item.Module,
		&item.Action,
		&targetType,
		&targetID,
		&targetName,
		&item.Result,
		&item.Summary,
		&detailsJSON,
		&ipAddress,
		&userAgent,
		&requestMethod,
		&requestPath,
		&createdAt,
	); err != nil {
		return nil, err
	}

	if actorUserID.Valid {
		item.ActorUserID = int64Ptr(actorUserID.Int64)
	}
	item.ActorUsername = actorUsername.String
	item.ActorDisplayName = actorDisplayName.String
	item.ActorRole = actorRole.String
	item.TargetType = targetType.String
	if targetID.Valid {
		item.TargetID = int64Ptr(targetID.Int64)
	}
	item.TargetName = targetName.String
	item.IPAddress = ipAddress.String
	item.UserAgent = userAgent.String
	item.RequestMethod = requestMethod.String
	item.RequestPath = requestPath.String
	item.CreatedAt = mustParseTime(createdAt)
	if detailsJSON.Valid && strings.TrimSpace(detailsJSON.String) != "" {
		_ = json.Unmarshal([]byte(detailsJSON.String), &item.Details)
	}
	return &item, nil
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func clientIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return ""
	}
	if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}


func scanDefectFlowLogs(rows *sql.Rows) ([]DefectFlowLog, error) {
	items := []DefectFlowLog{}
	for rows.Next() {
		var item DefectFlowLog
		var comment sql.NullString
		var createdAt string
		if err := rows.Scan(&item.ID, &item.DefectID, &item.Action, &item.FromStatus, &item.ToStatus, &item.OperatorID, &item.OperatorName, &item.OperatorRole, &comment, &createdAt); err != nil {
			return nil, err
		}
		item.Comment = nullStringPtr(comment)
		item.CreatedAt = mustParseTime(createdAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (app *App) handleDefectFlowLogs(w http.ResponseWriter, r *http.Request, id int64) {
	rows, err := app.db.Query(`
		SELECT id, defect_id, action, from_status, to_status, operator_id, operator_name, operator_role, comment, created_at
		FROM defect_flow_logs WHERE defect_id = ?
		ORDER BY datetime(created_at) ASC, id ASC
	`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询流转记录失败")
		return
	}
	defer rows.Close()
	items, err := scanDefectFlowLogs(rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取流转记录失败")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (app *App) handleDefectAction(w http.ResponseWriter, r *http.Request, id int64) {
	var payload DefectActionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	updated, err := app.applyDefectAction(currentUser(r.Context()), id, payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷记录不存在")
			return
		}
		if strings.Contains(err.Error(), "权限") || strings.Contains(err.Error(), "仅") {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated.AvailableActions = app.availableDefectActions(currentUser(r.Context()), updated)
	writeJSON(w, http.StatusOK, updated)
}

func (app *App) applyDefectAction(user *User, id int64, payload DefectActionRequest) (*Defect, error) {
	if user == nil {
		return nil, errors.New("缺少登录凭证")
	}
	defect, err := app.findDefect(id)
	if err != nil {
		return nil, err
	}
	action := strings.TrimSpace(payload.Action)
	nextStatus, err := nextDefectStatus(defect.Status, action)
	if err != nil {
		return nil, err
	}
	if err := app.validateDefectActionPermission(user, defect, action); err != nil {
		return nil, err
	}
	now := nowString()
	assigneeID := defect.AssigneeID
	assigneeName := defect.AssigneeName
	acceptorID := defect.AcceptorID
	acceptorName := defect.AcceptorName
	assignedAt := ptrTimeString(defect.AssignedAt)
	processedAt := ptrTimeString(defect.ProcessedAt)
	acceptedAt := ptrTimeString(defect.AcceptedAt)
	closedAt := ptrTimeString(defect.ClosedAt)
	processResult := ptrValue(defect.ProcessResult)
	acceptanceComment := ptrValue(defect.AcceptanceComment)

	switch action {
	case "assign", "reassign":
		if payload.AssigneeID == nil {
			return nil, errors.New("请选择处理人")
		}
		assignee, err := app.findUser(*payload.AssigneeID)
		if err != nil {
			return nil, errors.New("处理人不存在")
		}
		assigneeID = int64Ptr(assignee.ID)
		assigneeName = stringPtr(assignee.DisplayName)
		if payload.AcceptorID != nil {
			acceptor, err := app.findUser(*payload.AcceptorID)
			if err != nil {
				return nil, errors.New("验收人不存在")
			}
			acceptorID = int64Ptr(acceptor.ID)
			acceptorName = stringPtr(acceptor.DisplayName)
		}
		assignedAt = now
		processedAt = ""
		acceptedAt = ""
		closedAt = ""
	case "submit_for_acceptance":
		result := strings.TrimSpace(ptrValue(payload.ProcessResult))
		if result == "" {
			return nil, errors.New("请填写处理结果")
		}
		processResult = result
		processedAt = now
	case "accept":
		acceptedAt = now
		acceptanceComment = strings.TrimSpace(ptrValue(payload.AcceptanceComment))
	case "reject":
		feedback := strings.TrimSpace(ptrValue(payload.AcceptanceComment))
		if feedback == "" {
			return nil, errors.New("请填写退回原因")
		}
		acceptanceComment = feedback
		acceptedAt = ""
	case "close":
		closedAt = now
	}

	_, err = app.db.Exec(`
		UPDATE defects
		SET status = ?, assignee_id = ?, assignee_name = ?, acceptor_id = ?, acceptor_name = ?,
			assigned_at = ?, processed_at = ?, accepted_at = ?, closed_at = ?, process_result = ?, acceptance_comment = ?, updated_at = ?
		WHERE id = ?
	`, nextStatus, nullInt64Value(assigneeID), ptrValue(assigneeName), nullInt64Value(acceptorID), ptrValue(acceptorName), assignedAt, processedAt, acceptedAt, closedAt, processResult, acceptanceComment, now, id)
	if err != nil {
		return nil, err
	}
	comment := firstNonEmpty(strings.TrimSpace(ptrValue(payload.Comment)), strings.TrimSpace(ptrValue(payload.AcceptanceComment)), strings.TrimSpace(ptrValue(payload.ProcessResult)))
	if err := app.insertDefectFlowLog(id, action, defect.Status, nextStatus, user, nullableString(comment)); err != nil {
		return nil, err
	}
	return app.findDefect(id)
}

func (app *App) insertDefectFlowLog(defectID int64, action, fromStatus, toStatus string, user *User, comment *string) error {
	_, err := app.db.Exec(`
		INSERT INTO defect_flow_logs (defect_id, action, from_status, to_status, operator_id, operator_name, operator_role, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, defectID, action, fromStatus, toStatus, user.ID, userDisplayName(user), user.Role, ptrValue(comment), nowString())
	return err
}

func nextDefectStatus(currentStatus, action string) (string, error) {
	transitions, ok := defectActionTransitions[action]
	if !ok {
		return "", errors.New("不支持的流程操作")
	}
	nextStatus, ok := transitions[currentStatus]
	if !ok {
		return "", errors.New("当前状态不允许执行该操作")
	}
	return nextStatus, nil
}

func (app *App) validateDefectActionPermission(user *User, defect *Defect, action string) error {
	if user == nil {
		return errors.New("缺少登录凭证")
	}
	if user.Role == "admin" {
		return nil
	}
	if user.Role != "inspector" {
		return errors.New("当前账号没有权限执行该操作")
	}
	switch action {
	case "assign", "reassign", "close":
		return errors.New("当前账号没有权限执行该操作")
	case "start_process", "submit_for_acceptance":
		if defect.AssigneeID == nil || *defect.AssigneeID != user.ID {
			return errors.New("仅处理人可执行该操作")
		}
	case "accept", "reject":
		if defect.AcceptorID == nil || *defect.AcceptorID != user.ID {
			return errors.New("仅验收人可执行该操作")
		}
	}
	return nil
}

func (app *App) availableDefectActions(user *User, defect *Defect) []string {
	items := []string{}
	for action, transitions := range defectActionTransitions {
		if _, ok := transitions[defect.Status]; !ok {
			continue
		}
		if app.validateDefectActionPermission(user, defect, action) == nil {
			items = append(items, action)
		}
	}
	sort.Strings(items)
	return items
}

func ptrTimeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func nullInt64Value(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
