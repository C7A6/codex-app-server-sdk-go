# codex-app-server-sdk-go

[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](#installation)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](#license)
[![CI](https://img.shields.io/badge/ci-placeholder-lightgrey.svg)](#contributing)

Go SDK for `codex app-server`.

This project provides a typed Go client for the Codex App Server JSON-RPC API over stdio. It covers connection lifecycle management, typed request and notification models, high-level conversation workflows, process restart handling, and integration-tested wrappers around the app-server surface.

## Features

- Stdio-based `codex app-server` startup and initialization
- Typed request/response APIs for threads, turns, commands, config, filesystem, skills, plugins, apps, MCP, and account reads
- Typed notification decoding for thread, turn, item, account, MCP, Windows sandbox, and fuzzy-search events
- High-level workflows such as `SendMessageAndWait`, `StreamTurn`, and `QuickThread`
- Generic typed event registration with `OnEvent[T]`
- Structured JSON-RPC errors with predicate helpers
- Pagination helpers for threads and models
- Real-`codex` integration tests and golden protocol fixtures

## Installation

```bash
go get github.com/C7A6/codex-app-server-sdk-go/pkg/appserver
```

Requirements:

- Go 1.26+
- `codex` CLI installed and available on `PATH` ([installation guide placeholder](https://developers.openai.com/codex))

## Quick Start

### Create a client and connect

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/C7A6/codex-app-server-sdk-go/pkg/appserver"
)

func main() {
	ctx := context.Background()

	client, initResult, err := appserver.NewClient(ctx, appserver.StartOptions{
		ClientInfo: appserver.ClientInfo{
			Name:    "example_client",
			Title:   "Example Client",
			Version: "0.1.0",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("connected:", initResult.UserAgent, initResult.PlatformOS)
}
```

### Start a thread and send a message

```go
// Following examples assume client is already created as shown above.
thread, err := client.StartThread(ctx, appserver.ThreadStartParams{})
if err != nil {
	log.Fatal(err)
}

turn, err := client.StartTurn(ctx, appserver.TurnStartParams{
	ThreadID: thread.Thread.ID,
	Input: []appserver.TurnStartInputItem{
		{"type": "text", "text": "Summarize this repository."},
	},
})
if err != nil {
	log.Fatal(err)
}

log.Printf("turn started: %s", turn.Turn.ID)
```

### Use `SendMessageAndWait`

```go
// Following examples assume client is already created as shown above.
thread, err := client.StartThread(ctx, appserver.ThreadStartParams{})
if err != nil {
	log.Fatal(err)
}

waitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
defer cancel()

completed, err := client.SendMessageAndWait(waitCtx, thread.Thread.ID, "Explain the purpose of this SDK.")
if err != nil {
	log.Fatal(err)
}

log.Printf("turn completed: %s (%s)", completed.Turn.ID, completed.Turn.Status)
```

### Use `OnEvent[T]` typed event handlers

```go
// Following examples assume client is already created as shown above.
unregister, err := appserver.OnEvent(client, func(ctx context.Context, event *appserver.ThreadStartedEvent) {
	log.Printf("thread started: %s", event.Thread.ID)
})
if err != nil {
	log.Fatal(err)
}
defer unregister()

if _, err := client.StartThread(ctx, appserver.ThreadStartParams{}); err != nil {
	log.Fatal(err)
}
```

### Use `StreamTurn` for streaming

```go
// Following examples assume client is already created as shown above.
thread, err := client.StartThread(ctx, appserver.ThreadStartParams{})
if err != nil {
	log.Fatal(err)
}

streamCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
defer cancel()

events, _, err := client.StreamTurn(streamCtx, appserver.TurnStartParams{
	ThreadID: thread.Thread.ID,
	Input: []appserver.TurnStartInputItem{
		{"type": "text", "text": "Stream your progress while solving this task."},
	},
})
if err != nil {
	log.Fatal(err)
}

for event := range events {
	switch e := event.(type) {
	case *appserver.TurnStartedEvent:
		log.Printf("turn started: %s", e.Turn.ID)
	case *appserver.ItemStartedEvent:
		log.Printf("item started: %s (%s)", e.Item.ID, e.Item.Type)
	case *appserver.AgentMessageDeltaEvent:
		log.Print(e.Delta)
	case *appserver.TurnCompletedEvent:
		log.Printf("turn completed: %s", e.Turn.Status)
	}
}
```

## API Reference

The SDK exposes low-level request methods plus higher-level workflow helpers.

### Connection and lifecycle

Start, initialize, manage, and shut down a `codex app-server` session.

- `NewClient`
- `StartStdio`
- `Close`
- `Call`
- `Notify`
- `OnProcessExit`

### Initialization and capabilities

Control the handshake and per-connection capability flags.

- `Initialize`
- `Initialized`
- `StartOptions.SetExperimentalAPI`
- `StartOptions.SetNotificationOptOut`

### Threads and turns

Create, resume, fork, read, and manage conversation threads and turns.

- `StartThread`, `ResumeThread`, `ForkThread`, `ReadThread`
- `ListThreads`, `ListLoadedThreads`, `ListAllThreads`
- `SetThreadName`, `ArchiveThread`, `UnarchiveThread`, `UnsubscribeThread`
- `CompactThread`, `RollbackThread`
- `StartTurn`, `SteerTurn`, `InterruptTurn`, `StartReview`

### High-level workflows

Run common end-to-end conversation flows with less manual event wiring.

- `SendMessageAndWait`
- `StreamTurn`
- `QuickThread`

### Events and notifications

Subscribe to typed server notifications and item delta streams.

- `RegisterNotificationHandler`
- `OnEvent[T]`
- `OnTurnCompleted`
- `OnItemDelta`
- typed event structs such as `ThreadStartedEvent`, `TurnCompletedEvent`, `AgentMessageDeltaEvent`

### Discovery and account

Discover models and capabilities, and inspect account and rate-limit state.

- `ListModels`, `ListAllModels`
- `ListExperimentalFeatures`
- `ListCollaborationModes`
- `ReadAccount`
- `ReadRateLimits`

### Command execution

Run sandboxed commands outside thread flows and manage interactive sessions.

- `ExecCommand`
- `WriteCommandStdin`
- `ResizeCommandPTY`
- `TerminateCommand`

### Configuration and filesystem

Read and update config, requirements, external-agent migration data, and filesystem state.

- `ReadConfig`, `WriteConfigValue`, `BatchWriteConfig`, `ReadConfigRequirements`
- `DetectExternalAgentConfig`, `ImportExternalAgentConfig`
- `ReadFile`, `WriteFile`, `CreateDirectory`, `GetMetadata`, `ReadDirectory`, `RemovePath`, `CopyPath`

### Skills, plugins, apps, and MCP

Work with skills, plugin metadata, apps/connectors, and MCP server integration.

- `ListSkills`, `WriteSkillsConfig`
- `ListPlugins`, `ReadPlugin`
- `ListApps`
- `StartMCPOAuthLogin`, `ReloadMCPServerConfig`, `ListMCPServerStatus`

### Feedback and platform helpers

Submit feedback and trigger platform-specific helper flows.

- `UploadFeedback`
- `StartWindowsSandboxSetup`

## Error Handling

Server-side JSON-RPC failures are wrapped as `*appserver.RPCError`.

```go
var rpcErr *appserver.RPCError
if errors.As(err, &rpcErr) {
	log.Printf("rpc error: code=%d message=%s", rpcErr.Code, rpcErr.Message)
}
```

Helper predicates are available for common cases:

- `appserver.IsValidationError(err)`
- `appserver.IsNotInitializedError(err)`
- `appserver.IsRateLimitError(err)`

Process-level failures are reported separately as `*appserver.ProcessExitError`.

## Configuration

`StartOptions` controls how the client launches and manages `codex app-server`.

Common fields:

- `Command`, `Args`, `Dir`, `Env`, `Stderr`
- `ClientInfo`
- `Capabilities`
- `RestartOnFailure`
- `RestartPolicy`

Enable experimental endpoints and fields:

```go
opts := appserver.StartOptions{
	ClientInfo: appserver.ClientInfo{
		Name:    "example_client",
		Title:   "Example Client",
		Version: "0.1.0",
	},
}.SetExperimentalAPI(true)
```

Opt out of specific notifications for one connection:

```go
opts := appserver.StartOptions{
	ClientInfo: appserver.ClientInfo{
		Name:    "example_client",
		Title:   "Example Client",
		Version: "0.1.0",
	},
}.SetNotificationOptOut(
	appserver.MethodThreadStarted,
	appserver.MethodItemAgentMessageDelta,
)
```

## Contributing

Contributions are welcome.

Recommended workflow:

1. Open an issue or discussion for non-trivial changes.
2. Keep changes aligned with `ROADMAP.md` and the Codex App Server schema under `api/codex-app-server/`.
3. Add or update tests for new behavior.
4. Run:

```bash
go test ./...
```

Useful references:

- [`ROADMAP.md`](./ROADMAP.md)
- [`docs/codex-app-server/260322-codex-app-server.md`](./docs/codex-app-server/260322-codex-app-server.md)
- [`api/codex-app-server/`](./api/codex-app-server/)

## License

Apache License 2.0. See [`LICENSE`](./LICENSE).

## Tasks

* Below is for [xc](https://github.com/joerdav/xc)

### sync-codex-app-schema

* Sync codex app schema to [api/codex-app-server](./api/codex-app-server)

```bash
#!/usr/bin/env bash

codex app-server generate-json-schema --out api/codex-app-server
```
