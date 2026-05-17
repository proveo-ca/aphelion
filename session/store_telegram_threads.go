//go:build linux

package session

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/core"
)

type TelegramThreadStatus string

const (
	TelegramThreadStatusOpen   TelegramThreadStatus = "open"
	TelegramThreadStatusClosed TelegramThreadStatus = "closed"
)

type TelegramThread struct {
	ChatID              int64
	ThreadID            int64
	Status              TelegramThreadStatus
	CreatedBySenderID   int64
	CreatedFromUpdateID int64
	CreatedMessageID    int64
	CreatedText         string
	LastActivityAt      time.Time
	ClosedAt            time.Time
	AbsorbSummary       string
	AbsorbedAt          time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (t TelegramThread) Open() bool {
	return normalizeTelegramThreadStatus(t.Status) == TelegramThreadStatusOpen
}

func (s *SQLiteStore) CreateTelegramThreadForUpdate(chatID int64, senderID int64, updateID int64, messageID int64, text string, now time.Time) (TelegramThread, bool, error) {
	if chatID == 0 {
		return TelegramThread{}, false, fmt.Errorf("telegram thread chat id is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if updateID > 0 {
		thread, ok, err := s.TelegramThreadByCreatedUpdate(chatID, updateID)
		if err != nil {
			return TelegramThread{}, false, err
		}
		if ok {
			return thread, false, nil
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return TelegramThread{}, false, fmt.Errorf("begin telegram thread create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if updateID > 0 {
		thread, ok, err := telegramThreadByCreatedUpdateTx(tx, chatID, updateID)
		if err != nil {
			return TelegramThread{}, false, err
		}
		if ok {
			if err := tx.Commit(); err != nil {
				return TelegramThread{}, false, fmt.Errorf("commit telegram thread lookup: %w", err)
			}
			return thread, false, nil
		}
	}

	var nextThreadID int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(thread_id), 0) + 1 FROM telegram_threads WHERE chat_id = ?`, chatID).Scan(&nextThreadID); err != nil {
		return TelegramThread{}, false, fmt.Errorf("allocate telegram thread id: %w", err)
	}
	if nextThreadID <= 0 {
		nextThreadID = 1
	}
	nowRaw := now.UTC().Format(time.RFC3339Nano)
	if _, err := tx.Exec(`
		INSERT INTO telegram_threads(
			chat_id, thread_id, status, created_by_sender_id, created_from_update_id,
			created_message_id, created_text, last_activity_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, chatID, nextThreadID, string(TelegramThreadStatusOpen), senderID, updateID, messageID, clampStoreText(text, 2000), nowRaw, nowRaw, nowRaw); err != nil {
		return TelegramThread{}, false, fmt.Errorf("insert telegram thread: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return TelegramThread{}, false, fmt.Errorf("commit telegram thread create: %w", err)
	}
	thread, ok, err := s.TelegramThread(chatID, nextThreadID)
	if err != nil {
		return TelegramThread{}, false, err
	}
	if !ok {
		return TelegramThread{}, false, fmt.Errorf("telegram thread %d/%d missing after create", chatID, nextThreadID)
	}
	return thread, true, nil
}

func (s *SQLiteStore) TelegramThread(chatID int64, threadID int64) (TelegramThread, bool, error) {
	if chatID == 0 || threadID <= 0 {
		return TelegramThread{}, false, nil
	}
	row := s.db.QueryRow(telegramThreadSelectSQL()+` WHERE chat_id = ? AND thread_id = ?`, chatID, threadID)
	thread, err := scanTelegramThread(row)
	if err == sql.ErrNoRows {
		return TelegramThread{}, false, nil
	}
	if err != nil {
		return TelegramThread{}, false, err
	}
	return thread, true, nil
}

func (s *SQLiteStore) TelegramThreadByCreatedUpdate(chatID int64, updateID int64) (TelegramThread, bool, error) {
	if chatID == 0 || updateID <= 0 {
		return TelegramThread{}, false, nil
	}
	return telegramThreadByCreatedUpdateDB(s.db, chatID, updateID)
}

func (s *SQLiteStore) ListTelegramThreads(chatID int64, limit int) ([]TelegramThread, error) {
	if chatID == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.Query(telegramThreadSelectSQL()+`
		WHERE chat_id = ?
		ORDER BY
			CASE status WHEN 'open' THEN 0 ELSE 1 END,
			thread_id DESC
		LIMIT ?
	`, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("list telegram threads: %w", err)
	}
	defer rows.Close()
	return scanTelegramThreadRows(rows)
}

func (s *SQLiteStore) TouchTelegramThread(chatID int64, threadID int64, at time.Time) error {
	if chatID == 0 || threadID <= 0 {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atRaw := at.UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`
		UPDATE telegram_threads
		SET last_activity_at = ?, updated_at = ?
		WHERE chat_id = ? AND thread_id = ? AND status = ?
	`, atRaw, atRaw, chatID, threadID, string(TelegramThreadStatusOpen)); err != nil {
		return fmt.Errorf("touch telegram thread: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CloseTelegramThread(chatID int64, threadID int64, summary string, at time.Time) (TelegramThread, bool, error) {
	if chatID == 0 || threadID <= 0 {
		return TelegramThread{}, false, nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atRaw := at.UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`
		UPDATE telegram_threads
		SET status = ?, closed_at = COALESCE(closed_at, ?), absorbed_at = COALESCE(absorbed_at, ?),
			absorb_summary = ?, updated_at = ?
		WHERE chat_id = ? AND thread_id = ? AND status = ?
	`, string(TelegramThreadStatusClosed), atRaw, atRaw, clampStoreText(summary, 4000), atRaw, chatID, threadID, string(TelegramThreadStatusOpen))
	if err != nil {
		return TelegramThread{}, false, fmt.Errorf("close telegram thread: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return TelegramThread{}, false, fmt.Errorf("close telegram thread rows affected: %w", err)
	}
	thread, ok, err := s.TelegramThread(chatID, threadID)
	if err != nil {
		return TelegramThread{}, false, err
	}
	return thread, ok && affected > 0, nil
}

func (s *SQLiteStore) RecordTelegramThreadAbsorb(chatID int64, threadID int64, summary string, defaultSession *Session, newMessages []Message, at time.Time) (TelegramThread, bool, error) {
	if chatID == 0 || threadID <= 0 {
		return TelegramThread{}, false, nil
	}
	if defaultSession == nil {
		return TelegramThread{}, false, fmt.Errorf("record telegram thread absorb: default session is required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	prepareSessionForSave(defaultSession, core.TokenUsage{}, at)

	tx, err := s.db.Begin()
	if err != nil {
		return TelegramThread{}, false, fmt.Errorf("begin telegram thread absorb: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	atRaw := at.UTC().Format(time.RFC3339Nano)
	res, err := tx.Exec(`
		UPDATE telegram_threads
		SET status = ?, closed_at = COALESCE(closed_at, ?), absorbed_at = COALESCE(absorbed_at, ?),
			absorb_summary = ?, updated_at = ?
		WHERE chat_id = ? AND thread_id = ? AND status = ?
	`, string(TelegramThreadStatusClosed), atRaw, atRaw, clampStoreText(summary, 4000), atRaw, chatID, threadID, string(TelegramThreadStatusOpen))
	if err != nil {
		return TelegramThread{}, false, fmt.Errorf("record telegram thread absorb close: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return TelegramThread{}, false, fmt.Errorf("record telegram thread absorb rows affected: %w", err)
	}
	if affected == 0 {
		thread, ok, err := telegramThreadTx(tx, chatID, threadID)
		if err != nil {
			return TelegramThread{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return TelegramThread{}, false, fmt.Errorf("commit telegram thread absorb no-op: %w", err)
		}
		return thread, ok && affected > 0, nil
	}
	if err := saveSessionInTx(tx, defaultSession, newMessages, at); err != nil {
		return TelegramThread{}, false, fmt.Errorf("record telegram thread absorb note: %w", err)
	}
	thread, ok, err := telegramThreadTx(tx, chatID, threadID)
	if err != nil {
		return TelegramThread{}, false, err
	}
	if !ok {
		return TelegramThread{}, false, fmt.Errorf("telegram thread %d/%d missing after absorb", chatID, threadID)
	}
	if err := tx.Commit(); err != nil {
		return TelegramThread{}, false, fmt.Errorf("commit telegram thread absorb: %w", err)
	}
	return thread, true, nil
}

func (s *SQLiteStore) TelegramThreadIDForReplyMessage(chatID int64, messageID int64) (int64, bool, error) {
	if chatID == 0 || messageID <= 0 {
		return 0, false, nil
	}
	var callbackThreadID int64
	err := s.db.QueryRow(`
		SELECT thread_id
		FROM telegram_callback_messages
		WHERE chat_id = ? AND message_id = ? AND thread_id > 0
		LIMIT 1
	`, chatID, messageID).Scan(&callbackThreadID)
	if err != nil && err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("lookup telegram callback message thread: %w", err)
	}
	if err == nil {
		return callbackThreadID, true, nil
	}

	for _, lookup := range []struct {
		name  string
		query string
	}{
		{
			name: "pending decision",
			query: `
				SELECT session_id
				FROM pending_decisions
				WHERE chat_id = ? AND delivery_message_id = ? AND session_id != ''
				ORDER BY decision_seq DESC
				LIMIT 1
			`,
		},
		{
			name: "outbound",
			query: `
				SELECT session_id
				FROM outbound_messages
				WHERE chat_id = ? AND telegram_msg_id = ?
				ORDER BY id DESC
				LIMIT 1
			`,
		},
		{
			name: "ingress",
			query: `
				SELECT session_id
				FROM telegram_ingress_updates
				WHERE chat_id = ? AND message_id = ? AND session_id != ''
				ORDER BY update_id DESC
				LIMIT 1
			`,
		},
		{
			name: "progress",
			query: `
				SELECT session_id
				FROM turn_runs
				WHERE chat_id = ? AND progress_message_id = ? AND session_id != ''
				ORDER BY id DESC
				LIMIT 1
			`,
		},
	} {
		var sessionID string
		err := s.db.QueryRow(lookup.query, chatID, messageID).Scan(&sessionID)
		if err != nil && err != sql.ErrNoRows {
			return 0, false, fmt.Errorf("lookup telegram reply %s message: %w", lookup.name, err)
		}
		if err == nil {
			if threadID, ok := telegramThreadIDFromSessionID(chatID, sessionID); ok {
				return threadID, true, nil
			}
			continue
		}
	}

	var threadID int64
	err = s.db.QueryRow(`
		SELECT thread_id
		FROM telegram_threads
		WHERE chat_id = ? AND created_message_id = ?
		ORDER BY thread_id DESC
		LIMIT 1
	`, chatID, messageID).Scan(&threadID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lookup telegram reply created thread message: %w", err)
	}
	return threadID, threadID > 0, nil
}

func (s *SQLiteStore) RecordTelegramCallbackMessageThread(chatID int64, messageID int64, threadID int64, surface string, at time.Time) error {
	if chatID == 0 || messageID <= 0 || threadID <= 0 {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atRaw := at.UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`
		INSERT INTO telegram_callback_messages(chat_id, message_id, thread_id, surface, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, message_id) DO UPDATE SET
			thread_id = excluded.thread_id,
			surface = excluded.surface,
			updated_at = excluded.updated_at
	`, chatID, messageID, threadID, clampStoreText(surface, 120), atRaw, atRaw); err != nil {
		return fmt.Errorf("record telegram callback message thread: %w", err)
	}
	return nil
}

func telegramThreadIDFromSessionID(chatID int64, sessionID string) (int64, bool) {
	sessionID = strings.TrimSpace(sessionID)
	const prefix = "telegram_thread:"
	if !strings.HasPrefix(sessionID, prefix) {
		return 0, false
	}
	raw := strings.TrimPrefix(sessionID, prefix)
	if idx := strings.Index(raw, "/"); idx >= 0 {
		raw = raw[:idx]
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, false
	}
	sessionChatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || sessionChatID != chatID {
		return 0, false
	}
	threadID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || threadID <= 0 {
		return 0, false
	}
	return threadID, true
}

func (s *SQLiteStore) RebindTelegramIngressSession(surface string, updateID int64, sessionID string, inboundJSON string, updatedAt time.Time) error {
	surface = strings.TrimSpace(surface)
	sessionID = strings.TrimSpace(sessionID)
	if surface == "" || updateID <= 0 || sessionID == "" {
		return nil
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`
		UPDATE telegram_ingress_updates
		SET
			session_id = ?,
			inbound_json = CASE
				WHEN ? != '' THEN ?
				ELSE inbound_json
			END,
			updated_at = ?
		WHERE surface = ? AND update_id = ? AND status IN (?, ?)
	`, sessionID, strings.TrimSpace(inboundJSON), strings.TrimSpace(inboundJSON), updatedAt.UTC().Format(time.RFC3339Nano),
		surface, updateID, string(TelegramIngressUpdateAccepted), string(TelegramIngressUpdateQueued))
	if err != nil {
		return fmt.Errorf("rebind telegram ingress session: %w", err)
	}
	return nil
}

func telegramThreadByCreatedUpdateDB(db *sql.DB, chatID int64, updateID int64) (TelegramThread, bool, error) {
	row := db.QueryRow(telegramThreadSelectSQL()+` WHERE chat_id = ? AND created_from_update_id = ?`, chatID, updateID)
	thread, err := scanTelegramThread(row)
	if err == sql.ErrNoRows {
		return TelegramThread{}, false, nil
	}
	if err != nil {
		return TelegramThread{}, false, err
	}
	return thread, true, nil
}

func telegramThreadByCreatedUpdateTx(tx *sql.Tx, chatID int64, updateID int64) (TelegramThread, bool, error) {
	row := tx.QueryRow(telegramThreadSelectSQL()+` WHERE chat_id = ? AND created_from_update_id = ?`, chatID, updateID)
	thread, err := scanTelegramThread(row)
	if err == sql.ErrNoRows {
		return TelegramThread{}, false, nil
	}
	if err != nil {
		return TelegramThread{}, false, err
	}
	return thread, true, nil
}

func telegramThreadTx(tx *sql.Tx, chatID int64, threadID int64) (TelegramThread, bool, error) {
	row := tx.QueryRow(telegramThreadSelectSQL()+` WHERE chat_id = ? AND thread_id = ?`, chatID, threadID)
	thread, err := scanTelegramThread(row)
	if err == sql.ErrNoRows {
		return TelegramThread{}, false, nil
	}
	if err != nil {
		return TelegramThread{}, false, err
	}
	return thread, true, nil
}

func telegramThreadSelectSQL() string {
	return `SELECT chat_id, thread_id, status, created_by_sender_id, created_from_update_id,
		created_message_id, created_text, last_activity_at, closed_at, absorb_summary,
		absorbed_at, created_at, updated_at
		FROM telegram_threads`
}

func scanTelegramThreadRows(rows *sql.Rows) ([]TelegramThread, error) {
	var out []TelegramThread
	for rows.Next() {
		thread, err := scanTelegramThread(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, thread)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate telegram threads: %w", err)
	}
	return out, nil
}

func scanTelegramThread(scanner interface{ Scan(dest ...any) error }) (TelegramThread, error) {
	var thread TelegramThread
	var statusRaw string
	var lastActivityRaw string
	var closedAtRaw sql.NullString
	var absorbedAtRaw sql.NullString
	var createdAtRaw string
	var updatedAtRaw string
	if err := scanner.Scan(
		&thread.ChatID,
		&thread.ThreadID,
		&statusRaw,
		&thread.CreatedBySenderID,
		&thread.CreatedFromUpdateID,
		&thread.CreatedMessageID,
		&thread.CreatedText,
		&lastActivityRaw,
		&closedAtRaw,
		&thread.AbsorbSummary,
		&absorbedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return TelegramThread{}, err
	}
	lastActivity, err := parseSQLiteTime(lastActivityRaw)
	if err != nil {
		return TelegramThread{}, fmt.Errorf("parse telegram thread last_activity_at: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdAtRaw)
	if err != nil {
		return TelegramThread{}, fmt.Errorf("parse telegram thread created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return TelegramThread{}, fmt.Errorf("parse telegram thread updated_at: %w", err)
	}
	thread.Status = normalizeTelegramThreadStatus(TelegramThreadStatus(statusRaw))
	thread.LastActivityAt = lastActivity
	thread.CreatedAt = createdAt
	thread.UpdatedAt = updatedAt
	if closedAtRaw.Valid && strings.TrimSpace(closedAtRaw.String) != "" {
		closedAt, err := parseSQLiteTime(closedAtRaw.String)
		if err != nil {
			return TelegramThread{}, fmt.Errorf("parse telegram thread closed_at: %w", err)
		}
		thread.ClosedAt = closedAt
	}
	if absorbedAtRaw.Valid && strings.TrimSpace(absorbedAtRaw.String) != "" {
		absorbedAt, err := parseSQLiteTime(absorbedAtRaw.String)
		if err != nil {
			return TelegramThread{}, fmt.Errorf("parse telegram thread absorbed_at: %w", err)
		}
		thread.AbsorbedAt = absorbedAt
	}
	return thread, nil
}

func normalizeTelegramThreadStatus(status TelegramThreadStatus) TelegramThreadStatus {
	switch TelegramThreadStatus(strings.TrimSpace(strings.ToLower(string(status)))) {
	case TelegramThreadStatusOpen:
		return TelegramThreadStatusOpen
	case TelegramThreadStatusClosed:
		return TelegramThreadStatusClosed
	default:
		return ""
	}
}
