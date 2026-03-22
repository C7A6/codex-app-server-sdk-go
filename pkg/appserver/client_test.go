package appserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"
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

func TestNewClientInitializesConnection(t *testing.T) {
	requireCodex(t)

	client, result, err := NewClient(context.Background(), StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if result == nil || result.UserAgent == "" {
		t.Fatalf("expected initialized result: %#v", result)
	}
}

func TestInitializeAndInitializedWithRealCodex(t *testing.T) {
	requireCodex(t)

	conn, cleanup := startUninitializedTestConn(t)
	defer cleanup()

	result, err := Initialize(context.Background(), conn, InitializeParams{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if result == nil || result.UserAgent == "" {
		t.Fatalf("expected initialize result: %#v", result)
	}

	if err := Initialized(context.Background(), conn); err != nil {
		t.Fatalf("Initialized returned error: %v", err)
	}
}

func TestStartOptionsCapabilitySetters(t *testing.T) {
	opts := StartOptions{}
	opts = opts.SetExperimentalAPI(true)
	opts = opts.SetNotificationOptOut("thread/updated", "item/updated")

	if opts.Capabilities == nil {
		t.Fatal("expected capabilities to be allocated")
	}
	if !opts.Capabilities.ExperimentalAPI {
		t.Fatal("expected experimental api to be enabled")
	}
	if len(opts.Capabilities.OptOutNotificationMethods) != 2 {
		t.Fatalf("expected notification opt-out methods: %#v", opts.Capabilities.OptOutNotificationMethods)
	}
	if opts.Capabilities.OptOutNotificationMethods[0] != "thread/updated" {
		t.Fatalf("unexpected opt-out methods: %#v", opts.Capabilities.OptOutNotificationMethods)
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

func TestListExperimentalFeaturesWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startExperimentalTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	limit := uint32(10)
	result, err := client.ListExperimentalFeatures(context.Background(), ExperimentalFeatureListParams{
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("ListExperimentalFeatures returned error: %v", err)
	}
	if result == nil || len(result.Data) == 0 {
		t.Fatalf("expected experimental feature data: %#v", result)
	}
	for _, feature := range result.Data {
		if feature.Name == "" {
			t.Fatalf("expected feature name: %#v", feature)
		}
		switch feature.Stage {
		case ExperimentalFeatureStageBeta, ExperimentalFeatureStageUnderDevelopment, ExperimentalFeatureStageStable, ExperimentalFeatureStageDeprecated, ExperimentalFeatureStageRemoved:
		default:
			t.Fatalf("unexpected feature stage: %#v", feature)
		}
	}
}

func TestListCollaborationModesWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startExperimentalTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.ListCollaborationModes(context.Background())
	if err != nil {
		t.Fatalf("ListCollaborationModes returned error: %v", err)
	}
	if result == nil || len(result.Data) == 0 {
		t.Fatalf("expected collaboration mode data: %#v", result)
	}
	for _, mode := range result.Data {
		if mode.Name == "" {
			t.Fatalf("expected collaboration mode name: %#v", mode)
		}
		if mode.Mode != nil {
			switch *mode.Mode {
			case CollaborationModeKindPlan, CollaborationModeKindDefault:
			default:
				t.Fatalf("unexpected collaboration mode kind: %#v", mode)
			}
		}
	}
}

func TestFeedbackAndWindowsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	feedbackResult, err := client.UploadFeedback(context.Background(), FeedbackUploadParams{
		Classification: "bug",
		IncludeLogs:    false,
	})
	if err != nil {
		t.Fatalf("UploadFeedback returned error: %v", err)
	}
	if feedbackResult == nil || feedbackResult.ThreadID == "" {
		t.Fatalf("expected feedback upload result: %#v", feedbackResult)
	}

	windowsResult, err := client.StartWindowsSandboxSetup(context.Background(), WindowsSandboxSetupStartParams{
		Mode: WindowsSandboxSetupModeUnelevated,
	})
	if err != nil {
		t.Fatalf("StartWindowsSandboxSetup returned error: %v", err)
	}
	if windowsResult == nil || !windowsResult.Started {
		t.Fatalf("expected windows sandbox setup result: %#v", windowsResult)
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

func TestTerminateCommandWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("LookPath returned error: %v", err)
	}

	processID := "terminate-test-process"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultCh := make(chan *CommandExecResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := client.ExecCommand(ctx, CommandExecParams{
			Command:            []string{bashPath, "-lc", "sleep 30"},
			ProcessID:          &processID,
			StreamStdoutStderr: true,
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err = client.TerminateCommand(ctx, CommandExecTerminateParams{
			ProcessID: processID,
		})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("TerminateCommand returned error: %v", err)
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
	case <-ctx.Done():
		t.Fatalf("timed out waiting for command result: %v", ctx.Err())
	}
}

func TestListSkillsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	result, err := client.ListSkills(context.Background(), SkillsListParams{
		Cwds:        []string{cwd},
		ForceReload: true,
	})
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if result == nil || len(result.Data) == 0 {
		t.Fatalf("expected skills list payload: %#v", result)
	}
	if result.Data[0].Cwd == "" {
		t.Fatalf("expected cwd in first entry: %#v", result.Data[0])
	}
}

func TestWriteSkillsConfigWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	listResult, err := client.ListSkills(context.Background(), SkillsListParams{
		Cwds:        []string{cwd},
		ForceReload: true,
	})
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if listResult == nil || len(listResult.Data) == 0 || len(listResult.Data[0].Skills) == 0 {
		t.Fatalf("expected at least one skill to write config for: %#v", listResult)
	}

	skill := listResult.Data[0].Skills[0]
	writeResult, err := client.WriteSkillsConfig(context.Background(), SkillsConfigWriteParams{
		Path:    skill.Path,
		Enabled: skill.Enabled,
	})
	if err != nil {
		t.Fatalf("WriteSkillsConfig returned error: %v", err)
	}
	if writeResult == nil || writeResult.EffectiveEnabled != skill.Enabled {
		t.Fatalf("expected effective enabled %v, got %#v", skill.Enabled, writeResult)
	}
}

func TestReadConfigWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	result, err := client.ReadConfig(context.Background(), ConfigReadParams{
		Cwd:           &cwd,
		IncludeLayers: true,
	})
	if err != nil {
		t.Fatalf("ReadConfig returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected config read result")
	}
	if result.Config == nil || result.Origins == nil {
		t.Fatalf("expected config and origins: %#v", result)
	}
}

func TestWriteConfigValueWithRealCodex(t *testing.T) {
	requireCodex(t)

	tempDir := t.TempDir()
	configHome := tempDir + "/codex-home"
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	client, _, err := StartStdio(context.Background(), StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		Env: append(os.Environ(), "CODEX_HOME="+configHome),
	})
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	configPath := configHome + "/config.toml"
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := client.WriteConfigValue(context.Background(), ConfigWriteParams{
		KeyPath:       "model",
		MergeStrategy: ConfigMergeStrategyReplace,
		Value:         "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("WriteConfigValue returned error: %v", err)
	}
	if result == nil || result.FilePath == "" || result.Version == "" {
		t.Fatalf("expected config write result metadata: %#v", result)
	}
	if result.Status != ConfigWriteStatusOK && result.Status != ConfigWriteStatusOKOverridden {
		t.Fatalf("unexpected config write status: %#v", result)
	}
	if result.FilePath != configPath {
		t.Fatalf("expected config path %q, got %#v", configPath, result)
	}
}

func TestBatchWriteConfigWithRealCodex(t *testing.T) {
	requireCodex(t)

	tempDir := t.TempDir()
	configHome := tempDir + "/codex-home"
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	client, _, err := StartStdio(context.Background(), StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		Env: append(os.Environ(), "CODEX_HOME="+configHome),
	})
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	configPath := configHome + "/config.toml"
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := client.BatchWriteConfig(context.Background(), ConfigBatchWriteParams{
		Edits: []ConfigEdit{
			{
				KeyPath:       "model",
				MergeStrategy: ConfigMergeStrategyReplace,
				Value:         "gpt-5.4",
			},
			{
				KeyPath:       "model_provider",
				MergeStrategy: ConfigMergeStrategyUpsert,
				Value:         "openai",
			},
		},
	})
	if err != nil {
		t.Fatalf("BatchWriteConfig returned error: %v", err)
	}
	if result == nil || result.FilePath == "" || result.Version == "" {
		t.Fatalf("expected config batch write result metadata: %#v", result)
	}
	if result.Status != ConfigWriteStatusOK && result.Status != ConfigWriteStatusOKOverridden {
		t.Fatalf("unexpected config batch write status: %#v", result)
	}
	if result.FilePath != configPath {
		t.Fatalf("expected config path %q, got %#v", configPath, result)
	}
}

func TestReadConfigRequirementsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.ReadConfigRequirements(context.Background())
	if err != nil {
		t.Fatalf("ReadConfigRequirements returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected config requirements result")
	}
	if result.Requirements != nil {
		for _, mode := range result.Requirements.AllowedSandboxModes {
			switch mode {
			case "read-only", "workspace-write", "danger-full-access":
			default:
				t.Fatalf("unexpected sandbox mode: %q", mode)
			}
		}
	}
}

func TestDetectExternalAgentConfigWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	result, err := client.DetectExternalAgentConfig(context.Background(), ExternalAgentConfigDetectParams{
		Cwds:        []string{cwd},
		IncludeHome: false,
	})
	if err != nil {
		t.Fatalf("DetectExternalAgentConfig returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected external agent detect result")
	}
	for _, item := range result.Items {
		if item.Description == "" {
			t.Fatalf("expected migration description: %#v", item)
		}
		switch item.ItemType {
		case ExternalAgentConfigMigrationItemTypeAgentsMD, ExternalAgentConfigMigrationItemTypeConfig, ExternalAgentConfigMigrationItemTypeSkills, ExternalAgentConfigMigrationItemTypeMCPServerConfig:
		default:
			t.Fatalf("unexpected migration item type: %#v", item)
		}
	}
}

func TestImportExternalAgentConfigWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.ImportExternalAgentConfig(context.Background(), ExternalAgentConfigImportParams{
		MigrationItems: []ExternalAgentConfigMigrationItem{},
	})
	if err != nil {
		t.Fatalf("ImportExternalAgentConfig returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected external agent import result")
	}
}

func TestFilesystemOperationsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	root := t.TempDir()
	nestedDir := root + "/nested/dir"
	filePath := nestedDir + "/hello.txt"
	copyPath := nestedDir + "/copy.txt"
	payload := []byte("hello via fs api")
	payloadBase64 := base64.StdEncoding.EncodeToString(payload)

	recursive := true
	createResult, err := client.CreateDirectory(context.Background(), FSCreateDirectoryParams{
		Path:      nestedDir,
		Recursive: &recursive,
	})
	if err != nil {
		t.Fatalf("CreateDirectory returned error: %v", err)
	}
	if createResult == nil {
		t.Fatal("expected create directory result")
	}

	writeResult, err := client.WriteFile(context.Background(), FSWriteFileParams{
		Path:       filePath,
		DataBase64: payloadBase64,
	})
	if err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if writeResult == nil {
		t.Fatal("expected write file result")
	}

	readResult, err := client.ReadFile(context.Background(), FSReadFileParams{Path: filePath})
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if readResult == nil || readResult.DataBase64 != payloadBase64 {
		t.Fatalf("unexpected read file result: %#v", readResult)
	}

	metadataResult, err := client.GetMetadata(context.Background(), FSGetMetadataParams{Path: filePath})
	if err != nil {
		t.Fatalf("GetMetadata returned error: %v", err)
	}
	if metadataResult == nil || !metadataResult.IsFile || metadataResult.IsDirectory {
		t.Fatalf("unexpected metadata result: %#v", metadataResult)
	}

	dirResult, err := client.ReadDirectory(context.Background(), FSReadDirectoryParams{Path: nestedDir})
	if err != nil {
		t.Fatalf("ReadDirectory returned error: %v", err)
	}
	if dirResult == nil {
		t.Fatal("expected read directory result")
	}
	var foundOriginal bool
	for _, entry := range dirResult.Entries {
		if entry.FileName == "hello.txt" {
			foundOriginal = true
			if !entry.IsFile || entry.IsDirectory {
				t.Fatalf("unexpected directory entry: %#v", entry)
			}
		}
	}
	if !foundOriginal {
		t.Fatalf("expected hello.txt entry: %#v", dirResult)
	}

	copyResult, err := client.CopyPath(context.Background(), FSCopyPathParams{
		SourcePath:      filePath,
		DestinationPath: copyPath,
		Recursive:       false,
	})
	if err != nil {
		t.Fatalf("CopyPath returned error: %v", err)
	}
	if copyResult == nil {
		t.Fatal("expected copy path result")
	}

	copiedReadResult, err := client.ReadFile(context.Background(), FSReadFileParams{Path: copyPath})
	if err != nil {
		t.Fatalf("ReadFile copied path returned error: %v", err)
	}
	if copiedReadResult == nil || copiedReadResult.DataBase64 != payloadBase64 {
		t.Fatalf("unexpected copied read result: %#v", copiedReadResult)
	}

	force := false
	removeResult, err := client.RemovePath(context.Background(), FSRemovePathParams{
		Path:  copyPath,
		Force: &force,
	})
	if err != nil {
		t.Fatalf("RemovePath returned error: %v", err)
	}
	if removeResult == nil {
		t.Fatal("expected remove path result")
	}

	dirResultAfterRemove, err := client.ReadDirectory(context.Background(), FSReadDirectoryParams{Path: nestedDir})
	if err != nil {
		t.Fatalf("ReadDirectory after remove returned error: %v", err)
	}
	for _, entry := range dirResultAfterRemove.Entries {
		if entry.FileName == "copy.txt" {
			t.Fatalf("expected copy.txt to be removed: %#v", dirResultAfterRemove)
		}
	}
}

func TestListPluginsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	result, err := client.ListPlugins(context.Background(), PluginListParams{
		Cwds: []string{cwd},
	})
	if err != nil {
		t.Fatalf("ListPlugins returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected plugin list result")
	}
	for _, marketplace := range result.Marketplaces {
		if marketplace.Name == "" || marketplace.Path == "" {
			t.Fatalf("expected marketplace metadata: %#v", marketplace)
		}
	}
}

func TestReadPluginWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	listResult, err := client.ListPlugins(context.Background(), PluginListParams{
		Cwds: []string{cwd},
	})
	if err != nil {
		t.Fatalf("ListPlugins returned error: %v", err)
	}

	var marketplacePath string
	var pluginName string
	for _, marketplace := range listResult.Marketplaces {
		if len(marketplace.Plugins) > 0 {
			marketplacePath = marketplace.Path
			pluginName = marketplace.Plugins[0].Name
			break
		}
	}
	if marketplacePath == "" || pluginName == "" {
		t.Skip("no readable plugin available in current environment")
	}

	readResult, err := client.ReadPlugin(context.Background(), PluginReadParams{
		MarketplacePath: marketplacePath,
		PluginName:      pluginName,
	})
	if err != nil {
		t.Fatalf("ReadPlugin returned error: %v", err)
	}
	if readResult == nil || readResult.Plugin.Summary.Name == "" || readResult.Plugin.MarketplacePath == "" {
		t.Fatalf("expected plugin detail payload: %#v", readResult)
	}
}

func TestListAppsWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	limit := uint32(10)
	forceRefetch := true
	result, err := client.ListApps(context.Background(), AppsListParams{
		Limit:        &limit,
		ForceRefetch: &forceRefetch,
	})
	if err != nil {
		t.Fatalf("ListApps returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected apps list result")
	}
	for _, app := range result.Data {
		if app.ID == "" || app.Name == "" {
			t.Fatalf("expected app identity fields: %#v", app)
		}
	}
}

func TestStartMCPOAuthLoginWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	var statusResult struct {
		Data []struct {
			AuthStatus string `json:"authStatus"`
			Name       string `json:"name"`
		} `json:"data"`
	}
	if err := client.Call(context.Background(), "mcpServerStatus/list", map[string]any{}, &statusResult); err != nil {
		t.Fatalf("mcpServerStatus/list returned error: %v", err)
	}

	var serverName string
	for _, status := range statusResult.Data {
		if status.AuthStatus == "notLoggedIn" || status.AuthStatus == "oAuth" {
			serverName = status.Name
			break
		}
	}
	if serverName == "" {
		t.Skip("no oauth-capable MCP server available in current environment")
	}

	timeoutSecs := int64(1)
	result, err := client.StartMCPOAuthLogin(context.Background(), MCPOAuthLoginParams{
		Name:        serverName,
		TimeoutSecs: &timeoutSecs,
	})
	if err != nil {
		t.Fatalf("StartMCPOAuthLogin returned error: %v", err)
	}
	if result == nil || result.AuthorizationURL == "" {
		t.Fatalf("expected authorization URL: %#v", result)
	}
}

func TestReloadMCPServerConfigWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	result, err := client.ReloadMCPServerConfig(context.Background())
	if err != nil {
		t.Fatalf("ReloadMCPServerConfig returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected reload result")
	}
}

func TestListMCPServerStatusWithRealCodex(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	limit := uint32(10)
	result, err := client.ListMCPServerStatus(context.Background(), MCPServerStatusListParams{
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("ListMCPServerStatus returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected MCP server status result")
	}
	for _, server := range result.Data {
		if server.Name == "" {
			t.Fatalf("expected MCP server name: %#v", server)
		}
		switch server.AuthStatus {
		case MCPAuthStatusUnsupported, MCPAuthStatusNotLoggedIn, MCPAuthStatusBearerToken, MCPAuthStatusOAuth:
		default:
			t.Fatalf("unexpected MCP auth status: %#v", server)
		}
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

func TestOnProcessExitReceivesProcessFailure(t *testing.T) {
	requireCodex(t)

	client, _ := startTestClient(t, false)
	defer func() {
		_ = client.Close()
	}()

	exitCh := make(chan error, 1)
	unregister, err := client.OnProcessExit(func(err error) {
		exitCh <- err
	})
	if err != nil {
		t.Fatalf("OnProcessExit returned error: %v", err)
	}
	defer unregister()

	killActiveProcess(t, client)

	select {
	case err := <-exitCh:
		if err == nil {
			t.Fatal("expected process exit error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process exit callback")
	}
}

func TestRestartPolicyAlwaysOverridesLegacyRestartFlag(t *testing.T) {
	requireCodex(t)

	client, _, err := StartStdio(context.Background(), StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		RestartOnFailure: false,
		RestartPolicy:    RestartPolicyAlways,
	})
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	initialPID := currentPID(t, client)
	killActiveProcess(t, client)

	account, err := client.ReadAccount(context.Background(), AccountReadParams{RefreshToken: false})
	if err != nil {
		t.Fatalf("ReadAccount returned error after restart policy recovery: %v", err)
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

	testCases := []struct {
		name   string
		input  Notification
		assert func(t *testing.T, event NotificationEvent)
	}{
		{
			name: "account login completed",
			input: Notification{
				Method: MethodAccountLoginCompleted,
				Params: json.RawMessage(`{"loginId":"login-123","success":true,"error":null}`),
			},
			assert: func(t *testing.T, event NotificationEvent) {
				t.Helper()
				loginCompleted, ok := event.(*AccountLoginCompletedEvent)
				if !ok {
					t.Fatalf("expected *AccountLoginCompletedEvent, got %T", event)
				}
				if loginCompleted.LoginID == nil || *loginCompleted.LoginID != "login-123" || !loginCompleted.Success {
					t.Fatalf("unexpected login completed event: %#v", loginCompleted)
				}
			},
		},
		{
			name: "windows sandbox completed",
			input: Notification{
				Method: MethodWindowsSandboxCompleted,
				Params: json.RawMessage(`{"mode":"unelevated","success":true,"error":null}`),
			},
			assert: func(t *testing.T, event NotificationEvent) {
				t.Helper()
				completed, ok := event.(*WindowsSandboxSetupCompletedEvent)
				if !ok {
					t.Fatalf("expected *WindowsSandboxSetupCompletedEvent, got %T", event)
				}
				if completed.Mode != WindowsSandboxSetupModeUnelevated || !completed.Success {
					t.Fatalf("unexpected windows sandbox event: %#v", completed)
				}
			},
		},
		{
			name: "fuzzy session updated",
			input: Notification{
				Method: MethodFuzzyFileSearchUpdated,
				Params: json.RawMessage(`{"sessionId":"sess-1","query":"roadmap","files":[{"file_name":"ROADMAP.md","path":"ROADMAP.md","root":"/tmp","score":99}]}`),
			},
			assert: func(t *testing.T, event NotificationEvent) {
				t.Helper()
				updated, ok := event.(*FuzzyFileSearchSessionUpdatedEvent)
				if !ok {
					t.Fatalf("expected *FuzzyFileSearchSessionUpdatedEvent, got %T", event)
				}
				if updated.SessionID != "sess-1" || len(updated.Files) != 1 || updated.Files[0].FileName != "ROADMAP.md" {
					t.Fatalf("unexpected fuzzy session update: %#v", updated)
				}
			},
		},
		{
			name: "fuzzy session completed",
			input: Notification{
				Method: MethodFuzzyFileSearchCompleted,
				Params: json.RawMessage(`{"sessionId":"sess-1"}`),
			},
			assert: func(t *testing.T, event NotificationEvent) {
				t.Helper()
				completed, ok := event.(*FuzzyFileSearchSessionCompletedEvent)
				if !ok {
					t.Fatalf("expected *FuzzyFileSearchSessionCompletedEvent, got %T", event)
				}
				if completed.SessionID != "sess-1" {
					t.Fatalf("unexpected fuzzy session completed: %#v", completed)
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := tc.input.DecodeEvent()
			if err != nil {
				t.Fatalf("DecodeEvent returned error: %v", err)
			}
			tc.assert(t, event)
		})
	}
}

func TestToolRequestUserInputHandler(t *testing.T) {
	t.Parallel()

	client := &Client{}
	if err := client.RegisterToolRequestUserInputHandler(func(ctx context.Context, params ToolRequestUserInputParams) (*ToolRequestUserInputResult, error) {
		if params.ItemID != "item-1" || len(params.Questions) != 1 {
			t.Fatalf("unexpected params: %#v", params)
		}
		return &ToolRequestUserInputResult{
			Answers: map[string]ToolRequestUserInputAnswer{
				"answer": {Answers: []string{"selected"}},
			},
		}, nil
	}); err != nil {
		t.Fatalf("RegisterToolRequestUserInputHandler returned error: %v", err)
	}

	params := json.RawMessage(`{
		"itemId":"item-1",
		"threadId":"thr-1",
		"turnId":"turn-1",
		"questions":[{"header":"Pick","id":"answer","question":"Choose one","options":[{"label":"A","description":"opt a"}]}]
	}`)
	result, handled, err := client.handleServerRequest(context.Background(), &jsonrpc2.Request{
		Method: MethodToolRequestUserInput,
		Params: &params,
	})
	if err != nil {
		t.Fatalf("handleServerRequest returned error: %v", err)
	}
	if !handled {
		t.Fatal("expected request to be handled")
	}

	typedResult, ok := result.(*ToolRequestUserInputResult)
	if !ok {
		t.Fatalf("expected *ToolRequestUserInputResult, got %T", result)
	}
	if len(typedResult.Answers["answer"].Answers) != 1 || typedResult.Answers["answer"].Answers[0] != "selected" {
		t.Fatalf("unexpected request user input result: %#v", typedResult)
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

	var configWriteResult ConfigWriteResult
	if err := json.Unmarshal([]byte(`{
		"filePath":"/tmp/config.toml",
		"status":"ok",
		"version":"v1"
	}`), &configWriteResult); err != nil {
		t.Fatalf("unmarshal config write result: %v", err)
	}
	if configWriteResult.FilePath != "/tmp/config.toml" || configWriteResult.Status != ConfigWriteStatusOK || configWriteResult.Version != "v1" {
		t.Fatalf("unexpected config write result: %#v", configWriteResult)
	}

	var configBatchWriteResult ConfigWriteResult
	if err := json.Unmarshal([]byte(`{
		"filePath":"/tmp/config.toml",
		"status":"okOverridden",
		"version":"v2"
	}`), &configBatchWriteResult); err != nil {
		t.Fatalf("unmarshal config batch write result: %v", err)
	}
	if configBatchWriteResult.FilePath != "/tmp/config.toml" || configBatchWriteResult.Status != ConfigWriteStatusOKOverridden || configBatchWriteResult.Version != "v2" {
		t.Fatalf("unexpected config batch write result: %#v", configBatchWriteResult)
	}

	var configRequirementsResult ConfigRequirementsReadResult
	if err := json.Unmarshal([]byte(`{
		"requirements":{
			"allowedSandboxModes":["workspace-write"],
			"allowedWebSearchModes":["live"],
			"enforceResidency":"us",
			"featureRequirements":{"experimental_api":true}
		}
	}`), &configRequirementsResult); err != nil {
		t.Fatalf("unmarshal config requirements result: %v", err)
	}
	if configRequirementsResult.Requirements == nil || len(configRequirementsResult.Requirements.AllowedSandboxModes) != 1 || configRequirementsResult.Requirements.EnforceResidency == nil || *configRequirementsResult.Requirements.EnforceResidency != ConfigRequirementsResidencyUS {
		t.Fatalf("unexpected config requirements result: %#v", configRequirementsResult)
	}

	var detectResult ExternalAgentConfigDetectResult
	if err := json.Unmarshal([]byte(`{
		"items":[{
			"cwd":"/workspace/repo",
			"description":"Import AGENTS.md guidance",
			"itemType":"AGENTS_MD"
		}]
	}`), &detectResult); err != nil {
		t.Fatalf("unmarshal external agent detect result: %v", err)
	}
	if len(detectResult.Items) != 1 || detectResult.Items[0].ItemType != ExternalAgentConfigMigrationItemTypeAgentsMD {
		t.Fatalf("unexpected external agent detect result: %#v", detectResult)
	}

	var importResult ExternalAgentConfigImportResult
	if err := json.Unmarshal([]byte(`{}`), &importResult); err != nil {
		t.Fatalf("unmarshal external agent import result: %v", err)
	}

	var fsReadFileResult FSReadFileResult
	if err := json.Unmarshal([]byte(`{"dataBase64":"aGVsbG8="}`), &fsReadFileResult); err != nil {
		t.Fatalf("unmarshal fs read file result: %v", err)
	}
	if fsReadFileResult.DataBase64 != "aGVsbG8=" {
		t.Fatalf("unexpected fs read file result: %#v", fsReadFileResult)
	}

	var fsGetMetadataResult FSGetMetadataResult
	if err := json.Unmarshal([]byte(`{
		"createdAtMs":1,
		"isDirectory":false,
		"isFile":true,
		"modifiedAtMs":2
	}`), &fsGetMetadataResult); err != nil {
		t.Fatalf("unmarshal fs metadata result: %v", err)
	}
	if !fsGetMetadataResult.IsFile || fsGetMetadataResult.IsDirectory {
		t.Fatalf("unexpected fs metadata result: %#v", fsGetMetadataResult)
	}

	var fsReadDirectoryResult FSReadDirectoryResult
	if err := json.Unmarshal([]byte(`{
		"entries":[{"fileName":"hello.txt","isDirectory":false,"isFile":true}]
	}`), &fsReadDirectoryResult); err != nil {
		t.Fatalf("unmarshal fs read directory result: %v", err)
	}
	if len(fsReadDirectoryResult.Entries) != 1 || fsReadDirectoryResult.Entries[0].FileName != "hello.txt" {
		t.Fatalf("unexpected fs read directory result: %#v", fsReadDirectoryResult)
	}

	var experimentalFeatureListResult ExperimentalFeatureListResult
	if err := json.Unmarshal([]byte(`{
		"data":[{
			"name":"unified_exec",
			"stage":"beta",
			"displayName":"Unified exec",
			"description":"Improved execution path",
			"announcement":"Try it now",
			"enabled":false,
			"defaultEnabled":false
		}],
		"nextCursor":"cursor_2"
	}`), &experimentalFeatureListResult); err != nil {
		t.Fatalf("unmarshal experimental feature list result: %v", err)
	}
	if len(experimentalFeatureListResult.Data) != 1 || experimentalFeatureListResult.Data[0].Stage != ExperimentalFeatureStageBeta {
		t.Fatalf("unexpected experimental feature list result: %#v", experimentalFeatureListResult)
	}

	var collaborationModeListResult CollaborationModeListResult
	if err := json.Unmarshal([]byte(`{
		"data":[
			{"name":"Plan","mode":"plan","reasoning_effort":"medium"},
			{"name":"Default","mode":"default","model":"gpt-5.4"}
		]
	}`), &collaborationModeListResult); err != nil {
		t.Fatalf("unmarshal collaboration mode list result: %v", err)
	}
	if len(collaborationModeListResult.Data) != 2 || collaborationModeListResult.Data[0].Mode == nil || *collaborationModeListResult.Data[0].Mode != CollaborationModeKindPlan {
		t.Fatalf("unexpected collaboration mode list result: %#v", collaborationModeListResult)
	}

	var feedbackUploadResult FeedbackUploadResult
	if err := json.Unmarshal([]byte(`{"threadId":"thr_feedback"}`), &feedbackUploadResult); err != nil {
		t.Fatalf("unmarshal feedback upload result: %v", err)
	}
	if feedbackUploadResult.ThreadID != "thr_feedback" {
		t.Fatalf("unexpected feedback upload result: %#v", feedbackUploadResult)
	}

	var windowsSandboxResult WindowsSandboxSetupStartResult
	if err := json.Unmarshal([]byte(`{"started":true}`), &windowsSandboxResult); err != nil {
		t.Fatalf("unmarshal windows sandbox result: %v", err)
	}
	if !windowsSandboxResult.Started {
		t.Fatalf("unexpected windows sandbox result: %#v", windowsSandboxResult)
	}

	var toolRequestResult ToolRequestUserInputResult
	if err := json.Unmarshal([]byte(`{"answers":{"q1":{"answers":["opt1"]}}}`), &toolRequestResult); err != nil {
		t.Fatalf("unmarshal tool request user input result: %v", err)
	}
	if len(toolRequestResult.Answers["q1"].Answers) != 1 || toolRequestResult.Answers["q1"].Answers[0] != "opt1" {
		t.Fatalf("unexpected tool request user input result: %#v", toolRequestResult)
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

	var commandExecTerminateResult CommandExecTerminateResult
	if err := json.Unmarshal([]byte(`{}`), &commandExecTerminateResult); err != nil {
		t.Fatalf("unmarshal command exec terminate result: %v", err)
	}

	var skillsListResult SkillsListResult
	if err := json.Unmarshal([]byte(`{
		"data":[{
			"cwd":"/tmp/repo",
			"errors":[],
			"skills":[{
				"name":"build",
				"description":"Build the project",
				"enabled":true,
				"path":"/tmp/repo/.codex/skills/build",
				"scope":"repo"
			}]
		}]
	}`), &skillsListResult); err != nil {
		t.Fatalf("unmarshal skills list result: %v", err)
	}
	if len(skillsListResult.Data) != 1 || len(skillsListResult.Data[0].Skills) != 1 || skillsListResult.Data[0].Skills[0].Name != "build" {
		t.Fatalf("unexpected skills list result: %#v", skillsListResult)
	}

	var skillsConfigWriteResult SkillsConfigWriteResult
	if err := json.Unmarshal([]byte(`{"effectiveEnabled":true}`), &skillsConfigWriteResult); err != nil {
		t.Fatalf("unmarshal skills config write result: %v", err)
	}
	if !skillsConfigWriteResult.EffectiveEnabled {
		t.Fatalf("unexpected skills config write result: %#v", skillsConfigWriteResult)
	}

	var pluginListResult PluginListResult
	if err := json.Unmarshal([]byte(`{
		"marketplaces":[{
			"name":"official",
			"path":"/tmp/marketplace",
			"plugins":[{
				"id":"plugin_1",
				"name":"demo",
				"authPolicy":"ON_USE",
				"installPolicy":"AVAILABLE",
				"installed":false,
				"enabled":false,
				"source":{"type":"local","path":"/tmp/plugin"}
			}]
		}],
		"remoteSyncError":null
	}`), &pluginListResult); err != nil {
		t.Fatalf("unmarshal plugin list result: %v", err)
	}
	if len(pluginListResult.Marketplaces) != 1 || len(pluginListResult.Marketplaces[0].Plugins) != 1 || pluginListResult.Marketplaces[0].Plugins[0].Name != "demo" {
		t.Fatalf("unexpected plugin list result: %#v", pluginListResult)
	}

	var pluginReadResult PluginReadResult
	if err := json.Unmarshal([]byte(`{
		"plugin":{
			"marketplaceName":"official",
			"marketplacePath":"/tmp/marketplace",
			"mcpServers":["demo-server"],
			"apps":[{"id":"app_1","name":"Demo App"}],
			"skills":[{"name":"demo-skill","description":"Demo skill","path":"/tmp/skill"}],
			"summary":{
				"id":"plugin_1",
				"name":"demo",
				"authPolicy":"ON_USE",
				"installPolicy":"AVAILABLE",
				"installed":false,
				"enabled":false,
				"source":{"type":"local","path":"/tmp/plugin"}
			}
		}
	}`), &pluginReadResult); err != nil {
		t.Fatalf("unmarshal plugin read result: %v", err)
	}
	if pluginReadResult.Plugin.Summary.Name != "demo" || pluginReadResult.Plugin.MarketplaceName != "official" {
		t.Fatalf("unexpected plugin read result: %#v", pluginReadResult)
	}

	var appsListResult AppsListResult
	if err := json.Unmarshal([]byte(`{
		"data":[{
			"id":"app_1",
			"name":"Demo App",
			"isAccessible":true,
			"isEnabled":true,
			"pluginDisplayNames":["demo"],
			"branding":{"isDiscoverableApp":true,"category":"productivity"},
			"appMetadata":{
				"categories":["utilities"],
				"review":{"status":"approved"},
				"screenshots":[{"userPrompt":"Show the home screen","url":"https://example.com/shot.png"}]
			}
		}],
		"nextCursor":"cursor_2"
	}`), &appsListResult); err != nil {
		t.Fatalf("unmarshal apps list result: %v", err)
	}
	if len(appsListResult.Data) != 1 || appsListResult.Data[0].ID != "app_1" || appsListResult.NextCursor == nil || *appsListResult.NextCursor != "cursor_2" {
		t.Fatalf("unexpected apps list result: %#v", appsListResult)
	}

	var mcpOAuthLoginResult MCPOAuthLoginResult
	if err := json.Unmarshal([]byte(`{
		"authorizationUrl":"https://example.com/oauth/authorize?state=demo"
	}`), &mcpOAuthLoginResult); err != nil {
		t.Fatalf("unmarshal mcp oauth login result: %v", err)
	}
	if mcpOAuthLoginResult.AuthorizationURL == "" {
		t.Fatalf("unexpected mcp oauth login result: %#v", mcpOAuthLoginResult)
	}

	var mcpServerRefreshResult MCPServerRefreshResult
	if err := json.Unmarshal([]byte(`{}`), &mcpServerRefreshResult); err != nil {
		t.Fatalf("unmarshal mcp server refresh result: %v", err)
	}

	var mcpServerStatusListResult MCPServerStatusListResult
	if err := json.Unmarshal([]byte(`{
		"data":[{
			"name":"demo-server",
			"authStatus":"notLoggedIn",
			"resourceTemplates":[{"name":"repo","uriTemplate":"repo://{id}"}],
			"resources":[{"name":"readme","uri":"file:///README.md","mimeType":"text/markdown"}],
			"tools":{
				"search":{
					"name":"search",
					"inputSchema":{"type":"object"}
				}
			}
		}],
		"nextCursor":"cursor_2"
	}`), &mcpServerStatusListResult); err != nil {
		t.Fatalf("unmarshal mcp server status list result: %v", err)
	}
	if len(mcpServerStatusListResult.Data) != 1 || mcpServerStatusListResult.Data[0].Name != "demo-server" || mcpServerStatusListResult.Data[0].AuthStatus != MCPAuthStatusNotLoggedIn {
		t.Fatalf("unexpected mcp server status result: %#v", mcpServerStatusListResult)
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

func startUninitializedTestConn(t *testing.T) (*jsonrpc2.Conn, func()) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "codex", "app-server")

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

func startExperimentalTestClient(t *testing.T, restartOnFailure bool) (*Client, *InitializeResult) {
	t.Helper()

	client, result, err := StartStdio(context.Background(), StartOptions{
		ClientInfo: ClientInfo{
			Name:    "sdk_test",
			Title:   "SDK Test",
			Version: "0.1.0",
		},
		Capabilities: &Capabilities{
			ExperimentalAPI: true,
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
