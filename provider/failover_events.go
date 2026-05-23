//go:build linux

package provider

import (
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/core"
)

func recordProviderRetryEvent(events *[]core.ProviderEvent, provider string, attempt int, err error) {
	if events == nil {
		return
	}
	kind := core.ProviderFailureKind(err)
	*events = append(*events, core.ProviderEvent{
		EventType:        core.ExecutionEventProviderAttemptRetried,
		ObservedAt:       time.Now().UTC(),
		Provider:         strings.TrimSpace(provider),
		Attempt:          attempt,
		MaxRetries:       failoverMaxRetries,
		Error:            trimProviderEventError(err),
		FailureKind:      kind,
		Retryable:        core.ProviderFailureRetryable(kind),
		FailoverEligible: core.ProviderFailureFailoverEligible(kind),
	})
}

func recordProviderFailedEvent(events *[]core.ProviderEvent, provider string, err error) {
	if events == nil {
		return
	}
	kind := core.ProviderFailureKind(err)
	*events = append(*events, core.ProviderEvent{
		EventType:        core.ExecutionEventProviderAttemptFailed,
		ObservedAt:       time.Now().UTC(),
		Provider:         strings.TrimSpace(provider),
		Error:            trimProviderEventError(err),
		FailureKind:      kind,
		Retryable:        core.ProviderFailureRetryable(kind),
		FailoverEligible: core.ProviderFailureFailoverEligible(kind),
	})
}

type providerFailureCoder interface {
	ProviderFailureCode() string
}

type providerRetryAfterer interface {
	ProviderRetryAfter() time.Duration
}

func recordProviderPartialEvent(events *[]core.ProviderEvent, provider string, err error) {
	if events == nil {
		return
	}
	partial, responseID, reason, ok := partialProviderSnapshot(err)
	if !ok {
		return
	}
	kind := core.ProviderFailureKind(err)
	event := core.ProviderEvent{
		EventType:        core.ExecutionEventProviderPartial,
		ObservedAt:       time.Now().UTC(),
		Provider:         strings.TrimSpace(provider),
		Reason:           reason,
		ResponseID:       responseID,
		Error:            trimProviderEventError(err),
		FailureKind:      kind,
		Retryable:        core.ProviderFailureRetryable(kind),
		FailoverEligible: core.ProviderFailureFailoverEligible(kind),
	}
	if partial != nil {
		event.PartialContentChars = len(strings.TrimSpace(partial.Content))
		event.PartialToolCalls = len(partial.ToolCalls)
	}
	*events = append(*events, event)
}

func recordProviderFailoverEvent(events *[]core.ProviderEvent, from string, to string, err error) {
	if events == nil || strings.TrimSpace(to) == "" {
		return
	}
	kind := core.ProviderFailureKind(err)
	*events = append(*events, core.ProviderEvent{
		EventType:        core.ExecutionEventProviderFailoverEngaged,
		ObservedAt:       time.Now().UTC(),
		FromProvider:     strings.TrimSpace(from),
		ToProvider:       strings.TrimSpace(to),
		Error:            trimProviderEventError(err),
		FailureKind:      kind,
		Retryable:        core.ProviderFailureRetryable(kind),
		FailoverEligible: core.ProviderFailureFailoverEligible(kind),
	})
}

func trimProviderEventError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if len(text) > 500 {
		return text[:500] + "..."
	}
	return text
}
