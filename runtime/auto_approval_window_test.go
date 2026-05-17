//go:build linux

package runtime

import (
	"context"
	"github.com/idolum-ai/aphelion/decision"
	"strings"
	"testing"
	"time"
)

func TestRuntimeAutoApprovalDoubleExtendsWindowAndPreservesRemainingUses(t *testing.T) {
	t.Parallel()

	cfg, store, provider, sender := buildRuntimeFixtures(t)
	rt, err := New(cfg, store, provider, nil, sender)
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	if _, err := rt.ConfigureAutonomy(context.Background(), 99201, 1001, "leased 15m all"); err != nil {
		t.Fatalf("ConfigureAutonomy() err = %v", err)
	}
	if _, err := rt.ConfigureAutoApproval(context.Background(), 99201, 1001, "15m all uses=3 double window"); err != nil {
		t.Fatalf("ConfigureAutoApproval(enable) err = %v", err)
	}
	result, err := rt.AutoResolveDecision(context.Background(), decision.PendingDecision{
		ID: "dec-double-uses",
		Request: decision.Request{
			Kind:          decision.KindProposalApproval,
			ChatID:        99201,
			SenderID:      1002,
			Prompt:        "Approve this proposal?",
			Details:       "Run a bounded workspace check.",
			Choices:       []decision.Choice{{ID: "deny", Label: "Deny"}, {ID: "approve", Label: "Approve"}},
			DefaultChoice: "deny",
		},
	})
	if err != nil {
		t.Fatalf("AutoResolveDecision() err = %v", err)
	}
	if result.Choice != "approve" {
		t.Fatalf("auto resolution = %#v, want approve", result)
	}

	text, err := rt.ConfigureAutoApproval(context.Background(), 99201, 1001, "double")
	if err != nil {
		t.Fatalf("ConfigureAutoApproval(double) err = %v", err)
	}
	if !strings.Contains(text, "Doubled: 15m0s → 30m0s") || !strings.Contains(text, "Use budget remaining: 2 approval(s).") {
		t.Fatalf("double text = %q, want doubled duration and remaining-use budget", text)
	}
	leases, err := store.ActiveOperatorAutoApprovalLeases(99201, time.Now().UTC())
	if err != nil {
		t.Fatalf("ActiveOperatorAutoApprovalLeases() err = %v", err)
	}
	if len(leases) != 1 || leases[0].MaxUses != 2 || leases[0].UsedCount != 0 {
		t.Fatalf("leases = %#v, want fresh grant with 2 remaining uses and no used count", leases)
	}
	assertOperatorWindowDuration(t, leases[0].CreatedAt, leases[0].ExpiresAt, 30*time.Minute)

	if _, err := rt.ConfigureAutoApproval(context.Background(), 99201, 1001, "double"); err != nil {
		t.Fatalf("ConfigureAutoApproval(double again) err = %v", err)
	}
	leases, err = store.ActiveOperatorAutoApprovalLeases(99201, time.Now().UTC())
	if err != nil {
		t.Fatalf("ActiveOperatorAutoApprovalLeases(second) err = %v", err)
	}
	if len(leases) != 1 || leases[0].MaxUses != 2 || leases[0].UsedCount != 0 {
		t.Fatalf("leases second = %#v, want remaining use budget preserved", leases)
	}
	assertOperatorWindowDuration(t, leases[0].CreatedAt, leases[0].ExpiresAt, time.Hour)
}

func TestRuntimeAutoApprovalDoubleCapsAtTwentyFourHours(t *testing.T) {
	t.Parallel()

	cfg, store, provider, sender := buildRuntimeFixtures(t)
	rt, err := New(cfg, store, provider, nil, sender)
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	if _, err := rt.ConfigureAutoApproval(context.Background(), 99202, 1001, "20h all"); err != nil {
		t.Fatalf("ConfigureAutoApproval(enable) err = %v", err)
	}
	text, err := rt.ConfigureAutoApproval(context.Background(), 99202, 1001, "double")
	if err != nil {
		t.Fatalf("ConfigureAutoApproval(double) err = %v", err)
	}
	if !strings.Contains(text, "Doubled: 20h0m0s → 24h0m0s") {
		t.Fatalf("double text = %q, want capped 24h duration", text)
	}
	leases, err := store.ActiveOperatorAutoApprovalLeases(99202, time.Now().UTC())
	if err != nil {
		t.Fatalf("ActiveOperatorAutoApprovalLeases() err = %v", err)
	}
	if len(leases) != 1 {
		t.Fatalf("leases = %#v, want one active capped lease", leases)
	}
	assertOperatorWindowDuration(t, leases[0].CreatedAt, leases[0].ExpiresAt, 24*time.Hour)
}

func TestRuntimeAutoApprovalDoubleRequiresActiveGrant(t *testing.T) {
	t.Parallel()

	cfg, store, provider, sender := buildRuntimeFixtures(t)
	rt, err := New(cfg, store, provider, nil, sender)
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	_, err = rt.ConfigureAutoApproval(context.Background(), 99203, 1001, "double")
	if err == nil || !strings.Contains(err.Error(), "no active auto approval to double") {
		t.Fatalf("ConfigureAutoApproval(double) err = %v, want no-active grant error", err)
	}
}

func TestRuntimeAutoModeDoubleExtendsWindowAndRespectsCap(t *testing.T) {
	t.Parallel()

	cfg, store, provider, sender := buildRuntimeFixtures(t)
	cfg.Autonomy.Ceiling = "leased"
	cfg.Autonomy.AllowLiveOverrides = true
	cfg.Autonomy.MaxOverrideDuration = "45m"
	rt, err := New(cfg, store, provider, nil, sender)
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	if _, err := rt.ConfigureAutonomy(context.Background(), 99204, 1001, "leased 15m all"); err != nil {
		t.Fatalf("ConfigureAutonomy(enable) err = %v", err)
	}
	text, err := rt.ConfigureAutonomy(context.Background(), 99204, 1001, "double")
	if err != nil {
		t.Fatalf("ConfigureAutonomy(double) err = %v", err)
	}
	if !strings.Contains(text, "Doubled: 15m0s → 30m0s") {
		t.Fatalf("double text = %q, want 15m to 30m", text)
	}
	overrides, err := store.ActiveOperatorAutonomyOverrides(99204, time.Now().UTC())
	if err != nil {
		t.Fatalf("ActiveOperatorAutonomyOverrides() err = %v", err)
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides = %#v, want one active override", overrides)
	}
	assertOperatorWindowDuration(t, overrides[0].CreatedAt, overrides[0].ExpiresAt, 30*time.Minute)

	text, err = rt.ConfigureAutonomy(context.Background(), 99204, 1001, "double")
	if err != nil {
		t.Fatalf("ConfigureAutonomy(double capped) err = %v", err)
	}
	if !strings.Contains(text, "Doubled: 30m0s → 45m0s") {
		t.Fatalf("double text = %q, want cap at 45m", text)
	}
	overrides, err = store.ActiveOperatorAutonomyOverrides(99204, time.Now().UTC())
	if err != nil {
		t.Fatalf("ActiveOperatorAutonomyOverrides(capped) err = %v", err)
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides capped = %#v, want one active override", overrides)
	}
	assertOperatorWindowDuration(t, overrides[0].CreatedAt, overrides[0].ExpiresAt, 45*time.Minute)
}

func TestRuntimeAutoModeDoubleRequiresActiveOverride(t *testing.T) {
	t.Parallel()

	cfg, store, provider, sender := buildRuntimeFixtures(t)
	rt, err := New(cfg, store, provider, nil, sender)
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	_, err = rt.ConfigureAutonomy(context.Background(), 99205, 1001, "double")
	if err == nil || !strings.Contains(err.Error(), "no active auto mode to double") {
		t.Fatalf("ConfigureAutonomy(double) err = %v, want no-active mode error", err)
	}
}

func assertOperatorWindowDuration(t *testing.T, created time.Time, expires time.Time, want time.Duration) {
	t.Helper()
	got := expires.Sub(created)
	if got < want-time.Second || got > want+time.Second {
		t.Fatalf("window duration = %s, want approximately %s (created=%s expires=%s)", got, want, created, expires)
	}
}

func TestParseOperatorAutoApprovalDurationCapAllowsTwentyFourHours(t *testing.T) {
	t.Parallel()

	action, spec, err := parseOperatorAutoApprovalCommand("24h all")
	if err != nil {
		t.Fatalf("parseOperatorAutoApprovalCommand(24h) err = %v", err)
	}
	if action != "enable" || spec.Duration != 24*time.Hour {
		t.Fatalf("action/spec = %q/%#v, want enable with 24h duration", action, spec)
	}

	_, _, err = parseOperatorAutoApprovalCommand("25h all")
	if err == nil || !strings.Contains(err.Error(), "24h0m0s") {
		t.Fatalf("parseOperatorAutoApprovalCommand(25h) err = %v, want 24h cap error", err)
	}
}
