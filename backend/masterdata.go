package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
)

func (app *App) handleProductionLines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := app.listProductionLines(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "查询产线失败")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var payload masterDataPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "JSON 格式不正确")
			return
		}
		item, err := app.createProductionLine(payload)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleProductionLineByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/production-lines/")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusBadRequest, "产线ID无效")
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	var payload masterDataPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	item, err := app.updateProductionLine(id, payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "产线不存在")
			return
		}
		status := http.StatusBadRequest
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (app *App) handleEquipment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := app.listEquipment(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "查询设备失败")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var payload equipmentPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "JSON 格式不正确")
			return
		}
		item, err := app.createEquipment(payload)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleEquipmentByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/equipment/")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusBadRequest, "设备ID无效")
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	var payload equipmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	item, err := app.updateEquipment(id, payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "设备不存在")
			return
		}
		status := http.StatusBadRequest
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (app *App) handleDefectTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := app.listDefectTypes(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "查询缺陷类型失败")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var payload defectTypePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "JSON 格式不正确")
			return
		}
		item, err := app.createDefectType(payload)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (app *App) handleDefectTypeByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/defect-types/")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusBadRequest, "缺陷类型ID无效")
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	var payload defectTypePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式不正确")
		return
	}
	item, err := app.updateDefectType(id, payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷类型不存在")
			return
		}
		status := http.StatusBadRequest
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func parseIDFromPath(path, prefix string) (int64, error) {
	idStr := strings.TrimPrefix(path, prefix)
	if idStr == "" || strings.Contains(strings.Trim(idStr, "/"), "/") {
		return 0, sql.ErrNoRows
	}
	id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func normalizeMasterStatus(status *string) string {
	if status == nil || strings.TrimSpace(*status) == "" {
		return "active"
	}
	value := strings.TrimSpace(*status)
	if value != "active" && value != "inactive" {
		return ""
	}
	return value
}

func validateCodeName(code, name string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" {
		return errors.New("编码和名称不能为空")
	}
	return nil
}

func parseOptionalInt64(value string) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (app *App) validateDefectMasterData(defectTypeID, productionLineID, equipmentID *int64) (*DefectType, *ProductionLine, *Equipment, error) {
	var defectType *DefectType
	var productionLine *ProductionLine
	var equipment *Equipment

	if defectTypeID != nil {
		item, err := app.findDefectType(*defectTypeID)
		if err != nil || item.Status != "active" {
			return nil, nil, nil, errors.New("缺陷类型不存在或已停用")
		}
		defectType = item
	}
	if productionLineID != nil {
		item, err := app.findProductionLine(*productionLineID)
		if err != nil || item.Status != "active" {
			return nil, nil, nil, errors.New("产线不存在或已停用")
		}
		productionLine = item
	}
	if equipmentID != nil {
		item, err := app.findEquipment(*equipmentID)
		if err != nil || item.Status != "active" {
			return nil, nil, nil, errors.New("设备不存在或已停用")
		}
		equipment = item
		if productionLine != nil && equipment.ProductionLineID != productionLine.ID {
			return nil, nil, nil, errors.New("设备不属于所选产线")
		}
	}

	return defectType, productionLine, equipment, nil
}

func (app *App) createProductionLine(payload masterDataPayload) (*ProductionLine, error) {
	code := strings.TrimSpace(payload.Code)
	name := strings.TrimSpace(payload.Name)
	if err := validateCodeName(code, name); err != nil {
		return nil, err
	}
	status := normalizeMasterStatus(payload.Status)
	if status == "" {
		return nil, errors.New("状态不合法")
	}
	res, err := app.db.Exec(`INSERT INTO production_lines (code, name, description, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, code, name, ptrValue(payload.Description), status, nowString(), nowString())
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return app.findProductionLine(id)
}

func (app *App) updateProductionLine(id int64, payload masterDataPayload) (*ProductionLine, error) {
	item, err := app.findProductionLine(id)
	if err != nil {
		return nil, err
	}
	code, name := item.Code, item.Name
	description, status := item.Description, item.Status
	if strings.TrimSpace(payload.Code) != "" {
		code = strings.TrimSpace(payload.Code)
	}
	if strings.TrimSpace(payload.Name) != "" {
		name = strings.TrimSpace(payload.Name)
	}
	if payload.Description != nil {
		description = payload.Description
	}
	if payload.Status != nil {
		status = normalizeMasterStatus(payload.Status)
		if status == "" {
			return nil, errors.New("状态不合法")
		}
	}
	if err := validateCodeName(code, name); err != nil {
		return nil, err
	}
	_, err = app.db.Exec(`UPDATE production_lines SET code = ?, name = ?, description = ?, status = ?, updated_at = ? WHERE id = ?`, code, name, ptrValue(description), status, nowString(), id)
	if err != nil {
		return nil, err
	}
	return app.findProductionLine(id)
}

func (app *App) listProductionLines(r *http.Request) ([]ProductionLine, error) {
	query := `SELECT id, code, name, description, status, created_at, updated_at FROM production_lines WHERE 1=1`
	args := []any{}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(r.URL.Query().Get("keyword")); keyword != "" {
		query += ` AND (code LIKE ? OR name LIKE ?)`
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	query += ` ORDER BY id DESC`
	rows, err := app.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ProductionLine
	for rows.Next() {
		item, err := scanProductionLine(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (app *App) findProductionLine(id int64) (*ProductionLine, error) {
	row := app.db.QueryRow(`SELECT id, code, name, description, status, created_at, updated_at FROM production_lines WHERE id = ?`, id)
	return scanProductionLine(row)
}

func (app *App) createEquipment(payload equipmentPayload) (*Equipment, error) {
	code := strings.TrimSpace(payload.Code)
	name := strings.TrimSpace(payload.Name)
	if err := validateCodeName(code, name); err != nil {
		return nil, err
	}
	if payload.ProductionLineID <= 0 {
		return nil, errors.New("请选择所属产线")
	}
	line, err := app.findProductionLine(payload.ProductionLineID)
	if err != nil || line.Status != "active" {
		return nil, errors.New("所属产线不存在或已停用")
	}
	status := normalizeMasterStatus(payload.Status)
	if status == "" {
		return nil, errors.New("状态不合法")
	}
	res, err := app.db.Exec(`INSERT INTO equipment (code, name, description, status, production_line_id, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, code, name, ptrValue(payload.Description), status, payload.ProductionLineID, ptrValue(payload.Model), nowString(), nowString())
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return app.findEquipment(id)
}

func (app *App) updateEquipment(id int64, payload equipmentPayload) (*Equipment, error) {
	item, err := app.findEquipment(id)
	if err != nil {
		return nil, err
	}
	code, name := item.Code, item.Name
	description, status, lineID, model := item.Description, item.Status, item.ProductionLineID, item.Model
	if strings.TrimSpace(payload.Code) != "" {
		code = strings.TrimSpace(payload.Code)
	}
	if strings.TrimSpace(payload.Name) != "" {
		name = strings.TrimSpace(payload.Name)
	}
	if payload.Description != nil {
		description = payload.Description
	}
	if payload.Status != nil {
		status = normalizeMasterStatus(payload.Status)
		if status == "" {
			return nil, errors.New("状态不合法")
		}
	}
	if payload.ProductionLineID > 0 {
		lineID = payload.ProductionLineID
	}
	if payload.Model != nil {
		model = payload.Model
	}
	line, err := app.findProductionLine(lineID)
	if err != nil || line.Status != "active" {
		return nil, errors.New("所属产线不存在或已停用")
	}
	if err := validateCodeName(code, name); err != nil {
		return nil, err
	}
	_, err = app.db.Exec(`UPDATE equipment SET code = ?, name = ?, description = ?, status = ?, production_line_id = ?, model = ?, updated_at = ? WHERE id = ?`, code, name, ptrValue(description), status, lineID, ptrValue(model), nowString(), id)
	if err != nil {
		return nil, err
	}
	return app.findEquipment(id)
}

func (app *App) listEquipment(r *http.Request) ([]Equipment, error) {
	query := `SELECT e.id, e.code, e.name, e.description, e.status, e.production_line_id, p.name, e.model, e.created_at, e.updated_at FROM equipment e LEFT JOIN production_lines p ON p.id = e.production_line_id WHERE 1=1`
	args := []any{}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		query += ` AND e.status = ?`
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(r.URL.Query().Get("keyword")); keyword != "" {
		query += ` AND (e.code LIKE ? OR e.name LIKE ?)`
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	if lineID := strings.TrimSpace(r.URL.Query().Get("production_line_id")); lineID != "" {
		query += ` AND e.production_line_id = ?`
		args = append(args, lineID)
	}
	query += ` ORDER BY e.id DESC`
	rows, err := app.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Equipment
	for rows.Next() {
		item, err := scanEquipment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (app *App) findEquipment(id int64) (*Equipment, error) {
	row := app.db.QueryRow(`SELECT e.id, e.code, e.name, e.description, e.status, e.production_line_id, p.name, e.model, e.created_at, e.updated_at FROM equipment e LEFT JOIN production_lines p ON p.id = e.production_line_id WHERE e.id = ?`, id)
	return scanEquipment(row)
}

func (app *App) createDefectType(payload defectTypePayload) (*DefectType, error) {
	code := strings.TrimSpace(payload.Code)
	name := strings.TrimSpace(payload.Name)
	if err := validateCodeName(code, name); err != nil {
		return nil, err
	}
	status := normalizeMasterStatus(payload.Status)
	if status == "" {
		return nil, errors.New("状态不合法")
	}
	var severity string
	if payload.Severity != nil && strings.TrimSpace(*payload.Severity) != "" {
		normalized, err := normalizeSeverity(*payload.Severity)
		if err != nil {
			return nil, err
		}
		severity = normalized
	}
	res, err := app.db.Exec(`INSERT INTO defect_types (code, name, description, severity, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, code, name, ptrValue(payload.Description), severity, status, nowString(), nowString())
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return app.findDefectType(id)
}

func (app *App) updateDefectType(id int64, payload defectTypePayload) (*DefectType, error) {
	item, err := app.findDefectType(id)
	if err != nil {
		return nil, err
	}
	code, name := item.Code, item.Name
	description, status, severity := item.Description, item.Status, item.Severity
	if strings.TrimSpace(payload.Code) != "" {
		code = strings.TrimSpace(payload.Code)
	}
	if strings.TrimSpace(payload.Name) != "" {
		name = strings.TrimSpace(payload.Name)
	}
	if payload.Description != nil {
		description = payload.Description
	}
	if payload.Status != nil {
		status = normalizeMasterStatus(payload.Status)
		if status == "" {
			return nil, errors.New("状态不合法")
		}
	}
	if payload.Severity != nil {
		if strings.TrimSpace(*payload.Severity) == "" {
			severity = nil
		} else {
			normalized, err := normalizeSeverity(*payload.Severity)
			if err != nil {
				return nil, err
			}
			severity = stringPtr(normalized)
		}
	}
	if err := validateCodeName(code, name); err != nil {
		return nil, err
	}
	_, err = app.db.Exec(`UPDATE defect_types SET code = ?, name = ?, description = ?, severity = ?, status = ?, updated_at = ? WHERE id = ?`, code, name, ptrValue(description), ptrValue(severity), status, nowString(), id)
	if err != nil {
		return nil, err
	}
	return app.findDefectType(id)
}

func (app *App) listDefectTypes(r *http.Request) ([]DefectType, error) {
	query := `SELECT id, code, name, description, severity, status, created_at, updated_at FROM defect_types WHERE 1=1`
	args := []any{}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(r.URL.Query().Get("keyword")); keyword != "" {
		query += ` AND (code LIKE ? OR name LIKE ?)`
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	query += ` ORDER BY id DESC`
	rows, err := app.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DefectType
	for rows.Next() {
		item, err := scanDefectType(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (app *App) findDefectType(id int64) (*DefectType, error) {
	row := app.db.QueryRow(`SELECT id, code, name, description, severity, status, created_at, updated_at FROM defect_types WHERE id = ?`, id)
	return scanDefectType(row)
}

func scanProductionLine(s scanner) (*ProductionLine, error) {
	var item ProductionLine
	var description sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(&item.ID, &item.Code, &item.Name, &description, &item.Status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	item.Description = nullStringPtr(description)
	item.CreatedAt = mustParseTime(createdAt)
	item.UpdatedAt = mustParseTime(updatedAt)
	return &item, nil
}

func scanEquipment(s scanner) (*Equipment, error) {
	var item Equipment
	var description, lineName, model sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(&item.ID, &item.Code, &item.Name, &description, &item.Status, &item.ProductionLineID, &lineName, &model, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	item.Description = nullStringPtr(description)
	item.ProductionLineName = nullStringPtr(lineName)
	item.Model = nullStringPtr(model)
	item.CreatedAt = mustParseTime(createdAt)
	item.UpdatedAt = mustParseTime(updatedAt)
	return &item, nil
}

func scanDefectType(s scanner) (*DefectType, error) {
	var item DefectType
	var description, severity sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(&item.ID, &item.Code, &item.Name, &description, &severity, &item.Status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	item.Description = nullStringPtr(description)
	item.Severity = nullStringPtr(severity)
	item.CreatedAt = mustParseTime(createdAt)
	item.UpdatedAt = mustParseTime(updatedAt)
	return &item, nil
}

func summarizeUserAction(action string, user *User) string {
	name := "用户"
	if user != nil {
		name = firstNonEmpty(user.DisplayName, user.Username)
	}
	switch action {
	case "create":
		return fmt.Sprintf("创建用户 %s", name)
	case "disable":
		return fmt.Sprintf("停用用户 %s", name)
	case "enable":
		return fmt.Sprintf("启用用户 %s", name)
	default:
		return fmt.Sprintf("更新用户 %s", name)
	}
}

func diffUsers(before, after *User, passwordChanged bool) map[string]any {
	result := map[string]any{}
	if before == nil || after == nil {
		return result
	}
	if before.DisplayName != after.DisplayName {
		result["display_name"] = map[string]any{"before": before.DisplayName, "after": after.DisplayName}
	}
	if before.Role != after.Role {
		result["role"] = map[string]any{"before": before.Role, "after": after.Role}
	}
	if before.IsActive != after.IsActive {
		result["is_active"] = map[string]any{"before": before.IsActive, "after": after.IsActive}
	}
	if passwordChanged {
		result["password"] = "changed"
	}
	return result
}

func (app *App) handleDefectExportPDF(w http.ResponseWriter, r *http.Request, id int64) {
	_ = r
	defect, err := app.findDefect(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "缺陷记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "生成 PDF 报告失败")
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8Font(pdfChineseFontName, "", pdfChineseFontPath)
	pdf.SetFont(pdfChineseFontName, "", 16)
	pdf.SetTitle(reportSystemName+" - 缺陷详情报告", false)
	pdf.AddPage()
	pdf.CellFormat(190, 10, reportSystemName+" - 缺陷详情报告", "", 1, "C", false, 0, "")
	pdf.SetFont(pdfChineseFontName, "", 11)
	pdf.CellFormat(190, 8, fmt.Sprintf("导出时间：%s", time.Now().Format("2006-01-02 15:04:05")), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	lines := []struct{ label, value string }{
		{"缺陷ID", fmt.Sprintf("%d", defect.ID)},
		{"标题", defect.Name},
		{"类型", defect.DefectType},
		{"等级", severityLabelMapForExport(defect.Severity)},
		{"状态", statusLabelForExport(defect.Status)},
		{"位置", firstNonEmpty(ptrValue(defect.Location), "-")},
		{"创建时间", defect.CreatedAt.Format("2006-01-02 15:04:05")},
		{"更新时间", defect.UpdatedAt.Format("2006-01-02 15:04:05")},
	}
	for _, line := range lines {
		pdf.SetFont(pdfChineseFontName, "", 11)
		pdf.CellFormat(32, 8, line.label, "1", 0, "L", false, 0, "")
		pdf.CellFormat(158, 8, line.value, "1", 1, "L", false, 0, "")
	}
	pdf.Ln(3)
	pdf.SetFont(pdfChineseFontName, "", 12)
	pdf.CellFormat(190, 8, "缺陷说明", "", 1, "L", false, 0, "")
	pdf.SetFont(pdfChineseFontName, "", 11)
	pdf.MultiCell(190, 7, firstNonEmpty(ptrValue(defect.Description), "暂无描述"), "1", "L", false)
	if defect.ImagePath != nil && *defect.ImagePath != "" {
		pdf.Ln(3)
		pdf.SetFont(pdfChineseFontName, "", 12)
		pdf.CellFormat(190, 8, "缺陷图片", "", 1, "L", false, 0, "")
		imageFile := filepath.Join(uploadDir, filepath.Base(*defect.ImagePath))
		if _, statErr := os.Stat(imageFile); statErr == nil {
			opts := gofpdf.ImageOptions{ImageType: strings.TrimPrefix(strings.ToUpper(filepath.Ext(imageFile)), "."), ReadDpi: true}
			pdf.ImageOptions(imageFile, 20, pdf.GetY(), 80, 0, false, opts, 0, "")
			pdf.Ln(62)
		} else {
			pdf.MultiCell(190, 7, "图片路径："+*defect.ImagePath, "1", "L", false)
		}
	}

	buffer := &bytes.Buffer{}
	if err := pdf.Output(buffer); err != nil {
		writeError(w, http.StatusInternalServerError, "生成 PDF 报告失败")
		return
	}
	filename := fmt.Sprintf("defect_%d_report.pdf", id)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buffer.Bytes())
}
