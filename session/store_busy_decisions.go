//go:build linux

package session

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) UpsertPendingBusyDecision(record PendingBusyDecisionRecord) error {
	record.OwnerKey = strings.TrimSpace(record.OwnerKey)
	if record.OwnerKey == "" {
		return fmt.Errorf("pending busy decision owner_key is required")
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
		INSERT INTO pending_busy_decisions(
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
		return fmt.Errorf("upsert pending busy decision: %w", err)
	}
	return nil
}

func (s *SQLiteStore) PendingBusyDecision(ownerKey string) (*PendingBusyDecisionRecord, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil, sql.ErrNoRows
	}
	var record PendingBusyDecisionRecord
	var createdAtRaw, updatedAtRaw string
	err := s.db.QueryRow(`
		SELECT owner_key, chat_id, sender_id, session_id, scope_kind, scope_id, durable_agent_id, message_id, inbound_message_json, created_at, updated_at
		FROM pending_busy_decisions
		WHERE owner_key = ?
	`, ownerKey).Scan(&record.OwnerKey, &record.ChatID, &record.SenderID, &record.SessionID, &record.ScopeKind, &record.ScopeID, &record.DurableAgentID, &record.MessageID, &record.InboundMessageJSON, &createdAtRaw, &updatedAtRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("load pending busy decision: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse pending busy decision created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse pending busy decision updated_at: %w", err)
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return &record, nil
}

func (s *SQLiteStore) PendingBusyDecisions() ([]PendingBusyDecisionRecord, error) {
	rows, err := s.db.Query(`
		SELECT owner_key, chat_id, sender_id, session_id, scope_kind, scope_id, durable_agent_id, message_id, inbound_message_json, created_at, updated_at
		FROM pending_busy_decisions
		ORDER BY updated_at ASC, owner_key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query pending busy decisions: %w", err)
	}
	defer rows.Close()

	var out []PendingBusyDecisionRecord
	for rows.Next() {
		record, err := scanPendingBusyDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending busy decisions: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) DeletePendingBusyDecision(ownerKey string) error {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil
	}
	if _, err := s.db.Exec(`DELETE FROM pending_busy_decisions WHERE owner_key = ?`, ownerKey); err != nil {
		return fmt.Errorf("delete pending busy decision: %w", err)
	}
	return nil
}

func scanPendingBusyDecision(scanner interface{ Scan(dest ...any) error }) (PendingBusyDecisionRecord, error) {
	var record PendingBusyDecisionRecord
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
		return PendingBusyDecisionRecord{}, fmt.Errorf("scan pending busy decision: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdAtRaw)
	if err != nil {
		return PendingBusyDecisionRecord{}, fmt.Errorf("parse pending busy decision created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return PendingBusyDecisionRecord{}, fmt.Errorf("parse pending busy decision updated_at: %w", err)
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return record, nil
}
