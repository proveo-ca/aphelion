//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/principal"
	"github.com/idolum-ai/aphelion/session"
	"log"
	"strconv"
	"strings"
	"time"
)

func (c telegramCommandControl) RunDurableWizard(ctx context.Context, chatID int64, senderID int64, action string, agentID string, wizardAnswers map[string]any) (string, error) {
	if c.durableTools == nil {
		return "", fmt.Errorf("durable wizard controls are unavailable")
	}
	if !c.CanRestart(senderID) {
		return "", fmt.Errorf("durable wizard controls are admin only")
	}

	actor := principal.Principal{
		Role:           principal.RoleAdmin,
		TelegramUserID: senderID,
	}
	if c.resolver != nil {
		resolved, ok := c.resolver.ResolveTelegramUser(senderID)
		if !ok || resolved.Role != principal.RoleAdmin {
			return "", fmt.Errorf("durable wizard controls are admin only")
		}
		actor = resolved
	}

	request := map[string]any{
		"action":   strings.TrimSpace(action),
		"agent_id": strings.TrimSpace(agentID),
	}
	if len(wizardAnswers) > 0 {
		request["wizard_answers"] = wizardAnswers
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	key := session.SessionKey{
		ChatID: chatID,
		UserID: 0,
		Scope: session.ScopeRef{
			Kind: session.ScopeKindTelegramDM,
			ID:   strconv.FormatInt(chatID, 10),
		},
	}
	return c.durableTools.ExecuteForSessionPrincipal(ctx, actor, key, "durable_agent", payload)
}

func (c telegramCommandControl) DurableAgentsList(senderID int64) ([]core.DurableAgentStatusSnapshot, error) {
	if !c.CanRestart(senderID) {
		return nil, fmt.Errorf("durable-agent controls are admin only")
	}
	if c.rt == nil {
		return nil, nil
	}
	snapshot, err := c.rt.DurableAgentsStatusSnapshot()
	if err != nil {
		return nil, err
	}
	return append([]core.DurableAgentStatusSnapshot(nil), snapshot.Agents...), nil
}

func (c telegramCommandControl) StartDurableAgentConversation(ctx context.Context, chatID int64, senderID int64, agentID string) (string, error) {
	if c.durableTools == nil {
		return "", fmt.Errorf("durable-agent controls are unavailable")
	}
	if !c.CanRestart(senderID) {
		return "", fmt.Errorf("durable-agent controls are admin only")
	}

	actor := principal.Principal{
		Role:           principal.RoleAdmin,
		TelegramUserID: senderID,
	}
	if c.resolver != nil {
		resolved, ok := c.resolver.ResolveTelegramUser(senderID)
		if !ok || resolved.Role != principal.RoleAdmin {
			return "", fmt.Errorf("durable-agent controls are admin only")
		}
		actor = resolved
	}

	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", fmt.Errorf("durable agent id is required")
	}
	request := map[string]any{
		"action":   "conversation_send",
		"agent_id": agentID,
		"message":  "Scheduled parent-child check-in from /agents. Share current status, blockers, and concrete next actions.",
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	key := session.SessionKey{
		ChatID: chatID,
		UserID: 0,
		Scope: session.ScopeRef{
			Kind: session.ScopeKindTelegramDM,
			ID:   strconv.FormatInt(chatID, 10),
		},
	}
	if _, err := c.durableTools.ExecuteForSessionPrincipal(ctx, actor, key, "durable_agent", payload); err != nil {
		return "", err
	}

	wakeStatus := "queued for next child wake"
	if c.rt != nil {
		go func(agent string) {
			wakeCtx, cancel := newTurnContext(context.Background(), turnTimeout)
			defer cancel()
			if err := c.rt.RunDurableAgentChildWake(wakeCtx, agent, time.Now().UTC()); err != nil {
				log.Printf("WARN durable-agent background wake failed agent_id=%s err=%v", strings.TrimSpace(agent), err)
			}
		}(agentID)
		wakeStatus = "wake attempt started"
	}
	return fmt.Sprintf("Started background conversation with durable agent %s (%s).", agentID, wakeStatus), nil
}
