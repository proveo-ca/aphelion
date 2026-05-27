# Contributing

Aphelion is a Linux-first, Telegram-controlled, governed outpost for personal
agents. Contributions should preserve that shape: local operation, explicit
authority, typed evidence, and short repair paths.

The practical contributor path is
[docs/guides/contributor-handbook.md](docs/guides/contributor-handbook.md).

## Local Setup

On Linux, run the normal local loop:

```bash
go test ./...
make architecture
make build
```

On macOS or another non-Linux host, `make test` and `make architecture`
intentionally stop with a Linux-only message. Use `make verify-linux-compile`
for a local compile-only check, then get the full Linux loop run before merge.

Use `config.example.toml` as a reference. Do not commit live config, tokens,
session databases, logs, transcripts, or local artifacts.

## Design Rules

- Keep Aphelion an outpost, not a platform.
- Telegram is the primary radio link; avoid channel abstraction unless a concrete
  governed workflow needs it.
- Text is presentation. Authority, consent, leases, grants, and evidence should
  be typed records or compiled contracts.
- Prefer durable concepts and stable repeated behavior over one-off structures.
- Update architecture docs when changing `runtime`, `turn`, `pipeline`,
  `session`, or `durableagent` ownership boundaries.

## Pull Requests

Before opening a PR from Linux, run:

```bash
make public-readiness
make architecture
go test ./...
make build
git diff --check
```

PRs must have Linux verification before merge. If you developed from a
non-Linux host, include the `make verify-linux-compile` result and say who will
run or has run the Linux loop.

If `gitleaks` is available, also run:

```bash
make secrets
```

For security-sensitive changes, describe the authority boundary being changed and
the tests that prove it remains bounded.
