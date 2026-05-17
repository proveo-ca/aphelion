//go:build linux

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/decision"
	"github.com/idolum-ai/aphelion/session"
)

func TestDecisionResumeStatusTreatsCanonicalTerminalsAsTerminal(t *testing.T) {
	t.Parallel()

	for _, status := range []session.TelegramIngressUpdateStatus{
		session.TelegramIngressUpdateCompleted,
		session.TelegramIngressUpdateSkipped,
		session.TelegramIngressUpdateFailed,
		session.TelegramIngressUpdateDropped,
		session.TelegramIngressUpdateInterrupted,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			store, err := session.NewSQLiteStore(filepath.Join(t.TempDir(), "sessions.db"))
			if err != nil {
				t.Fatalf("NewSQLiteStore() err = %v", err)
			}
			t.Cleanup(func() { _ = store.Close() })

			msg := core.InboundMessage{ChatID: 7, SenderID: 42, MessageID: 99, IngressUpdateID: 9101, Text: "resume status"}
			if err := store.RecordTelegramIngressTerminal(session.TelegramIngressUpdateRecord{
				Surface:     telegramBusyDecisionResumeIngressSurface,
				UpdateID:    telegramDecisionResumeUpdateID(msg, telegramBusyDecisionResumeIngressSurface),
				UpdateKind:  "decision_resume_busy",
				ChatID:      msg.ChatID,
				SenderID:    msg.SenderID,
				MessageID:   msg.MessageID,
				Status:      status,
				ErrorText:   "terminal",
				CompletedAt: time.Now().UTC(),
			}); err != nil {
				t.Fatalf("RecordTelegramIngressTerminal() err = %v", err)
			}

			got, err := (&telegramDecisionHandler{store: store}).decisionResumeStatus(msg, telegramBusyDecisionResumeIngressSurface)
			if err != nil {
				t.Fatalf("decisionResumeStatus() err = %v", err)
			}
			if got != telegramDecisionResumeTerminal {
				t.Fatalf("decisionResumeStatus(%s) = %v, want terminal", status, got)
			}
		})
	}
}

func TestRestartReconciliationCleansBusyDecisionForTerminalResumeIngress(t *testing.T) {
	t.Parallel()

	for _, status := range []session.TelegramIngressUpdateStatus{
		session.TelegramIngressUpdateFailed,
		session.TelegramIngressUpdateDropped,
		session.TelegramIngressUpdateInterrupted,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			store, err := session.NewSQLiteStore(filepath.Join(t.TempDir(), "sessions.db"))
			if err != nil {
				t.Fatalf("NewSQLiteStore() err = %v", err)
			}
			t.Cleanup(func() { _ = store.Close() })

			msg := core.InboundMessage{ChatID: 7, SenderID: 42, MessageID: 99, IngressSurface: telegramPrimaryIngressSurface, IngressUpdateID: 9201, Text: "stale busy message"}
			ownerKey := telegramSessionOwnerKey(msg)
			raw, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Marshal() err = %v", err)
			}
			if err := store.UpsertPendingBusyDecision(session.PendingBusyDecisionRecord{
				OwnerKey:           ownerKey,
				ChatID:             msg.ChatID,
				SenderID:           msg.SenderID,
				SessionID:          core.SessionIDForInboundMessage(msg),
				ScopeKind:          string(session.ScopeKindTelegramDM),
				ScopeID:            "7",
				MessageID:          msg.MessageID,
				InboundMessageJSON: string(raw),
			}); err != nil {
				t.Fatalf("UpsertPendingBusyDecision() err = %v", err)
			}
			seedRestartLoadedPendingDecision(t, store, "old-busy-terminal", ownerKey, msg, decision.KindInterrupt)
			if err := store.RecordTelegramIngressTerminal(session.TelegramIngressUpdateRecord{
				Surface:     telegramBusyDecisionResumeIngressSurface,
				UpdateID:    telegramDecisionResumeUpdateID(msg, telegramBusyDecisionResumeIngressSurface),
				UpdateKind:  "decision_resume_busy",
				ChatID:      msg.ChatID,
				SenderID:    msg.SenderID,
				MessageID:   msg.MessageID,
				Status:      status,
				ErrorText:   "terminal",
				CompletedAt: time.Now().UTC(),
			}); err != nil {
				t.Fatalf("RecordTelegramIngressTerminal() err = %v", err)
			}

			sender := &decisionTestSender{}
			router := &decisionAcceptedTestRouter{decisionTestRouter: &decisionTestRouter{}}
			broker := newTelegramDecisionBroker(sender, decision.WithDurableStore(newTelegramDecisionDurableStore(store)))
			if err := broker.Load(context.Background()); err != nil {
				t.Fatalf("Load() err = %v", err)
			}
			handler := newTelegramDecisionHandler(sender, router, broker, store)
			if err := handler.ReconcileRestartLoadedDecisions(context.Background()); err != nil {
				t.Fatalf("ReconcileRestartLoadedDecisions() err = %v", err)
			}
			if len(sender.inline) != 0 || len(router.accepted) != 0 {
				t.Fatalf("sender inline=%#v accepted=%#v, want no reissue or default route", sender.inline, router.accepted)
			}
			if _, ok := broker.Peek("old-busy-terminal"); ok {
				t.Fatal("restart-loaded busy decision remained active")
			}
			if _, err := store.PendingBusyDecision(ownerKey); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("PendingBusyDecision() err = %v, want cleared", err)
			}
		})
	}
}

func TestRestartReconciliationCleansArtifactDecisionForTerminalResumeIngress(t *testing.T) {
	t.Parallel()

	for _, status := range []session.TelegramIngressUpdateStatus{
		session.TelegramIngressUpdateFailed,
		session.TelegramIngressUpdateDropped,
		session.TelegramIngressUpdateInterrupted,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			store, err := session.NewSQLiteStore(filepath.Join(t.TempDir(), "sessions.db"))
			if err != nil {
				t.Fatalf("NewSQLiteStore() err = %v", err)
			}
			t.Cleanup(func() { _ = store.Close() })

			msg := core.InboundMessage{
				ChatID:          7,
				SenderID:        42,
				MessageID:       99,
				IngressSurface:  telegramPrimaryIngressSurface,
				IngressUpdateID: 9301,
				Text:            "stale artifact message",
				Artifacts: []core.Artifact{{
					ID:       "archive-1",
					Channel:  "telegram",
					RemoteID: "file-1",
					Kind:     "archive",
					Filename: "bundle.zip",
				}},
			}
			ownerKey := telegramSessionOwnerKey(msg)
			raw, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Marshal() err = %v", err)
			}
			if err := store.UpsertPendingArtifactRetention(session.PendingArtifactRetentionRecord{
				OwnerKey:           ownerKey,
				ChatID:             msg.ChatID,
				SenderID:           msg.SenderID,
				SessionID:          core.SessionIDForInboundMessage(msg),
				ScopeKind:          string(session.ScopeKindTelegramDM),
				ScopeID:            "7",
				MessageID:          msg.MessageID,
				InboundMessageJSON: string(raw),
			}); err != nil {
				t.Fatalf("UpsertPendingArtifactRetention() err = %v", err)
			}
			seedRestartLoadedPendingDecision(t, store, "old-artifact-terminal", ownerKey, msg, decision.KindArtifactRetention)
			if err := store.RecordTelegramIngressTerminal(session.TelegramIngressUpdateRecord{
				Surface:     telegramArtifactRetentionDecisionResumeIngressSurface,
				UpdateID:    telegramDecisionResumeUpdateID(msg, telegramArtifactRetentionDecisionResumeIngressSurface),
				UpdateKind:  "decision_resume_artifact_retention",
				ChatID:      msg.ChatID,
				SenderID:    msg.SenderID,
				MessageID:   msg.MessageID,
				Status:      status,
				ErrorText:   "terminal",
				CompletedAt: time.Now().UTC(),
			}); err != nil {
				t.Fatalf("RecordTelegramIngressTerminal() err = %v", err)
			}

			sender := &decisionTestSender{}
			router := &decisionAcceptedTestRouter{decisionTestRouter: &decisionTestRouter{}}
			broker := newTelegramDecisionBroker(sender, decision.WithDurableStore(newTelegramDecisionDurableStore(store)))
			if err := broker.Load(context.Background()); err != nil {
				t.Fatalf("Load() err = %v", err)
			}
			handler := newTelegramDecisionHandler(sender, router, broker, store)
			if err := handler.ReconcileRestartLoadedDecisions(context.Background()); err != nil {
				t.Fatalf("ReconcileRestartLoadedDecisions() err = %v", err)
			}
			if len(sender.inline) != 0 || len(router.accepted) != 0 {
				t.Fatalf("sender inline=%#v accepted=%#v, want no reissue or default route", sender.inline, router.accepted)
			}
			if _, ok := broker.Peek("old-artifact-terminal"); ok {
				t.Fatal("restart-loaded artifact decision remained active")
			}
			if _, err := store.PendingArtifactRetention(ownerKey); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("PendingArtifactRetention() err = %v, want cleared", err)
			}
		})
	}
}

func seedRestartLoadedPendingDecision(t *testing.T, store *session.SQLiteStore, id string, ownerKey string, msg core.InboundMessage, kind decision.Kind) {
	t.Helper()
	choicesJSON, err := json.Marshal([]decision.Choice{{ID: "queue", Label: "Queue"}})
	if err != nil {
		t.Fatalf("Marshal(choices) err = %v", err)
	}
	if err := store.UpsertPendingDecision(session.PendingDecisionRecord{
		ID:                id,
		Sequence:          50,
		OwnerKey:          ownerKey,
		SessionID:         core.SessionIDForInboundMessage(msg),
		ScopeKind:         string(session.ScopeKindTelegramDM),
		ScopeID:           "7",
		Kind:              string(kind),
		ChatID:            msg.ChatID,
		SenderID:          msg.SenderID,
		MessageID:         msg.MessageID,
		Prompt:            "Restart-loaded prompt",
		ChoicesJSON:       string(choicesJSON),
		DefaultChoice:     "queue",
		TimeoutNanos:      int64(time.Hour),
		DeliveryMessageID: 7005,
	}); err != nil {
		t.Fatalf("UpsertPendingDecision() err = %v", err)
	}
}
