package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	toolCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_tool_calls_total",
			Help: "Total tool calls by tool name and outcome (ok, error, rate_limited, auth_failed)",
		},
		[]string{"tool", "outcome"},
	)

	toolCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_tool_call_duration_seconds",
			Help:    "Duration of MCP tool call handling by tool name",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"tool"},
	)

	rateLimitRejectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_rate_limit_rejections_total",
			Help: "Total rate limit rejections by scope (ip, subaccount) and layer (http, tool)",
		},
		[]string{"scope", "layer"},
	)

	activeSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mcp_active_sessions",
			Help: "Number of currently active MCP sessions in the session store",
		},
	)

	sessionEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_session_events_total",
			Help: "Session lifecycle events by type (created, authenticated, expired, deleted)",
		},
		[]string{"event"},
	)

	activeSubscriptions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mcp_active_subscriptions",
			Help: "Total number of active streaming subscriptions across all sessions",
		},
	)

	subscriptionEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_subscription_events_total",
			Help: "Subscription lifecycle events by type (subscribed, unsubscribed)",
		},
		[]string{"event"},
	)
)

func ToolCallsTotal(tool, outcome string) prometheus.Counter {
	return toolCallsTotal.WithLabelValues(tool, outcome)
}

func ToolCallDuration(tool string) prometheus.Observer {
	return toolCallDuration.WithLabelValues(tool)
}

func RateLimitRejectionsTotal(scope, layer string) prometheus.Counter {
	return rateLimitRejectionsTotal.WithLabelValues(scope, layer)
}

func ActiveSessions() prometheus.Gauge {
	return activeSessions
}

func SessionEventsTotal(event string) prometheus.Counter {
	return sessionEventsTotal.WithLabelValues(event)
}

func ActiveSubscriptions() prometheus.Gauge {
	return activeSubscriptions
}

func SubscriptionEventsTotal(event string) prometheus.Counter {
	return subscriptionEventsTotal.WithLabelValues(event)
}
