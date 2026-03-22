# ROADMAP

* The version of the SDK follows the version of the Codex CLI for consistency.

## v0.116.0

- [ ] Implementation Using `github.com/sourcegraph/jsonrpc2`
- [ ] ONLY Support Stdio Protocol
- [x] Implement a stdio-based function that starts and initializes `codex app-server`.
- [ ] Handle process-level failures by returning an error or restarting automatically when the `codex` binary is missing or the app-server exits unexpectedly.
- [ ] Implement `initialize`, `account/read`, and `account/rateLimits/read` support with unit tests that verify successful responses.
