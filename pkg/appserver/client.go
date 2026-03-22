package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

const defaultCommand = "codex"

var (
	ErrClientClosed      = errors.New("appserver: client is closed")
	errNilHandler        = errors.New("appserver: notification handler is nil")
	errMissingClientInfo = errors.New("appserver: client info is required")
)

type ProcessExitError struct {
	Err error
}

func (e *ProcessExitError) Error() string {
	if e == nil || e.Err == nil {
		return "appserver: process exited unexpectedly"
	}
	return fmt.Sprintf("appserver: process exited unexpectedly: %v", e.Err)
}

func (e *ProcessExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ClientInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Capabilities struct {
	ExperimentalAPI           bool     `json:"experimentalApi,omitempty"`
	OptOutNotificationMethods []string `json:"optOutNotificationMethods,omitempty"`
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

type Account struct {
	Type     string `json:"type"`
	Email    string `json:"email,omitempty"`
	PlanType string `json:"planType,omitempty"`
}

type AccountReadParams struct {
	RefreshToken bool `json:"refreshToken"`
}

type AccountReadResult struct {
	Account            *Account `json:"account"`
	RequiresOpenAIAuth bool     `json:"requiresOpenaiAuth"`
}

type RateLimitWindow struct {
	UsedPercent        int   `json:"usedPercent"`
	WindowDurationMins int   `json:"windowDurationMins"`
	ResetsAt           int64 `json:"resetsAt"`
}

type RateLimitBucket struct {
	LimitID   string           `json:"limitId"`
	LimitName *string          `json:"limitName"`
	Primary   *RateLimitWindow `json:"primary"`
	Secondary *RateLimitWindow `json:"secondary"`
}

type RateLimitsReadResult struct {
	RateLimits          *RateLimitBucket           `json:"rateLimits"`
	RateLimitsByLimitID map[string]RateLimitBucket `json:"rateLimitsByLimitId,omitempty"`
}

type StartOptions struct {
	Command          string
	Args             []string
	Dir              string
	Env              []string
	Stderr           io.Writer
	ClientInfo       ClientInfo
	Capabilities     *Capabilities
	RestartOnFailure bool
}

type Notification struct {
	Method string
	Params json.RawMessage
}

func (n Notification) DecodeParams(v any) error {
	if len(n.Params) == 0 {
		return nil
	}
	return json.Unmarshal(n.Params, v)
}

type NotificationHandler func(context.Context, Notification)

type Client struct {
	mu                   sync.Mutex
	opts                 StartOptions
	session              *session
	closed               bool
	nextHandlerID        uint64
	notificationHandlers map[string]map[uint64]NotificationHandler
}

func StartStdio(ctx context.Context, opts StartOptions) (*Client, *InitializeResult, error) {
	if opts.ClientInfo.Name == "" || opts.ClientInfo.Title == "" || opts.ClientInfo.Version == "" {
		return nil, nil, errMissingClientInfo
	}

	client := &Client{
		opts:                 opts,
		notificationHandlers: make(map[string]map[uint64]NotificationHandler),
	}

	sess, result, err := startSession(ctx, client, opts)
	if err != nil {
		return nil, nil, err
	}
	client.session = sess

	return client, result, nil
}

func (c *Client) RegisterNotificationHandler(method string, handler NotificationHandler) (func(), error) {
	if handler == nil {
		return nil, errNilHandler
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	c.nextHandlerID++
	handlerID := c.nextHandlerID
	if c.notificationHandlers[method] == nil {
		c.notificationHandlers[method] = make(map[uint64]NotificationHandler)
	}
	c.notificationHandlers[method][handlerID] = handler

	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		handlers := c.notificationHandlers[method]
		if handlers == nil {
			return
		}
		delete(handlers, handlerID)
		if len(handlers) == 0 {
			delete(c.notificationHandlers, method)
		}
	}, nil
}

func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	return c.call(ctx, func(sess *session) error {
		return sess.conn.Call(ctx, method, params, result)
	})
}

func (c *Client) Notify(ctx context.Context, method string, params any) error {
	return c.call(ctx, func(sess *session) error {
		return sess.conn.Notify(ctx, method, params)
	})
}

func (c *Client) ReadAccount(ctx context.Context, params AccountReadParams) (*AccountReadResult, error) {
	result := &AccountReadResult{}
	if err := c.Call(ctx, "account/read", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ReadRateLimits(ctx context.Context) (*RateLimitsReadResult, error) {
	result := &RateLimitsReadResult{}
	if err := c.Call(ctx, "account/rateLimits/read", nil, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListModels(ctx context.Context, params ModelListParams) (*ModelListResult, error) {
	result := &ModelListResult{}
	if err := c.Call(ctx, "model/list", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) StartThread(ctx context.Context, params ThreadStartParams) (*ThreadStartResult, error) {
	result := &ThreadStartResult{}
	if err := c.Call(ctx, "thread/start", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ResumeThread(ctx context.Context, params ThreadResumeParams) (*ThreadResumeResult, error) {
	result := &ThreadResumeResult{}
	if err := c.Call(ctx, "thread/resume", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ForkThread(ctx context.Context, params ThreadForkParams) (*ThreadForkResult, error) {
	result := &ThreadForkResult{}
	if err := c.Call(ctx, "thread/fork", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ReadThread(ctx context.Context, params ThreadReadParams) (*ThreadReadResult, error) {
	result := &ThreadReadResult{}
	if err := c.Call(ctx, "thread/read", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListThreads(ctx context.Context, params ThreadListParams) (*ThreadListResult, error) {
	result := &ThreadListResult{}
	if err := c.Call(ctx, "thread/list", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListLoadedThreads(ctx context.Context, params ThreadLoadedListParams) (*ThreadLoadedListResult, error) {
	result := &ThreadLoadedListResult{}
	if err := c.Call(ctx, "thread/loaded/list", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SetThreadName(ctx context.Context, params ThreadSetNameParams) (*ThreadSetNameResult, error) {
	result := &ThreadSetNameResult{}
	if err := c.Call(ctx, "thread/name/set", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ArchiveThread(ctx context.Context, params ThreadArchiveParams) (*ThreadArchiveResult, error) {
	result := &ThreadArchiveResult{}
	if err := c.Call(ctx, "thread/archive", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UnarchiveThread(ctx context.Context, params ThreadUnarchiveParams) (*ThreadUnarchiveResult, error) {
	result := &ThreadUnarchiveResult{}
	if err := c.Call(ctx, "thread/unarchive", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UnsubscribeThread(ctx context.Context, params ThreadUnsubscribeParams) (*ThreadUnsubscribeResult, error) {
	result := &ThreadUnsubscribeResult{}
	if err := c.Call(ctx, "thread/unsubscribe", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CompactThread(ctx context.Context, params ThreadCompactStartParams) (*ThreadCompactStartResult, error) {
	result := &ThreadCompactStartResult{}
	if err := c.Call(ctx, "thread/compact/start", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) RollbackThread(ctx context.Context, params ThreadRollbackParams) (*ThreadRollbackResult, error) {
	result := &ThreadRollbackResult{}
	if err := c.Call(ctx, "thread/rollback", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) StartTurn(ctx context.Context, params TurnStartParams) (*TurnStartResult, error) {
	result := &TurnStartResult{}
	if err := c.Call(ctx, "turn/start", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SteerTurn(ctx context.Context, params TurnSteerParams) (*TurnSteerResult, error) {
	result := &TurnSteerResult{}
	if err := c.Call(ctx, "turn/steer", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) InterruptTurn(ctx context.Context, params TurnInterruptParams) (*TurnInterruptResult, error) {
	result := &TurnInterruptResult{}
	if err := c.Call(ctx, "turn/interrupt", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) StartReview(ctx context.Context, params ReviewStartParams) (*ReviewStartResult, error) {
	result := &ReviewStartResult{}
	if err := c.Call(ctx, "review/start", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ExecCommand(ctx context.Context, params CommandExecParams) (*CommandExecResult, error) {
	result := &CommandExecResult{}
	if err := c.Call(ctx, "command/exec", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) WriteCommandStdin(ctx context.Context, params CommandExecWriteParams) (*CommandExecWriteResult, error) {
	result := &CommandExecWriteResult{}
	if err := c.Call(ctx, "command/exec/write", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ResizeCommandPTY(ctx context.Context, params CommandExecResizeParams) (*CommandExecResizeResult, error) {
	result := &CommandExecResizeResult{}
	if err := c.Call(ctx, "command/exec/resize", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) TerminateCommand(ctx context.Context, params CommandExecTerminateParams) (*CommandExecTerminateResult, error) {
	result := &CommandExecTerminateResult{}
	if err := c.Call(ctx, "command/exec/terminate", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListSkills(ctx context.Context, params SkillsListParams) (*SkillsListResult, error) {
	result := &SkillsListResult{}
	if err := c.Call(ctx, "skills/list", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) WriteSkillsConfig(ctx context.Context, params SkillsConfigWriteParams) (*SkillsConfigWriteResult, error) {
	result := &SkillsConfigWriteResult{}
	if err := c.Call(ctx, "skills/config/write", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListPlugins(ctx context.Context, params PluginListParams) (*PluginListResult, error) {
	result := &PluginListResult{}
	if err := c.Call(ctx, "plugin/list", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ReadPlugin(ctx context.Context, params PluginReadParams) (*PluginReadResult, error) {
	result := &PluginReadResult{}
	if err := c.Call(ctx, "plugin/read", params, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	sess := c.session
	c.session = nil
	c.mu.Unlock()

	if sess != nil {
		return sess.close()
	}
	return nil
}

func (c *Client) call(ctx context.Context, invoke func(*session) error) error {
	for attempt := 0; attempt < 2; attempt++ {
		sess, err := c.activeSession(ctx)
		if err != nil {
			return err
		}

		err = invoke(sess)
		if err == nil {
			return nil
		}

		if !errors.Is(err, jsonrpc2.ErrClosed) {
			if sess.done() {
				return sess.processExitError()
			}
			return err
		}

		if !c.opts.RestartOnFailure {
			if sess.done() {
				return sess.processExitError()
			}
			return err
		}

		c.invalidateSession(sess)
	}

	sess, err := c.activeSession(ctx)
	if err != nil {
		return err
	}
	return sess.processExitError()
}

func (c *Client) activeSession(ctx context.Context) (*session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if c.session != nil && !c.session.done() {
		return c.session, nil
	}

	if c.session != nil && !c.opts.RestartOnFailure {
		return nil, c.session.processExitError()
	}

	sess, _, err := startSession(ctx, c, c.opts)
	if err != nil {
		return nil, err
	}
	c.session = sess
	return sess, nil
}

func (c *Client) invalidateSession(target *session) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == target {
		c.session = nil
	}
}

func startSession(ctx context.Context, client *Client, opts StartOptions) (*session, *InitializeResult, error) {
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
	conn := jsonrpc2.NewConn(ctx, jsonrpc2.NewPlainObjectStream(stdio), &clientHandler{client: client})

	sess := newSession(cmd, conn, stdio)

	result := &InitializeResult{}
	if err := conn.Call(ctx, "initialize", InitializeParams{
		ClientInfo:   opts.ClientInfo,
		Capabilities: opts.Capabilities,
	}, result); err != nil {
		_ = sess.close()
		if sess.done() {
			return nil, nil, sess.processExitError()
		}
		return nil, nil, err
	}

	if err := conn.Notify(ctx, "initialized", struct{}{}); err != nil {
		_ = sess.close()
		if sess.done() {
			return nil, nil, sess.processExitError()
		}
		return nil, nil, err
	}

	return sess, result, nil
}

type session struct {
	cmd   *exec.Cmd
	conn  *jsonrpc2.Conn
	stdio io.ReadWriteCloser

	doneCh    chan struct{}
	closeOnce sync.Once
	waitOnce  sync.Once

	mu      sync.RWMutex
	exitErr error
}

func newSession(cmd *exec.Cmd, conn *jsonrpc2.Conn, stdio io.ReadWriteCloser) *session {
	sess := &session{
		cmd:    cmd,
		conn:   conn,
		stdio:  stdio,
		doneCh: make(chan struct{}),
	}

	go func() {
		sess.setExit(cmd.Wait())
	}()

	return sess
}

func (s *session) done() bool {
	select {
	case <-s.doneCh:
		return true
	default:
		return false
	}
}

func (s *session) processExitError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &ProcessExitError{Err: s.exitErr}
}

func (s *session) setExit(err error) {
	s.waitOnce.Do(func() {
		s.mu.Lock()
		s.exitErr = err
		s.mu.Unlock()
		close(s.doneCh)
	})
}

func (s *session) close() error {
	s.closeOnce.Do(func() {
		if s.conn != nil {
			_ = s.conn.Close()
		}
		if s.stdio != nil {
			_ = s.stdio.Close()
		}
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		<-s.doneCh
	})

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exitErr
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

type clientHandler struct {
	client *Client
}

func (h *clientHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Notif {
		h.client.dispatchNotification(ctx, req)
		return
	}

	_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: "appserver: server request method is not supported yet",
	})
}

func (c *Client) dispatchNotification(ctx context.Context, req *jsonrpc2.Request) {
	notification := Notification{
		Method: req.Method,
	}
	if req.Params != nil {
		notification.Params = append(json.RawMessage(nil), (*req.Params)...)
	}

	handlers := c.handlersForMethod(req.Method)
	if len(handlers) == 0 {
		return
	}

	go func() {
		for _, handler := range handlers {
			handler(ctx, notification)
		}
	}()
}

func (c *Client) handlersForMethod(method string) []NotificationHandler {
	c.mu.Lock()
	defer c.mu.Unlock()

	registered := c.notificationHandlers[method]
	if len(registered) == 0 {
		return nil
	}

	handlers := make([]NotificationHandler, 0, len(registered))
	for _, handler := range registered {
		handlers = append(handlers, handler)
	}
	return handlers
}
