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
		if len(event.Thread) == 0 {
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
