//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/decision"
	"github.com/idolum-ai/aphelion/session"
	"log"
	"strings"
	"time"
)

func mergeStopResults(a core.StopResult, b core.StopResult) core.StopResult {
	return core.StopResult{
		ActiveCanceled:      a.ActiveCanceled || b.ActiveCanceled,
		QueuedDropped:       a.QueuedDropped || b.QueuedDropped,
		ContinuationRevoked: a.ContinuationRevoked || b.ContinuationRevoked,
		ContinuationLabel:   firstNonEmpty(a.ContinuationLabel, b.ContinuationLabel),
	}
}

func mergeSessionStatus(a core.SessionStatus, b core.SessionStatus) core.SessionStatus {
	return core.SessionStatus{
		Active:      a.Active || b.Active,
		Queued:      a.Queued || b.Queued || a.QueueDepth+b.QueueDepth > 0,
		QueueDepth:  a.QueueDepth + b.QueueDepth,
		Diagnostics: append(append([]string(nil), a.Diagnostics...), b.Diagnostics...),
	}
}

func mergeRouterStatusSnapshots(a core.RouterStatusSnapshot, b core.RouterStatusSnapshot) core.RouterStatusSnapshot {
	if a.ActiveTurnsByChat == nil {
		a.ActiveTurnsByChat = make(map[int64][]uint64)
	}
	if a.QueueDepthByChat == nil {
		a.QueueDepthByChat = make(map[int64]int)
	}
	for chatID, ids := range b.ActiveTurnsByChat {
		a.ActiveTurnsByChat[chatID] = append(a.ActiveTurnsByChat[chatID], ids...)
	}
	for chatID, depth := range b.QueueDepthByChat {
		a.QueueDepthByChat[chatID] += depth
	}
	return a
}

func (c telegramCommandControl) Stop(chatID int64) core.StopResult {
	result := core.StopResult{}
	if c.ingress != nil {
		result = mergeStopResults(result, c.ingress.Stop(chatID))
	}
	if c.router != nil {
		result = mergeStopResults(result, c.router.Stop(chatID))
	}
	if c.rt != nil {
		revoke, err := c.rt.RevokeContinuation(chatID)
		if err == nil {
			result.ContinuationRevoked = revoke.Revoked
			result.ContinuationLabel = revoke.ContinuationLabel
		}
	}
	c.maybeFlushMemory(chatID, "stop")
	return result
}

func (c telegramCommandControl) StopForMessage(msg core.InboundMessage) core.StopResult {
	result := core.StopResult{}
	if c.ingress != nil {
		result = mergeStopResults(result, c.ingress.StopForMessage(msg))
	}
	if c.router != nil {
		result = mergeStopResults(result, c.router.StopForMessage(msg))
	}
	if c.rt != nil {
		revoke, err := c.rt.RevokeContinuationForKey(session.SessionKey{ChatID: msg.ChatID, UserID: 0, Scope: telegramCommandMessageScope(msg)})
		if err == nil {
			result.ContinuationRevoked = revoke.Revoked
			result.ContinuationLabel = revoke.ContinuationLabel
		}
	}
	return result
}

func (c telegramCommandControl) MarkDroppedIngress(messages []core.InboundMessage) {
	if c.store == nil || len(messages) == 0 {
		return
	}
	seen := make(map[ingressIdentity]struct{}, len(messages))
	now := time.Now().UTC()
	for _, msg := range messages {
		identity, ok := ingressIdentityForMessage(msg)
		if !ok {
			continue
		}
		if _, ok := seen[identity]; ok {
			continue
		}
		seen[identity] = struct{}{}
		if _, err := c.store.MarkTelegramIngressDroppedIfDispatchable(identity.surface, identity.updateID, "operator_session_stop", now); err != nil {
			log.Printf("WARN mark dropped telegram ingress failed surface=%s update_id=%d err=%v", identity.surface, identity.updateID, err)
		}
	}
}

func (c telegramCommandControl) Route(ctx context.Context, msg core.InboundMessage) {
	if err := c.RouteAccepted(ctx, msg); err != nil {
		log.Printf("WARN telegram ingress route failed chat_id=%d message_id=%d err=%v", msg.ChatID, msg.MessageID, err)
	}
}

func (c telegramCommandControl) RouteAccepted(ctx context.Context, msg core.InboundMessage) error {
	if err := c.rebindTelegramIngressForMessage(msg); err != nil {
		return err
	}
	if c.store != nil && strings.TrimSpace(msg.IngressSurface) != "" && msg.IngressUpdateID > 0 {
		result, err := c.store.MarkTelegramIngressQueued(msg.IngressSurface, msg.IngressUpdateID, time.Now().UTC())
		if err != nil {
			return err
		}
		if !result.Found {
			return fmt.Errorf("telegram ingress update %s/%d is not accepted", strings.TrimSpace(msg.IngressSurface), msg.IngressUpdateID)
		}
		if !result.Dispatch {
			return nil
		}
	}
	if c.ingress != nil {
		if err := c.ingress.Enqueue(ctx, msg); err != nil {
			return err
		}
		return nil
	}
	if c.router != nil {
		c.router.Route(ctx, msg)
	}
	return nil
}

func (c telegramCommandControl) rebindTelegramIngressForMessage(msg core.InboundMessage) error {
	if c.store == nil || msg.TelegramThreadID <= 0 || strings.TrimSpace(msg.IngressSurface) == "" || msg.IngressUpdateID <= 0 {
		return nil
	}
	encoded := ""
	if raw, err := json.Marshal(msg); err == nil {
		encoded = string(raw)
	}
	return c.store.RebindTelegramIngressSession(msg.IngressSurface, msg.IngressUpdateID, core.SessionIDForInboundMessage(msg), encoded, time.Now().UTC())
}

func (c telegramCommandControl) MarkStreamControlStopping(streamID string, chatID int64) bool {
	if c.rt == nil {
		return false
	}
	return c.rt.MarkStreamControlStopping(streamID, chatID)
}

func (c telegramCommandControl) StopRun(runID int64, senderID int64) (core.StopResult, bool, error) {
	if c.store == nil || runID <= 0 {
		return core.StopResult{}, false, nil
	}
	run, err := c.store.TurnRun(runID)
	if err != nil {
		return core.StopResult{}, false, err
	}
	if strings.TrimSpace(string(run.Status)) != string(session.TurnRunStatusRunning) {
		return core.StopResult{}, false, nil
	}
	return c.StopForMessage(telegramInboundForTurnRun(*run, senderID)), true, nil
}

func (c telegramCommandControl) DetachRun(runID int64, senderID int64) (core.DetachResult, bool, error) {
	if c.store == nil || runID <= 0 {
		return core.DetachResult{}, false, nil
	}
	run, err := c.store.TurnRun(runID)
	if err != nil {
		return core.DetachResult{}, false, err
	}
	if strings.TrimSpace(string(run.Status)) != string(session.TurnRunStatusRunning) {
		return core.DetachResult{}, false, nil
	}
	result, err := c.DetachForMessage(telegramInboundForTurnRun(*run, senderID))
	if err != nil {
		return core.DetachResult{}, false, err
	}
	return result, true, nil
}

func (c telegramCommandControl) New(chatID int64, senderID int64) (core.NewSessionResult, error) {
	stopped := c.Stop(chatID)
	result := core.NewSessionResult{
		ActiveCanceled:      stopped.ActiveCanceled,
		QueuedDropped:       stopped.QueuedDropped,
		ContinuationRevoked: stopped.ContinuationRevoked,
	}
	if c.decisionDetacher != nil {
		removed, err := 0, error(nil)
		if detacher, ok := c.decisionDetacher.(pendingDecisionChatSenderDetacher); ok {
			removed, err = detacher.DetachByChatSender(context.Background(), chatID, senderID)
		} else {
			ownerKey := decision.OwnerKey(chatID, senderID)
			removed, err = c.decisionDetacher.DetachByOwner(context.Background(), ownerKey)
		}
		if err != nil {
			return core.NewSessionResult{}, err
		}
		result.PendingDecisionsDetached = removed
	}
	if c.rt != nil {
		cleared, err := c.rt.ClearChatSessionContext(chatID)
		if err != nil {
			return core.NewSessionResult{}, err
		}
		result.ContextCleared = cleared
	}
	return result, nil
}

func (c telegramCommandControl) NewForMessage(msg core.InboundMessage) (core.NewSessionResult, error) {
	stopped := c.StopForMessage(msg)
	result := core.NewSessionResult{
		ActiveCanceled:      stopped.ActiveCanceled,
		QueuedDropped:       stopped.QueuedDropped,
		ContinuationRevoked: stopped.ContinuationRevoked,
	}
	if c.decisionDetacher != nil {
		ownerKey := telegramSessionOwnerKey(msg)
		removed, err := c.decisionDetacher.DetachByOwner(context.Background(), ownerKey)
		if err != nil {
			return core.NewSessionResult{}, err
		}
		result.PendingDecisionsDetached = removed
	}
	if c.rt != nil {
		cleared, err := c.rt.ClearSessionContextForKey(session.SessionKey{ChatID: msg.ChatID, UserID: 0, Scope: telegramCommandMessageScope(msg)})
		if err != nil {
			return core.NewSessionResult{}, err
		}
		result.ContextCleared = cleared
	}
	return result, nil
}

func (c telegramCommandControl) Detach(chatID int64, senderID int64) (core.DetachResult, error) {
	stopped := c.Stop(chatID)
	result := core.DetachResult{
		ActiveCanceled:      stopped.ActiveCanceled,
		QueuedDropped:       stopped.QueuedDropped,
		ContinuationRevoked: stopped.ContinuationRevoked,
	}
	if c.decisionDetacher == nil {
		return result, nil
	}
	removed, err := 0, error(nil)
	if detacher, ok := c.decisionDetacher.(pendingDecisionChatSenderDetacher); ok {
		removed, err = detacher.DetachByChatSender(context.Background(), chatID, senderID)
	} else {
		ownerKey := decision.OwnerKey(chatID, senderID)
		removed, err = c.decisionDetacher.DetachByOwner(context.Background(), ownerKey)
	}
	if err != nil {
		return core.DetachResult{}, err
	}
	result.PendingDecisionsDetached = removed
	return result, nil
}

func (c telegramCommandControl) DetachForMessage(msg core.InboundMessage) (core.DetachResult, error) {
	stopped := c.StopForMessage(msg)
	result := core.DetachResult{
		ActiveCanceled:      stopped.ActiveCanceled,
		QueuedDropped:       stopped.QueuedDropped,
		ContinuationRevoked: stopped.ContinuationRevoked,
	}
	if c.decisionDetacher == nil {
		return result, nil
	}
	ownerKey := telegramSessionOwnerKey(msg)
	removed, err := c.decisionDetacher.DetachByOwner(context.Background(), ownerKey)
	if err != nil {
		return core.DetachResult{}, err
	}
	result.PendingDecisionsDetached = removed
	return result, nil
}

func (c telegramCommandControl) Restart(chatID int64) error {
	if c.detachPendingOnRestart && c.decisionDetacher != nil {
		removed, err := c.decisionDetacher.DetachAll(context.Background())
		if err != nil {
			log.Printf("WARN restart detach pending decisions failed err=%v", err)
		} else if removed > 0 {
			log.Printf("WARN restart detached %d pending decision(s) before exit", removed)
		}
	}
	log.Printf("WARN restart requested via telegram chat_id=%d", chatID)
	if c.rt != nil {
		c.rt.BeginShutdown()
	}
	go func() {
		time.Sleep(restartExitWait)
		processExit(exitCodeFailure)
	}()
	return nil
}

func (c telegramCommandControl) maybeFlushMemory(chatID int64, reason string) {
	if c.rt == nil {
		return
	}
	if err := c.rt.FlushChatMemory(context.Background(), chatID, reason); err != nil {
		log.Printf("WARN memory flush skipped chat_id=%d reason=%s err=%v", chatID, strings.TrimSpace(reason), err)
		c.rt.ReportOperationalIssue(context.Background(), "memory_flush", fmt.Errorf("chat_id=%d reason=%s: %w", chatID, strings.TrimSpace(reason), err))
	}
}
