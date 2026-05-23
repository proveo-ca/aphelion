//go:build linux

package core

import (
	"errors"
	"testing"
)

func TestProviderFailureKindClassifiesTransportTimeout(t *testing.T) {
	err := errors.New(`codex: request: Post "https://chatgpt.com/backend-api/codex/responses": http2: timeout awaiting response headers`)
	if got := ProviderFailureKind(err); got != ProviderFailureTransportTimeout {
		t.Fatalf("ProviderFailureKind() = %q, want %q", got, ProviderFailureTransportTimeout)
	}
	if !ProviderFailureRetryable(ProviderFailureKind(err)) {
		t.Fatal("transport timeout should be retryable")
	}
	if !ProviderFailureFailoverEligible(ProviderFailureKind(err)) {
		t.Fatal("transport timeout should be failover eligible")
	}
}

func TestProviderFailureKindClassifiesBudgetAndBuffer(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "context budget",
			err:  errors.New("context_budget_exceeded: estimated_input_tokens=300 hard_limit_tokens=200 context_window=180"),
			want: ProviderFailureContextBudget,
		},
		{
			name: "request buffer",
			err:  errors.New("codex: status 507 server_error: exceeded request buffer limit while retrying upstream"),
			want: ProviderFailureRequestBuffer,
		},
		{
			name: "continuation",
			err:  errors.New("codex: stored-response continuation rejected after incomplete response"),
			want: ProviderFailureContinuationRejected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProviderFailureKind(tt.err); got != tt.want {
				t.Fatalf("ProviderFailureKind() = %q, want %q", got, tt.want)
			}
		})
	}
}
