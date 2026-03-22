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
)

var errUnsupportedNotificationEvent = errors.New("appserver: unsupported notification event")

type NotificationEvent interface {
	NotificationMethod() string
}

type ThreadStartedEvent struct {
	Thread json.RawMessage `json:"thread"`
}

func (ThreadStartedEvent) NotificationMethod() string { return MethodThreadStarted }

type ThreadStatusChangedEvent struct {
	ThreadID string          `json:"threadId"`
	Status   json.RawMessage `json:"status"`
}

func (ThreadStatusChangedEvent) NotificationMethod() string { return MethodThreadStatusChanged }

type ThreadArchivedEvent struct {
	ThreadID string          `json:"threadId,omitempty"`
	Thread   json.RawMessage `json:"thread,omitempty"`
}

func (ThreadArchivedEvent) NotificationMethod() string { return MethodThreadArchived }

type ThreadUnarchivedEvent struct {
	Thread json.RawMessage `json:"thread"`
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
	ThreadID string          `json:"threadId"`
	Turn     json.RawMessage `json:"turn"`
}

func (TurnStartedEvent) NotificationMethod() string { return MethodTurnStarted }

type TurnCompletedEvent struct {
	ThreadID string          `json:"threadId"`
	Turn     json.RawMessage `json:"turn"`
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
	ThreadID string          `json:"threadId"`
	TurnID   string          `json:"turnId"`
	Item     json.RawMessage `json:"item"`
}

func (ItemStartedEvent) NotificationMethod() string { return MethodItemStarted }

type ItemCompletedEvent struct {
	ThreadID string          `json:"threadId"`
	TurnID   string          `json:"turnId"`
	Item     json.RawMessage `json:"item"`
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
	SummaryIndex int    `json:"summaryIndex"`
}

func (ReasoningSummaryTextDeltaEvent) NotificationMethod() string {
	return MethodItemReasoningSummaryDelta
}

type ReasoningSummaryPartAddedEvent struct {
	ItemID       string `json:"itemId"`
	SummaryIndex int    `json:"summaryIndex"`
}

func (ReasoningSummaryPartAddedEvent) NotificationMethod() string {
	return MethodItemReasoningPartAdded
}

type ReasoningTextDeltaEvent struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

func (ReasoningTextDeltaEvent) NotificationMethod() string { return MethodItemReasoningTextDelta }

type CommandExecutionOutputDeltaEvent struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
	Stream string `json:"stream,omitempty"`
}

func (CommandExecutionOutputDeltaEvent) NotificationMethod() string {
	return MethodItemCommandOutputDelta
}

type FileChangeOutputDeltaEvent struct {
	ItemID string          `json:"itemId"`
	Delta  json.RawMessage `json:"delta"`
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
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedNotificationEvent, notification.Method)
	}

	if err := notification.DecodeParams(event); err != nil {
		return nil, err
	}

	return event, nil
}
