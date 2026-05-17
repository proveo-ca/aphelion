//go:build linux

package main

import (
	"context"
	"fmt"
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/session"
	"time"
)

func (c telegramCommandControl) MemoryReviewSnapshot(ctx context.Context, chatID int64, senderID int64, source memoryReviewSource) (memoryReviewSnapshot, error) {
	if c.rt == nil {
		return memoryReviewSnapshot{
			GeneratedAt: time.Now().UTC(),
			Source:      core.NormalizeMemoryReviewSource(string(source)),
			Query:       "",
		}, nil
	}
	return c.rt.MemoryReviewSnapshot(ctx, chatID, senderID, core.NormalizeMemoryReviewSource(string(source)))
}

func (c telegramCommandControl) MemoryReviewSnapshotForMessage(ctx context.Context, msg core.InboundMessage, source memoryReviewSource) (memoryReviewSnapshot, error) {
	if c.rt == nil {
		return memoryReviewSnapshot{
			GeneratedAt: time.Now().UTC(),
			Source:      core.NormalizeMemoryReviewSource(string(source)),
			Query:       "",
		}, nil
	}
	return c.rt.MemoryReviewSnapshotForKey(ctx, session.SessionKey{ChatID: msg.ChatID, UserID: 0, Scope: telegramCommandMessageScope(msg)}, msg.SenderID, core.NormalizeMemoryReviewSource(string(source)))
}

func (c telegramCommandControl) MissionCommand(ctx context.Context, chatID int64, senderID int64, args string) (string, error) {
	if c.rt == nil {
		return "Mission Ledger is unavailable.", nil
	}
	return c.rt.MissionCommand(ctx, chatID, senderID, args)
}

func (c telegramCommandControl) MissionHome(ctx context.Context, chatID int64, senderID int64) ([]session.MissionState, session.WorkingObjective, bool, error) {
	if c.rt == nil {
		return nil, session.WorkingObjective{}, false, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.MissionHome(ctx, chatID, senderID)
}

func (c telegramCommandControl) MissionDetails(ctx context.Context, chatID int64, senderID int64, missionID string) (session.MissionState, []session.MissionEvent, error) {
	if c.rt == nil {
		return session.MissionState{}, nil, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.MissionDetails(ctx, chatID, senderID, missionID)
}

func (c telegramCommandControl) SetMissionPinned(ctx context.Context, chatID int64, senderID int64, missionID string, pinned bool) (session.MissionState, error) {
	if c.rt == nil {
		return session.MissionState{}, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.SetMissionPinned(ctx, chatID, senderID, missionID, pinned)
}

func (c telegramCommandControl) UpdateMissionStatus(ctx context.Context, chatID int64, senderID int64, missionID string, status session.MissionStatus) (session.MissionState, error) {
	if c.rt == nil {
		return session.MissionState{}, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.UpdateMissionStatus(ctx, chatID, senderID, missionID, status)
}

func (c telegramCommandControl) MissionLedgerHealth(ctx context.Context, senderID int64) (session.MissionLedgerHealth, error) {
	if c.rt == nil {
		return session.MissionLedgerHealth{}, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.MissionLedgerHealth(ctx, senderID)
}

func (c telegramCommandControl) MissionActionProposal(ctx context.Context, chatID int64, senderID int64, missionID string) (session.ActionProposal, error) {
	if c.rt == nil {
		return session.ActionProposal{}, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.MissionActionProposal(ctx, chatID, senderID, missionID)
}

func (c telegramCommandControl) ApplyMissionActionProposalDecision(ctx context.Context, chatID int64, senderID int64, missionID string, choice string) (session.MissionState, bool, error) {
	if c.rt == nil {
		return session.MissionState{}, false, fmt.Errorf("Mission Ledger is unavailable.")
	}
	return c.rt.ApplyMissionActionProposalDecision(ctx, chatID, senderID, missionID, choice)
}

func (c telegramCommandControl) MemoryFocus(chatID int64) (core.MemoryFocus, bool) {
	if c.rt == nil {
		return core.MemoryFocus{}, false
	}
	return c.rt.MemoryFocus(chatID)
}

func (c telegramCommandControl) MemoryFocusForMessage(msg core.InboundMessage) (core.MemoryFocus, bool) {
	if c.rt == nil {
		return core.MemoryFocus{}, false
	}
	return c.rt.MemoryFocusForKey(session.SessionKey{ChatID: msg.ChatID, UserID: 0, Scope: telegramCommandMessageScope(msg)})
}

func (c telegramCommandControl) SetMemoryFocus(chatID int64, focus core.MemoryFocus) {
	if c.rt == nil {
		return
	}
	c.rt.SetMemoryFocus(chatID, focus)
}

func (c telegramCommandControl) SetMemoryFocusForMessage(msg core.InboundMessage, focus core.MemoryFocus) {
	if c.rt == nil {
		return
	}
	c.rt.SetMemoryFocusForKey(session.SessionKey{ChatID: msg.ChatID, UserID: 0, Scope: telegramCommandMessageScope(msg)}, focus)
}

func (c telegramCommandControl) ClearMemoryFocus(chatID int64) bool {
	if c.rt == nil {
		return false
	}
	return c.rt.ClearMemoryFocus(chatID)
}

func (c telegramCommandControl) ClearMemoryFocusForMessage(msg core.InboundMessage) bool {
	if c.rt == nil {
		return false
	}
	return c.rt.ClearMemoryFocusForKey(session.SessionKey{ChatID: msg.ChatID, UserID: 0, Scope: telegramCommandMessageScope(msg)})
}
