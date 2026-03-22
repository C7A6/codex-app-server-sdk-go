package appserver

import (
	"context"
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
