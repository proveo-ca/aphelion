//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/decision"
	"github.com/idolum-ai/aphelion/session"
)

func (h *telegramDecisionHandler) ReconcileRestartLoadedDecisions(ctx context.Context) error {
	if h == nil || h.store == nil || h.broker == nil {
		return nil
	}
	now := time.Now().UTC()
	if err := h.reconcilePendingBusyDecisions(ctx, now); err != nil {
		return err
	}
	if err := h.reconcilePendingArtifactRetentionDecisions(ctx, now); err != nil {
		return err
	}
	return h.detachNonResumableRestartLoadedDecisions(ctx)
}

func (h *telegramDecisionHandler) reconcilePendingBusyDecisions(ctx context.Context, now time.Time) error {
	records, err := h.store.PendingBusyDecisions()
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		msg, err := pendingBusyDecisionMessage(record)
		if err != nil {
			return err
		}
		ownerKey := firstNonEmpty(strings.TrimSpace(record.OwnerKey), telegramSessionOwnerKey(msg))
		if ownerKey == "" {
			return fmt.Errorf("pending busy decision owner key is required")
		}
		req := h.busyDecisionRequest(msg, ownerKey)
		pending, found := h.broker.PendingByOwnerKind(ownerKey, req.Kind)
		status, err := h.decisionResumeStatus(msg, telegramBusyDecisionResumeIngressSurface)
		if err != nil {
			return err
		}
		switch status {
		case telegramDecisionResumeInProgress:
			if found && pending.LoadedFromDurable {
				if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_resume_in_progress"); err != nil {
					return err
				}
			}
			continue
		case telegramDecisionResumeTerminal:
			if found && pending.LoadedFromDurable {
				if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_resume_terminal"); err != nil {
					return err
				}
			}
			if err := h.store.DeletePendingBusyDecision(ownerKey); err != nil {
				return err
			}
			continue
		}
		if found && !pending.LoadedFromDurable {
			continue
		}
		if decisionRecordExpired(record.CreatedAt, req.Timeout, now) {
			if found {
				if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_timeout"); err != nil {
					return err
				}
			}
			if err := h.resumePendingBusyDecision(ctx, ownerKey, decision.Result{
				DecisionID: pending.ID,
				Choice:     strings.TrimSpace(req.DefaultChoice),
				Delivery:   pending.Delivery,
				TimedOut:   true,
			}); err != nil {
				return err
			}
			continue
		}
		if found {
			if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_reissued"); err != nil {
				return err
			}
		}
		go h.awaitBusyDecision(context.Background(), ownerKey, req)
	}
	return nil
}

func (h *telegramDecisionHandler) reconcilePendingArtifactRetentionDecisions(ctx context.Context, now time.Time) error {
	records, err := h.store.PendingArtifactRetentions()
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		msg, err := pendingArtifactRetentionMessage(record)
		if err != nil {
			return err
		}
		if !hasArtifactRetentionApprovalCandidates(msg) {
			continue
		}
		ownerKey := firstNonEmpty(strings.TrimSpace(record.OwnerKey), telegramSessionOwnerKey(msg))
		if ownerKey == "" {
			return fmt.Errorf("pending artifact retention owner key is required")
		}
		req := h.artifactRetentionDecisionRequest(msg, ownerKey)
		pending, found := h.broker.PendingByOwnerKind(ownerKey, req.Kind)
		status, err := h.decisionResumeStatus(msg, telegramArtifactRetentionDecisionResumeIngressSurface)
		if err != nil {
			return err
		}
		switch status {
		case telegramDecisionResumeInProgress:
			if found && pending.LoadedFromDurable {
				if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_resume_in_progress"); err != nil {
					return err
				}
			}
			continue
		case telegramDecisionResumeTerminal:
			if found && pending.LoadedFromDurable {
				if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_resume_terminal"); err != nil {
					return err
				}
			}
			if err := h.store.DeletePendingArtifactRetention(ownerKey); err != nil {
				return err
			}
			continue
		}
		if found && !pending.LoadedFromDurable {
			continue
		}
		if decisionRecordExpired(record.CreatedAt, req.Timeout, now) {
			if found {
				if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_timeout"); err != nil {
					return err
				}
			}
			if err := h.resumePendingArtifactRetention(ctx, ownerKey, decision.Result{
				DecisionID: pending.ID,
				Choice:     strings.TrimSpace(req.DefaultChoice),
				Delivery:   pending.Delivery,
				TimedOut:   true,
			}); err != nil {
				return err
			}
			continue
		}
		if found {
			if _, _, err := h.broker.DetachDecision(ctx, pending.ID, "restart_loaded_reissued"); err != nil {
				return err
			}
		}
		go h.awaitArtifactRetentionDecision(context.Background(), ownerKey, req)
	}
	return nil
}

func (h *telegramDecisionHandler) detachNonResumableRestartLoadedDecisions(ctx context.Context) error {
	if h == nil || h.broker == nil {
		return nil
	}
	pending, err := h.broker.PendingDecisions(ctx)
	if err != nil {
		return err
	}
	for _, decision := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !decision.LoadedFromDurable || h.canResumeRestartLoadedDecision(decision) {
			continue
		}
		if _, _, err := h.broker.DetachDecision(ctx, decision.ID, "restart_loaded_non_resumable"); err != nil {
			return err
		}
	}
	return nil
}

type telegramDecisionResumeStatus int

const (
	telegramDecisionResumeMissing telegramDecisionResumeStatus = iota
	telegramDecisionResumeInProgress
	telegramDecisionResumeTerminal
)

func (h *telegramDecisionHandler) decisionResumeStatus(msg core.InboundMessage, surface string) (telegramDecisionResumeStatus, error) {
	if h == nil || h.store == nil {
		return telegramDecisionResumeMissing, nil
	}
	updateID := telegramDecisionResumeUpdateID(msg, surface)
	record, ok, err := h.store.TelegramIngressUpdate(surface, updateID)
	if err != nil || !ok {
		return telegramDecisionResumeMissing, err
	}
	switch record.Status {
	case session.TelegramIngressUpdateAccepted, session.TelegramIngressUpdateQueued, session.TelegramIngressUpdateRunning:
		return telegramDecisionResumeInProgress, nil
	default:
		if session.TelegramIngressUpdateStatusTerminal(record.Status) {
			return telegramDecisionResumeTerminal, nil
		}
		return telegramDecisionResumeMissing, nil
	}
}

func (h *telegramDecisionHandler) busyDecisionRequest(msg core.InboundMessage, ownerKey string) decision.Request {
	target := telegramSessionTargetForMessage(msg)
	req := decision.Request{
		ChatID:        msg.ChatID,
		SenderID:      msg.SenderID,
		MessageID:     msg.MessageID,
		OwnerKey:      strings.TrimSpace(ownerKey),
		SessionID:     target.SessionID,
		ScopeKind:     string(target.Scope.Kind),
		ScopeID:       target.Scope.ID,
		Choices:       []decision.Choice{{ID: "stop", Label: stopChoiceLabel(msg.Text)}, {ID: "queue", Label: queueChoiceLabel(msg.Text)}},
		DefaultChoice: "queue",
	}
	if isStopWord(msg.Text) {
		req.Kind = decision.KindStopWord
		req.Prompt = "Stop the current task?"
		req.Timeout = h.stopWordTimeout
	} else {
		req.Kind = decision.KindInterrupt
		req.Prompt = "I'm still working on the previous request. What would you like to do?"
		req.Timeout = h.interruptTimeout
	}
	return req
}

func (h *telegramDecisionHandler) artifactRetentionDecisionRequest(msg core.InboundMessage, ownerKey string) decision.Request {
	target := telegramSessionTargetForMessage(msg)
	return decision.Request{
		Kind:           decision.KindArtifactRetention,
		ChatID:         msg.ChatID,
		SenderID:       msg.SenderID,
		MessageID:      msg.MessageID,
		OwnerKey:       strings.TrimSpace(ownerKey),
		SessionID:      target.SessionID,
		ScopeKind:      string(target.Scope.Kind),
		ScopeID:        target.Scope.ID,
		DurableAgentID: strings.TrimSpace(target.Scope.DurableAgentID),
		Prompt:         "How should I retain this inbound file?",
		Details:        formatArtifactRetentionDetails(msg),
		Choices:        artifactRetentionChoices(),
		DefaultChoice:  "session",
		Timeout:        h.artifactRetentionTimeout,
	}
}

func pendingBusyDecisionMessage(record session.PendingBusyDecisionRecord) (core.InboundMessage, error) {
	var msg core.InboundMessage
	if strings.TrimSpace(record.InboundMessageJSON) == "" {
		return core.InboundMessage{}, fmt.Errorf("pending busy decision has no inbound payload")
	}
	if err := json.Unmarshal([]byte(record.InboundMessageJSON), &msg); err != nil {
		return core.InboundMessage{}, fmt.Errorf("decode pending busy decision: %w", err)
	}
	return msg, nil
}

func pendingArtifactRetentionMessage(record session.PendingArtifactRetentionRecord) (core.InboundMessage, error) {
	var msg core.InboundMessage
	if strings.TrimSpace(record.InboundMessageJSON) == "" {
		return core.InboundMessage{}, fmt.Errorf("pending artifact retention has no inbound payload")
	}
	if err := json.Unmarshal([]byte(record.InboundMessageJSON), &msg); err != nil {
		return core.InboundMessage{}, fmt.Errorf("decode pending artifact retention: %w", err)
	}
	return msg, nil
}

func decisionRecordExpired(createdAt time.Time, timeout time.Duration, now time.Time) bool {
	if timeout < 0 || createdAt.IsZero() {
		return false
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return !createdAt.UTC().Add(timeout).After(now.UTC())
}
