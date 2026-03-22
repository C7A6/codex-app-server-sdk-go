package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
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

	turnCompleted := make(chan *TurnCompletedEvent, 1)
	unregister, err := client.RegisterNotificationHandler(MethodTurnCompleted, func(ctx context.Context, notification Notification) {
		event, err := notification.DecodeEvent()
		if err != nil {
			t.Errorf("DecodeEvent returned error: %v", err)
			return
		}

		completed, ok := event.(*TurnCompletedEvent)
		if !ok || completed.ThreadID != started.Thread.ID {
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

	var turnStartResult struct {
		Turn Turn `json:"turn"`
	}
	if err := client.Call(ctx, "turn/start", map[string]any{
		"threadId": started.Thread.ID,
		"input": []map[string]any{
			{
				"type": "text",
				"text": "hello",
			},
		},
	}, &turnStartResult); err != nil {
		t.Fatalf("turn/start returned error: %v", err)
	}

	select {
	case <-turnCompleted:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for turn completion: %v", ctx.Err())
	}

	return started
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
