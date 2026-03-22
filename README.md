# codex-app-server-sdk-go

[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](#installation)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](#license)
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
- `codex` CLI installed and available on `PATH`

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
package main

import (
	"context"
	"log"

	"github.com/C7A6/codex-app-server-sdk-go/pkg/appserver"
)

func main() {
	ctx := context.Background()

	client, _, err := appserver.NewClient(ctx, appserver.StartOptions{
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
}
```

### Use `SendMessageAndWait`

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/C7A6/codex-app-server-sdk-go/pkg/appserver"
)

func main() {
	client, _, err := appserver.NewClient(context.Background(), appserver.StartOptions{
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

	thread, err := client.StartThread(context.Background(), appserver.ThreadStartParams{})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	completed, err := client.SendMessageAndWait(ctx, thread.Thread.ID, "Explain the purpose of this SDK.")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("turn completed: %s (%s)", completed.Turn.ID, completed.Turn.Status)
}
```

### Use `OnEvent[T]` typed event handlers

```go
package main

import (
	"context"
	"log"

	"github.com/C7A6/codex-app-server-sdk-go/pkg/appserver"
)

func main() {
	client, _, err := appserver.NewClient(context.Background(), appserver.StartOptions{
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

	unregister, err := appserver.OnEvent(client, func(ctx context.Context, event *appserver.ThreadStartedEvent) {
		log.Printf("thread started: %s", event.Thread.ID)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer unregister()

	if _, err := client.StartThread(context.Background(), appserver.ThreadStartParams{}); err != nil {
		log.Fatal(err)
	}
}
```

### Use `StreamTurn` for streaming

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/C7A6/codex-app-server-sdk-go/pkg/appserver"
)

func main() {
	client, _, err := appserver.NewClient(context.Background(), appserver.StartOptions{
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

	thread, err := client.StartThread(context.Background(), appserver.ThreadStartParams{})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	events, _, err := client.StreamTurn(ctx, appserver.TurnStartParams{
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
}
```

## API Reference

The SDK exposes low-level request methods plus higher-level workflow helpers.

### Connection and lifecycle

- `NewClient`
- `StartStdio`
- `Close`
- `Call`
- `Notify`
- `OnProcessExit`

### Initialization and capabilities

- `Initialize`
- `Initialized`
- `StartOptions.SetExperimentalAPI`
- `StartOptions.SetNotificationOptOut`

### Threads and turns

- `StartThread`, `ResumeThread`, `ForkThread`, `ReadThread`
- `ListThreads`, `ListLoadedThreads`, `ListAllThreads`
- `SetThreadName`, `ArchiveThread`, `UnarchiveThread`, `UnsubscribeThread`
- `CompactThread`, `RollbackThread`
- `StartTurn`, `SteerTurn`, `InterruptTurn`, `StartReview`

### High-level workflows

- `SendMessageAndWait`
- `StreamTurn`
- `QuickThread`

### Events and notifications

- `RegisterNotificationHandler`
- `OnEvent[T]`
- `OnTurnCompleted`
- `OnItemDelta`
- typed event structs such as `ThreadStartedEvent`, `TurnCompletedEvent`, `AgentMessageDeltaEvent`

### Discovery and account

- `ListModels`, `ListAllModels`
- `ListExperimentalFeatures`
- `ListCollaborationModes`
- `ReadAccount`
- `ReadRateLimits`

### Command execution

- `ExecCommand`
- `WriteCommandStdin`
- `ResizeCommandPTY`
- `TerminateCommand`

### Configuration and filesystem

- `ReadConfig`, `WriteConfigValue`, `BatchWriteConfig`, `ReadConfigRequirements`
- `DetectExternalAgentConfig`, `ImportExternalAgentConfig`
- `ReadFile`, `WriteFile`, `CreateDirectory`, `GetMetadata`, `ReadDirectory`, `RemovePath`, `CopyPath`

### Skills, plugins, apps, and MCP

- `ListSkills`, `WriteSkillsConfig`
- `ListPlugins`, `ReadPlugin`
- `ListApps`
- `StartMCPOAuthLogin`, `ReloadMCPServerConfig`, `ListMCPServerStatus`

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

MIT. See [`LICENSE`](./LICENSE).
