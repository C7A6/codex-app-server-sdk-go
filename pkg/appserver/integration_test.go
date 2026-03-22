package appserver

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

func TestRejectsRequestsBeforeInitialization(t *testing.T) {
	requireCodex(t)

	conn, cleanup := startTempDirUninitializedConn(t)
	defer cleanup()

	var result ThreadStartResult
	err := conn.Call(context.Background(), "thread/start", map[string]any{}, &result)
	if err == nil {
		t.Fatal("expected not initialized error")
	}
	if !strings.Contains(err.Error(), "Not initialized") {
		t.Fatalf("expected not initialized error, got: %v", err)
	}
}

func TestRejectsDuplicateInitializeOnSameConnection(t *testing.T) {
	requireCodex(t)

	conn, cleanup := startTempDirUninitializedConn(t)
	defer cleanup()

	_, err := Initialize(context.Background(), conn, InitializeParams{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if err := Initialized(context.Background(), conn); err != nil {
		t.Fatalf("Initialized returned error: %v", err)
	}

	_, err = Initialize(context.Background(), conn, InitializeParams{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
	})
	if err == nil {
		t.Fatal("expected already initialized error")
	}
	if !strings.Contains(err.Error(), "Already initialized") {
		t.Fatalf("expected already initialized error, got: %v", err)
	}
}

func TestNotificationOptOutSuppressesExactMethodOnly(t *testing.T) {
	requireCodex(t)

	opts := StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		Dir: t.TempDir(),
	}
	opts = opts.SetNotificationOptOut(MethodThreadStarted)

	client, _, err := StartStdio(context.Background(), opts)
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	threadStartedCh := make(chan *ThreadStartedEvent, 1)
	unregisterThread, err := client.RegisterNotificationHandler(MethodThreadStarted, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}
		if typed, ok := event.(*ThreadStartedEvent); ok {
			threadStartedCh <- typed
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler thread/started returned error: %v", err)
	}
	defer unregisterThread()

	turnStartedCh := make(chan *TurnStartedEvent, 1)
	unregisterTurn, err := client.RegisterNotificationHandler(MethodTurnStarted, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}
		if typed, ok := event.(*TurnStartedEvent); ok {
			turnStartedCh <- typed
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler turn/started returned error: %v", err)
	}
	defer unregisterTurn()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	if _, err := client.StartTurn(context.Background(), TurnStartParams{
		ThreadID: started.Thread.ID,
		Input:    []TurnStartInputItem{{"type": "text", "text": "say hello"}},
	}); err != nil {
		t.Fatalf("StartTurn returned error: %v", err)
	}

	select {
	case turnStarted := <-turnStartedCh:
		if turnStarted.ThreadID != started.Thread.ID {
			t.Fatalf("unexpected turn started event: %#v", turnStarted)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for turn/started notification")
	}

	select {
	case event := <-threadStartedCh:
		t.Fatalf("unexpected thread/started notification despite opt-out: %#v", event)
	case <-time.After(500 * time.Millisecond):
	}
}

func TestUnknownNotificationOptOutMethodIsIgnored(t *testing.T) {
	requireCodex(t)

	opts := StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		Dir: t.TempDir(),
	}
	opts = opts.SetNotificationOptOut("does/not/exist")

	client, _, err := StartStdio(context.Background(), opts)
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	threadStartedCh := make(chan *ThreadStartedEvent, 1)
	unregister, err := client.RegisterNotificationHandler(MethodThreadStarted, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}
		if typed, ok := event.(*ThreadStartedEvent); ok {
			threadStartedCh <- typed
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler returned error: %v", err)
	}
	defer unregister()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	select {
	case event := <-threadStartedCh:
		if event.Thread.ID != started.Thread.ID {
			t.Fatalf("unexpected thread started event: %#v", event)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for thread/started notification")
	}
}

func TestUnsubscribeEmitsThreadClosedNotifications(t *testing.T) {
	requireCodex(t)

	client, _ := startTempDirClient(t, false, nil)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	statusChangedCh := make(chan *ThreadStatusChangedEvent, 1)
	unregisterStatus, err := client.RegisterNotificationHandler(MethodThreadStatusChanged, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}
		if typed, ok := event.(*ThreadStatusChangedEvent); ok && typed.ThreadID == started.Thread.ID {
			statusChangedCh <- typed
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler status changed returned error: %v", err)
	}
	defer unregisterStatus()

	closedCh := make(chan *ThreadClosedEvent, 1)
	unregisterClosed, err := client.RegisterNotificationHandler(MethodThreadClosed, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}
		if typed, ok := event.(*ThreadClosedEvent); ok && typed.ThreadID == started.Thread.ID {
			closedCh <- typed
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler thread closed returned error: %v", err)
	}
	defer unregisterClosed()

	result, err := client.UnsubscribeThread(context.Background(), ThreadUnsubscribeParams{ThreadID: started.Thread.ID})
	if err != nil {
		t.Fatalf("UnsubscribeThread returned error: %v", err)
	}
	if result.Status != ThreadUnsubscribeStatusUnsubscribed {
		t.Fatalf("unexpected unsubscribe status: %#v", result)
	}

	select {
	case event := <-statusChangedCh:
		if event.Status.Type != "notLoaded" {
			t.Fatalf("unexpected status changed event: %#v", event)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for thread/status/changed notification")
	}

	select {
	case event := <-closedCh:
		if event.ThreadID != started.Thread.ID {
			t.Fatalf("unexpected thread closed event: %#v", event)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for thread/closed notification")
	}
}

func TestArchiveAndUnarchiveEmitNotifications(t *testing.T) {
	requireCodex(t)

	client, _ := startTempDirClient(t, false, nil)
	defer func() {
		_ = client.Close()
	}()

	started := createPersistedThread(t, client)

	archivedCh := make(chan string, 1)
	unregisterArchived, err := client.RegisterNotificationHandler(MethodThreadArchived, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}
		if typed, ok := event.(*ThreadArchivedEvent); ok {
			if typed.ThreadID != "" {
				archivedCh <- typed.ThreadID
				return
			}
			if typed.Thread != nil {
				archivedCh <- typed.Thread.ID
			}
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler thread archived returned error: %v", err)
	}
	defer unregisterArchived()

	unarchivedCh := make(chan string, 1)
	unregisterUnarchived, err := client.RegisterNotificationHandler(MethodThreadUnarchived, func(ctx context.Context, notification Notification) {
		var payload struct {
			ThreadID string  `json:"threadId"`
			Thread   *Thread `json:"thread,omitempty"`
		}
		if decodeErr := notification.DecodeParams(&payload); decodeErr != nil {
			return
		}
		if payload.ThreadID != "" {
			unarchivedCh <- payload.ThreadID
			return
		}
		if payload.Thread != nil {
			unarchivedCh <- payload.Thread.ID
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler thread unarchived returned error: %v", err)
	}
	defer unregisterUnarchived()

	if _, err := client.ArchiveThread(context.Background(), ThreadArchiveParams{ThreadID: started.Thread.ID}); err != nil {
		t.Fatalf("ArchiveThread returned error: %v", err)
	}

	select {
	case threadID := <-archivedCh:
		if threadID != started.Thread.ID {
			t.Fatalf("unexpected archived thread id: %q", threadID)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for thread/archived notification")
	}

	if _, err := client.UnarchiveThread(context.Background(), ThreadUnarchiveParams{ThreadID: started.Thread.ID}); err != nil {
		t.Fatalf("UnarchiveThread returned error: %v", err)
	}

	select {
	case threadID := <-unarchivedCh:
		if threadID != started.Thread.ID {
			t.Fatalf("unexpected unarchived thread id: %q", threadID)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for thread/unarchived notification")
	}
}

func TestCommandExecRejectsEmptyCommand(t *testing.T) {
	requireCodex(t)

	client, _ := startTempDirClient(t, false, nil)
	defer func() {
		_ = client.Close()
	}()

	_, err := client.ExecCommand(context.Background(), CommandExecParams{Command: []string{}})
	if err == nil {
		t.Fatal("expected validation error for empty command")
	}

	var exitErr *ProcessExitError
	if errors.As(err, &exitErr) {
		t.Fatalf("expected protocol validation error, got process exit error: %v", err)
	}
}

func TestSendMessageAndWaitWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTempDirClient(t, false, nil)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	completed, err := client.SendMessageAndWait(ctx, started.Thread.ID, "Say hello in one sentence.")
	if err != nil {
		t.Fatalf("SendMessageAndWait returned error: %v", err)
	}
	if completed.ThreadID != started.Thread.ID {
		t.Fatalf("unexpected thread id: %#v", completed)
	}
	if completed.Turn.ID == "" || completed.Turn.Status == "" {
		t.Fatalf("unexpected completed turn payload: %#v", completed)
	}
}

func TestStreamTurnWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTempDirClient(t, false, nil)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	events, turnStart, err := client.StreamTurn(ctx, TurnStartParams{
		ThreadID: started.Thread.ID,
		Input: []TurnStartInputItem{
			{"type": "text", "text": "Summarize this thread."},
		},
	})
	if err != nil {
		t.Fatalf("StreamTurn returned error: %v", err)
	}
	if turnStart == nil || turnStart.Turn.ID == "" {
		t.Fatalf("unexpected turn start result: %#v", turnStart)
	}

	sawCompleted := false
	sawTurnLifecycle := false
	sawItemLifecycle := false

	for event := range events {
		switch typed := event.(type) {
		case *TurnStartedEvent:
			if typed.ThreadID == started.Thread.ID {
				sawTurnLifecycle = true
			}
		case *TurnCompletedEvent:
			if typed.ThreadID == started.Thread.ID && typed.Turn.ID == turnStart.Turn.ID {
				sawCompleted = true
			}
		case *ItemStartedEvent:
			if typed.ThreadID == started.Thread.ID && typed.TurnID == turnStart.Turn.ID {
				sawItemLifecycle = true
			}
		case *ItemCompletedEvent:
			if typed.ThreadID == started.Thread.ID && typed.TurnID == turnStart.Turn.ID {
				sawItemLifecycle = true
			}
		}
	}

	if !sawCompleted {
		t.Fatal("expected streamed turn/completed event")
	}
	if !sawTurnLifecycle {
		t.Fatal("expected streamed turn lifecycle event")
	}
	if !sawItemLifecycle {
		t.Fatal("expected streamed item lifecycle event")
	}
}

func TestQuickThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTempDirClient(t, false, nil)
	defer func() {
		_ = client.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	thread, completed, err := client.QuickThread(ctx, ThreadStartParams{}, "Explain what this assistant does.")
	if err != nil {
		t.Fatalf("QuickThread returned error: %v", err)
	}
	if thread == nil || thread.Thread.ID == "" {
		t.Fatalf("unexpected thread result: %#v", thread)
	}
	if completed == nil || completed.ThreadID != thread.Thread.ID {
		t.Fatalf("unexpected completed turn result: %#v", completed)
	}
}

func startTempDirClient(t *testing.T, restartOnFailure bool, mutate func(*StartOptions)) (*Client, *InitializeResult) {
	t.Helper()

	opts := StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		Dir:              t.TempDir(),
		RestartOnFailure: restartOnFailure,
	}
	if mutate != nil {
		mutate(&opts)
	}

	client, result, err := StartStdio(context.Background(), opts)
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	return client, result
}

func startTempDirUninitializedConn(t *testing.T) (*jsonrpc2.Conn, func()) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "codex", "app-server")
	cmd.Dir = t.TempDir()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe returned error: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe returned error: %v", err)
	}

	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start returned error: %v", err)
	}

	stdio := &processStdio{
		stdin:  stdin,
		stdout: stdout,
	}
	handlerClient := &Client{
		notificationHandlers: make(map[string]map[uint64]NotificationHandler),
		processExitHandlers:  make(map[uint64]ProcessExitHandler),
	}
	conn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewPlainObjectStream(stdio), &clientHandler{client: handlerClient})

	cleanup := func() {
		_ = conn.Close()
		_ = stdio.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}

	return conn, cleanup
}
