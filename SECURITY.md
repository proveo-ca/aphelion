# Security Policy

Aphelion is a governed personal-agent runtime. It can execute local tools,
manage local files, interact with Telegram, and connect to model providers. Treat
it as security-sensitive infrastructure.

## Supported Versions

Security fixes are handled on the default branch until the project publishes a
stable release policy. Public releases before a stable version should be treated
as alpha software.

## Reporting a Vulnerability

Please do not report vulnerabilities in public issues.

Use GitHub private vulnerability reporting or a private GitHub Security Advisory
for the repository. Include:

- affected commit or release
- configuration surface involved
- whether local tool execution, Telegram control, provider credentials, durable
  children, or stored transcripts are affected
- a minimal reproduction when possible

## Operational Guidance

- Never commit live `aphelion.toml`, provider API keys, Telegram bot tokens,
  session databases, transcript exports, logs, or forensic artifacts.
- Keep runtime config and token files mode `0600`.
- Review sandbox profiles before enabling tool execution for non-admin
  principals or durable children.
- Use explicit leases/grants for deploys, account access, public contact, and
  other high-risk actions.

## Public Release History

Do not publish private pre-public Git history as the canonical public release.
Public source cuts should start from a reviewed clean tree and should not carry
private operational paths, account identifiers, transcript fragments, local
state, or credential-location details from historical development.

The historical archive, when present, is separate from the canonical public
source. Treat anything found in private history as potentially exposed and review
or rotate affected operational material before publication.
