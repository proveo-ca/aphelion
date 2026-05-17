//go:build linux

package session

import (
	"testing"
	"time"
)

func TestOperatorAutoApprovalLeaseLifecycle(t *testing.T) {
	t.Parallel()

	store := newTestSQLiteStore(t)
	defer store.Close()

	now := time.Date(2026, 5, 4, 22, 0, 0, 0, time.UTC)
	created, err := store.CreateOperatorAutoApprovalLease(OperatorAutoApprovalLease{
		ID:          "auto-test",
		AdminUserID: 1001,
		ChatID:      7001,
		Scope:       "workspace",
		MaxUses:     1,
		CreatedAt:   now,
		ExpiresAt:   now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateOperatorAutoApprovalLease() err = %v", err)
	}
	if created.Scope != OperatorAutoApprovalScopeWorkspace || !created.ActiveAt(now) {
		t.Fatalf("created lease = %#v, want active workspace lease", created)
	}

	active, err := store.ActiveOperatorAutoApprovalLeases(7001, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ActiveOperatorAutoApprovalLeases() err = %v", err)
	}
	if len(active) != 1 || active[0].ID != "auto-test" {
		t.Fatalf("active leases = %#v, want auto-test", active)
	}
	listed, err := store.OperatorAutoApprovalLeases(10, now.Add(time.Minute), true)
	if err != nil {
		t.Fatalf("OperatorAutoApprovalLeases(activeOnly) err = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "auto-test" {
		t.Fatalf("listed active leases = %#v, want auto-test", listed)
	}
	used, ok, err := store.IncrementOperatorAutoApprovalUse("auto-test", now.Add(2*time.Minute))
	if err != nil || !ok {
		t.Fatalf("IncrementOperatorAutoApprovalUse() = lease:%#v ok:%v err:%v, want ok", used, ok, err)
	}
	if used.UsedCount != 1 || used.ActiveAt(now.Add(3*time.Minute)) {
		t.Fatalf("used lease = %#v, want exhausted after one use", used)
	}
	active, err = store.ActiveOperatorAutoApprovalLeases(7001, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("ActiveOperatorAutoApprovalLeases(after use) err = %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active leases after use = %#v, want none", active)
	}
	allListed, err := store.OperatorAutoApprovalLeases(10, now.Add(3*time.Minute), false)
	if err != nil {
		t.Fatalf("OperatorAutoApprovalLeases(all) err = %v", err)
	}
	if len(allListed) != 1 || allListed[0].ID != "auto-test" || allListed[0].UsedCount != 1 {
		t.Fatalf("all listed leases = %#v, want exhausted auto-test", allListed)
	}

	revoked, err := store.RevokeOperatorAutoApprovalLeases(7001, 1001, now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("RevokeOperatorAutoApprovalLeases() err = %v", err)
	}
	if len(revoked) != 1 || revoked[0].ID != "auto-test" {
		t.Fatalf("revoked = %#v, want auto-test", revoked)
	}
}

func TestOperatorAutonomyOverrideLifecycle(t *testing.T) {
	t.Parallel()

	store := newTestSQLiteStore(t)
	defer store.Close()

	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	created, err := store.CreateOperatorAutonomyOverride(OperatorAutonomyOverride{
		ID:          "mode-test",
		AdminUserID: 1001,
		ChatID:      7002,
		Mode:        "leased",
		Scope:       "deploy",
		Reason:      "bounded release",
		CreatedAt:   now,
		ExpiresAt:   now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateOperatorAutonomyOverride() err = %v", err)
	}
	if created.Scope != OperatorAutoApprovalScopeDeploy || created.Mode != "leased" || !created.ActiveAt(now) {
		t.Fatalf("created override = %#v, want active deploy leased override", created)
	}

	active, err := store.ActiveOperatorAutonomyOverrides(7002, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ActiveOperatorAutonomyOverrides() err = %v", err)
	}
	if len(active) != 1 || active[0].ID != "mode-test" {
		t.Fatalf("active overrides = %#v, want mode-test", active)
	}
	latest, ok, err := store.LatestOperatorAutonomyOverride(7002, 1001)
	if err != nil || !ok {
		t.Fatalf("LatestOperatorAutonomyOverride() = override:%#v ok:%v err:%v, want ok", latest, ok, err)
	}
	if latest.ID != "mode-test" {
		t.Fatalf("latest override = %#v, want mode-test", latest)
	}

	revoked, err := store.RevokeOperatorAutonomyOverrides(7002, 1001, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("RevokeOperatorAutonomyOverrides() err = %v", err)
	}
	if len(revoked) != 1 || revoked[0].ID != "mode-test" {
		t.Fatalf("revoked = %#v, want mode-test", revoked)
	}
	active, err = store.ActiveOperatorAutonomyOverrides(7002, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("ActiveOperatorAutonomyOverrides(after revoke) err = %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active overrides after revoke = %#v, want none", active)
	}
}
