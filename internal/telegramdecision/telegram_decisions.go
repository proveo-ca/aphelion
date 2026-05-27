//go:build linux

package telegramdecision

import (
	"context"
	"time"

	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/decision"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/telegram"
)

const (
	defaultInterruptTimeout         = DefaultInterruptTimeout
	defaultStopWordTimeout          = DefaultStopWordTimeout
	defaultUserApprovalTimeout      = DefaultUserApprovalTimeout
	defaultExecApprovalTimeout      = DefaultExecApprovalTimeout
	defaultArtifactRetentionTimeout = DefaultArtifactRetentionTimeout
	defaultMemoryDelegationTimeout  = DefaultMemoryDelegationTimeout
	defaultSnapshotRestoreTimeout   = DefaultSnapshotRestoreTimeout
)

type telegramDecisionSender = DecisionSender
type telegramDecisionKeyboardEditor = DecisionKeyboardEditor
type telegramDecisionKeyboardClearer = DecisionKeyboardClearer
type telegramDecisionRouter = Router
type telegramDecisionMessageStatusRouter = MessageStatusRouter
type telegramDecisionMessageStopRouter = MessageStopRouter
type telegramPermanentArtifactKeeper = PermanentArtifactKeeper
type telegramDecisionSummaryFunc = SummaryFunc
type telegramDecisionBrokerUIOptions = BrokerUIOptions

type DecisionHandler struct {
	*Handler
	sender                   telegramDecisionSender
	broker                   *decision.Broker
	store                    *session.SQLiteStore
	router                   telegramDecisionRouter
	interruptTimeout         time.Duration
	stopWordTimeout          time.Duration
	artifactRetentionTimeout time.Duration
}

func editDecisionMessageClearingInlineKeyboard(ctx context.Context, sender telegramDecisionSender, chatID int64, messageID int64, text string) error {
	return EditDecisionMessageClearingInlineKeyboard(ctx, sender, chatID, messageID, text)
}

func NewDecisionHandler(sender DecisionSender, router Router, broker *decision.Broker, store *session.SQLiteStore, keepers ...PermanentArtifactKeeper) *DecisionHandler {
	inner := NewHandler(sender, router, broker, store, keepers...)
	return &DecisionHandler{
		Handler:                  inner,
		sender:                   sender,
		broker:                   broker,
		store:                    store,
		router:                   router,
		interruptTimeout:         defaultInterruptTimeout,
		stopWordTimeout:          defaultStopWordTimeout,
		artifactRetentionTimeout: defaultArtifactRetentionTimeout,
	}
}

func newTelegramDecisionHandler(sender telegramDecisionSender, router telegramDecisionRouter, broker *decision.Broker, store *session.SQLiteStore, keepers ...telegramPermanentArtifactKeeper) *DecisionHandler {
	return NewDecisionHandler(sender, router, broker, store, keepers...)
}

func newTelegramDecisionBroker(sender telegramDecisionSender, opts ...decision.BrokerOption) *decision.Broker {
	return NewBroker(sender, opts...)
}

func newTelegramDecisionBrokerWithSummary(sender telegramDecisionSender, summarize telegramDecisionSummaryFunc, ui telegramDecisionBrokerUIOptions, opts ...decision.BrokerOption) *decision.Broker {
	return NewBrokerWithSummaryAndUI(sender, summarize, ui, opts...)
}

func (h *DecisionHandler) syncDecisionHandler() *Handler {
	if h == nil {
		return nil
	}
	if h.Handler == nil {
		h.Handler = NewHandler(h.sender, h.router, h.broker, h.store)
	}
	if h.Handler != nil {
		h.Handler.SetRouter(h.router)
		h.Handler.SetInterruptTimeout(h.interruptTimeout)
		h.Handler.SetStopWordTimeout(h.stopWordTimeout)
		h.Handler.SetArtifactRetentionTimeout(h.artifactRetentionTimeout)
	}
	return h.Handler
}

func (h *DecisionHandler) SetRouter(router telegramDecisionRouter) {
	if h != nil {
		h.router = router
		if h.Handler != nil {
			h.Handler.SetRouter(router)
		}
	}
}

func (h *DecisionHandler) SetInterruptTimeout(timeout time.Duration) {
	if h != nil {
		h.interruptTimeout = timeout
		if h.Handler != nil {
			h.Handler.SetInterruptTimeout(timeout)
		}
	}
}

func (h *DecisionHandler) SetStopWordTimeout(timeout time.Duration) {
	if h != nil {
		h.stopWordTimeout = timeout
		if h.Handler != nil {
			h.Handler.SetStopWordTimeout(timeout)
		}
	}
}

func (h *DecisionHandler) SetArtifactRetentionTimeout(timeout time.Duration) {
	if h != nil {
		h.artifactRetentionTimeout = timeout
		if h.Handler != nil {
			h.Handler.SetArtifactRetentionTimeout(timeout)
		}
	}
}

func (h *DecisionHandler) InterruptTimeout() time.Duration {
	if h == nil {
		return 0
	}
	return h.interruptTimeout
}

func (h *DecisionHandler) StopWordTimeout() time.Duration {
	if h == nil {
		return 0
	}
	return h.stopWordTimeout
}

func (h *DecisionHandler) ArtifactRetentionTimeout() time.Duration {
	if h == nil {
		return 0
	}
	return h.artifactRetentionTimeout
}

func (h *DecisionHandler) HandleBusyMessage(ctx context.Context, msg core.InboundMessage) (bool, error) {
	inner := h.syncDecisionHandler()
	if inner == nil {
		return false, nil
	}
	return inner.HandleBusyMessage(ctx, msg)
}

func (h *DecisionHandler) HandleArtifactRetentionMessage(ctx context.Context, msg core.InboundMessage) (bool, error) {
	inner := h.syncDecisionHandler()
	if inner == nil {
		return false, nil
	}
	return inner.HandleArtifactRetentionMessage(ctx, msg)
}

func (h *DecisionHandler) ResumePendingBusyDecision(ctx context.Context, ownerKey string, result decision.Result) error {
	inner := h.syncDecisionHandler()
	if inner == nil {
		return nil
	}
	return inner.ResumePendingBusyDecision(ctx, ownerKey, result)
}

func (h *DecisionHandler) resumePendingBusyDecision(ctx context.Context, ownerKey string, result decision.Result) error {
	return h.ResumePendingBusyDecision(ctx, ownerKey, result)
}

func (h *DecisionHandler) ResumePendingArtifactRetention(ctx context.Context, ownerKey string, result decision.Result) error {
	inner := h.syncDecisionHandler()
	if inner == nil {
		return nil
	}
	return inner.ResumePendingArtifactRetention(ctx, ownerKey, result)
}

func (h *DecisionHandler) resumePendingArtifactRetention(ctx context.Context, ownerKey string, result decision.Result) error {
	return h.ResumePendingArtifactRetention(ctx, ownerKey, result)
}

func (h *DecisionHandler) ReconcileRestartLoadedDecisions(ctx context.Context) error {
	inner := h.syncDecisionHandler()
	if inner == nil {
		return nil
	}
	return inner.ReconcileRestartLoadedDecisions(ctx)
}

func (h *DecisionHandler) DecisionResumeStatus(msg core.InboundMessage, surface string) (DecisionResumeStatus, error) {
	inner := h.syncDecisionHandler()
	if inner == nil {
		return DecisionResumeMissing, nil
	}
	return inner.DecisionResumeStatus(msg, surface)
}

func (h *DecisionHandler) decisionResumeStatus(msg core.InboundMessage, surface string) (DecisionResumeStatus, error) {
	return h.DecisionResumeStatus(msg, surface)
}

func (h *DecisionHandler) HandleCallbackQuery(ctx context.Context, cb telegram.CallbackQuery) error {
	if eventID, action, ok := core.DecodeReviewEventCallbackData(cb.Data); ok {
		return h.handleReviewEventCallback(ctx, cb, eventID, action)
	}
	if h == nil || h.Handler == nil {
		return nil
	}
	return h.Handler.HandleCallbackQuery(ctx, cb)
}

func callbackChatID(cb telegram.CallbackQuery) int64   { return CallbackChatID(cb) }
func callbackSenderID(cb telegram.CallbackQuery) int64 { return CallbackSenderID(cb) }
func callbackMessageID(cb telegram.CallbackQuery) int64 {
	return CallbackMessageID(cb)
}
