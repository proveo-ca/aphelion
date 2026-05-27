//go:build linux

package doctor

import (
	"github.com/idolum-ai/aphelion/agent"
	"github.com/idolum-ai/aphelion/config"
	"github.com/idolum-ai/aphelion/principal"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/tool/sandbox"
	"strings"
	"time"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func callString(fn func() string) string {
	if fn == nil {
		return ""
	}
	return fn()
}

func firstConfiguredWorkExecutor(cfg config.WorkConfig) string {
	if trimmed := strings.TrimSpace(cfg.Executor); trimmed != "" {
		return trimmed
	}
	for _, candidate := range cfg.AutoOrder {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return "native"
}

func reasoningOptionsForRun(r *Runtime, kind session.TurnRunKind) *agent.CompleteOptions {
	if r == nil || r.reasoningOptionsForRun == nil {
		return nil
	}
	return r.reasoningOptionsForRun(kind)
}

func recordExecutionEvent(r *Runtime, key session.SessionKey, eventType string, stage string, status string, payload map[string]any, createdAt time.Time) {
	if r == nil || r.recordExecutionEvent == nil {
		return
	}
	r.recordExecutionEvent(key, eventType, stage, status, payload, createdAt)
}

func reportOperationalIssueAsync(r *Runtime, component string, err error) {
	if r == nil || r.reportOperationalIssueAsync == nil {
		return
	}
	r.reportOperationalIssueAsync(component, err)
}

func truncatePreview(raw string, limit int) string {
	raw = strings.TrimSpace(raw)
	if limit <= 0 || len(raw) <= limit {
		return raw
	}
	if limit <= 3 {
		return raw[:limit]
	}
	return raw[:limit-3] + "..."
}

func trimError(raw string) string { return truncatePreview(strings.TrimSpace(raw), 400) }

func roundDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d >= time.Hour {
		return d.Round(time.Minute).String()
	}
	return d.Round(time.Second).String()
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func DynamicPromptRoot(scope sandbox.Scope) string {
	if scope.Principal.Role == principal.RoleApprovedUser && strings.TrimSpace(scope.UserMemory) != "" {
		return scope.UserMemory
	}
	if strings.TrimSpace(scope.SharedMemoryRoot) != "" {
		return scope.SharedMemoryRoot
	}
	return scope.WorkingRoot
}

func ContinuationWorkMode(state session.ContinuationState) string {
	state = session.NormalizeContinuationState(state)
	if phase, ok := currentContinuationBundlePhase(state.ApprovalBundle); ok {
		if mode := workModeFromStructuredAuthority(phase.AuthorityClass); mode != "" {
			return mode
		}
	}
	proposal := session.NormalizeActionProposal(state.ActionProposal)
	mode := strongestWorkMode(
		workModeFromStructuredAuthority(proposal.RiskClass),
		workModeFromStructuredAuthorityList(proposal.AllowedActions),
		workModeFromStructuredAuthorityList(state.ContinuationLease.AllowedActions),
	)
	if mode != "" {
		return mode
	}
	lower := strings.ToLower(strings.Join([]string{proposal.Summary, proposal.BoundedEffect, state.StageSummary}, " "))
	if strings.Contains(lower, "read_only") || strings.Contains(lower, "status_check") {
		return "read_only"
	}
	return ""
}

func currentContinuationBundlePhase(bundle session.ContinuationApprovalBundle) (session.ContinuationApprovalBundlePhase, bool) {
	bundle = session.NormalizeContinuationApprovalBundle(bundle)
	for _, phase := range bundle.Phases {
		if phase.Status == session.ContinuationLeaseStatusActive || phase.Status == session.ContinuationLeaseStatusPending {
			return phase, true
		}
	}
	return session.ContinuationApprovalBundlePhase{}, false
}

func workModeFromStructuredAuthority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "read_only", "read-only", "read_only_review", "status_check", "status-check":
		return "read_only"
	case "workspace_write", "workspace-write", "write", "edit":
		return "workspace_write"
	case "commit":
		return "commit"
	case "deploy", "restart", "service_restart":
		return "deploy"
	default:
		return ""
	}
}

func workModeFromStructuredAuthorityList(values []string) string {
	mode := ""
	for _, value := range values {
		mode = strongestWorkMode(mode, workModeFromStructuredAuthority(value))
	}
	return mode
}

func strongestWorkMode(left string, values ...string) string {
	strongest := left
	for _, value := range values {
		if workModeRank(value) > workModeRank(strongest) {
			strongest = value
		}
	}
	return strongest
}

func workModeRank(mode string) int {
	switch strings.TrimSpace(mode) {
	case "read_only":
		return 1
	case "workspace_write":
		return 2
	case "commit":
		return 3
	case "deploy":
		return 4
	default:
		return 0
	}
}
