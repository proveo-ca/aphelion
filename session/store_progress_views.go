//go:build linux

package session

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func NormalizeTurnProgressView(view string) string {
	switch strings.TrimSpace(view) {
	case TurnProgressViewDetails:
		return TurnProgressViewDetails
	default:
		return TurnProgressViewSummary
	}
}

func (s *SQLiteStore) TurnProgressView(runID int64) (TurnProgressViewState, bool, error) {
	if runID <= 0 {
		return TurnProgressViewState{}, false, nil
	}
	row := s.db.QueryRow(`
		SELECT run_id, message_id, selected_view, summary_text, details_text, updated_at
		FROM turn_progress_views
		WHERE run_id = ?
	`, runID)
	state, err := scanTurnProgressView(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TurnProgressViewState{}, false, nil
	}
	if err != nil {
		return TurnProgressViewState{}, false, err
	}
	return state, true, nil
}

func (s *SQLiteStore) SetTurnProgressSelectedView(runID int64, messageID int64, view string) error {
	if runID <= 0 {
		return fmt.Errorf("turn run id is required")
	}
	view = NormalizeTurnProgressView(view)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`
		INSERT INTO turn_progress_views(
			run_id, message_id, selected_view, updated_at
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			message_id = CASE
				WHEN excluded.message_id > 0 THEN excluded.message_id
				ELSE turn_progress_views.message_id
			END,
			selected_view = excluded.selected_view,
			updated_at = excluded.updated_at
	`, runID, positiveInt64(messageID), view, now)
	if err != nil {
		return fmt.Errorf("set turn progress selected view: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveTurnProgressRender(runID int64, messageID int64, selectedView string, summaryText string, detailsText string) error {
	if runID <= 0 {
		return fmt.Errorf("turn run id is required")
	}
	selectedView = NormalizeTurnProgressView(selectedView)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`
		INSERT INTO turn_progress_views(
			run_id, message_id, selected_view, summary_text, details_text, updated_at
		)
		SELECT ?, ?, ?, ?, ?, ?
		WHERE EXISTS (SELECT 1 FROM turn_runs WHERE id = ?)
		ON CONFLICT(run_id) DO UPDATE SET
			message_id = CASE
				WHEN excluded.message_id > 0 THEN excluded.message_id
				ELSE turn_progress_views.message_id
			END,
			selected_view = turn_progress_views.selected_view,
			summary_text = excluded.summary_text,
			details_text = excluded.details_text,
			updated_at = excluded.updated_at
	`, runID, positiveInt64(messageID), selectedView, summaryText, detailsText, now, runID)
	if err != nil {
		return fmt.Errorf("save turn progress render: %w", err)
	}
	return nil
}

func positiveInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func scanTurnProgressView(scanner interface{ Scan(dest ...any) error }) (TurnProgressViewState, error) {
	var (
		state        TurnProgressViewState
		selectedView string
		updatedAtRaw string
	)
	if err := scanner.Scan(&state.RunID, &state.MessageID, &selectedView, &state.SummaryText, &state.DetailsText, &updatedAtRaw); err != nil {
		return TurnProgressViewState{}, fmt.Errorf("scan turn progress view: %w", err)
	}
	state.SelectedView = NormalizeTurnProgressView(selectedView)
	updatedAt, err := parseSQLiteTime(updatedAtRaw)
	if err != nil {
		return TurnProgressViewState{}, fmt.Errorf("parse turn progress view updated_at: %w", err)
	}
	state.UpdatedAt = updatedAt
	return state, nil
}
