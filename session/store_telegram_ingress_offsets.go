//go:build linux

package session

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) TelegramIngressNextUpdateID(surface string) (int64, error) {
	surface = strings.TrimSpace(surface)
	if surface == "" {
		return 0, fmt.Errorf("telegram ingress offset surface is required")
	}
	var next int64
	if err := s.db.QueryRow(`
		SELECT next_update_id
		FROM telegram_ingress_offsets
		WHERE surface = ?
	`, surface).Scan(&next); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("query telegram ingress offset: %w", err)
	}
	return next, nil
}

func (s *SQLiteStore) SaveTelegramIngressNextUpdateID(surface string, nextUpdateID int64, updatedAt time.Time) error {
	surface = strings.TrimSpace(surface)
	if surface == "" {
		return fmt.Errorf("telegram ingress offset surface is required")
	}
	if nextUpdateID < 0 {
		nextUpdateID = 0
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`
		INSERT INTO telegram_ingress_offsets(surface, next_update_id, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(surface) DO UPDATE SET
			next_update_id = CASE
				WHEN excluded.next_update_id > telegram_ingress_offsets.next_update_id THEN excluded.next_update_id
				ELSE telegram_ingress_offsets.next_update_id
			END,
			updated_at = excluded.updated_at
	`, surface, nextUpdateID, updatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save telegram ingress offset: %w", err)
	}
	return nil
}
