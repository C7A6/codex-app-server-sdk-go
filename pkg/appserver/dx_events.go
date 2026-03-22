package appserver

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

var errInvalidEventType = errors.New("appserver: invalid event type")

type ItemDeltaEvent interface {
	NotificationEvent
	itemDeltaEvent()
}

func OnEvent[T NotificationEvent](client *Client, handler func(context.Context, T)) (func(), error) {
	if handler == nil {
		return nil, errNilHandler
	}

	method, err := notificationMethodForType[T]()
	if err != nil {
		return nil, err
	}

	return client.RegisterNotificationHandler(method, func(ctx context.Context, notification Notification) {
		event, decodeErr := notification.DecodeEvent()
		if decodeErr != nil {
			return
		}

		typed, ok := event.(T)
		if !ok {
			return
		}
		handler(ctx, typed)
	})
}

func (c *Client) OnTurnCompleted(handler func(context.Context, *TurnCompletedEvent)) (func(), error) {
	return OnEvent(c, handler)
}

func (c *Client) OnItemDelta(handler func(context.Context, ItemDeltaEvent)) (func(), error) {
	if handler == nil {
		return nil, errNilHandler
	}

	methods := []string{
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
	unregisterAll := func() {
		once.Do(func() {
			for _, unregister := range unregisters {
				if unregister != nil {
					unregister()
				}
			}
		})
	}

	for _, method := range methods {
		unregister, err := c.RegisterNotificationHandler(method, func(ctx context.Context, notification Notification) {
			event, decodeErr := notification.DecodeEvent()
			if decodeErr != nil {
				return
			}

			delta, ok := event.(ItemDeltaEvent)
			if !ok {
				return
			}
			handler(ctx, delta)
		})
		if err != nil {
			unregisterAll()
			return nil, err
		}
		unregisters = append(unregisters, unregister)
	}

	return unregisterAll, nil
}

func notificationMethodForType[T NotificationEvent]() (string, error) {
	typeOfT := reflect.TypeOf((*T)(nil)).Elem()
	if typeOfT == nil {
		return "", errInvalidEventType
	}

	value, err := zeroEventValue(typeOfT)
	if err != nil {
		return "", err
	}

	event, ok := value.(NotificationEvent)
	if !ok {
		return "", errInvalidEventType
	}
	return event.NotificationMethod(), nil
}

func zeroEventValue(t reflect.Type) (any, error) {
	switch t.Kind() {
	case reflect.Pointer:
		if t.Elem().Kind() != reflect.Struct {
			return nil, errInvalidEventType
		}
		return reflect.New(t.Elem()).Interface(), nil
	case reflect.Struct:
		return reflect.New(t).Elem().Interface(), nil
	default:
		return nil, errInvalidEventType
	}
}
