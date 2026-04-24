package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

const (
	grpcRequestIDMetadataKey = "x-request-id"
)

// UnaryRequestIDInterceptor is a gRPC server interceptor that extracts or generates
// a request ID from incoming metadata and stores it in the context.
// The request ID is also returned in the response metadata.
func UnaryRequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		requestID := extractRequestIDFromMetadata(ctx)
		if requestID == "" {
			requestID = string(snx_lib_request.NewRequestID())
		}

		ctx = context.WithValue(ctx, requestIDContextKey{}, requestID)

		// Non-fatal: proceed even if we can't set response headers
		grpc.SetHeader(ctx, metadata.Pairs(grpcRequestIDMetadataKey, requestID)) //nolint:errcheck

		return handler(ctx, req)
	}
}

type requestIDContextKey struct{}

// GetGRPCRequestID retrieves the request ID from a gRPC context.
// Returns an empty string if no request ID is present.
func GetGRPCRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return requestID
	}
	return ""
}

func extractRequestIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(grpcRequestIDMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
