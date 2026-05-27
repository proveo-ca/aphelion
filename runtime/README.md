# runtime

`runtime` is Aphelion's long-lived house shell.

## Live Ownership

`runtime` owns:

- Telegram ingress and outbound adapter wiring
- principal resolution, scope resolution, and session locking
- pre-turn shell handoff into species assemblers (`interactive_dm`, `maintenance`, and durable-group adapters)
- background loops (heartbeat, cron, startup recovery, idle expiry)
- durable-agent lifecycle wiring
- assembly of concrete governor/face/persistence/delivery ports for `turn`

## Boundaries

`runtime` is not the main owner of one-turn stage order. Turn sequencing runs
through `turn.Machine`, while conversational mechanics are delegated to
`pipeline`.

Authoritative split for interactive DM turns:

- runtime shell (`turn.go`) owns principal admission, scope resolution, chat
  action loops, session locks, and transport event awareness.
- interactive DM species assembly (`interactive_dm_turn.go`) owns one-turn
  construction: shared interactive-like assembly, coordinator/port wiring, and
  `turn.Machine` invocation.
- `turn` owns stage order once invoked.

Authoritative split for maintenance turns (heartbeat/cron/recovery):

- runtime maintenance loops (`heartbeat.go`, `cron.go`, `recovery.go`) own
  maintenance-specific request synthesis, hidden-input gathering, and post-turn
  fanout behavior.
- maintenance species assembly (`maintenance_turn_assembly.go`) owns one-turn
  construction for the maintenance family: coordinator/port wiring and
  `turn.Machine` invocation.
- `turn` owns stage order once invoked.

## Package Map

- `runtime.go`: runtime construction, loops, and process wiring
- `interactive_like_assembly.go`: shared interactive-like turn assembly spine used by DM and durable-group turns
- `interactive_dm_turn.go`: interactive DM species assembler boundary and one-turn construction
- `maintenance_turn_assembly.go`: maintenance execution-family assembly boundary for heartbeat/cron/recovery turns
- `turn.go`, `turn_finalize.go`, `turn_monitor_events.go`, `interactive_dm_turn.go`, `maintenance_turn.go`, `durable_wake_turn.go`: adapters from runtime facts into `turn`
- `turn_coordinator_common.go`, `turn_coordinator_interactive.go`, `turn_coordinator_durable.go`: shared and species-specific coordinator adapters
- `durable_wake.go`: pluggable durable wake ingress adapters and shared wake-turn substrate
- `external_channel_runtime.go`: shared external-channel lifecycle helpers for poll due checks, command attempt/success/failure state, backoff, and adapter state containment
- `codex_app_server_channel.go`: top-level wiring for the Codex app-server external-channel adapter; helper/client/artifact/work-event mechanics live in `runtime/codex/`
- `durable_group.go`, `durable_group_context.go`, `durable_group_review.go`, `durable_wake.go`, `durable_wake_loop.go`, `durable_wake_scheduled_review.go`: durable-agent channel runtimes and channel adapters
- `interactive_dm_turn_runtime_test.go`, `durable_group_runtime_test.go`, `durable_wake_runtime_test.go`, `startup_recovery_runtime_test.go`, `status_runtime_test.go`, `doctor_runtime_test.go`: runtime-domain integration suites (by concern)

## Leaf Packages

- `runtime/codex`: bounded Codex app-server helper package. It owns the app-server client, status prompt helpers, artifact manifest helpers, work-event projection helpers, and command-effect taxonomy used by the Codex work lane. Top-level `runtime` still owns durable-agent wake wiring, executor selection, lifecycle loops, and authority state integration.
