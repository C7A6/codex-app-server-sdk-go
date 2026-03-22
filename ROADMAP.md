# ROADMAP

* The version of the SDK follows the version of the Codex CLI for consistency.

## v0.116.0

- [x] Implementation Using `github.com/sourcegraph/jsonrpc2`
- [x] ONLY Support Stdio Protocol
- [x] Implement a stdio-based function that starts and initializes `codex app-server`.
- [x] Handle process-level failures by returning an error or restarting automatically when the `codex` binary is missing or the app-server exits unexpectedly.
- [x] Implement `initialize`, `account/read`, and `account/rateLimits/read` support with unit tests that verify successful responses.

### Recommended Implementation Priority

1. `RegisterNotificationHandler` + `DefineEventTypes`: Why app-server is fundamentally notification-driven, and thread, turn, auth, review, and MCP flows all depend on typed event intake. How implement a method-based notification dispatcher first, then map each server notification schema in `api/codex-app-server/` to typed Go payloads.
2. `DefineCoreTypes`: Why request and response payloads must stabilize before higher-level APIs multiply. How use the JSON Schema files under `api/codex-app-server/v1/` and `api/codex-app-server/v2/` to define reusable Go structs for models, threads, turns, items, commands, and config data.
3. `ListModels`: Why model discovery is low-risk, high-signal, and a good proving ground for schema-driven pagination and nested payload decoding. How implement it directly from `ModelListParams.json` and `ModelListResponse.json`, then use the same patterns for later list APIs.
4. `StartThread` + `StartTurn`: Why these two methods create the minimum useful conversational workflow for SDK consumers. How implement typed request and response wrappers from the thread and turn start schemas and rely on the notification handler for streamed progress.
5. `HandleTurnEvents` + `HandleItemEvents` + `HandleItemDeltas`: Why turn execution is incomplete without streamed server updates for progress, output, and final state. How wire typed notification handlers for `turn/*`, `item/*`, and delta notifications after the dispatcher exists.
6. `ResumeThread` + `ReadThread` + `ListThreads`: Why practical clients need to reopen, inspect, and page through thread history after basic conversation start works. How reuse the shared thread types and pagination conventions established by earlier work.
7. `ExecCommand` family: Why command execution is a major app-server capability but depends less on conversational state once core event plumbing is in place. How implement command request methods and stream output through the same item and command delta handling path.
8. Auth login and logout methods: Why read-only auth status is already covered, but login flows need both request wrappers and asynchronous notification handling to feel complete. How add login, cancel, logout, and token refresh support after account notifications are typed.
9. Filesystem and config APIs: Why they are valuable but less foundational than conversation, event, and auth primitives. How implement them as typed request-response wrappers once the shared type system is established.
10. Plugins, skills, apps, MCP, feedback, and Windows-specific APIs: Why these are important extensions but should follow after the core transport, conversation, and state-management layers are reliable. How implement them incrementally on top of the same schema-driven request and notification foundation.

### Core Transport And Session

- [ ] `NewClient`: Why expose a stable SDK entry point for stdio app-server sessions. How build and validate client configuration, then prepare the JSON-RPC connection wrapper.
- [ ] `StartStdio`: Why start a local `codex app-server` process over stdio. How spawn `codex app-server`, wire stdin/stdout to `jsonrpc2`, and complete the initialization handshake.
- [ ] `Close`: Why callers need deterministic process and connection shutdown. How close the JSON-RPC connection, close pipes, terminate the process, and wait for exit once.
- [ ] `Call`: Why every request method should share one lifecycle-safe call path. How centralize session lookup, JSON-RPC request dispatch, and process-exit error translation.
- [ ] `Notify`: Why notification sending should follow the same lifecycle rules as requests. How reuse the active session and forward notification payloads without response handling.
- [ ] `OnProcessExit`: Why callers may need to observe unexpected process death. How expose a callback or channel that publishes process-level exit state.
- [ ] `RestartPolicy`: Why integrations need configurable recovery after process crashes. How add options for no-restart, always-restart, or bounded retry behavior around session recreation.
- [ ] `ProcessExitError`: Why protocol errors and process failures must be distinguished clearly. How wrap process exit status and expose `Unwrap` for downstream error inspection.

### Initialization And Capabilities

- [ ] `Initialize`: Why some callers may want an explicit handshake API in addition to startup helpers. How send `initialize` with `clientInfo` and optional capabilities, then decode runtime metadata.
- [ ] `Initialized`: Why the protocol requires an acknowledgement after `initialize`. How send the `initialized` notification once per connection after a successful handshake.
- [ ] `SetExperimentalAPI`: Why some endpoints and fields are gated by `experimentalApi`. How add an option builder that toggles `initialize.params.capabilities.experimentalApi`.
- [ ] `SetNotificationOptOut`: Why clients may want to suppress noisy notifications per connection. How add an option builder that fills `optOutNotificationMethods` exactly as documented.

### Model Discovery

- [x] `ListModels`: Why clients need model metadata before rendering selectors or picking defaults. How call `model/list`, support pagination flags, and decode reasoning and modality metadata.
- [ ] `ListExperimentalFeatures`: Why integrations may surface or gate preview functionality. How call `experimentalFeature/list` and map lifecycle metadata into typed results.
- [ ] `ListCollaborationModes`: Why callers may need to enumerate supported collaboration presets. How call `collaborationMode/list` and decode the non-paginated response.

### Thread Lifecycle

- [x] `StartThread`: Why a new conversation must begin with an explicit thread object. How call `thread/start`, accept model and thread settings, and decode the created thread payload.
- [x] `ResumeThread`: Why clients need to continue an existing conversation. How call `thread/resume` with a thread ID and return the resumed thread state.
- [x] `ForkThread`: Why callers may branch an existing conversation into a new history line. How call `thread/fork` and decode the new thread descriptor.
- [x] `ReadThread`: Why stored threads must be readable without loading them into memory. How call `thread/read` and support the `includeTurns` flag.
- [x] `ListThreads`: Why clients need pagination and filtering for thread history UIs. How call `thread/list` with cursor, limit, sort, provider, source, archive, and cwd filters.
- [x] `ListLoadedThreads`: Why callers may need to inspect in-memory thread state. How call `thread/loaded/list` and decode loaded thread IDs.
- [ ] `SetThreadName`: Why a thread needs a user-facing display name. How call `thread/name/set` with thread identity and the desired label.
- [ ] `ArchiveThread`: Why inactive threads should be removable from the active list without deletion. How call `thread/archive` and treat `{}` as success.
- [ ] `UnarchiveThread`: Why archived threads must be restorable into active storage. How call `thread/unarchive` and decode the restored thread object.
- [ ] `UnsubscribeThread`: Why clients may stop receiving turn and item events for a loaded thread. How call `thread/unsubscribe` and decode the unsubscribe result state.
- [ ] `CompactThread`: Why long conversations may need context compaction. How call `thread/compact/start` and rely on streamed turn and item notifications for progress.
- [ ] `RollbackThread`: Why a client may need to drop recent turns from working context. How call `thread/rollback` with a rollback count and decode the updated thread.

### Turn Lifecycle

- [ ] `StartTurn`: Why a thread needs a primary execution entry point for user input. How call `turn/start` with input items and optional overrides for model, cwd, sandbox, and output schema.
- [ ] `SteerTurn`: Why an in-flight turn may need incremental user steering. How call `turn/steer` with additional input and decode the accepted `turnId`.
- [ ] `InterruptTurn`: Why a caller must be able to cancel active generation. How call `turn/interrupt` and treat an empty success object as completion.
- [ ] `StartReview`: Why the SDK should expose the documented review workflow. How call `review/start` and decode the initial review response while consuming review-mode events.

### Command Execution

- [ ] `ExecCommand`: Why clients may need sandboxed command execution outside thread flows. How call `command/exec` with command, cwd, and sandbox policy and return the execution descriptor.
- [ ] `WriteCommandStdin`: Why long-running commands may require streamed stdin input. How call `command/exec/write` with bytes or close signals for an existing execution session.
- [ ] `ResizeCommandPTY`: Why PTY-backed commands need terminal resize support. How call `command/exec/resize` with rows and columns for the target execution session.
- [ ] `TerminateCommand`: Why callers need an explicit way to stop a running command. How call `command/exec/terminate` and treat `{}` as success.

### Skills, Plugins, Apps, And MCP

- [ ] `ListSkills`: Why UIs may need to show skills available for one or more working directories. How call `skills/list` with cwd lists, reload flags, and extra user roots.
- [ ] `WriteSkillsConfig`: Why users need SDK support for enabling or disabling skills. How call `skills/config/write` with path-based enablement changes.
- [ ] `ListPlugins`: Why clients may need marketplace and auth policy metadata for plugins. How call `plugin/list` and decode marketplace and plugin state data.
- [ ] `ReadPlugin`: Why callers may need detailed metadata for one plugin. How call `plugin/read` with marketplace path and plugin name and decode bundled assets.
- [ ] `ListApps`: Why connector UIs need a typed catalog of available apps. How call `app/list` with pagination and decode accessibility and enabled fields.
- [ ] `StartMCPOAuthLogin`: Why MCP servers may require an OAuth bootstrap flow. How call `mcpServer/oauth/login` and return the authorization URL and login context.
- [ ] `ReloadMCPServerConfig`: Why runtime MCP configuration may change on disk. How call `config/mcpServer/reload` and surface a simple success result.
- [ ] `ListMCPServerStatus`: Why operators need visibility into MCP auth state and exposed tools. How call `mcpServerStatus/list` with cursor and limit support.

### Configuration And Filesystem

- [ ] `ReadConfig`: Why callers may need the resolved effective app-server configuration. How call `config/read` and decode layered configuration values.
- [ ] `WriteConfigValue`: Why single configuration keys should be writable through the SDK. How call `config/value/write` with a typed key and value payload.
- [ ] `BatchWriteConfig`: Why multiple config edits should be applied atomically. How call `config/batchWrite` and decode the committed configuration response.
- [ ] `ReadConfigRequirements`: Why enterprise clients may need requirement and policy metadata. How call `configRequirements/read` and decode allow-lists and residency constraints.
- [ ] `DetectExternalAgentConfig`: Why migration tooling needs to discover importable external-agent artifacts. How call `externalAgentConfig/detect` with home and cwd controls.
- [ ] `ImportExternalAgentConfig`: Why detected external-agent settings should be importable through the SDK. How call `externalAgentConfig/import` with explicit migration items.
- [ ] `ReadFile`: Why the app-server exposes filesystem access over JSON-RPC v2 APIs. How call `fs/readFile` with an absolute path and decode file contents.
- [ ] `WriteFile`: Why clients need remote file writes through the app-server filesystem surface. How call `fs/writeFile` with absolute path and content payloads.
- [ ] `CreateDirectory`: Why callers may need to prepare filesystem paths before writing files. How call `fs/createDirectory` with an absolute path and creation options.
- [ ] `GetMetadata`: Why clients may need file type and stat metadata before operating on paths. How call `fs/getMetadata` and decode the metadata payload.
- [ ] `ReadDirectory`: Why directory listings are needed for file browser integrations. How call `fs/readDirectory` with an absolute path and decode child entries.
- [ ] `RemovePath`: Why SDK consumers need a deletion API for filesystem automation. How call `fs/remove` with an absolute path and removal options.
- [ ] `CopyPath`: Why file duplication should be available through the same transport. How call `fs/copy` with absolute source and destination paths.

### Auth And Account

- [ ] `ReadAccount`: Why integrations need to inspect the current authentication state. How call `account/read` with `refreshToken` support and decode account and provider metadata.
- [ ] `StartAPIKeyLogin`: Why clients should support direct API-key authentication. How call `account/login/start` with `type=apiKey` and verify the completion notifications.
- [ ] `StartChatGPTLogin`: Why browser-based ChatGPT auth is a documented first-class flow. How call `account/login/start` with `type=chatgpt` and return `loginId` and `authUrl`.
- [ ] `StartChatGPTTokenLogin`: Why host-managed ChatGPT tokens need a dedicated login path. How call `account/login/start` with `type=chatgptAuthTokens`, `idToken`, and `accessToken`.
- [ ] `CancelLogin`: Why a pending ChatGPT login must be cancellable. How call `account/login/cancel` with `loginId` and rely on completion notifications for the final result.
- [ ] `Logout`: Why callers need an explicit sign-out API. How call `account/logout` and observe `account/updated` for the post-logout auth mode.
- [ ] `ReadRateLimits`: Why ChatGPT-backed clients need current quota information. How call `account/rateLimits/read` and decode single-bucket and multi-bucket rate-limit payloads.
- [ ] `HandleChatGPTTokenRefresh`: Why external-token mode may require the host app to refresh tokens on demand. How register a server-request handler for `account/chatgptAuthTokens/refresh` and send refreshed tokens back.

### Feedback, Windows, And Approvals

- [ ] `UploadFeedback`: Why clients may need to submit bug reports and conversation feedback. How call `feedback/upload` with classification, reason, logs, and extra attachments.
- [ ] `StartWindowsSandboxSetup`: Why Windows clients may need elevated or unelevated setup flows. How call `windowsSandbox/setupStart` and follow completion events asynchronously.
- [ ] `RequestUserInput`: Why experimental tool flows may ask the host for guided user input. How call `tool/requestUserInput` with one to three questions and decode the selected answers.

### Notifications And Streaming Events

- [x] `RegisterNotificationHandler`: Why the protocol is bidirectional and event-heavy. How expose a dispatcher that routes server notifications by method name to typed handlers.
- [ ] `HandleThreadEvents`: Why thread creation, archive, unarchive, close, and status changes are server-driven. How decode `thread/started`, `thread/archived`, `thread/unarchived`, `thread/closed`, and `thread/status/changed`.
- [ ] `HandleTurnEvents`: Why turn progress must be streamed to clients in real time. How decode `turn/started`, `turn/completed`, `turn/diff/updated`, `turn/plan/updated`, and `thread/tokenUsage/updated`.
- [ ] `HandleItemEvents`: Why all granular work units arrive through item notifications. How decode `item/started` and `item/completed` into typed item unions.
- [ ] `HandleItemDeltas`: Why large text and command outputs are streamed incrementally. How decode item delta notifications such as agent text, reasoning deltas, command output, and file change output.
- [ ] `HandleAccountNotifications`: Why auth mode and rate-limit changes may happen independently of direct requests. How decode `account/login/completed`, `account/updated`, and `account/rateLimits/updated`.
- [ ] `HandleMCPOAuthCompletion`: Why OAuth login flows finish asynchronously. How decode `mcpServer/oauthLogin/completed` and correlate the result with the initiating request.
- [ ] `HandleWindowsSandboxCompletion`: Why Windows sandbox setup finishes via notification rather than inline response. How decode `windowsSandbox/setupCompleted` into a typed event.
- [ ] `HandleFuzzyFileSearchEvents`: Why experimental file-search sessions publish updates asynchronously. How decode `fuzzyFileSearch/sessionUpdated` and `fuzzyFileSearch/sessionCompleted`.

### Shared Type System And Test Coverage

- [x] `DefineCoreTypes`: Why request and response payloads should be strongly typed across the SDK. How add shared Go structs for threads, turns, items, reviews, commands, and config payloads from the document.
- [x] `DefineEventTypes`: Why notification payloads should not be handled as raw maps. How add typed event models keyed by method name for threads, turns, items, auth, and MCP flows.
- [ ] `AddIntegrationTests`: Why transport, auth, and session behavior should be verified against the real `codex` binary. How keep real-binary integration tests for initialization, auth reads, process failure, and restart scenarios.
- [ ] `AddGoldenProtocolTests`: Why stable payload encoding and decoding matters for SDK compatibility. How capture representative JSON-RPC request and response fixtures from the document and assert struct compatibility.
