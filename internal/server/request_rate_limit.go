package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
)

var errNoJSONRPCResponses = errors.New("no JSON-RPC responses to encode")

var requestRateLimitOperations = map[string]string{
	"initialize":               "initialize",
	"ping":                     "mcp_ping",
	"prompts/get":              "get_prompt",
	"prompts/list":             "list_prompts",
	"resources/list":           "list_resources",
	"resources/templates/list": "list_resource_templates",
	"tools/list":               "list_tools",
}

func wrapMCPHandlerWithRateLimit(
	logger snx_lib_logging.Logger,
	next http.Handler,
	limiter tools.ToolRateLimiter,
	trustedProxyPrefixes []netip.Prefix,
) http.Handler {
	if next == nil || limiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.Method != http.MethodPost || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if closeErr := r.Body.Close(); closeErr != nil {
			logger.Warn("failed to close request body after read", "error", closeErr)
		}
		if err != nil {
			http.Error(w, "Bad Request: failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		requestWithClientIP := r.WithContext(tools.WithClientIP(r.Context(), clientIPFromRequest(r, trustedProxyPrefixes)))

		plan, err := rateLimitRPCRequests(requestWithClientIP.Context(), body, limiter)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !plan.modified {
			next.ServeHTTP(w, requestWithClientIP)
			return
		}
		if len(plan.allowedPayloads) == 0 {
			if len(plan.blockedResponses) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if err := writeJSONRPCResponses(w, plan.blockedResponses, plan.wasBatch); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		filteredBody, err := encodeRawJSONRPCPayloads(plan.allowedPayloads, plan.wasBatch)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		filteredReq := requestWithClientIP.Clone(requestWithClientIP.Context())
		filteredReq.Body = io.NopCloser(bytes.NewReader(filteredBody))
		filteredReq.ContentLength = int64(len(filteredBody))

		recorder := &bufferingResponseWriter{header: make(http.Header)}
		next.ServeHTTP(recorder, filteredReq)
		if len(plan.blockedResponses) == 0 {
			if err := copyResponse(w, recorder.statusCode, recorder.header, recorder.body.Bytes()); err != nil {
				logger.Warn("failed to copy downstream response", "error", err)
			}
			return
		}

		downstreamResponses, err := decodeJSONRPCResponses(recorder.body.Bytes())
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := writeJSONRPCResponses(w, append(plan.blockedResponses, downstreamResponses...), plan.wasBatch); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

type rateLimitPlan struct {
	allowedPayloads  []json.RawMessage
	blockedResponses []*jsonrpc.Response
	modified         bool
	wasBatch         bool
}

func rateLimitRPCRequests(
	ctx context.Context,
	body []byte,
	limiter tools.ToolRateLimiter,
) (*rateLimitPlan, error) {
	payloads, wasBatch, err := decodeRawJSONRPCPayloads(body)
	if err != nil {
		return &rateLimitPlan{}, nil
	}
	if len(payloads) == 0 {
		return &rateLimitPlan{}, nil
	}

	plan := &rateLimitPlan{
		allowedPayloads: make([]json.RawMessage, 0, len(payloads)),
		wasBatch:        wasBatch,
	}

	for _, payload := range payloads {
		req, ok, err := decodeJSONRPCRequest(payload)
		if err != nil {
			return &rateLimitPlan{}, nil
		}
		if !ok || req == nil {
			plan.allowedPayloads = append(plan.allowedPayloads, payload)
			continue
		}
		operationName, shouldRateLimit := requestOperationForPayload(req)
		if !shouldRateLimit {
			plan.allowedPayloads = append(plan.allowedPayloads, payload)
			continue
		}
		if err := tools.MaybeRateLimitOperation(ctx, limiter, nil, operationName, 1); err != nil {
			plan.modified = true
			if req.IsCall() {
				plan.blockedResponses = append(plan.blockedResponses, &jsonrpc.Response{
					Error: tools.JSONRPCErrorForRateLimit(err),
					ID:    req.ID,
				})
			}
			continue
		}
		plan.allowedPayloads = append(plan.allowedPayloads, payload)
	}

	return plan, nil
}

func decodeRawJSONRPCPayloads(body []byte) ([]json.RawMessage, bool, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, false, nil
	}
	var rawBatch []json.RawMessage
	if err := json.Unmarshal(body, &rawBatch); err == nil {
		return rawBatch, true, nil
	}
	return []json.RawMessage{json.RawMessage(body)}, false, nil
}

func decodeJSONRPCRequest(payload json.RawMessage) (*jsonrpc.Request, bool, error) {
	msg, err := jsonrpc.DecodeMessage(payload)
	if err != nil {
		return nil, false, nil
	}
	req, ok := msg.(*jsonrpc.Request)
	return req, ok, nil
}

func encodeRawJSONRPCPayloads(payloads []json.RawMessage, wasBatch bool) ([]byte, error) {
	if !wasBatch && len(payloads) == 1 {
		return payloads[0], nil
	}
	return json.Marshal(payloads)
}

func decodeJSONRPCResponses(body []byte) ([]*jsonrpc.Response, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}

	var rawBatch []json.RawMessage
	if err := json.Unmarshal(body, &rawBatch); err == nil {
		responses := make([]*jsonrpc.Response, 0, len(rawBatch))
		for _, raw := range rawBatch {
			msg, err := jsonrpc.DecodeMessage(raw)
			if err != nil {
				return nil, err
			}
			response, ok := msg.(*jsonrpc.Response)
			if !ok {
				continue
			}
			responses = append(responses, response)
		}
		return responses, nil
	}

	msg, err := jsonrpc.DecodeMessage(body)
	if err != nil {
		return nil, err
	}
	response, ok := msg.(*jsonrpc.Response)
	if !ok {
		return nil, nil
	}
	return []*jsonrpc.Response{response}, nil
}

func copyResponse(w http.ResponseWriter, statusCode int, header http.Header, body []byte) error {
	for key, values := range header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if statusCode != 0 {
		w.WriteHeader(statusCode)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write downstream response body: %w", err)
	}
	return nil
}

func requestOperationForPayload(req *jsonrpc.Request) (string, bool) {
	if operationName, ok := requestRateLimitOperations[req.Method]; ok {
		return operationName, true
	}
	return "", false
}

func writeJSONRPCResponses(w http.ResponseWriter, responses []*jsonrpc.Response, wasBatch bool) error {
	payload, err := encodeJSONRPCResponses(responses, wasBatch)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(payload)
	return err
}

func encodeJSONRPCResponses(responses []*jsonrpc.Response, wasBatch bool) ([]byte, error) {
	if !wasBatch && len(responses) == 1 {
		return jsonrpc.EncodeMessage(responses[0])
	}

	rawBatch := make([]json.RawMessage, 0, len(responses))
	for _, response := range responses {
		payload, err := jsonrpc.EncodeMessage(response)
		if err != nil {
			return nil, err
		}
		rawBatch = append(rawBatch, payload)
	}

	if len(rawBatch) == 0 {
		return nil, errNoJSONRPCResponses
	}

	return json.Marshal(rawBatch)
}

// bufferingResponseWriter captures a downstream handler's response in memory
// so the middleware can inspect and merge it before writing to the real
// ResponseWriter. It implements http.Flusher so that handlers that type-assert
// for Flusher do not panic, but flushes are no-ops since the body is buffered.
type bufferingResponseWriter struct {
	body       bytes.Buffer
	header     http.Header
	statusCode int
}

func (b *bufferingResponseWriter) Header() http.Header {
	return b.header
}

func (b *bufferingResponseWriter) Write(p []byte) (int, error) {
	return b.body.Write(p)
}

func (b *bufferingResponseWriter) WriteHeader(code int) {
	if b.statusCode == 0 {
		b.statusCode = code
	}
}

// Flush is a no-op: the response is fully buffered and will be forwarded once
// the middleware has merged the downstream and blocked-request responses.
func (b *bufferingResponseWriter) Flush() {}
