package utils

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Creates a custom dialer that only uses IPv4 addresses.
// This prevents "network is unreachable" errors when IPv6 is not properly configured.
func IPv4OnlyDialer(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext(ctx, "tcp4", addr)
}

// Returns the standard dial options for gRPC clients,
// including IPv4-only dialing to avoid IPv6 connectivity issues.
//
// Note: gRPC connections are designed to be long-lived and multiplexed.
// A single connection can handle thousands of concurrent RPCs via HTTP/2.
// Connection pooling is handled internally by gRPC, so manual pooling is not needed.
func DefaultGRPCDialOptions(additionalOpts ...grpc.DialOption) []grpc.DialOption {
	baseOpts := []grpc.DialOption{
		grpc.WithContextDialer(IPv4OnlyDialer),
		// Keep connection alive to detect broken connections
		// 30s interval matches DefaultGRPCServerOptions enforcement
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // Send keepalive every 30s
			Timeout:             20 * time.Second, // Wait 20s for keepalive ack
			PermitWithoutStream: true,             // Send pings even without active streams
		}),
	}
	return append(baseOpts, additionalOpts...)
}

// Returns the standard server options for gRPC servers with keepalive enforcement
// that matches DefaultGRPCDialOptions client settings.
//
// These settings prevent "ENHANCE_YOUR_CALM - too_many_pings" errors by:
// - Allowing clients to ping every 20s+ (matches client's 30s pings)
// - Permitting pings without active streams
// - Setting reasonable connection lifetime limits
//
// All gRPC servers should use these options for consistency.
func DefaultGRPCServerOptions(additionalOpts ...grpc.ServerOption) []grpc.ServerOption {
	baseOpts := []grpc.ServerOption{
		// Enforcement policy - what we require from clients
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second, // Allow pings no more frequent than every 20s
			PermitWithoutStream: true,             // Allow pings even when no streams are active
		}),
		// Server parameters - when we ping clients
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute, // Max time a connection can be idle before server closes it
			MaxConnectionAge:      30 * time.Minute, // Max connection lifetime before server closes it
			MaxConnectionAgeGrace: 5 * time.Second,  // Grace period for active RPCs after MaxConnectionAge
			Time:                  5 * time.Minute,  // Ping client if no activity for 5 min
			Timeout:               20 * time.Second, // Wait 20s for ping ack before closing connection
		}),
	}
	return append(baseOpts, additionalOpts...)
}
