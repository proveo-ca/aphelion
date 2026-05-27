# Architecture Waivers

This file is the single ledger for temporary architecture waivers.

Rules:
- Every waiver must include `owner` and `expires_on`.
- Expired waivers must be removed or renewed in an explicit follow-up commit.
- Waivers are temporary seams, not permanent abstractions.

## Active Waivers

### WAIVER-2026-05-root-shims

- **owner:** maintainers
- **expires_on:** 2026-08-25
- **Status:** active
- **Surface:** root `*_shims.go` files:
  `commands_shims.go`, `maintenance_durable_agent_shims.go`,
  `maintenance_memory_shims.go`, `maintenance_repair_shims.go`,
  `standalone_cli_shims.go`, `telegram_decision_shims.go`, and
  `telegram_runtime_shims.go`.
- **Why it exists:** these files are compatibility seams left after moving CLI,
  Telegram, and maintenance behavior into narrower internal packages while the
  root package remains the single-binary composition layer.
- **Boundary:** shims may forward to owned packages, alias public helper types,
  and preserve test/caller compatibility. They must not grow new behavior,
  authority checks, persistence contracts, or transport logic in root.
- **Exit gate:** remove the shims after callers/tests move to package-owned
  entrypoints, or replace this waiver with an explicit root-facade ownership
  note if the facade is still intentional at expiry.
