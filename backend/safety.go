package main

import (
	"database/sql"
	"errors"
	"net/http"
)

var errDefectConflict = errors.New("缺陷已被其他操作更新，请刷新后重试")

// Claim the version in the same transaction as the state update and flow log.
func advanceDefectVersion(tx *sql.Tx, id, version int64) error {
	result, err := tx.Exec(`UPDATE defects SET workflow_version = workflow_version + 1 WHERE id = ? AND workflow_version = ?`, id, version)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errDefectConflict
	}
	return nil
}

func (app *App) replacePassword(id int64, hash string) error {
	tx, err := app.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, nowString(), id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func writeBodyError(w http.ResponseWriter, err error, message string) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "请求内容过大")
		return
	}
	writeError(w, http.StatusBadRequest, message)
}
