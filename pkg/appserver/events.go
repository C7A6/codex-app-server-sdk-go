package appserver

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	MethodThreadStarted             = "thread/started"
	MethodThreadStatusChanged       = "thread/status/changed"
	MethodThreadArchived            = "thread/archived"
	MethodThreadUnarchived          = "thread/unarchived"
	MethodThreadClosed              = "thread/closed"
	MethodThreadNameUpdated         = "thread/name/updated"
	MethodThreadTokenUsageUpdated   = "thread/tokenUsage/updated"
	MethodTurnStarted               = "turn/started"
	MethodTurnCompleted             = "turn/completed"
	MethodTurnDiffUpdated           = "turn/diff/updated"
	MethodTurnPlanUpdated           = "turn/plan/updated"
	MethodItemStarted               = "item/started"
	MethodItemCompleted             = "item/completed"
	MethodItemAgentMessageDelta     = "item/agentMessage/delta"
	MethodItemPlanDelta             = "item/plan/delta"
	MethodItemReasoningSummaryDelta = "item/reasoning/summaryTextDelta"
	MethodItemReasoningPartAdded    = "item/reasoning/summaryPartAdded"
	MethodItemReasoningTextDelta    = "item/reasoning/textDelta"
	MethodItemCommandOutputDelta    = "item/commandExecution/outputDelta"
	MethodItemFileChangeOutputDelta = "item/fileChange/outputDelta"
	MethodAccountLoginCompleted     = "account/login/completed"
	MethodAccountUpdated            = "account/updated"
	MethodAccountRateLimitsUpdated  = "account/rateLimits/updated"
	MethodMCPOAuthLoginCompleted    = "mcpServer/oauthLogin/completed"
	MethodWindowsSandboxCompleted   = "windowsSandbox/setupCompleted"
	MethodFuzzyFileSearchUpdated    = "fuzzyFileSearch/sessionUpdated"
	MethodFuzzyFileSearchCompleted  = "fuzzyFileSearch/sessionCompleted"
)

var errUnsupportedNotificationEvent = errors.New("appserver: unsupported notification event")

type NotificationEvent interface {
	NotificationMethod() string
}

type ThreadStartedEvent struct {
	Thread Thread `json:"thread"`
}

func (ThreadStartedEvent) NotificationMethod() string { return MethodThreadStarted }

type ThreadStatusChangedEvent struct {
	ThreadID string       `json:"threadId"`
	Status   ThreadStatus `json:"status"`
}

func (ThreadStatusChangedEvent) NotificationMethod() string { return MethodThreadStatusChanged }

type ThreadArchivedEvent struct {
	ThreadID string  `json:"threadId,omitempty"`
	Thread   *Thread `json:"thread,omitempty"`
}

func (ThreadArchivedEvent) NotificationMethod() string { return MethodThreadArchived }

type ThreadUnarchivedEvent struct {
	Thread Thread `json:"thread"`
}

func (ThreadUnarchivedEvent) NotificationMethod() string { return MethodThreadUnarchived }

type ThreadClosedEvent struct {
	ThreadID string `json:"threadId"`
}

func (ThreadClosedEvent) NotificationMethod() string { return MethodThreadClosed }

type ThreadNameUpdatedEvent struct {
	ThreadID string `json:"threadId"`
	Name     string `json:"name"`
}

func (ThreadNameUpdatedEvent) NotificationMethod() string { return MethodThreadNameUpdated }

type ThreadTokenUsageUpdatedEvent struct {
	ThreadID   string          `json:"threadId"`
	TokenUsage json.RawMessage `json:"tokenUsage"`
}

func (ThreadTokenUsageUpdatedEvent) NotificationMethod() string { return MethodThreadTokenUsageUpdated }

type TurnStartedEvent struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

func (TurnStartedEvent) NotificationMethod() string { return MethodTurnStarted }

type TurnCompletedEvent struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

func (TurnCompletedEvent) NotificationMethod() string { return MethodTurnCompleted }

type TurnDiffUpdatedEvent struct {
	ThreadID string          `json:"threadId"`
	TurnID   string          `json:"turnId"`
	Diff     json.RawMessage `json:"diff"`
}

func (TurnDiffUpdatedEvent) NotificationMethod() string { return MethodTurnDiffUpdated }

type TurnPlanUpdatedEvent struct {
	TurnID      string          `json:"turnId"`
	Explanation *string         `json:"explanation,omitempty"`
	Plan        json.RawMessage `json:"plan"`
}

func (TurnPlanUpdatedEvent) NotificationMethod() string { return MethodTurnPlanUpdated }

type ItemStartedEvent struct {
	ThreadID string     `json:"threadId"`
	TurnID   string     `json:"turnId"`
	Item     ThreadItem `json:"item"`
}

func (ItemStartedEvent) NotificationMethod() string { return MethodItemStarted }

type ItemCompletedEvent struct {
	ThreadID string     `json:"threadId"`
	TurnID   string     `json:"turnId"`
	Item     ThreadItem `json:"item"`
}

func (ItemCompletedEvent) NotificationMethod() string { return MethodItemCompleted }

type AgentMessageDeltaEvent struct {
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
	TurnID   string `json:"turnId,omitempty"`
	ThreadID string `json:"threadId,omitempty"`
}

func (AgentMessageDeltaEvent) NotificationMethod() string { return MethodItemAgentMessageDelta }

type PlanDeltaEvent struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
	TurnID string `json:"turnId,omitempty"`
}

func (PlanDeltaEvent) NotificationMethod() string { return MethodItemPlanDelta }

type ReasoningSummaryTextDeltaEvent struct {
	ItemID       string `json:"itemId"`
	Delta        string `json:"delta"`
	SummaryIndex int64  `json:"summaryIndex"`
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
}

func (ReasoningSummaryTextDeltaEvent) NotificationMethod() string {
	return MethodItemReasoningSummaryDelta
}

type ReasoningSummaryPartAddedEvent struct {
	ItemID       string `json:"itemId"`
	SummaryIndex int64  `json:"summaryIndex"`
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
}

func (ReasoningSummaryPartAddedEvent) NotificationMethod() string {
	return MethodItemReasoningPartAdded
}

type ReasoningTextDeltaEvent struct {
	ContentIndex int64  `json:"contentIndex"`
	ItemID       string `json:"itemId"`
	Delta        string `json:"delta"`
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
}

func (ReasoningTextDeltaEvent) NotificationMethod() string { return MethodItemReasoningTextDelta }

type CommandExecutionOutputDeltaEvent struct {
	ItemID   string `json:"itemId,omitempty"`
	Delta    string `json:"delta,omitempty"`
	Stream   string `json:"stream,omitempty"`
	ThreadID string `json:"threadId,omitempty"`
	TurnID   string `json:"turnId,omitempty"`
}

func (CommandExecutionOutputDeltaEvent) NotificationMethod() string {
	return MethodItemCommandOutputDelta
}

type FileChangeOutputDeltaEvent struct {
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

func (FileChangeOutputDeltaEvent) NotificationMethod() string { return MethodItemFileChangeOutputDelta }

type AccountLoginCompletedEvent struct {
	LoginID *string `json:"loginId"`
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

func (AccountLoginCompletedEvent) NotificationMethod() string { return MethodAccountLoginCompleted }

type AccountUpdatedEvent struct {
	AuthMode *string `json:"authMode"`
	PlanType *string `json:"planType"`
}

func (AccountUpdatedEvent) NotificationMethod() string { return MethodAccountUpdated }

type AccountRateLimitsUpdatedEvent struct {
	RateLimits json.RawMessage `json:"rateLimits"`
}

func (AccountRateLimitsUpdatedEvent) NotificationMethod() string {
	return MethodAccountRateLimitsUpdated
}

type MCPOAuthLoginCompletedEvent struct {
	Name    string  `json:"name"`
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

func (MCPOAuthLoginCompletedEvent) NotificationMethod() string { return MethodMCPOAuthLoginCompleted }

type WindowsSandboxSetupCompletedEvent struct {
	Mode    WindowsSandboxSetupMode `json:"mode"`
	Success bool                    `json:"success"`
	Error   *string                 `json:"error"`
}

func (WindowsSandboxSetupCompletedEvent) NotificationMethod() string {
	return MethodWindowsSandboxCompleted
}

type FuzzyFileSearchResult struct {
	FileName string   `json:"file_name"`
	Indices  []uint32 `json:"indices,omitempty"`
	Path     string   `json:"path"`
	Root     string   `json:"root"`
	Score    uint32   `json:"score"`
}

type FuzzyFileSearchSessionUpdatedEvent struct {
	Files     []FuzzyFileSearchResult `json:"files"`
	Query     string                  `json:"query"`
	SessionID string                  `json:"sessionId"`
}

func (FuzzyFileSearchSessionUpdatedEvent) NotificationMethod() string {
	return MethodFuzzyFileSearchUpdated
}

type FuzzyFileSearchSessionCompletedEvent struct {
	SessionID string `json:"sessionId"`
}

func (FuzzyFileSearchSessionCompletedEvent) NotificationMethod() string {
	return MethodFuzzyFileSearchCompleted
}

func (n Notification) DecodeEvent() (NotificationEvent, error) {
	return DecodeNotificationEvent(n)
}

func DecodeNotificationEvent(notification Notification) (NotificationEvent, error) {
	var event NotificationEvent

	switch notification.Method {
	case MethodThreadStarted:
		event = &ThreadStartedEvent{}
	case MethodThreadStatusChanged:
		event = &ThreadStatusChangedEvent{}
	case MethodThreadArchived:
		event = &ThreadArchivedEvent{}
	case MethodThreadUnarchived:
		event = &ThreadUnarchivedEvent{}
	case MethodThreadClosed:
		event = &ThreadClosedEvent{}
	case MethodThreadNameUpdated:
		event = &ThreadNameUpdatedEvent{}
	case MethodThreadTokenUsageUpdated:
		event = &ThreadTokenUsageUpdatedEvent{}
	case MethodTurnStarted:
		event = &TurnStartedEvent{}
	case MethodTurnCompleted:
		event = &TurnCompletedEvent{}
	case MethodTurnDiffUpdated:
		event = &TurnDiffUpdatedEvent{}
	case MethodTurnPlanUpdated:
		event = &TurnPlanUpdatedEvent{}
	case MethodItemStarted:
		event = &ItemStartedEvent{}
	case MethodItemCompleted:
		event = &ItemCompletedEvent{}
	case MethodItemAgentMessageDelta:
		event = &AgentMessageDeltaEvent{}
	case MethodItemPlanDelta:
		event = &PlanDeltaEvent{}
	case MethodItemReasoningSummaryDelta:
		event = &ReasoningSummaryTextDeltaEvent{}
	case MethodItemReasoningPartAdded:
		event = &ReasoningSummaryPartAddedEvent{}
	case MethodItemReasoningTextDelta:
		event = &ReasoningTextDeltaEvent{}
	case MethodItemCommandOutputDelta:
		event = &CommandExecutionOutputDeltaEvent{}
	case MethodItemFileChangeOutputDelta:
		event = &FileChangeOutputDeltaEvent{}
	case MethodAccountLoginCompleted:
		event = &AccountLoginCompletedEvent{}
	case MethodAccountUpdated:
		event = &AccountUpdatedEvent{}
	case MethodAccountRateLimitsUpdated:
		event = &AccountRateLimitsUpdatedEvent{}
	case MethodMCPOAuthLoginCompleted:
		event = &MCPOAuthLoginCompletedEvent{}
	case MethodWindowsSandboxCompleted:
		event = &WindowsSandboxSetupCompletedEvent{}
	case MethodFuzzyFileSearchUpdated:
		event = &FuzzyFileSearchSessionUpdatedEvent{}
	case MethodFuzzyFileSearchCompleted:
		event = &FuzzyFileSearchSessionCompletedEvent{}
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedNotificationEvent, notification.Method)
	}

	if err := notification.DecodeParams(event); err != nil {
		return nil, err
	}

	return event, nil
}
