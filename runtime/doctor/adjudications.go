//go:build linux

package doctor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/session"
)

func statusAdjudicationsFromExecutionEvents(events []session.ExecutionEvent, limit int) []core.AdjudicationStatusSnapshot {
	if len(events) == 0 || limit == 0 {
		return nil
	}
	if limit < 0 {
		limit = len(events)
	}
	ordered := append([]session.ExecutionEvent(nil), events...)
	sort.Slice(ordered, func(i, j int) bool { return executionEventBefore(ordered[i], ordered[j]) })
	out := make([]core.AdjudicationStatusSnapshot, 0, minInt(limit, len(ordered)))
	for i := len(ordered) - 1; i >= 0; i-- {
		event := ordered[i]
		eventType := strings.TrimSpace(event.EventType)
		if eventType != core.ExecutionEventReplyClaimAdjudicated && eventType != core.ExecutionEventContinuationAdjudicated {
			continue
		}
		adjudication, ok := runtimeAdjudicationFromExecutionEvent(event)
		if !ok {
			continue
		}
		out = append(out, core.AdjudicationStatusSnapshot{SessionID: strings.TrimSpace(event.SessionID), ChatID: event.ChatID, Seq: event.Seq, Kind: adjudication.Kind, Surface: adjudication.Surface, SubjectID: adjudication.SubjectID, OperatorLabel: adjudication.OperatorLabel, VisibleAction: adjudication.VisibleAction, Findings: append([]core.RuntimeFinding(nil), adjudication.Findings...), EvidenceRefs: append([]string(nil), adjudication.EvidenceRefs...), CreatedAt: event.CreatedAt})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func executionEventBefore(left session.ExecutionEvent, right session.ExecutionEvent) bool {
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.Before(right.CreatedAt)
	}
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	return left.Seq < right.Seq
}

func runtimeAdjudicationFromExecutionEvent(event session.ExecutionEvent) (core.RuntimeAdjudication, bool) {
	payload := executionEventPayload(event.PayloadJSON)
	if len(payload) == 0 {
		return core.RuntimeAdjudication{}, false
	}
	findings := payloadRuntimeFindings(payload, "findings")
	if len(findings) == 0 {
		claimTypes := payloadStringSlice(payload, "claim_types")
		details := payloadStringSlice(payload, "details")
		for i, claimType := range claimTypes {
			detail := ""
			if i < len(details) {
				detail = details[i]
			}
			findings = append(findings, core.RuntimeFinding{Kind: claimType, ClaimType: claimType, EvidenceStatus: "not_observed_in_current_turn", Detail: detail})
		}
	}
	adjudication := core.NormalizeRuntimeAdjudication(core.RuntimeAdjudication{Kind: firstNonEmpty(payloadString(payload, "adjudication_kind"), "execution_claim"), Surface: firstNonEmpty(payloadString(payload, "surface"), "final_reply"), SubjectID: firstNonEmpty(payloadString(payload, "subject_id"), "latest_turn"), OperatorLabel: firstNonEmpty(payloadString(payload, "operator_label"), executionClaimOperatorLabel(payloadString(payload, "visible_action"))), Findings: findings, EvidenceRefs: payloadStringSlice(payload, "evidence_refs"), VisibleAction: payloadString(payload, "visible_action"), CreatedAt: event.CreatedAt})
	if adjudication.Kind == "" && len(adjudication.Findings) == 0 {
		return core.RuntimeAdjudication{}, false
	}
	return adjudication, true
}

func executionEventPayload(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	return payload
}
func payloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	value, ok := payload[strings.TrimSpace(key)]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
func payloadStringSlice(payload map[string]any, key string) []string {
	if len(payload) == 0 {
		return nil
	}
	raw, ok := payload[strings.TrimSpace(key)]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if v := strings.TrimSpace(fmt.Sprint(item)); v != "" {
				out = append(out, v)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if v := strings.TrimSpace(item); v != "" {
				out = append(out, v)
			}
		}
		return out
	default:
		return splitCSV(strings.TrimSpace(fmt.Sprint(typed)))
	}
}
func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
func payloadRuntimeFindings(payload map[string]any, key string) []core.RuntimeFinding {
	raw, ok := payload[strings.TrimSpace(key)]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]core.RuntimeFinding, 0, len(items))
	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var finding core.RuntimeFinding
		if err := json.Unmarshal(b, &finding); err != nil {
			continue
		}
		out = append(out, finding)
	}
	return out
}
func executionClaimOperatorLabel(visibleAction string) string {
	switch strings.TrimSpace(visibleAction) {
	case "repair_requested":
		return "Reply claim needs repair"
	case "persona_repaired":
		return "Reply claim repaired"
	case "fallback_neutralized":
		return "Reply claim neutralized"
	default:
		return "Reply claim adjudicated"
	}
}
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
