//go:build linux

package session

import (
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) RecordTelegramIngressFailure(record TelegramIngressFailureRecord) error {
	record.Surface = strings.TrimSpace(record.Surface)
	if record.Surface == "" {
		return fmt.Errorf("telegram ingress failure surface is required")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`
		INSERT INTO telegram_ingress_failures(
			surface, update_id, update_kind, chat_id, sender_id, message_id,
			error_text, payload_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.Surface,
		record.UpdateID,
		strings.TrimSpace(record.UpdateKind),
		record.ChatID,
		record.SenderID,
		record.MessageID,
		clampStoreText(record.ErrorText, 2000),
		clampStoreText(record.Payload, 8000),
		record.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record telegram ingress failure: %w", err)
	}
	return nil
}

func (s *SQLiteStore) RecentTelegramIngressFailures(limit int) ([]TelegramIngressFailureRecord, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := s.db.Query(`
		SELECT id, surface, update_id, update_kind, chat_id, sender_id, message_id,
			error_text, payload_json, created_at
		FROM telegram_ingress_failures
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query telegram ingress failures: %w", err)
	}
	defer rows.Close()

	var out []TelegramIngressFailureRecord
	for rows.Next() {
		var record TelegramIngressFailureRecord
		var createdAtRaw string
		if err := rows.Scan(
			&record.ID,
			&record.Surface,
			&record.UpdateID,
			&record.UpdateKind,
			&record.ChatID,
			&record.SenderID,
			&record.MessageID,
			&record.ErrorText,
			&record.Payload,
			&createdAtRaw,
		); err != nil {
			return nil, fmt.Errorf("scan telegram ingress failure: %w", err)
		}
		createdAt, err := parseSQLiteTime(createdAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse telegram ingress failure created_at: %w", err)
		}
		record.CreatedAt = createdAt
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate telegram ingress failures: %w", err)
	}
	return out, nil
}
