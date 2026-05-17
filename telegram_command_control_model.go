//go:build linux

package main

import (
	"fmt"
	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/session"
	"time"
)

func (c telegramCommandControl) ModelSlotStatuses() ([]core.ModelSlotStatus, error) {
	if c.rt == nil {
		return nil, fmt.Errorf("runtime is not configured")
	}
	return c.rt.ModelSlotStatuses()
}

func (c telegramCommandControl) ValidateModelSlotConfig(cfg core.ModelSlotConfig) core.ModelValidation {
	if c.rt == nil {
		validation := core.ValidateModelSlotConfig(cfg, core.ModelSlotUsesTools(cfg.Slot))
		validation.Valid = false
		validation.Error = "runtime is not configured"
		return validation
	}
	return c.rt.ValidateModelSlotConfig(cfg)
}

func (c telegramCommandControl) SetModelSlotConfig(cfg core.ModelSlotConfig, actor string, reason string, ttl time.Duration) (core.ModelSlotStatus, error) {
	if c.rt == nil {
		return core.ModelSlotStatus{}, fmt.Errorf("runtime is not configured")
	}
	return c.rt.SetModelSlotOverride(cfg, actor, reason, ttl)
}

func (c telegramCommandControl) RollbackModelSlot(slot string, actor string, reason string) (core.ModelSlotStatus, error) {
	if c.rt == nil {
		return core.ModelSlotStatus{}, fmt.Errorf("runtime is not configured")
	}
	return c.rt.RollbackModelSlot(slot, actor, reason)
}

func (c telegramCommandControl) ClearModelSlot(slot string, actor string, reason string) (core.ModelSlotStatus, error) {
	if c.rt == nil {
		return core.ModelSlotStatus{}, fmt.Errorf("runtime is not configured")
	}
	return c.rt.ClearModelSlot(slot, actor, reason)
}

func (c telegramCommandControl) ModelSlotHistory(slot string, limit int) ([]session.ModelSlotOverrideRecord, error) {
	if c.rt == nil {
		return nil, fmt.Errorf("runtime is not configured")
	}
	return c.rt.ModelSlotHistory(slot, limit)
}
