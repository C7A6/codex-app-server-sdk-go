package appserver

import (
	"errors"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

type RPCError struct {
	Code    int64
	Message string
	Data    any
}

func (e *RPCError) Error() string {
	if e == nil {
		return "appserver: rpc error"
	}
	return e.Message
}

func IsValidationError(err error) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}

	return rpcErr.Code == int64(jsonrpc2.CodeInvalidParams) || rpcErr.Code == int64(jsonrpc2.CodeInvalidRequest)
}

func IsNotInitializedError(err error) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}

	return strings.Contains(strings.ToLower(rpcErr.Message), "not initialized")
}

func IsRateLimitError(err error) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}

	message := strings.ToLower(rpcErr.Message)
	return strings.Contains(message, "rate limit") || strings.Contains(message, "usage limit") || strings.Contains(message, "retry later")
}

func wrapRPCError(err error) error {
	if err == nil {
		return nil
	}

	var rpcErr *jsonrpc2.Error
	if !errors.As(err, &rpcErr) {
		return err
	}

	return &RPCError{
		Code:    int64(rpcErr.Code),
		Message: rpcErr.Message,
		Data:    rpcErr.Data,
	}
}
