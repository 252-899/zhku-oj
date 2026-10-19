package api

import "errors"

var (
	ErrAssistantNotRegistered = errors.New("assistant not registered")
	ErrAssistantNotConfigured = errors.New("assistant not configured")
	ErrModuleNotRegistered    = errors.New("module not registered")
	ErrInvalidRuntimeRequest  = errors.New("invalid runtime request")
	ErrTaskAlreadyRunning     = errors.New("root task already running")
	ErrTaskNotFound           = errors.New("task not found")
	ErrChildTaskNotFound      = errors.New("child task not found")
	ErrStopNotImplemented     = errors.New("stop not implemented")
)
