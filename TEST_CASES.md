# TEST CASES

This document lists additional test cases that are still worth implementing after the current coverage in [`pkg/appserver/client_test.go`](/home/nixos/go/src/github.com/C7A6/codex-app-server-sdk-go/pkg/appserver/client_test.go).

It intentionally excludes scenarios that are already covered by the existing suite, such as:

- basic stdio startup and initialization success
- missing binary failure
- explicit `initialize` and `initialized` helpers
- basic account reads and rate-limit reads
- basic model, thread, turn, review, command, skills, plugin, app, MCP, config, and filesystem happy paths
- process-exit restart and non-restart behavior
- notification registration and typed notification decoding

## Integration Tests

### Initialization Handshake Enforcement

#### Reject requests before initialization

- Why: The spec says every connection must send `initialize` and then `initialized` before any other request. The client currently proves the happy path, but it does not prove server-side enforcement.
- Scenario: Open a raw stdio JSON-RPC connection to `codex app-server`, send `thread/start` before `initialize`, and capture the response.
- Expected: The server returns a JSON-RPC error indicating `Not initialized`, and no thread is created.

#### Reject duplicate initialize on one connection

- Why: The spec explicitly says repeated `initialize` calls on the same connection must fail with `Already initialized`.
- Scenario: Open a raw stdio connection, send a valid `initialize`, send `initialized`, then send a second `initialize` request on the same connection.
- Expected: The second `initialize` call returns a JSON-RPC error indicating `Already initialized`.

### Capabilities And Gating

#### Experimental method is rejected without `experimentalApi`

- Why: The document says experimental methods and fields must be rejected unless the connection opted into `capabilities.experimentalApi`.
- Scenario: Start a client without experimental capability and call `experimentalFeature/list` or `collaborationMode/list`.
- Expected: The server returns an error whose message states that the method requires `experimentalApi`.

#### Experimental method succeeds with `experimentalApi`

- Why: The SDK already checks option plumbing, but it does not yet assert the gating boundary directly on the same endpoint pair.
- Scenario: Run the same experimental endpoint once without capability and once with `SetExperimentalAPI(true)`.
- Expected: The non-experimental connection fails with the capability error, and the experimental connection succeeds.

#### Experimental field on `thread/start` is rejected without opt-in

- Why: The spec documents `dynamicTools` as an experimental field on an otherwise stable method, which is a distinct compatibility rule from an experimental method.
- Scenario: Send `thread/start` with `dynamicTools` on a connection that did not enable `experimentalApi`.
- Expected: The server rejects the request with an error indicating the field requires `experimentalApi`.

### Notification Opt-Out

#### Exact-match notification opt-out suppresses only the named method

- Why: The spec says `optOutNotificationMethods` uses exact matching, not prefix matching. This is a wire-level contract and is not covered today.
- Scenario: Initialize a connection with `optOutNotificationMethods: ["thread/started"]`, register handlers for both `thread/started` and `turn/started`, then start a thread and a turn.
- Expected: `thread/started` is not observed on that connection, while `turn/started` still arrives normally.

#### Unknown opt-out method names are ignored

- Why: The spec says unknown names are accepted and ignored; a regression here would silently suppress unrelated traffic or reject initialization unnecessarily.
- Scenario: Initialize with `optOutNotificationMethods: ["does/not/exist"]`, then start a thread.
- Expected: Initialization succeeds, and normal notifications such as `thread/started` still arrive.

### Thread Lifecycle Notifications

#### Unsubscribe emits `thread/status/changed` and `thread/closed` when last subscriber leaves

- Why: The current suite checks the return status from `thread/unsubscribe`, but not the documented side effects when the last subscriber is removed.
- Scenario: Start a thread on a single connection, register handlers for `thread/status/changed` and `thread/closed`, then call `thread/unsubscribe`.
- Expected: The response status is `unsubscribed`, the server emits `thread/status/changed` with `type: "notLoaded"`, and then emits `thread/closed` for the same `threadId`.

#### Archive and unarchive emit lifecycle notifications

- Why: The current tests verify state through follow-up reads and lists, but they do not assert the documented `thread/archived` and `thread/unarchived` notifications.
- Scenario: Register handlers for `thread/archived` and `thread/unarchived`, archive a persisted thread, then unarchive it.
- Expected: Each operation emits the matching notification with the target `threadId`.

#### Compaction emits `contextCompaction` item lifecycle

- Why: The spec deprecates legacy `thread/compacted` in favor of `item/started` and `item/completed` with `type: "contextCompaction"`. That replacement behavior should be validated.
- Scenario: Start a persisted thread, register item handlers, call `thread/compact/start`, and wait for the resulting item lifecycle.
- Expected: The server emits `item/started` and `item/completed` for a `contextCompaction` item on the target thread.

### Turn Streaming

#### Turn completion after interrupt ends with `status: "interrupted"`

- Why: The current interrupt test only accepts request success or active-turn race behavior. It does not verify the documented terminal turn status when interruption actually takes effect.
- Scenario: Start a turn that runs long enough to interrupt, register for `turn/completed`, then call `turn/interrupt`.
- Expected: The final `turn/completed` notification reports `turn.status == "interrupted"`.

#### `turn/plan/updated` streams plan state independently of final items

- Why: The spec explicitly says `turn/plan/updated` may include empty `items` arrays and that item notifications remain authoritative. This subtle contract is not covered.
- Scenario: Trigger a turn in plan mode or another flow that emits plan updates, and capture `turn/plan/updated`.
- Expected: The notification decodes successfully even when `items` is empty, and the plan entries use only the documented statuses: `pending`, `inProgress`, `completed`.

#### `turn/diff/updated` emits aggregated unified diff

- Why: The diff update notification is documented as the canonical aggregated file-change diff for a turn, and it is not asserted by the current suite.
- Scenario: Run a turn that makes a file edit, register for `turn/diff/updated`, and wait for diff notifications.
- Expected: At least one `turn/diff/updated` notification arrives with the matching `threadId`, `turnId`, and a non-empty unified diff payload.

### Command Streaming And Validation

#### `command/exec` rejects an empty command array

- Why: The spec explicitly documents this validation rule, but the current tests only cover successful command execution.
- Scenario: Call `command/exec` with `command: []`.
- Expected: The server returns a validation error and does not start a process.

#### `command/exec` with `streamStdoutStderr` emits output delta notifications

- Why: The SDK currently validates buffered command execution and stdin/resize/terminate flows, but not streamed `command/exec/outputDelta` semantics.
- Scenario: Start a command with `streamStdoutStderr: true` and a known `processId`, register for output delta notifications, and run a command that writes to stdout in multiple chunks.
- Expected: One or more output delta notifications arrive in order, and the final command result is still returned.

### Review And Detached Execution

#### Detached review creates a distinct review thread

- Why: The current review test covers the basic response shape, but not the spec’s detached review behavior.
- Scenario: Start a normal thread, call `review/start` with `delivery: "detached"`, and register for `thread/started`.
- Expected: The response contains a `reviewThreadId` different from the original thread id, and a `thread/started` notification is emitted for the new review thread before review events stream.

#### Review mode item lifecycle is emitted

- Why: The spec documents `enteredReviewMode` and `exitedReviewMode` item notifications as the UI contract for reviewer state.
- Scenario: Start a review and collect `item/started` and `item/completed` events.
- Expected: The stream includes an `enteredReviewMode` item when review begins and an `exitedReviewMode` item when it finishes.

### App And MCP Update Notifications

#### `app/list/updated` is emitted during app refresh

- Why: The current test validates `app/list` responses, but not the asynchronous refresh notification that the spec documents for accessible and directory app sources.
- Scenario: Register a handler for `app/list/updated`, call `app/list` with `forceRefetch: true`, and wait for refresh notifications.
- Expected: At least one `app/list/updated` notification arrives with a decodable app list payload.

#### `mcpServer/oauthLogin/completed` notification is emitted when OAuth flow finishes

- Why: The current MCP OAuth test only validates the initial login bootstrap result or skip conditions. The completion notification is the client-facing end of that workflow.
- Scenario: In an environment with an OAuth-capable MCP server, register for `mcpServer/oauthLogin/completed`, start the login flow, finish the authorization externally, and wait for notification delivery.
- Expected: The notification arrives with the configured MCP server name and either `success: true` or a typed error payload.

## Golden Protocol Tests

### Initialization Fixtures

#### Golden fixture: initialize request with capabilities

- Why: The initialization payload is the first compatibility boundary for every client. A stable fixture guards field names and nested capability encoding.
- Scenario: Encode `InitializeParams` with `clientInfo`, `experimentalApi: true`, and two `optOutNotificationMethods`.
- Expected: The serialized JSON exactly matches the documented wire shape and field casing.

#### Golden fixture: initialize success result

- Why: The SDK should keep decoding the documented runtime metadata fields even if the implementation is refactored.
- Scenario: Decode the documented `initialize` success payload containing `userAgent`, `platformFamily`, and `platformOs`.
- Expected: All fields populate the Go struct exactly.

#### Golden fixture: not-initialized error response

- Why: The document defines a protocol-level handshake failure that callers need to distinguish from transport failures.
- Scenario: Store a representative JSON-RPC error object returned for a request made before initialization.
- Expected: The fixture decodes into the SDK’s error path without losing the server message.

#### Golden fixture: already-initialized error response

- Why: Duplicate handshake handling is another explicit protocol contract, and it should remain stable across refactors.
- Scenario: Store a representative error object for a second `initialize` request on the same connection.
- Expected: The fixture decodes with the correct JSON-RPC error code and message.

### Thread And Turn Fixtures

#### Golden fixture: `thread/start` response plus `thread/started` notification

- Why: The same logical operation produces both a response and a notification, and the SDK should preserve compatibility for both payload shapes.
- Scenario: Capture the documented `thread/start` success result and follow-up `thread/started` notification.
- Expected: Both fixtures decode into the expected typed structs, with `name` absent or null tolerated on first creation.

#### Golden fixture: `thread/read` with `includeTurns: true`

- Why: The spec shows `thread/read` returning runtime `status` and full `turns`, which differs from the lighter summary shape used elsewhere.
- Scenario: Decode the documented `thread/read` response with `status.type = "notLoaded"` and populated `turns`.
- Expected: The thread, status union, and turn slice decode exactly.

#### Golden fixture: `thread/unsubscribe` side-effect notifications

- Why: `thread/status/changed` and `thread/closed` are lifecycle-critical notifications, and the current decode coverage does not pin their exact wire shape.
- Scenario: Add fixtures for the unsubscribe response and both follow-up notifications.
- Expected: The response and notifications decode into the expected status and thread-id fields.

#### Golden fixture: `turn/completed` failed payload with `codexErrorInfo`

- Why: Failed turns carry nested error metadata, including optional HTTP status, which is easy to regress silently.
- Scenario: Decode a representative `turn/completed` notification with `status: "failed"` and a populated `codexErrorInfo`.
- Expected: The nested error object, error kind, and optional HTTP status all survive decoding.

#### Golden fixture: `turn/plan/updated` with empty `items`

- Why: The spec explicitly calls out that `items` may be empty even while plan updates stream. That quirk should be frozen in a fixture.
- Scenario: Decode a `turn/plan/updated` notification whose `items` is empty and whose `plan` contains multiple statuses.
- Expected: Decoding succeeds, and the plan statuses remain ordered and typed.

#### Golden fixture: `turn/diff/updated`

- Why: Aggregated unified diff payloads are a separate event shape from file-change items and deserve independent fixture coverage.
- Scenario: Decode a representative `turn/diff/updated` notification with `threadId`, `turnId`, and a unified diff string.
- Expected: All fields decode exactly, and the diff text is preserved verbatim.

### Item And Delta Fixtures

#### Golden fixture: reasoning summary boundaries

- Why: Reasoning streaming uses multiple notification shapes that have ordering semantics (`summaryPartAdded`, `summaryTextDelta`, `textDelta`). Those shapes should be pinned explicitly.
- Scenario: Add fixtures for `item/reasoning/summaryPartAdded`, `item/reasoning/summaryTextDelta`, and `item/reasoning/textDelta`.
- Expected: Each delta fixture decodes into the correct typed event with `itemId`, indices, and text preserved.

#### Golden fixture: command output delta

- Why: Command streaming is currently unpinned at the wire level even though the spec defines ordered stdout/stderr deltas.
- Scenario: Decode a representative `item/commandExecution/outputDelta` notification.
- Expected: The item id, stream kind, and delta content decode exactly.

#### Golden fixture: file-change output delta

- Why: The spec says `item/fileChange/outputDelta` carries the underlying `apply_patch` tool-call response, which is a distinct payload from the final `fileChange` item.
- Scenario: Decode a representative `item/fileChange/outputDelta` notification.
- Expected: The item id and tool-response content decode without lossy coercion.

#### Golden fixture: review-mode items

- Why: `enteredReviewMode` and `exitedReviewMode` are user-visible milestones in review flows and should stay wire-compatible.
- Scenario: Add fixtures for both item payloads as seen in `item/started` and `item/completed`.
- Expected: The item type, id, and `review` text decode exactly.

### Approval And Server-Request Fixtures

#### Golden fixture: command approval request with network context

- Why: Approval payloads are one of the most nested parts of the protocol, and `networkApprovalContext` changes how clients must render the prompt.
- Scenario: Decode a representative `item/commandExecution/requestApproval` server request containing `networkApprovalContext`, `availableDecisions`, and optional command metadata.
- Expected: All nested fields decode correctly, including host, protocol, optional port, and decision list.

#### Golden fixture: file-change approval request

- Why: File-change approvals are a second server-request family with a different payload shape and should be frozen separately.
- Scenario: Decode a representative `item/fileChange/requestApproval` server request with `reason` and `grantRoot`.
- Expected: The request fields decode exactly and preserve absolute-path values.

#### Golden fixture: `serverRequest/resolved`

- Why: The server uses the same notification both for successful client responses and for cleanup when pending prompts are cleared. The notification shape should be fixed.
- Scenario: Decode a representative `serverRequest/resolved` notification containing `threadId` and `requestId`.
- Expected: The request-resolution payload decodes exactly.

#### Golden fixture: `tool/requestUserInput`

- Why: The SDK already handles this request path, but a golden fixture is still useful because question arrays, options, and `isOther` are UI-facing protocol data.
- Scenario: Decode a representative `item/tool/requestUserInput` server request with multiple questions and options.
- Expected: All questions, options, and optional `isOther` flags decode exactly.

### Auth And Rate-Limit Fixtures

#### Golden fixture: `account/read` response variants

- Why: The spec documents multiple legitimate shapes for `account/read`: unauthenticated, API-key, and ChatGPT account results. The decoder should be pinned against all of them.
- Scenario: Add fixtures for `account: null`, `account.type: "apiKey"`, and `account.type: "chatgpt"` with `email` and `planType`.
- Expected: Each variant decodes without requiring fields that are intentionally absent in other modes.

#### Golden fixture: multi-bucket `account/rateLimits/read`

- Why: The current suite reads rate limits from a live server, but a golden fixture should pin the documented `rateLimitsByLimitId` shape and backward-compatible single-bucket view.
- Scenario: Decode the multi-bucket rate-limit example from the spec.
- Expected: `rateLimits` and `rateLimitsByLimitId` both decode exactly, including nullable `limitName` and `secondary`.

#### Golden fixture: `account/rateLimits/updated` notification

- Why: The update notification is documented separately from the read response and should be pinned independently.
- Scenario: Decode the example `account/rateLimits/updated` notification.
- Expected: The notification decodes into the typed rate-limit update event with the nested primary window intact.

### App, MCP, And Windows Fixtures

#### Golden fixture: `app/list/updated`

- Why: App refresh notifications are an async protocol surface not covered by the current decode fixtures.
- Scenario: Decode a representative `app/list/updated` notification with one app entry.
- Expected: The app list payload decodes exactly, including nullable branding and metadata fields.

#### Golden fixture: `mcpServer/oauthLogin/completed`

- Why: The OAuth completion notification closes the MCP login flow and should remain wire-compatible.
- Scenario: Decode both a success and failure notification fixture for `mcpServer/oauthLogin/completed`.
- Expected: The name, success flag, and optional error field decode exactly.

#### Golden fixture: `windowsSandbox/setupCompleted`

- Why: Windows sandbox setup is asynchronous and notification-driven; the completion payload should be pinned separately from the request result.
- Scenario: Decode a representative `windowsSandbox/setupCompleted` notification.
- Expected: `mode`, `success`, and nullable `error` decode exactly.
