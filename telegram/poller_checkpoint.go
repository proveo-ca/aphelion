//go:build linux

package telegram

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/core"
)

type PollerCheckpoint interface {
	NextUpdateID(ctx context.Context) (int64, error)
	SaveNextUpdateID(ctx context.Context, nextUpdateID int64) error
	UpdateState(ctx context.Context, updateID int64) (PollerUpdateState, error)
	RecordAccepted(ctx context.Context, accepted PollerAccepted) (PollerAcceptResult, error)
	RecordHandled(ctx context.Context, updateID int64) error
	RecordFailure(ctx context.Context, failure PollerFailure) error
	RecordTerminal(ctx context.Context, terminal PollerTerminal) error
}

const (
	PollerTerminalCompleted = "completed"
	PollerTerminalSkipped   = "skipped"
)

type PollerAccepted struct {
	UpdateID    int64
	UpdateKind  string
	ChatID      int64
	SenderID    int64
	MessageID   int64
	SessionID   string
	InboundJSON string
	Payload     string
	CreatedAt   time.Time
}

type PollerUpdateState struct {
	Found    bool
	Terminal bool
	Status   string
}

type PollerAcceptResult struct {
	Dispatch bool
	Terminal bool
	Status   string
}

type PollerFailure struct {
	UpdateID   int64
	UpdateKind string
	ChatID     int64
	SenderID   int64
	MessageID  int64
	ErrorText  string
	Payload    string
	CreatedAt  time.Time
}

type PollerTerminal struct {
	UpdateID   int64
	UpdateKind string
	Status     string
	Reason     string
	ChatID     int64
	SenderID   int64
	MessageID  int64
	Payload    string
	CreatedAt  time.Time
}

func WithCheckpoint(checkpoint PollerCheckpoint) PollerOption {
	return func(p *Poller) {
		p.checkpoint = checkpoint
	}
}

func (p *Poller) advanceOffset(ctx context.Context, current int64, next int64) (int64, error) {
	if next <= current {
		return current, nil
	}
	if p != nil && p.checkpoint != nil {
		if err := p.checkpoint.SaveNextUpdateID(ctx, next); err != nil {
			return current, err
		}
	}
	return next, nil
}

func (p *Poller) recordFailure(ctx context.Context, upd Update, kind string, cause error) error {
	if p == nil || p.checkpoint == nil || cause == nil {
		return nil
	}
	failure := PollerFailure{
		UpdateID:   upd.UpdateID,
		UpdateKind: updateKind(upd, kind),
		ErrorText:  cause.Error(),
		CreatedAt:  time.Now().UTC(),
	}
	failure.ChatID, failure.SenderID, failure.MessageID = updateFailureRefs(upd)
	if raw, err := json.Marshal(upd); err == nil {
		failure.Payload = string(raw)
	}
	return p.checkpoint.RecordFailure(ctx, failure)
}

func (p *Poller) recordTerminal(ctx context.Context, upd Update, kind string, status string, reason string) error {
	if p == nil || p.checkpoint == nil || upd.UpdateID <= 0 {
		return nil
	}
	terminal := PollerTerminal{
		UpdateID:   upd.UpdateID,
		UpdateKind: updateKind(upd, kind),
		Status:     strings.TrimSpace(status),
		Reason:     strings.TrimSpace(reason),
		CreatedAt:  time.Now().UTC(),
	}
	terminal.ChatID, terminal.SenderID, terminal.MessageID = updateFailureRefs(upd)
	if raw, err := json.Marshal(upd); err == nil {
		terminal.Payload = string(raw)
	}
	return p.checkpoint.RecordTerminal(ctx, terminal)
}

func (p *Poller) updateState(ctx context.Context, updateID int64) (PollerUpdateState, error) {
	if p == nil || p.checkpoint == nil || updateID <= 0 {
		return PollerUpdateState{}, nil
	}
	return p.checkpoint.UpdateState(ctx, updateID)
}

func (p *Poller) recordAccepted(ctx context.Context, upd Update, kind string, inbound core.InboundMessage) (PollerAcceptResult, error) {
	if p == nil || p.checkpoint == nil {
		return PollerAcceptResult{Dispatch: true}, nil
	}
	accepted := PollerAccepted{
		UpdateID:   upd.UpdateID,
		UpdateKind: updateKind(upd, kind),
		ChatID:     inbound.ChatID,
		SenderID:   inbound.SenderID,
		MessageID:  inbound.MessageID,
		SessionID:  core.SessionIDForInboundMessage(inbound),
		CreatedAt:  time.Now().UTC(),
	}
	if encoded, err := json.Marshal(inbound); err == nil {
		accepted.InboundJSON = string(encoded)
	}
	if raw, err := json.Marshal(upd); err == nil {
		accepted.Payload = string(raw)
	}
	result, err := p.checkpoint.RecordAccepted(ctx, accepted)
	if err != nil {
		return PollerAcceptResult{}, err
	}
	return result, nil
}

func (p *Poller) recordHandled(ctx context.Context, updateID int64) error {
	if p == nil || p.checkpoint == nil || updateID <= 0 {
		return nil
	}
	return p.checkpoint.RecordHandled(ctx, updateID)
}

func updateKind(upd Update, fallback string) string {
	switch {
	case upd.MessageReaction != nil:
		return "message_reaction"
	case upd.CallbackQuery != nil:
		return "callback_query"
	case upd.Message != nil:
		return "message"
	default:
		return strings.TrimSpace(fallback)
	}
}

func updateFailureRefs(upd Update) (chatID int64, senderID int64, messageID int64) {
	if upd.MessageReaction != nil {
		if upd.MessageReaction.Chat != nil {
			chatID = upd.MessageReaction.Chat.ID
		}
		senderID = senderIDFromUser(upd.MessageReaction.User)
		messageID = upd.MessageReaction.MessageID
		return chatID, senderID, messageID
	}
	if upd.CallbackQuery != nil {
		senderID = senderIDFromUser(upd.CallbackQuery.From)
		if upd.CallbackQuery.Message != nil {
			messageID = upd.CallbackQuery.Message.MessageID
			if upd.CallbackQuery.Message.Chat != nil {
				chatID = upd.CallbackQuery.Message.Chat.ID
			}
		}
		return chatID, senderID, messageID
	}
	if upd.Message != nil {
		messageID = upd.Message.MessageID
		senderID = senderIDFromUser(upd.Message.From)
		if upd.Message.Chat != nil {
			chatID = upd.Message.Chat.ID
		}
	}
	return chatID, senderID, messageID
}

func senderIDFromUser(user *User) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}
