package appserver

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	defaultCommand = "codex"
)

var errMissingClientInfo = errors.New("appserver: client info is required")

type ClientInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Capabilities struct {
	ExperimentalAPI          bool     `json:"experimentalApi,omitempty"`
	OptOutNotificationMethod []string `json:"optOutNotificationMethods,omitempty"`
}

type InitializeParams struct {
	ClientInfo   ClientInfo    `json:"clientInfo"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

type InitializeResult struct {
	UserAgent      string `json:"userAgent"`
	PlatformFamily string `json:"platformFamily"`
	PlatformOS     string `json:"platformOs"`
}

type StartOptions struct {
	Command      string
	Args         []string
	Dir          string
	Env          []string
	Stderr       io.Writer
	ClientInfo   ClientInfo
	Capabilities *Capabilities
}

type Client struct {
	cmd   *exec.Cmd
	conn  *jsonrpc2.Conn
	stdio io.ReadWriteCloser

	closeOnce sync.Once
	waitOnce  sync.Once
	waitErr   error
}

func StartStdio(ctx context.Context, opts StartOptions) (*Client, *InitializeResult, error) {
	if opts.ClientInfo.Name == "" || opts.ClientInfo.Title == "" || opts.ClientInfo.Version == "" {
		return nil, nil, errMissingClientInfo
	}

	command := opts.Command
	if command == "" {
		command = defaultCommand
	}

	args := opts.Args
	if len(args) == 0 {
		args = []string{"app-server"}
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	stdio := &processStdio{
		stdin:  stdin,
		stdout: stdout,
	}
	conn := jsonrpc2.NewConn(ctx, jsonrpc2.NewPlainObjectStream(stdio), &noopHandler{})

	client := &Client{
		cmd:   cmd,
		conn:  conn,
		stdio: stdio,
	}

	result := &InitializeResult{}
	if err := conn.Call(ctx, "initialize", InitializeParams{
		ClientInfo:   opts.ClientInfo,
		Capabilities: opts.Capabilities,
	}, result); err != nil {
		client.Close()
		return nil, nil, err
	}

	if err := conn.Notify(ctx, "initialized", struct{}{}); err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, result, nil
}

func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	return c.conn.Call(ctx, method, params, result)
}

func (c *Client) Notify(ctx context.Context, method string, params any) error {
	return c.conn.Notify(ctx, method, params)
}

func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		if c.conn != nil {
			_ = c.conn.Close()
		}
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		if c.stdio != nil {
			_ = c.stdio.Close()
		}
		c.wait()
	})
	return nil
}

func (c *Client) wait() error {
	c.waitOnce.Do(func() {
		if c.cmd != nil {
			c.waitErr = c.cmd.Wait()
		}
	})
	return c.waitErr
}

type processStdio struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser

	closeOnce sync.Once
	closeErr  error
}

func (s *processStdio) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *processStdio) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *processStdio) Close() error {
	s.closeOnce.Do(func() {
		if err := s.stdin.Close(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
			s.closeErr = err
			return
		}
		if err := s.stdout.Close(); err != nil {
			s.closeErr = err
		}
	})
	return s.closeErr
}

type noopHandler struct{}

func (h *noopHandler) Handle(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) {}
