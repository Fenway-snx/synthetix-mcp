package halt

import (
	snx_lib_runtime_admin_jetstream_queues "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/admin/jetstream_queues"
)

type TargetState string

const (
	TargetState_Idle    TargetState = "IDLE"
	TargetState_Running TargetState = "RUNNING"
	TargetState_Stopped TargetState = "STOPPED"
)

type StateChangeRequest struct {
	Target         TargetState `json:"target"`
	TimeoutSeconds int         `json:"timeoutSeconds,omitempty"`
}

type StateEnvelope struct {
	Error           string         `json:"error,omitempty"`
	Metadata        StateMetadata  `json:"metadata"`
	Metrics         StateMetrics   `json:"metrics"`
	ServiceSpecific map[string]any `json:"serviceSpecific"`
}

type StateMetadata struct {
	ServiceId     string `json:"serviceId"`
	Status        string `json:"status"`
	TargetState   string `json:"targetState"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
	Version       string `json:"version"`
}

type JetStreamQueueDepthsMetrics struct {
	CollectPartial bool                                                         `json:"collectPartial,omitempty"`
	Queues         []snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepth `json:"queues"`
}

type StateMetrics struct {
	DrainDurationMs      int64                        `json:"drainDurationMs"`
	InFlightOps          int64                        `json:"inFlightOps"`
	JetStreamQueueDepths *JetStreamQueueDepthsMetrics `json:"jetStreamQueueDepths,omitempty"`
}
