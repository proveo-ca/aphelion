//go:build linux

package telegramdecision

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/decision"
	"github.com/idolum-ai/aphelion/internal/telegramruntime"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/telegram"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResumePendingArtifactRetentionRecordsSyntheticIngressAndClearsPending(t *testing.T) {
	t.Parallel()

	store, err := session.NewSQLiteStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	msg := core.InboundMessage{
		ChatID:           7,
		SenderID:         42,
		MessageID:        99,
		TelegramThreadID: 3,
		IngressSurface:   telegramruntime.PrimaryIngressSurface,
		IngressUpdateID:  801,
		Text:             "inspect this bundle",
		Artifacts: []core.Artifact{{
			ID:         "doc-1",
			Channel:    "telegram",
			RemoteID:   "file-1",
			Kind:       "document",
			SourceType: "document",
			Filename:   "bundle.zip",
			MimeType:   "application/zip",
		}},
	}
	ownerKey := telegramruntime.SessionOwnerKey(msg)
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() err = %v", err)
	}
	if err := store.UpsertPendingArtifactRetention(session.PendingArtifactRetentionRecord{
		OwnerKey:           ownerKey,
		ChatID:             msg.ChatID,
		SenderID:           msg.SenderID,
		SessionID:          core.SessionIDForInboundMessage(msg),
		ScopeKind:          string(session.ScopeKindTelegramThread),
		ScopeID:            "7:3",
		MessageID:          msg.MessageID,
		InboundMessageJSON: string(raw),
	}); err != nil {
		t.Fatalf("UpsertPendingArtifactRetention() err = %v", err)
	}

	router := &decisionAcceptedTestRouter{decisionTestRouter: &decisionTestRouter{}}
	handler := NewHandler(&decisionTestSender{}, router, decision.NewBroker(nil), store)
	if err := handler.ResumePendingArtifactRetention(context.Background(), ownerKey, decision.Result{Choice: "local"}); err != nil {
		t.Fatalf("ResumePendingArtifactRetention() err = %v", err)
	}

	if len(router.accepted) != 1 {
		t.Fatalf("accepted = %#v, want one synthetic artifact retention route", router.accepted)
	}
	routed := router.accepted[0]
	if routed.IngressSurface != telegramruntime.ArtifactRetentionDecisionResumeIngressSurface || routed.IngressUpdateID != 801 || routed.TelegramThreadID != 3 {
		t.Fatalf("routed = %#v, want thread-scoped artifact decision resume ingress", routed)
	}
	if got := routed.Artifacts[0].Metadata["aphelion_retention_choice"]; got != "local" {
		t.Fatalf("retention choice = %q, want local", got)
	}
	record, ok, err := store.TelegramIngressUpdate(telegramruntime.ArtifactRetentionDecisionResumeIngressSurface, 801)
	if err != nil || !ok {
		t.Fatalf("TelegramIngressUpdate() ok=%t err=%v", ok, err)
	}
	if record.Status != session.TelegramIngressUpdateAccepted || record.SessionID != "telegram_thread:7:3" || !strings.Contains(record.InboundJSON, "aphelion_retention_choice") {
		t.Fatalf("record = %#v, want recoverable accepted artifact ingress", record)
	}
	if _, err := store.PendingArtifactRetention(ownerKey); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("PendingArtifactRetention() err = %v, want sql.ErrNoRows after successful synthetic accept", err)
	}
}

func TestResumePendingArtifactRetentionKeepsPendingWhenSyntheticRouteFails(t *testing.T) {
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
		IngressSurface:  telegramruntime.PrimaryIngressSurface,
		IngressUpdateID: 802,
		Text:            "inspect this bundle",
		Artifacts: []core.Artifact{{
			ID:       "doc-1",
			Channel:  "telegram",
			RemoteID: "file-1",
			Kind:     "archive",
			Filename: "bundle.zip",
		}},
	}
	ownerKey := telegramruntime.SessionOwnerKey(msg)
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

	routeErr := errors.New("route unavailable")
	router := &decisionAcceptedTestRouter{decisionTestRouter: &decisionTestRouter{}, acceptedErr: routeErr}
	handler := NewHandler(&decisionTestSender{}, router, decision.NewBroker(nil), store)
	if err := handler.ResumePendingArtifactRetention(context.Background(), ownerKey, decision.Result{Choice: "session"}); !errors.Is(err, routeErr) {
		t.Fatalf("ResumePendingArtifactRetention() err = %v, want route error", err)
	}
	if _, err := store.PendingArtifactRetention(ownerKey); err != nil {
		t.Fatalf("PendingArtifactRetention() err = %v, want pending row retained for retry", err)
	}
	record, ok, err := store.TelegramIngressUpdate(telegramruntime.ArtifactRetentionDecisionResumeIngressSurface, 802)
	if err != nil || !ok {
		t.Fatalf("TelegramIngressUpdate() ok=%t err=%v", ok, err)
	}
	if record.Status != session.TelegramIngressUpdateAccepted {
		t.Fatalf("record status = %s, want accepted recoverable ingress despite route failure", record.Status)
	}
}

func TestRestartLoadedArtifactRetentionCallbackResumesPendingMessage(t *testing.T) {
	store, err := session.NewSQLiteStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	durable := NewDurableStore(store)
	senderBeforeRestart := &decisionTestSender{}
	brokerBeforeRestart := NewBroker(senderBeforeRestart, decision.WithDurableStore(durable))
	handlerBeforeRestart := NewHandler(senderBeforeRestart, &decisionTestRouter{}, brokerBeforeRestart, store)
	handlerBeforeRestart.artifactRetentionTimeout = time.Hour

	msg := core.InboundMessage{
		ChatID:          7,
		SenderID:        42,
		MessageID:       99,
		IngressSurface:  telegramruntime.PrimaryIngressSurface,
		IngressUpdateID: 9101,
		Text:            "please inspect this archive",
		Artifacts: []core.Artifact{{
			ID:       "archive-1",
			Channel:  "telegram",
			RemoteID: "file-archive",
			Kind:     "archive",
			Filename: "bundle.zip",
			MimeType: "application/zip",
		}},
	}
	handled, err := handlerBeforeRestart.HandleArtifactRetentionMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleArtifactRetentionMessage() err = %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want artifact retention prompt")
	}
	pending := waitForStoredPendingDecision(t, store, decision.KindArtifactRetention)

	senderAfterRestart := &decisionTestSender{}
	routerAfterRestart := &decisionAcceptedTestRouter{decisionTestRouter: &decisionTestRouter{}}
	brokerAfterRestart := NewBroker(senderAfterRestart, decision.WithDurableStore(durable))
	if err := brokerAfterRestart.Load(context.Background()); err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	loaded, ok := brokerAfterRestart.Peek(pending.ID)
	if !ok || !loaded.LoadedFromDurable {
		t.Fatalf("loaded pending = %#v, ok=%t; want restart-loaded artifact decision", loaded, ok)
	}
	handlerAfterRestart := NewHandler(senderAfterRestart, routerAfterRestart, brokerAfterRestart, store)

	err = handlerAfterRestart.HandleCallbackQuery(context.Background(), telegram.CallbackQuery{
		ID:       "cb-restart-artifact",
		Data:     decision.EncodeCallbackData(pending.ID, "local"),
		UpdateID: 9102,
		From:     &telegram.User{ID: 42},
		Message: &telegram.Message{
			MessageID: pending.DeliveryMessageID,
			Chat:      &telegram.Chat{ID: 7, Type: "private"},
		},
	})
	if err != nil {
		t.Fatalf("HandleCallbackQuery() err = %v", err)
	}
	if len(senderAfterRestart.answers) != 1 || senderAfterRestart.answers[0].text != "" {
		t.Fatalf("answers = %#v, want one success acknowledgement", senderAfterRestart.answers)
	}
	if len(routerAfterRestart.accepted) != 1 {
		t.Fatalf("accepted = %#v, want one resumed artifact synthetic ingress", routerAfterRestart.accepted)
	}
	routed := routerAfterRestart.accepted[0]
	if routed.IngressSurface != telegramruntime.ArtifactRetentionDecisionResumeIngressSurface || routed.IngressUpdateID != 9101 {
		t.Fatalf("routed = %#v, want artifact retention decision resume ingress", routed)
	}
	if len(routed.Artifacts) != 1 || routed.Artifacts[0].Metadata["aphelion_retention_choice"] != "local" {
		t.Fatalf("routed artifacts = %#v, want local retention choice", routed.Artifacts)
	}
	if _, err := store.PendingArtifactRetention(telegramruntime.SessionOwnerKey(msg)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("PendingArtifactRetention() err = %v, want cleared after restart callback resume", err)
	}
}
