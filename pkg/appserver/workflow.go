package appserver

import (
	"context"
	"sync"
)

func (c *Client) SendMessageAndWait(ctx context.Context, threadID string, text string) (*TurnCompletedEvent, error) {
	completedCh := make(chan *TurnCompletedEvent, 1)
	unregister, err := c.OnTurnCompleted(func(handlerCtx context.Context, event *TurnCompletedEvent) {
		if event.ThreadID != threadID {
			return
		}

		select {
		case completedCh <- event:
		default:
		}
	})
	if err != nil {
		return nil, err
	}
	defer unregister()

	if _, err := c.StartTurn(ctx, TurnStartParams{
		ThreadID: threadID,
		Input: []TurnStartInputItem{
			{
				"type": "text",
				"text": text,
			},
		},
	}); err != nil {
		return nil, err
	}

	select {
	case event := <-completedCh:
		return event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) StreamTurn(ctx context.Context, params TurnStartParams) (<-chan NotificationEvent, *TurnStartResult, error) {
	events := make(chan NotificationEvent, 32)

	var (
		mu     sync.RWMutex
		turnID string
	)

	sendEvent := func(event NotificationEvent) {
		if !eventMatchesTurn(event, params.ThreadID, func() string {
			mu.RLock()
			defer mu.RUnlock()
			return turnID
		}()) {
			return
		}

		select {
		case events <- event:
		case <-ctx.Done():
		}
	}

	methods := []string{
		MethodTurnStarted,
		MethodTurnCompleted,
		MethodTurnDiffUpdated,
		MethodTurnPlanUpdated,
		MethodItemStarted,
		MethodItemCompleted,
		MethodItemAgentMessageDelta,
		MethodItemPlanDelta,
		MethodItemReasoningSummaryDelta,
		MethodItemReasoningPartAdded,
		MethodItemReasoningTextDelta,
		MethodItemCommandOutputDelta,
		MethodItemFileChangeOutputDelta,
	}

	unregisters := make([]func(), 0, len(methods))
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			for _, unregister := range unregisters {
				if unregister != nil {
					unregister()
				}
			}
			close(events)
		})
	}

	for _, method := range methods {
		unregister, err := c.RegisterNotificationHandler(method, func(handlerCtx context.Context, notification Notification) {
			event, decodeErr := notification.DecodeEvent()
			if decodeErr != nil {
				return
			}

			sendEvent(event)

			if _, ok := event.(*TurnCompletedEvent); ok && eventMatchesTurn(event, params.ThreadID, func() string {
				mu.RLock()
				defer mu.RUnlock()
				return turnID
			}()) {
				cleanup()
			}
		})
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		unregisters = append(unregisters, unregister)
	}

	result, err := c.StartTurn(ctx, params)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	mu.Lock()
	turnID = result.Turn.ID
	mu.Unlock()

	go func() {
		<-ctx.Done()
		cleanup()
	}()

	return events, result, nil
}

func (c *Client) QuickThread(ctx context.Context, threadParams ThreadStartParams, text string) (*ThreadStartResult, *TurnCompletedEvent, error) {
	thread, err := c.StartThread(ctx, threadParams)
	if err != nil {
		return nil, nil, err
	}

	completed, err := c.SendMessageAndWait(ctx, thread.Thread.ID, text)
	if err != nil {
		return thread, nil, err
	}

	return thread, completed, nil
}

func eventMatchesTurn(event NotificationEvent, threadID string, turnID string) bool {
	eventThreadID, eventTurnID := eventIdentity(event)

	if threadID != "" && eventThreadID != "" && eventThreadID != threadID {
		return false
	}
	if turnID != "" && eventTurnID != "" && eventTurnID != turnID {
		return false
	}
	if turnID != "" && eventTurnID == turnID {
		return true
	}
	if threadID != "" && eventThreadID == threadID {
		return true
	}
	return threadID == "" && turnID == ""
}

func eventIdentity(event NotificationEvent) (threadID string, turnID string) {
	switch typed := event.(type) {
	case *TurnStartedEvent:
		return typed.ThreadID, typed.Turn.ID
	case *TurnCompletedEvent:
		return typed.ThreadID, typed.Turn.ID
	case *TurnDiffUpdatedEvent:
		return typed.ThreadID, typed.TurnID
	case *TurnPlanUpdatedEvent:
		return "", typed.TurnID
	case *ItemStartedEvent:
		return typed.ThreadID, typed.TurnID
	case *ItemCompletedEvent:
		return typed.ThreadID, typed.TurnID
	case *AgentMessageDeltaEvent:
		return typed.ThreadID, typed.TurnID
	case *PlanDeltaEvent:
		return "", typed.TurnID
	case *ReasoningSummaryTextDeltaEvent:
		return typed.ThreadID, typed.TurnID
	case *ReasoningSummaryPartAddedEvent:
		return typed.ThreadID, typed.TurnID
	case *ReasoningTextDeltaEvent:
		return typed.ThreadID, typed.TurnID
	case *CommandExecutionOutputDeltaEvent:
		return typed.ThreadID, typed.TurnID
	case *FileChangeOutputDeltaEvent:
		return typed.ThreadID, typed.TurnID
	default:
		return "", ""
	}
}
