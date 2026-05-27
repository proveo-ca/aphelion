//go:build linux

package codex

const (
	AdapterName       = "codex_app_server"
	WakeChannel       = "codex_app_server"
	MaxMessageBytes   = int64(1 << 20)
	StatusCommandName = "codex_app_server.status_heartbeat"
)
