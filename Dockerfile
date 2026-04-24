# Build argument to control Datadog inclusion
ARG ENABLE_DATADOG=false

# Build stage
FROM golang:1.26.1-alpine3.23 AS builder
ARG ENABLE_DATADOG
RUN if [ "$ENABLE_DATADOG" = "true" ]; then go install github.com/DataDog/orchestrion@v1.6.1; fi

# Download dependencies for lib
WORKDIR /app/lib
COPY lib/go.mod lib/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Download dependencies for MCP service
WORKDIR /app/services/mcp
COPY services/mcp/go.mod services/mcp/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source
WORKDIR /app
COPY lib/ ./lib/
COPY services/mcp/ ./services/mcp/

# Build binary
WORKDIR /app/services/mcp
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux $([ "$ENABLE_DATADOG" = "true" ] && echo "orchestrion") go build -o mcp-server ./cmd/server/main.go

# Final stage
FROM alpine:3.23.0

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    mkdir -p /app/services/mcp

WORKDIR /app
COPY --from=builder /app/services/mcp/mcp-server .
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8096

CMD ["./mcp-server"]
