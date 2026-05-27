//go:build linux

// Package codex owns the Codex app-server leaf mechanics used by runtime.
//
// It contains the WebSocket/JSON-RPC client, durable-child status prompts,
// heartbeat artifact helpers, work-event projection, and command-effect
// taxonomy for the Codex work lane. The package must stay a leaf under the
// runtime shell: it should not own executor selection, durable-agent state
// writes, continuation leases, approval grants, service lifecycle, or transport
// ingress. Top-level runtime keeps those authority and lifecycle boundaries.
package codex
