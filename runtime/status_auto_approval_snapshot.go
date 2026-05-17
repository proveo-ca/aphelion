//go:build linux

package runtime

import (
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/session"
	"strings"
	"time"
)

func (r *Runtime) autoApprovalStatusSnapshot(chatID int64, now time.Time) (*core.AutoApprovalStatusSnapshot, error) {
	if r == nil || r.store == nil || chatID == 0 {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	leases, err := r.store.ActiveOperatorAutoApprovalLeases(chatID, now)
	if err != nil {
		return nil, err
	}
	var selected *session.OperatorAutoApprovalLease
	for _, lease := range leases {
		lease = session.NormalizeOperatorAutoApprovalLease(lease)
		if !lease.ActiveAt(now) {
			continue
		}
		if selected == nil || lease.ExpiresAt.After(selected.ExpiresAt) {
			copied := lease
			selected = &copied
		}
	}
	if selected == nil {
		return nil, nil
	}
	blocked, err := r.operatorAutoApprovalBlockedReason(chatID, selected.AdminUserID, selected.Scope, now)
	if err != nil {
		return nil, err
	}
	return &core.AutoApprovalStatusSnapshot{
		Active:        true,
		Usable:        strings.TrimSpace(blocked) == "",
		BlockedReason: strings.TrimSpace(blocked),
		LeaseID:       strings.TrimSpace(selected.ID),
		AdminUserID:   selected.AdminUserID,
		Scope:         strings.TrimSpace(selected.Scope),
		UsedCount:     selected.UsedCount,
		MaxUses:       selected.MaxUses,
		Reason:        strings.TrimSpace(selected.Reason),
		CreatedAt:     selected.CreatedAt,
		UpdatedAt:     selected.UpdatedAt,
		ExpiresAt:     selected.ExpiresAt,
	}, nil
}
