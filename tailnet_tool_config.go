//go:build linux

package main

import (
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/config"
	"github.com/idolum-ai/aphelion/tailnet"
)

func remoteHostSSHTimeoutFromConfig(cfg *config.Config) time.Duration {
	if cfg == nil {
		return tailnet.DefaultCommandTimeout
	}
	raw := strings.TrimSpace(cfg.Tailscale.CommandTimeout)
	if raw == "" {
		return tailnet.DefaultCommandTimeout
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		return tailnet.DefaultCommandTimeout
	}
	return timeout
}
