//go:build linux

package session

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) UpsertPendingArtifactRetention(record PendingArtifactRetentionRecord) error {
	record.OwnerKey = strings.TrimSpace(record.OwnerKey)
	if record.OwnerKey == "" {
		return fmt.Errorf("pending artifact retention owner_key is required")
	}
	record.InboundMessageJSON = strings.TrimSpace(record.InboundMessageJSON)
	if record.InboundMessageJSON == "" {
		record.InboundMessageJSON = "{}"
	}
	record.SessionID = strings.TrimSpace(record.SessionID)
	record.ScopeKind = strings.TrimSpace(record.ScopeKind)
	record.ScopeID = strings.TrimSpace(record.ScopeID)
	record.DurableAgentID = strings.TrimSpace(record.DurableAgentID)
	now := time.Now().UTC()
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := record.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}
	_, err := s.db.Exec(`
		INSERT INTO pending_artifact_retention(
			owner_key, chat_id, sender_id, session_id, scope_kind, scope_id, durable_agent_id, message_id, inbound_message_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(owner_key) DO UPDATE SET
			chat_id = excluded.chat_id,
			sender_id = excluded.sender_id,
			session_id = excluded.session_id,
			scope_kind = excluded.scope_kind,
			scope_id = excluded.scope_id,
			durable_agent_id = excluded.durable_agent_id,
			message_id = excluded.message_id,
			inbound_message_json = excluded.inbound_message_json,
			updated_at = excluded.updated_at
	`, record.OwnerKey, record.ChatID, record.SenderID, record.SessionID, record.ScopeKind, record.ScopeID, record.DurableAgentID, record.MessageID, record.InboundMessageJSON, createdAt.UTC().Format(time.RFC3339Nano), updatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("upsert pending artifact retention: %w", err)
	}
	return nil
}

func (s *SQLiteStore) PendingArtifactRetention(ownerKey string) (*PendingArtifactRetentionRecord, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil, sql.ErrNoRows
	}
	var record PendingArtifactRetentionRecord
	var createdAtRaw, updatedAtRaw string
	err := s.db.QueryRow(`
		SELECT owner_key, chat_id, sender_id, session_id, scope_kind, scope_id, durable_agent_id, message_id, inbound_message_json, created_at, updated_at
		FROM pending_artifact_retention
		WHERE owner_key = ?
	`, ownerKey).Scan(&record.OwnerKey, &record.ChatID, &record.SenderID, &record.SessionID, &record.ScopeKind, &record.ScopeID, &record.DurableAgentID, &record.MessageID, &record.InboundMessageJSON, &createdAtRaw, &updatedAtRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("load pending artifact retention: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse pending artifact retention created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse pending artifact retention updated_at: %w", err)
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return &record, nil
}

func (s *SQLiteStore) PendingArtifactRetentions() ([]PendingArtifactRetentionRecord, error) {
	rows, err := s.db.Query(`
		SELECT owner_key, chat_id, sender_id, session_id, scope_kind, scope_id, durable_agent_id, message_id, inbound_message_json, created_at, updated_at
		FROM pending_artifact_retention
		ORDER BY updated_at ASC, owner_key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query pending artifact retention: %w", err)
	}
	defer rows.Close()

	var out []PendingArtifactRetentionRecord
	for rows.Next() {
		record, err := scanPendingArtifactRetention(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending artifact retention: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) PendingArtifactRetentionForMessage(chatID int64, senderID int64, messageID int64) (*PendingArtifactRetentionRecord, error) {
	if chatID == 0 || senderID == 0 || messageID == 0 {
		return nil, sql.ErrNoRows
	}
	var record PendingArtifactRetentionRecord
	var createdAtRaw, updatedAtRaw string
	err := s.db.QueryRow(`
		SELECT owner_key, chat_id, sender_id, session_id, scope_kind, scope_id, durable_agent_id, message_id, inbound_message_json, created_at, updated_at
		FROM pending_artifact_retention
		WHERE chat_id = ? AND sender_id = ? AND message_id = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`, chatID, senderID, messageID).Scan(&record.OwnerKey, &record.ChatID, &record.SenderID, &record.SessionID, &record.ScopeKind, &record.ScopeID, &record.DurableAgentID, &record.MessageID, &record.InboundMessageJSON, &createdAtRaw, &updatedAtRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("load pending artifact retention by message: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse pending artifact retention created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse pending artifact retention updated_at: %w", err)
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return &record, nil
}

func (s *SQLiteStore) DeletePendingArtifactRetention(ownerKey string) error {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil
	}
	if _, err := s.db.Exec(`DELETE FROM pending_artifact_retention WHERE owner_key = ?`, ownerKey); err != nil {
		return fmt.Errorf("delete pending artifact retention: %w", err)
	}
	return nil
}

func scanPendingArtifactRetention(scanner interface{ Scan(dest ...any) error }) (PendingArtifactRetentionRecord, error) {
	var record PendingArtifactRetentionRecord
	var createdAtRaw, updatedAtRaw string
	if err := scanner.Scan(
		&record.OwnerKey,
		&record.ChatID,
		&record.SenderID,
		&record.SessionID,
		&record.ScopeKind,
		&record.ScopeID,
		&record.DurableAgentID,
		&record.MessageID,
		&record.InboundMessageJSON,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return PendingArtifactRetentionRecord{}, fmt.Errorf("scan pending artifact retention: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdAtRaw)
	if err != nil {
		return PendingArtifactRetentionRecord{}, fmt.Errorf("parse pending artifact retention created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return PendingArtifactRetentionRecord{}, fmt.Errorf("parse pending artifact retention updated_at: %w", err)
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return record, nil
}
