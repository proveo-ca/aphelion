//go:build linux

package mission

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/agent"
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/face"
	"github.com/idolum-ai/aphelion/internal/telegrampresentation"
	"github.com/idolum-ai/aphelion/principal"
	"github.com/idolum-ai/aphelion/session"
)

var ErrPrincipalDenied = errors.New("principal is not admitted")

type Dependencies struct {
	Store                            *session.SQLiteStore
	Resolver                         *principal.Resolver
	Outbound                         any
	ModelSlotProvider                func(slot string) (agent.Provider, core.ModelSlotStatus, bool)
	RecordExecutionEvent             func(key session.SessionKey, eventType string, stage string, status string, payload map[string]any, createdAt time.Time)
	PrefixTelegramText               func(msg core.InboundMessage, text string) string
	OperationArtifactRequestUserText func(text string) string
}

type Runtime struct {
	store                                *session.SQLiteStore
	resolver                             *principal.Resolver
	outbound                             any
	modelSlotProviderFunc                func(slot string) (agent.Provider, core.ModelSlotStatus, bool)
	recordExecutionEventFunc             func(key session.SessionKey, eventType string, stage string, status string, payload map[string]any, createdAt time.Time)
	prefixTelegramTextFunc               func(msg core.InboundMessage, text string) string
	operationArtifactRequestUserTextFunc func(text string) string
}

func NewRuntime(deps Dependencies) *Runtime {
	return &Runtime{
		store:                                deps.Store,
		resolver:                             deps.Resolver,
		outbound:                             deps.Outbound,
		modelSlotProviderFunc:                deps.ModelSlotProvider,
		recordExecutionEventFunc:             deps.RecordExecutionEvent,
		prefixTelegramTextFunc:               deps.PrefixTelegramText,
		operationArtifactRequestUserTextFunc: deps.OperationArtifactRequestUserText,
	}
}

func (r *Runtime) modelSlotProvider(slot string) (agent.Provider, core.ModelSlotStatus, bool) {
	if r == nil || r.modelSlotProviderFunc == nil {
		return nil, core.ModelSlotStatus{}, false
	}
	return r.modelSlotProviderFunc(slot)
}

func (r *Runtime) recordExecutionEvent(key session.SessionKey, eventType string, stage string, status string, payload map[string]any, createdAt time.Time) {
	if r == nil || r.recordExecutionEventFunc == nil {
		return
	}
	r.recordExecutionEventFunc(key, eventType, stage, status, payload, createdAt)
}

func (r *Runtime) prefixTelegramText(msg core.InboundMessage, text string) string {
	if r != nil && r.prefixTelegramTextFunc != nil {
		return r.prefixTelegramTextFunc(msg, text)
	}
	return text
}

func (r *Runtime) operationArtifactRequestUserText(text string) string {
	if r != nil && r.operationArtifactRequestUserTextFunc != nil {
		return r.operationArtifactRequestUserTextFunc(text)
	}
	return operationArtifactRequestUserText(text)
}

func TelegramDMScopeRef(chatID int64) session.ScopeRef {
	if chatID == 0 {
		return session.ScopeRef{}
	}
	return session.ScopeRef{Kind: session.ScopeKindTelegramDM, ID: strconv.FormatInt(chatID, 10)}
}

func CommandOwner(actor principal.Principal, senderID int64) string {
	if strings.TrimSpace(actor.DurableAgentID) != "" {
		return "durable_agent:" + strings.TrimSpace(actor.DurableAgentID)
	}
	if actor.TelegramUserID > 0 {
		return "telegram:" + strconv.FormatInt(actor.TelegramUserID, 10)
	}
	if senderID > 0 {
		return "telegram:" + strconv.FormatInt(senderID, 10)
	}
	return "system"
}

func renderRuntimeCompactPanel(panel face.OperatorPanel) string {
	return face.RenderCompactOperatorPanel(panel, face.OperatorPanelCompactOptions{DetailLimit: 4, EvidenceLimit: 2})
}

func operationArtifactRequestUserText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	const replyContextMarker = "\n\nReply context:\n"
	if idx := strings.Index(text, replyContextMarker); idx >= 0 {
		return strings.TrimSpace(text[:idx])
	}
	return text
}

func prefixTelegramPresentationText(prefix string, text string) string {
	return telegrampresentation.PrefixText(prefix, text)
}

func unavailableStoreError(subject string) error {
	return fmt.Errorf("%s is unavailable: session store is not configured", subject)
}

func firstRuntimeNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func hiddenInputTokens(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(text)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-')
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, "_- ")
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}
