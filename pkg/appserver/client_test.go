package appserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStartStdioInitializesConnection(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(t.TempDir(), "fake-codex.sh")
	script := `#!/usr/bin/env bash
set -euo pipefail

read -r initialize_line
printf '%s' "$initialize_line" | grep -q '"method":"initialize"'
printf '%s' "$initialize_line" | grep -q '"name":"test-client"'
id=$(printf '%s' "$initialize_line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
printf '{"id":%s,"result":{"userAgent":"codex-cli/0.116.0","platformFamily":"unix","platformOs":"linux"}}\n' "$id"

read -r initialized_line
printf '%s' "$initialized_line" | grep -q '"method":"initialized"'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	client, result, err := StartStdio(context.Background(), StartOptions{
		Command: scriptPath,
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Title:   "Test Client",
			Version: "0.1.0",
		},
	})
	if err != nil {
		t.Fatalf("StartStdio returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	if result.UserAgent != "codex-cli/0.116.0" {
		t.Fatalf("unexpected user agent: %q", result.UserAgent)
	}
	if result.PlatformFamily != "unix" {
		t.Fatalf("unexpected platform family: %q", result.PlatformFamily)
	}
	if result.PlatformOS != "linux" {
		t.Fatalf("unexpected platform os: %q", result.PlatformOS)
	}
}

func TestStartStdioRequiresClientInfo(t *testing.T) {
	t.Parallel()

	_, _, err := StartStdio(context.Background(), StartOptions{})
	if err == nil {
		t.Fatal("expected missing client info error")
	}
}
