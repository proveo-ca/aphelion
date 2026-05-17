//go:build linux

package session

import (
	"path/filepath"
	"testing"
)

func TestTurnProgressViewStateCachesSelectedProjection(t *testing.T) {
	t.Parallel()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "progress-views.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	defer store.Close()

	key := SessionKey{ChatID: 71001, UserID: 1001}
	run, err := store.BeginTurnRun(key, TurnRunKindInteractive, "inspect progress")
	if err != nil {
		t.Fatalf("BeginTurnRun() err = %v", err)
	}
	if _, ok, err := store.TurnProgressView(run.ID); err != nil || ok {
		t.Fatalf("TurnProgressView(empty) = ok:%v err:%v, want missing nil", ok, err)
	}

	if err := store.SetTurnProgressSelectedView(run.ID, 42, TurnProgressViewDetails); err != nil {
		t.Fatalf("SetTurnProgressSelectedView(details) err = %v", err)
	}
	state, ok, err := store.TurnProgressView(run.ID)
	if err != nil || !ok {
		t.Fatalf("TurnProgressView(details) = ok:%v err:%v, want state", ok, err)
	}
	if state.SelectedView != TurnProgressViewDetails || state.MessageID != 42 {
		t.Fatalf("state = %#v, want details view and message 42", state)
	}

	if err := store.SaveTurnProgressRender(run.ID, 42, TurnProgressViewSummary, "Working...\n- Searching files", "Working...\n- exec rg progress"); err != nil {
		t.Fatalf("SaveTurnProgressRender() err = %v", err)
	}
	state, ok, err = store.TurnProgressView(run.ID)
	if err != nil || !ok {
		t.Fatalf("TurnProgressView(cached) = ok:%v err:%v, want state", ok, err)
	}
	if state.SelectedView != TurnProgressViewDetails {
		t.Fatalf("selected view = %q, want render cache save to preserve operator-selected details", state.SelectedView)
	}
	if state.SummaryText != "Working...\n- Searching files" || state.DetailsText != "Working...\n- exec rg progress" {
		t.Fatalf("cached text = summary:%q details:%q, want saved render pair", state.SummaryText, state.DetailsText)
	}

	if err := store.SetTurnProgressSelectedView(run.ID, 0, TurnProgressViewSummary); err != nil {
		t.Fatalf("SetTurnProgressSelectedView(summary) err = %v", err)
	}
	state, ok, err = store.TurnProgressView(run.ID)
	if err != nil || !ok {
		t.Fatalf("TurnProgressView(summary) = ok:%v err:%v, want state", ok, err)
	}
	if state.SelectedView != TurnProgressViewSummary || state.MessageID != 42 {
		t.Fatalf("state = %#v, want summary view while preserving message id", state)
	}
}
