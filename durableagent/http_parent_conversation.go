//go:build linux

package durableagent

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/core"
)

func (h *HTTPHandler) pendingParentConversation(agentID string, limit int) ([]core.DurableAgentConversationMessage, error) {
	if h == nil || h.store == nil {
		return nil, fmt.Errorf("durable agent control plane store is nil")
	}
	state, err := h.store.DurableAgentState(strings.TrimSpace(agentID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	continuity, err := core.ParseDurableAgentContinuityState(state.StateJSON)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	return continuity.PendingParentConversationMessages(limit), nil
}

func (h *HTTPHandler) acknowledgeParentConversation(agentID string, messageIDs []string, at time.Time) error {
	if h == nil || h.store == nil {
		return fmt.Errorf("durable agent control plane store is nil")
	}
	_, err := h.store.UpdateDurableAgentState(strings.TrimSpace(agentID), func(state *core.DurableAgentState) error {
		continuity, err := core.ParseDurableAgentContinuityState(state.StateJSON)
		if err != nil {
			return err
		}
		continuity, err = continuity.AcknowledgeParentConversationMessageIDs(messageIDs, at)
		if err != nil {
			return fmt.Errorf("%w: %v", errParentConversationAckRejected, err)
		}
		state.StateJSON, err = continuity.Marshal()
		return err
	})
	return err
}
