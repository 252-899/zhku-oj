package stream

import (
	aiapi "zk-code-arena-server/pkg/ai/api"
)

func Event(eventType aiapi.EventType, req *aiapi.RuntimeRequest, stage string, data any) *aiapi.RuntimeStreamEvent {
	if req != nil {
		req.Normalize()
	}
	return aiapi.NewRuntimeStreamEvent(eventType, req, stage, data)
}

func MetaEvent(req *aiapi.RuntimeRequest, data map[string]any) *aiapi.RuntimeStreamEvent {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["status"]; !ok {
		data["status"] = "running"
	}
	return Event(aiapi.EventMeta, req, "running", data)
}

func StateEvent(req *aiapi.RuntimeRequest, stage string, data map[string]any) *aiapi.RuntimeStreamEvent {
	if data == nil {
		data = map[string]any{}
	}
	if stage != "" {
		if _, ok := data["stage_label"]; !ok {
			data["stage_label"] = stage
		}
	}
	return Event(aiapi.EventState, req, stage, data)
}

func ErrorEvent(req *aiapi.RuntimeRequest, stage, message string, data map[string]any) *aiapi.RuntimeStreamEvent {
	event := Event(aiapi.EventError, req, stage, data)
	event.ErrorMessage = message
	return event
}

func DoneEvent(req *aiapi.RuntimeRequest, status string) *aiapi.RuntimeStreamEvent {
	if status == "" {
		status = "completed"
	}
	stage := status
	if stage == "completed" {
		stage = "completed"
	}
	return Event(aiapi.EventDone, req, stage, map[string]any{"status": status})
}
