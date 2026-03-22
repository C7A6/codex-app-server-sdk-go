package appserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestStartStdioInitializesConnection(t *testing.T) {
	requireCodex(t)

	client, result := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	if result.UserAgent == "" {
		t.Fatal("expected user agent")
	}
	if result.PlatformFamily == "" {
		t.Fatal("expected platform family")
	}
	if result.PlatformOS == "" {
		t.Fatal("expected platform os")
	}
}

func TestStartStdioRequiresClientInfo(t *testing.T) {
	_, _, err := StartStdio(context.Background(), StartOptions{})
	if err == nil {
		t.Fatal("expected missing client info error")
	}
}

func TestStartStdioReturnsErrorWhenBinaryIsMissing(t *testing.T) {
	_, _, err := StartStdio(context.Background(), StartOptions{
		Command: "/definitely-missing-codex-binary",
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Title:   "Test Client",
			Version: "0.1.0",
		},
	})
	if err == nil {
		t.Fatal("expected missing binary error")
	}
}

func TestReadAccountAndRateLimitsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	account, err := client.ReadAccount(context.Background(), AccountReadParams{RefreshToken: false})
	if err != nil {
		t.Fatalf("ReadAccount returned error: %v", err)
	}
	if account == nil {
		t.Fatal("expected account result")
	}

	rateLimits, err := client.ReadRateLimits(context.Background())
	if err != nil {
		t.Fatalf("ReadRateLimits returned error: %v", err)
	}
	if rateLimits == nil || rateLimits.RateLimits == nil || rateLimits.RateLimits.Primary == nil {
		t.Fatal("expected rate limits payload")
	}
}

func TestListModelsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	limit := uint32(10)
	includeHidden := false
	models, err := client.ListModels(context.Background(), ModelListParams{
		Limit:         &limit,
		IncludeHidden: &includeHidden,
	})
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}
	if models == nil || len(models.Data) == 0 {
		t.Fatal("expected at least one model")
	}
	if models.Data[0].ID == "" || models.Data[0].Model == "" || models.Data[0].DisplayName == "" {
		t.Fatalf("unexpected model payload: %#v", models.Data[0])
	}
	if len(models.Data[0].SupportedReasoningEfforts) == 0 {
		t.Fatalf("expected reasoning effort metadata: %#v", models.Data[0])
	}
}

func TestStartThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}
	if result == nil || result.Thread.ID == "" {
		t.Fatalf("expected thread in response: %#v", result)
	}
	if result.Model == "" || result.ModelProvider == "" || result.Cwd == "" {
		t.Fatalf("expected resolved thread defaults: %#v", result)
	}
}

func TestResumeThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)

	_ = client.Close()

	resumeClient, _ := startTestClient(t, false)
	defer func() {
		_ = resumeClient.Close()
	}()

	resumed, err := resumeClient.ResumeThread(context.Background(), ThreadResumeParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("ResumeThread returned error: %v", err)
	}
	if resumed == nil || resumed.Thread.ID != started.Thread.ID {
		t.Fatalf("expected resumed thread id %q, got %#v", started.Thread.ID, resumed)
	}
	if resumed.Model == "" || resumed.ModelProvider == "" || resumed.Cwd == "" {
		t.Fatalf("expected resolved resume defaults: %#v", resumed)
	}
}

func TestForkThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	_ = client.Close()

	forkClient, _ := startTestClient(t, false)
	defer func() {
		_ = forkClient.Close()
	}()

	forked, err := forkClient.ForkThread(context.Background(), ThreadForkParams{
		ThreadID:  started.Thread.ID,
		Ephemeral: false,
	})
	if err != nil {
		t.Fatalf("ForkThread returned error: %v", err)
	}
	if forked == nil || forked.Thread.ID == "" {
		t.Fatalf("expected forked thread response: %#v", forked)
	}
	if forked.Thread.ID == started.Thread.ID {
		t.Fatalf("expected new thread ID, got original %q", forked.Thread.ID)
	}
}

func TestReadThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	_ = client.Close()

	readClient, _ := startTestClient(t, false)
	defer func() {
		_ = readClient.Close()
	}()

	readResult, err := readClient.ReadThread(context.Background(), ThreadReadParams{
		ThreadID:     started.Thread.ID,
		IncludeTurns: false,
	})
	if err != nil {
		t.Fatalf("ReadThread returned error: %v", err)
	}
	if readResult == nil || readResult.Thread.ID != started.Thread.ID {
		t.Fatalf("expected thread id %q, got %#v", started.Thread.ID, readResult)
	}
}

func TestListThreadsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	_ = client.Close()

	listClient, _ := startTestClient(t, false)
	defer func() {
		_ = listClient.Close()
	}()

	limit := uint32(50)
	archived := false
	threads, err := listClient.ListThreads(context.Background(), ThreadListParams{
		Archived: &archived,
		Cwd:      &started.Cwd,
		Limit:    &limit,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if threads == nil || len(threads.Data) == 0 {
		t.Fatalf("expected at least one thread result: %#v", threads)
	}

	for _, thread := range threads.Data {
		if thread.ID == started.Thread.ID {
			return
		}
	}

	t.Fatalf("expected listed threads to include %q, got %#v", started.Thread.ID, threads.Data)
}

func TestListLoadedThreadsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	limit := uint32(50)
	loadedThreads, err := client.ListLoadedThreads(context.Background(), ThreadLoadedListParams{
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("ListLoadedThreads returned error: %v", err)
	}
	if loadedThreads == nil || len(loadedThreads.Data) == 0 {
		t.Fatalf("expected at least one loaded thread: %#v", loadedThreads)
	}

	for _, threadID := range loadedThreads.Data {
		if threadID == started.Thread.ID {
			return
		}
	}

	t.Fatalf("expected loaded threads to include %q, got %#v", started.Thread.ID, loadedThreads.Data)
}

func TestSetThreadNameWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	defer func() {
		_ = client.Close()
	}()

	name := "sdk rename test"
	result, err := client.SetThreadName(context.Background(), ThreadSetNameParams{
		ThreadID: started.Thread.ID,
		Name:     name,
	})
	if err != nil {
		t.Fatalf("SetThreadName returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil rename result")
	}

	readResult, err := client.ReadThread(context.Background(), ThreadReadParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("ReadThread returned error after rename: %v", err)
	}
	if readResult.Thread.Name == nil || *readResult.Thread.Name != name {
		t.Fatalf("expected thread name %q, got %#v", name, readResult.Thread.Name)
	}
}

func TestArchiveThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.ArchiveThread(context.Background(), ThreadArchiveParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("ArchiveThread returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil archive result")
	}

	limit := uint32(50)
	activeArchived := false
	activeList, err := client.ListThreads(context.Background(), ThreadListParams{
		Archived: &activeArchived,
		Cwd:      &started.Cwd,
		Limit:    &limit,
	})
	if err != nil {
		t.Fatalf("ListThreads active returned error: %v", err)
	}
	for _, thread := range activeList.Data {
		if thread.ID == started.Thread.ID {
			t.Fatalf("expected archived thread %q to be absent from active list", started.Thread.ID)
		}
	}

	archived := true
	archivedList, err := client.ListThreads(context.Background(), ThreadListParams{
		Archived: &archived,
		Cwd:      &started.Cwd,
		Limit:    &limit,
	})
	if err != nil {
		t.Fatalf("ListThreads archived returned error: %v", err)
	}
	for _, thread := range archivedList.Data {
		if thread.ID == started.Thread.ID {
			return
		}
	}

	t.Fatalf("expected archived thread %q in archived list, got %#v", started.Thread.ID, archivedList.Data)
}

func TestUnarchiveThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	defer func() {
		_ = client.Close()
	}()

	_, err := client.ArchiveThread(context.Background(), ThreadArchiveParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("ArchiveThread returned error: %v", err)
	}

	result, err := client.UnarchiveThread(context.Background(), ThreadUnarchiveParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("UnarchiveThread returned error: %v", err)
	}
	if result == nil || result.Thread.ID != started.Thread.ID {
		t.Fatalf("expected unarchived thread id %q, got %#v", started.Thread.ID, result)
	}

	limit := uint32(50)
	activeArchived := false
	activeList, err := client.ListThreads(context.Background(), ThreadListParams{
		Archived: &activeArchived,
		Cwd:      &started.Cwd,
		Limit:    &limit,
	})
	if err != nil {
		t.Fatalf("ListThreads active returned error: %v", err)
	}
	for _, thread := range activeList.Data {
		if thread.ID == started.Thread.ID {
			return
		}
	}

	t.Fatalf("expected unarchived thread %q in active list, got %#v", started.Thread.ID, activeList.Data)
}

func TestUnsubscribeThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	result, err := client.UnsubscribeThread(context.Background(), ThreadUnsubscribeParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("UnsubscribeThread returned error: %v", err)
	}
	switch result.Status {
	case ThreadUnsubscribeStatusUnsubscribed, ThreadUnsubscribeStatusNotSubscribed, ThreadUnsubscribeStatusNotLoaded:
	default:
		t.Fatalf("unexpected unsubscribe status: %#v", result)
	}
}

func TestCompactThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.CompactThread(context.Background(), ThreadCompactStartParams{
		ThreadID: started.Thread.ID,
	})
	if err != nil {
		t.Fatalf("CompactThread returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil compact result")
	}
}

func TestRollbackThreadWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	started := createPersistedThread(t, client)
	defer func() {
		_ = client.Close()
	}()

	startCompletedTurn(t, client, started.Thread.ID, "second turn")

	result, err := client.RollbackThread(context.Background(), ThreadRollbackParams{
		ThreadID: started.Thread.ID,
		NumTurns: 1,
	})
	if err != nil {
		t.Fatalf("RollbackThread returned error: %v", err)
	}
	if result == nil || result.Thread.ID != started.Thread.ID {
		t.Fatalf("expected rolled back thread id %q, got %#v", started.Thread.ID, result)
	}
	if len(result.Thread.Turns) == 0 {
		t.Fatalf("expected rollback result to include turns: %#v", result)
	}
	if len(result.Thread.Turns) != 1 {
		t.Fatalf("expected one remaining turn after rollback, got %#v", result.Thread.Turns)
	}
}

func TestStartTurnWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	result, err := client.StartTurn(context.Background(), TurnStartParams{
		ThreadID: started.Thread.ID,
		Input: []TurnStartInputItem{
			{
				"type": "text",
				"text": "hello from start turn test",
			},
		},
	})
	if err != nil {
		t.Fatalf("StartTurn returned error: %v", err)
	}
	if result == nil || result.Turn.ID == "" {
		t.Fatalf("expected turn in response: %#v", result)
	}
}

func TestSteerTurnWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	startTurnResult, err := client.StartTurn(ctx, TurnStartParams{
		ThreadID: started.Thread.ID,
		Input: []TurnStartInputItem{
			{
				"type": "text",
				"text": "Inspect every Go file in the current repository, summarize the package structure in detail, and include concrete file examples before giving a final answer.",
			},
		},
	})
	if err != nil {
		t.Fatalf("StartTurn returned error: %v", err)
	}
	if startTurnResult == nil || startTurnResult.Turn.ID == "" {
		t.Fatalf("expected active turn response: %#v", startTurnResult)
	}

	steerResult, err := client.SteerTurn(ctx, TurnSteerParams{
		ThreadID:       started.Thread.ID,
		ExpectedTurnID: startTurnResult.Turn.ID,
		Input: []TurnStartInputItem{
			{
				"type": "text",
				"text": "Keep it concise instead and end with a one-line summary.",
			},
		},
	})
	if err != nil {
		if !strings.Contains(err.Error(), "no active turn to steer") {
			t.Fatalf("SteerTurn returned unexpected error: %v", err)
		}
		return
	}
	if steerResult == nil || steerResult.TurnID != startTurnResult.Turn.ID {
		t.Fatalf("expected steer to target turn %q, got %#v", startTurnResult.Turn.ID, steerResult)
	}
}

func TestInterruptTurnWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	startTurnResult, err := client.StartTurn(ctx, TurnStartParams{
		ThreadID: started.Thread.ID,
		Input: []TurnStartInputItem{
			{
				"type": "text",
				"text": "Inspect every Go file in the current repository, summarize the package structure in detail, and include concrete file examples before giving a final answer.",
			},
		},
	})
	if err != nil {
		t.Fatalf("StartTurn returned error: %v", err)
	}
	if startTurnResult == nil || startTurnResult.Turn.ID == "" {
		t.Fatalf("expected active turn response: %#v", startTurnResult)
	}

	interruptResult, err := client.InterruptTurn(ctx, TurnInterruptParams{
		ThreadID: started.Thread.ID,
		TurnID:   startTurnResult.Turn.ID,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "no active turn to interrupt") && !strings.Contains(err.Error(), "turn not active") {
			t.Fatalf("InterruptTurn returned unexpected error: %v", err)
		}
		return
	}
	if interruptResult == nil {
		t.Fatal("expected non-nil interrupt result")
	}
}

func TestStartReviewWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	started, err := client.StartThread(context.Background(), ThreadStartParams{})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	delivery := ReviewDeliveryInline
	result, err := client.StartReview(context.Background(), ReviewStartParams{
		ThreadID: started.Thread.ID,
		Delivery: &delivery,
		Target: ReviewTarget{
			"type":         "custom",
			"instructions": "Review the current state briefly and report any obvious issues.",
		},
	})
	if err != nil {
		t.Fatalf("StartReview returned error: %v", err)
	}
	if result == nil || result.ReviewThreadID == "" || result.Turn.ID == "" {
		t.Fatalf("expected review start result payload: %#v", result)
	}
}

func TestExecCommandWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("LookPath returned error: %v", err)
	}

	result, err := client.ExecCommand(context.Background(), CommandExecParams{
		Command: []string{bashPath, "-lc", "pwd"},
		Cwd:     &cwd,
	})
	if err != nil {
		t.Fatalf("ExecCommand returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected command result")
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exitCode=0, got %#v", result)
	}
	if strings.TrimSpace(result.Stdout) != cwd {
		t.Fatalf("expected stdout to equal cwd %q, got %#v", cwd, result.Stdout)
	}
}

func TestWriteCommandStdinWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("LookPath returned error: %v", err)
	}

	processID := "stdin-test-process"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultCh := make(chan *CommandExecResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := client.ExecCommand(ctx, CommandExecParams{
			Command:     []string{bashPath, "-lc", "read line; printf '%s' \"$line\""},
			ProcessID:   &processID,
			StreamStdin: true,
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	payload := base64.StdEncoding.EncodeToString([]byte("hello stdin\n"))
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err = client.WriteCommandStdin(ctx, CommandExecWriteParams{
			ProcessID:   processID,
			DeltaBase64: &payload,
			CloseStdin:  true,
		})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("WriteCommandStdin returned error: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ExecCommand returned error: %v", err)
	case result := <-resultCh:
		if result == nil {
			t.Fatal("expected command result")
		}
		if result.ExitCode != 0 || result.Stdout != "hello stdin" {
			t.Fatalf("unexpected command result: %#v", result)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for command result: %v", ctx.Err())
	}
}

func TestResizeCommandPTYWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("LookPath returned error: %v", err)
	}

	processID := "resize-test-process"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultCh := make(chan *CommandExecResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := client.ExecCommand(ctx, CommandExecParams{
			Command:     []string{bashPath, "-lc", "sleep 1"},
			ProcessID:   &processID,
			TTY:         true,
			Size:        &CommandExecTerminalSize{Rows: 24, Cols: 80},
			StreamStdin: true,
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err = client.ResizeCommandPTY(ctx, CommandExecResizeParams{
			ProcessID: processID,
			Size: CommandExecTerminalSize{
				Rows: 40,
				Cols: 120,
			},
		})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("ResizeCommandPTY returned error: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ExecCommand returned error: %v", err)
	case <-resultCh:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for command result: %v", ctx.Err())
	}
}

func TestProcessExitReturnsErrorWhenRestartDisabled(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	killActiveProcess(t, client)

	_, err := client.ReadAccount(context.Background(), AccountReadParams{RefreshToken: false})
	if err == nil {
		t.Fatal("expected process exit error")
	}

	var exitErr *ProcessExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ProcessExitError, got %T: %v", err, err)
	}
}

func TestProcessExitRestartsWhenEnabled(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, true)
	defer func() {
		_ = client.Close()
	}()

	initialPID := currentPID(t, client)
	killActiveProcess(t, client)

	account, err := client.ReadAccount(context.Background(), AccountReadParams{RefreshToken: false})
	if err != nil {
		t.Fatalf("ReadAccount returned error after restart: %v", err)
	}
	if account == nil {
		t.Fatal("expected account result after restart")
	}

	restartedPID := currentPID(t, client)
	if restartedPID == initialPID {
		t.Fatalf("expected restarted process pid to change, got %d", restartedPID)
	}
}

func TestRegisterNotificationHandlerReceivesThreadStarted(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	type threadStartedNotification struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}

	received := make(chan threadStartedNotification, 1)
	unregister, err := client.RegisterNotificationHandler("thread/started", func(ctx context.Context, notification Notification) {
		var payload threadStartedNotification
		if err := notification.DecodeParams(&payload); err != nil {
			t.Errorf("DecodeParams returned error: %v", err)
			return
		}
		select {
		case received <- payload:
		default:
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler returned error: %v", err)
	}
	defer unregister()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result json.RawMessage
	if err := client.Call(ctx, "thread/start", map[string]any{}, &result); err != nil {
		t.Fatalf("thread/start returned error: %v", err)
	}

	select {
	case notification := <-received:
		if notification.Thread.ID == "" {
			t.Fatal("expected thread ID in notification")
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for thread/started notification: %v", ctx.Err())
	}
}

func TestDecodeNotificationEvent(t *testing.T) {
	t.Parallel()

	notification := Notification{
		Method: MethodAccountLoginCompleted,
		Params: json.RawMessage(`{"loginId":"login-123","success":true,"error":null}`),
	}

	event, err := notification.DecodeEvent()
	if err != nil {
		t.Fatalf("DecodeEvent returned error: %v", err)
	}

	loginCompleted, ok := event.(*AccountLoginCompletedEvent)
	if !ok {
		t.Fatalf("expected *AccountLoginCompletedEvent, got %T", event)
	}
	if loginCompleted.LoginID == nil || *loginCompleted.LoginID != "login-123" {
		t.Fatalf("unexpected login ID: %#v", loginCompleted.LoginID)
	}
	if !loginCompleted.Success {
		t.Fatal("expected success=true")
	}
}

func TestCoreTypeDecoding(t *testing.T) {
	t.Parallel()

	var review ReviewStartResult
	if err := json.Unmarshal([]byte(`{
		"reviewThreadId":"thr_review",
		"turn":{"id":"turn_1","status":"inProgress","items":[{"id":"item_1","type":"agentMessage","text":"hello"}]}
	}`), &review); err != nil {
		t.Fatalf("unmarshal review result: %v", err)
	}
	if review.ReviewThreadID != "thr_review" {
		t.Fatalf("unexpected review thread id: %q", review.ReviewThreadID)
	}
	if review.Turn.ID != "turn_1" || len(review.Turn.Items) != 1 {
		t.Fatalf("unexpected turn payload: %#v", review.Turn)
	}

	var commandResult CommandExecResult
	if err := json.Unmarshal([]byte(`{"exitCode":0,"stdout":"ok","stderr":""}`), &commandResult); err != nil {
		t.Fatalf("unmarshal command result: %v", err)
	}
	if commandResult.ExitCode != 0 || commandResult.Stdout != "ok" {
		t.Fatalf("unexpected command result: %#v", commandResult)
	}

	var configResult ConfigReadResult
	if err := json.Unmarshal([]byte(`{
		"config":{"model":"gpt-5.4"},
		"origins":{"model":{"source":"user"}}
	}`), &configResult); err != nil {
		t.Fatalf("unmarshal config result: %v", err)
	}
	if configResult.Config["model"] != "gpt-5.4" {
		t.Fatalf("unexpected config model: %#v", configResult.Config["model"])
	}

	var modelListResult ModelListResult
	if err := json.Unmarshal([]byte(`{
		"data":[{
			"id":"gpt-5.4",
			"model":"gpt-5.4",
			"displayName":"GPT-5.4",
			"description":"test model",
			"hidden":false,
			"isDefault":true,
			"defaultReasoningEffort":"medium",
			"supportedReasoningEfforts":[{"reasoningEffort":"low","description":"Lower latency"}],
			"inputModalities":["text","image"],
			"supportsPersonality":true
		}],
		"nextCursor":null
	}`), &modelListResult); err != nil {
		t.Fatalf("unmarshal model list result: %v", err)
	}
	if len(modelListResult.Data) != 1 || modelListResult.Data[0].ID != "gpt-5.4" {
		t.Fatalf("unexpected model list result: %#v", modelListResult)
	}

	var threadStartResult ThreadStartResult
	if err := json.Unmarshal([]byte(`{
		"approvalPolicy":"never",
		"approvalsReviewer":"user",
		"cwd":"/tmp",
		"model":"gpt-5.4",
		"modelProvider":"openai",
		"reasoningEffort":"medium",
		"sandbox":"workspace-write",
		"serviceTier":"fast",
		"thread":{"id":"thr_123","status":{"type":"idle"}}
	}`), &threadStartResult); err != nil {
		t.Fatalf("unmarshal thread start result: %v", err)
	}
	if threadStartResult.Thread.ID != "thr_123" || threadStartResult.Model != "gpt-5.4" {
		t.Fatalf("unexpected thread start result: %#v", threadStartResult)
	}

	var threadResumeResult ThreadResumeResult
	if err := json.Unmarshal([]byte(`{
		"approvalPolicy":"never",
		"approvalsReviewer":"user",
		"cwd":"/tmp",
		"model":"gpt-5.4",
		"modelProvider":"openai",
		"thread":{"id":"thr_123","status":{"type":"idle"}}
	}`), &threadResumeResult); err != nil {
		t.Fatalf("unmarshal thread resume result: %v", err)
	}
	if threadResumeResult.Thread.ID != "thr_123" || threadResumeResult.ModelProvider != "openai" {
		t.Fatalf("unexpected thread resume result: %#v", threadResumeResult)
	}

	var threadForkResult ThreadForkResult
	if err := json.Unmarshal([]byte(`{
		"approvalPolicy":"never",
		"approvalsReviewer":"user",
		"cwd":"/tmp",
		"model":"gpt-5.4",
		"modelProvider":"openai",
		"thread":{"id":"thr_forked","status":{"type":"idle"}}
	}`), &threadForkResult); err != nil {
		t.Fatalf("unmarshal thread fork result: %v", err)
	}
	if threadForkResult.Thread.ID != "thr_forked" || threadForkResult.Model != "gpt-5.4" {
		t.Fatalf("unexpected thread fork result: %#v", threadForkResult)
	}

	var threadReadResult ThreadReadResult
	if err := json.Unmarshal([]byte(`{
		"thread":{"id":"thr_read","status":{"type":"idle"}}
	}`), &threadReadResult); err != nil {
		t.Fatalf("unmarshal thread read result: %v", err)
	}
	if threadReadResult.Thread.ID != "thr_read" || threadReadResult.Thread.Status == nil || threadReadResult.Thread.Status.Type != "idle" {
		t.Fatalf("unexpected thread read result: %#v", threadReadResult)
	}

	var threadListResult ThreadListResult
	if err := json.Unmarshal([]byte(`{
		"data":[
			{"id":"thr_1","status":{"type":"idle"}},
			{"id":"thr_2","status":{"type":"running"}}
		],
		"nextCursor":"cursor_2"
	}`), &threadListResult); err != nil {
		t.Fatalf("unmarshal thread list result: %v", err)
	}
	if len(threadListResult.Data) != 2 || threadListResult.Data[1].ID != "thr_2" || threadListResult.NextCursor == nil || *threadListResult.NextCursor != "cursor_2" {
		t.Fatalf("unexpected thread list result: %#v", threadListResult)
	}

	var threadLoadedListResult ThreadLoadedListResult
	if err := json.Unmarshal([]byte(`{
		"data":["thr_1","thr_2"],
		"nextCursor":"cursor_3"
	}`), &threadLoadedListResult); err != nil {
		t.Fatalf("unmarshal thread loaded list result: %v", err)
	}
	if len(threadLoadedListResult.Data) != 2 || threadLoadedListResult.Data[0] != "thr_1" || threadLoadedListResult.NextCursor == nil || *threadLoadedListResult.NextCursor != "cursor_3" {
		t.Fatalf("unexpected thread loaded list result: %#v", threadLoadedListResult)
	}

	var threadSetNameResult ThreadSetNameResult
	if err := json.Unmarshal([]byte(`{}`), &threadSetNameResult); err != nil {
		t.Fatalf("unmarshal thread set name result: %v", err)
	}

	var threadArchiveResult ThreadArchiveResult
	if err := json.Unmarshal([]byte(`{}`), &threadArchiveResult); err != nil {
		t.Fatalf("unmarshal thread archive result: %v", err)
	}

	var threadUnarchiveResult ThreadUnarchiveResult
	if err := json.Unmarshal([]byte(`{
		"thread":{"id":"thr_restored","status":{"type":"idle"}}
	}`), &threadUnarchiveResult); err != nil {
		t.Fatalf("unmarshal thread unarchive result: %v", err)
	}
	if threadUnarchiveResult.Thread.ID != "thr_restored" || threadUnarchiveResult.Thread.Status == nil || threadUnarchiveResult.Thread.Status.Type != "idle" {
		t.Fatalf("unexpected thread unarchive result: %#v", threadUnarchiveResult)
	}

	var threadUnsubscribeResult ThreadUnsubscribeResult
	if err := json.Unmarshal([]byte(`{"status":"unsubscribed"}`), &threadUnsubscribeResult); err != nil {
		t.Fatalf("unmarshal thread unsubscribe result: %v", err)
	}
	if threadUnsubscribeResult.Status != ThreadUnsubscribeStatusUnsubscribed {
		t.Fatalf("unexpected thread unsubscribe result: %#v", threadUnsubscribeResult)
	}

	var threadCompactStartResult ThreadCompactStartResult
	if err := json.Unmarshal([]byte(`{}`), &threadCompactStartResult); err != nil {
		t.Fatalf("unmarshal thread compact start result: %v", err)
	}

	var threadRollbackResult ThreadRollbackResult
	if err := json.Unmarshal([]byte(`{
		"thread":{
			"id":"thr_rollback",
			"status":{"type":"idle"},
			"turns":[{"id":"turn_1","status":"completed"}]
		}
	}`), &threadRollbackResult); err != nil {
		t.Fatalf("unmarshal thread rollback result: %v", err)
	}
	if threadRollbackResult.Thread.ID != "thr_rollback" || len(threadRollbackResult.Thread.Turns) != 1 || threadRollbackResult.Thread.Turns[0].ID != "turn_1" {
		t.Fatalf("unexpected thread rollback result: %#v", threadRollbackResult)
	}

	var turnStartResult TurnStartResult
	if err := json.Unmarshal([]byte(`{
		"turn":{"id":"turn_started","status":"inProgress"}
	}`), &turnStartResult); err != nil {
		t.Fatalf("unmarshal turn start result: %v", err)
	}
	if turnStartResult.Turn.ID != "turn_started" || turnStartResult.Turn.Status != "inProgress" {
		t.Fatalf("unexpected turn start result: %#v", turnStartResult)
	}

	var turnSteerResult TurnSteerResult
	if err := json.Unmarshal([]byte(`{"turnId":"turn_started"}`), &turnSteerResult); err != nil {
		t.Fatalf("unmarshal turn steer result: %v", err)
	}
	if turnSteerResult.TurnID != "turn_started" {
		t.Fatalf("unexpected turn steer result: %#v", turnSteerResult)
	}

	var turnInterruptResult TurnInterruptResult
	if err := json.Unmarshal([]byte(`{}`), &turnInterruptResult); err != nil {
		t.Fatalf("unmarshal turn interrupt result: %v", err)
	}

	var reviewStartResult ReviewStartResult
	if err := json.Unmarshal([]byte(`{
		"reviewThreadId":"thr_review",
		"turn":{"id":"turn_review","status":"inProgress"}
	}`), &reviewStartResult); err != nil {
		t.Fatalf("unmarshal review start result: %v", err)
	}
	if reviewStartResult.ReviewThreadID != "thr_review" || reviewStartResult.Turn.ID != "turn_review" {
		t.Fatalf("unexpected review start result: %#v", reviewStartResult)
	}

	var commandExecResult CommandExecResult
	if err := json.Unmarshal([]byte(`{
		"exitCode":0,
		"stdout":"ok",
		"stderr":""
	}`), &commandExecResult); err != nil {
		t.Fatalf("unmarshal command exec result: %v", err)
	}
	if commandExecResult.ExitCode != 0 || commandExecResult.Stdout != "ok" {
		t.Fatalf("unexpected command exec result: %#v", commandExecResult)
	}

	var commandExecWriteResult CommandExecWriteResult
	if err := json.Unmarshal([]byte(`{}`), &commandExecWriteResult); err != nil {
		t.Fatalf("unmarshal command exec write result: %v", err)
	}

	var commandExecResizeResult CommandExecResizeResult
	if err := json.Unmarshal([]byte(`{}`), &commandExecResizeResult); err != nil {
		t.Fatalf("unmarshal command exec resize result: %v", err)
	}
}

func TestRegisterNotificationHandlerCanDecodeTypedEvent(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	received := make(chan *ThreadStartedEvent, 1)
	unregister, err := client.RegisterNotificationHandler(MethodThreadStarted, func(ctx context.Context, notification Notification) {
		event, err := notification.DecodeEvent()
		if err != nil {
			t.Errorf("DecodeEvent returned error: %v", err)
			return
		}

		threadStarted, ok := event.(*ThreadStartedEvent)
		if !ok {
			t.Errorf("expected *ThreadStartedEvent, got %T", event)
			return
		}

		select {
		case received <- threadStarted:
		default:
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler returned error: %v", err)
	}
	defer unregister()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result json.RawMessage
	if err := client.Call(ctx, "thread/start", map[string]any{}, &result); err != nil {
		t.Fatalf("thread/start returned error: %v", err)
	}

	select {
	case event := <-received:
		if event.Thread.ID == "" {
			t.Fatal("expected thread payload")
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for typed notification: %v", ctx.Err())
	}
}

func requireCodex(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("codex"); err != nil {
		t.Skipf("codex not found: %v", err)
	}
}

func startTestClient(t *testing.T, restartOnFailure bool) (*Client, *InitializeResult) {
	t.Helper()

	client, result, err := StartStdio(context.Background(), StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		RestartOnFailure: restartOnFailure,
	})
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	return client, result
}

func createPersistedThread(t *testing.T, client *Client) *ThreadStartResult {
	t.Helper()

	ephemeral := false
	started, err := client.StartThread(context.Background(), ThreadStartParams{
		Ephemeral: &ephemeral,
	})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	startCompletedTurn(t, client, started.Thread.ID, "hello")

	return started
}

func startCompletedTurn(t *testing.T, client *Client, threadID string, text string) {
	t.Helper()

	waitForCompletedTurn(t, client, threadID, text)
}

func waitForCompletedTurn(t *testing.T, client *Client, threadID string, text string) {
	t.Helper()

	turnCompleted := make(chan *TurnCompletedEvent, 1)
	unregister, err := client.RegisterNotificationHandler(MethodTurnCompleted, func(ctx context.Context, notification Notification) {
		event, err := notification.DecodeEvent()
		if err != nil {
			t.Errorf("DecodeEvent returned error: %v", err)
			return
		}

		completed, ok := event.(*TurnCompletedEvent)
		if !ok || completed.ThreadID != threadID {
			return
		}

		select {
		case turnCompleted <- completed:
		default:
		}
	})
	if err != nil {
		t.Fatalf("RegisterNotificationHandler returned error: %v", err)
	}
	defer unregister()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	turnStartResult, err := client.StartTurn(ctx, TurnStartParams{
		ThreadID: threadID,
		Input: []TurnStartInputItem{
			{
				"type": "text",
				"text": text,
			},
		},
	})
	if err != nil {
		t.Fatalf("StartTurn returned error: %v", err)
	}
	if turnStartResult == nil || turnStartResult.Turn.ID == "" {
		t.Fatalf("expected started turn response: %#v", turnStartResult)
	}

	select {
	case <-turnCompleted:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for turn completion: %v", ctx.Err())
	}
}

func killActiveProcess(t *testing.T, client *Client) {
	t.Helper()

	sess := currentSession(t, client)
	if err := sess.cmd.Process.Kill(); err != nil {
		t.Fatalf("kill process: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if sess.done() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("timed out waiting for process exit")
}

func currentPID(t *testing.T, client *Client) int {
	t.Helper()

	sess := currentSession(t, client)
	if sess.cmd == nil || sess.cmd.Process == nil {
		t.Fatal("expected active process")
	}
	return sess.cmd.Process.Pid
}

func currentSession(t *testing.T, client *Client) *session {
	t.Helper()

	client.mu.Lock()
	defer client.mu.Unlock()

	if client.session == nil {
		t.Fatal("expected active session")
	}
	return client.session
}
