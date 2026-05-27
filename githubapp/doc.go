//go:build linux

// Package githubapp owns Aphelion's narrow GitHub App credential helper.
//
// It parses configured RSA private keys, signs app JWTs, mints installation
// tokens through GitHub's App API, validates requested repository/permission
// scope, and redacts token-shaped output. The package must not decide whether a
// repository workflow is authorized, persist installation tokens, or broaden a
// caller's authority. Runtime and CLI callers remain responsible for config,
// operator intent, and any surrounding approval boundary.
package githubapp
