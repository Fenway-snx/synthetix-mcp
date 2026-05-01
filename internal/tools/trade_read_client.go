// Broker-signing shim around the /v1/trade REST transport.
// It hides envelope shape and returns typed unavailability errors.
package tools

import (
	"context"
	"errors"
	"fmt"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/synthetixio/synthetix-go/resttrade"
	"github.com/synthetixio/synthetix-go/types"
)

// Returned when an authenticated read cannot be produced for the session.
var ErrReadUnavailable = errors.New("tools: authenticated REST read unavailable for this session")

// Returned when broker and session subaccounts differ.
var ErrBrokerSubAccountMismatch = errors.New("tools: broker subaccount does not match session subaccount")

// Returned when a broker-signed write cannot be produced.
var ErrWriteUnavailable = errors.New("tools: broker-signed REST write unavailable for this session")

// Narrow broker surface for authenticated reads.
// Kept separate from writes so deployments can constrain signing scope.
type BrokerReadSigner interface {
	SignReadAction(subAccountID int64, action snx_lib_api_types.RequestAction) (snx_lib_auth.TradeSignature, int64, error)
	WalletAddress() string
	SubAccountID() int64
}

// Write-side broker surface for signing plus nonce allocation.
// The shim combines this with local validation before POST.
type BrokerTradeSigner interface {
	SignTradeAction(
		subAccountID int64,
		nonce int64,
		expiresAfter int64,
		action snx_lib_api_types.RequestAction,
		payload any,
	) (snx_lib_auth.TradeSignature, error)
	AllocateNonce() (int64, int64)
	WalletAddress() string
	SubAccountID() int64
}

// Local pre-POST hook for session and ownership validation.
type TradeActionValidator interface {
	ValidateTradeAction(
		sessionWalletAddress string,
		sessionSubAccountID int64,
		nonce int64,
		expiresAfter int64,
		action snx_lib_api_types.RequestAction,
		payload any,
		signature snx_lib_auth.TradeSignature,
	) error
}

// REST trade client with broker-backed read and write signing.
// Missing dependencies return typed unavailability errors.
type TradeReadClient struct {
	rest      *resttrade.Client
	broker    BrokerReadSigner
	writer    BrokerTradeSigner
	validator TradeActionValidator
	logger    snx_lib_logging.Logger
}

// In the production wiring writer and broker are the same
// underlying signer; the split exists so tests can stub the two
// surfaces independently.
func NewTradeReadClient(
	rest *resttrade.Client,
	broker BrokerReadSigner,
	writer BrokerTradeSigner,
	validator TradeActionValidator,
	logger snx_lib_logging.Logger,
) *TradeReadClient {
	return &TradeReadClient{
		rest:      rest,
		broker:    broker,
		writer:    writer,
		validator: validator,
		logger:    logger,
	}
}

// Signs + POSTs an authenticated read. Returns the api-service
// response verbatim; callers own the mapping to tool output shapes.
func (c *TradeReadClient) GetSubAccount(ctx context.Context, tc ToolContext) (*types.SubAccountResponse, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getSubAccount"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetSubAccount(ctx, req)
}

func (c *TradeReadClient) GetOpenOrders(ctx context.Context, tc ToolContext) ([]types.OpenOrder, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getOpenOrders"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetOpenOrders(ctx, req)
}

func (c *TradeReadClient) GetPositions(ctx context.Context, tc ToolContext) ([]types.Position, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getPositions"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetPositions(ctx, req)
}

// Passes filter params through the envelope without re-signing.
func (c *TradeReadClient) GetOrderHistory(ctx context.Context, tc ToolContext, params map[string]any) (types.OrderHistoryResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getOrderHistory"), params)
	if err != nil {
		return nil, err
	}
	return c.rest.GetOrderHistory(ctx, req)
}

// Params (symbol, orderId, startTime, endTime, offset, limit) pass
// through verbatim. The wrapped response carries total so callers
// can page.
func (c *TradeReadClient) GetTrades(ctx context.Context, tc ToolContext, params map[string]any) (*types.TradeHistoryResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getTrades"), params)
	if err != nil {
		return nil, err
	}
	return c.rest.GetTrades(ctx, req)
}

// Upstream caps limit at 1000; callers that need the full series
// must re-issue with a rolling time window.
func (c *TradeReadClient) GetFundingPayments(ctx context.Context, tc ToolContext, params map[string]any) (*types.FundingPaymentsResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getFundingPayments"), params)
	if err != nil {
		return nil, err
	}
	return c.rest.GetFundingPayments(ctx, req)
}

// Only the `period` key is consulted upstream; pass nil for the
// default ("day").
func (c *TradeReadClient) GetPerformanceHistory(ctx context.Context, tc ToolContext, params map[string]any) (*types.PerformanceHistoryResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getPerformanceHistory"), params)
	if err != nil {
		return nil, err
	}
	return c.rest.GetPerformanceHistory(ctx, req)
}

func (c *TradeReadClient) GetBalanceUpdates(ctx context.Context, tc ToolContext, params map[string]any) (types.BalanceUpdatesResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getBalanceUpdates"), params)
	if err != nil {
		return types.BalanceUpdatesResponse{}, err
	}
	return c.rest.GetBalanceUpdates(ctx, req)
}

func (c *TradeReadClient) GetDelegatedSigners(ctx context.Context, tc ToolContext) (types.DelegatedSignersResponse, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getDelegatedSigners"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetDelegatedSigners(ctx, req)
}

func (c *TradeReadClient) GetDelegationsForDelegate(ctx context.Context, tc ToolContext) (types.DelegationsForDelegateResponse, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getDelegationsForDelegate"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetDelegationsForDelegate(ctx, req)
}

func (c *TradeReadClient) GetFees(ctx context.Context, tc ToolContext) (*types.FeesResponse, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getFees"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetFees(ctx, req)
}

func (c *TradeReadClient) GetPortfolio(ctx context.Context, tc ToolContext) (*types.PortfolioResponse, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getPortfolio"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetPortfolio(ctx, req)
}

func (c *TradeReadClient) GetPositionHistory(ctx context.Context, tc ToolContext, params map[string]any) (types.PositionHistoryResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getPositionHistory"), params)
	if err != nil {
		return types.PositionHistoryResponse{}, err
	}
	return c.rest.GetPositionHistory(ctx, req)
}

func (c *TradeReadClient) GetRateLimits(ctx context.Context, tc ToolContext) (*types.RateLimitsResponse, error) {
	req, err := c.buildRequest(tc, snx_lib_api_types.RequestAction("getRateLimits"))
	if err != nil {
		return nil, err
	}
	return c.rest.GetRateLimits(ctx, req)
}

func (c *TradeReadClient) GetTradesForPosition(ctx context.Context, tc ToolContext, params map[string]any) (*types.TradesForPositionResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getTradesForPosition"), params)
	if err != nil {
		return nil, err
	}
	return c.rest.GetTradesForPosition(ctx, req)
}

func (c *TradeReadClient) GetTransfers(ctx context.Context, tc ToolContext, params map[string]any) (types.TransfersResponse, error) {
	req, err := c.buildRequestWithParams(tc, snx_lib_api_types.RequestAction("getTransfers"), params)
	if err != nil {
		return types.TransfersResponse{}, err
	}
	return c.rest.GetTransfers(ctx, req)
}

// Write methods gate broker availability, sign, validate locally, then POST.
// Sign payload and envelope payload are split so callers own their invariant.

func (c *TradeReadClient) PlaceOrders(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload any,
) (*types.PlaceOrdersResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("placeOrders"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.PlaceOrders(ctx, &types.PlaceOrdersRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) ModifyOrder(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload any,
) (*types.ModifyOrderResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("modifyOrder"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.ModifyOrder(ctx, &types.ModifyOrderRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) CancelOrders(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload any,
) (*types.CancelOrdersResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("cancelOrders"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.CancelOrders(ctx, &types.CancelOrdersRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) CancelAllOrders(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload any,
) (*types.CancelAllOrdersResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("cancelAllOrders"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.CancelAllOrders(ctx, &types.CancelAllOrdersRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) AddDelegatedSigner(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload types.AddDelegatedSignerAction,
) (*types.AddDelegatedSignerResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("addDelegatedSigner"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.AddDelegatedSigner(ctx, &types.AddDelegatedSignerRequest{
		Params:       envelopePayload,
		SubAccountID: fmtInt64(tc.State.SubAccountID),
		Nonce:        uint64(nonce),
		ExpiresAfter: expiresAfter,
		Signature:    sigToComponents(sig),
	})
}

func (c *TradeReadClient) RemoveAllDelegatedSigners(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload types.RemoveAllDelegatedSignersAction,
) (*types.RemoveAllDelegatedSignersResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("removeAllDelegatedSigners"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.RemoveAllDelegatedSigners(ctx, &types.RemoveAllDelegatedSignersRequest{
		Params:       envelopePayload,
		SubAccountID: fmtInt64(tc.State.SubAccountID),
		Nonce:        uint64(nonce),
		ExpiresAfter: expiresAfter,
		Signature:    sigToComponents(sig),
	})
}

func (c *TradeReadClient) RemoveDelegatedSigner(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload map[string]any,
) (*types.RemoveDelegatedSignerResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("removeDelegatedSigner"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.RemoveDelegatedSigner(ctx, &types.RemoveDelegatedSignerRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) ScheduleCancel(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload map[string]any,
) (*types.ScheduleCancelResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("scheduleCancel"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.ScheduleCancel(ctx, &types.ScheduleCancelRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) TransferCollateral(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload map[string]any,
) (*types.TransferCollateralResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("transferCollateral"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.TransferCollateral(ctx, &types.TransferCollateralRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) UpdateLeverage(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload map[string]any,
) (*types.UpdateLeverageResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("updateLeverage"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.UpdateLeverage(ctx, &types.UpdateLeverageRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

func (c *TradeReadClient) WithdrawCollateral(
	ctx context.Context,
	tc ToolContext,
	signPayload any,
	envelopePayload map[string]any,
) (*types.WithdrawCollateralResponse, error) {
	sig, nonce, expiresAfter, err := c.signWrite(tc, snx_lib_api_types.RequestAction("withdrawCollateral"), signPayload)
	if err != nil {
		return nil, err
	}
	return c.rest.WithdrawCollateral(ctx, &types.WithdrawCollateralRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: c.writer.WalletAddress(),
		Nonce:         uint64(nonce),
		ExpiresAfter:  expiresAfter,
		Signature:     sigToComponents(sig),
	})
}

// External-wallet writes accept caller-supplied signature material.
// The shim only handles envelope construction and REST dispatch.

// SignedWrite bundles the caller-provided signing outputs the
// ext-wallet write methods need. All four fields are required;
// WalletAddress must match the session's authenticated wallet.
type SignedWrite struct {
	WalletAddress string
	Nonce         int64
	ExpiresAfter  int64
	Signature     snx_lib_auth.TradeSignature
}

func (c *TradeReadClient) PlaceOrdersWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload any,
	sw SignedWrite,
) (*types.PlaceOrdersResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.PlaceOrders(ctx, &types.PlaceOrdersRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) ModifyOrderWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload any,
	sw SignedWrite,
) (*types.ModifyOrderResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.ModifyOrder(ctx, &types.ModifyOrderRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) CancelOrdersWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload any,
	sw SignedWrite,
) (*types.CancelOrdersResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.CancelOrders(ctx, &types.CancelOrdersRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) CancelAllOrdersWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload any,
	sw SignedWrite,
) (*types.CancelAllOrdersResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.CancelAllOrders(ctx, &types.CancelAllOrdersRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) AddDelegatedSignerWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload types.AddDelegatedSignerAction,
	sw SignedWrite,
) (*types.AddDelegatedSignerResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.AddDelegatedSigner(ctx, &types.AddDelegatedSignerRequest{
		Params:       envelopePayload,
		SubAccountID: fmtInt64(tc.State.SubAccountID),
		Nonce:        uint64(sw.Nonce),
		ExpiresAfter: sw.ExpiresAfter,
		Signature:    sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) RemoveAllDelegatedSignersWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload types.RemoveAllDelegatedSignersAction,
	sw SignedWrite,
) (*types.RemoveAllDelegatedSignersResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.RemoveAllDelegatedSigners(ctx, &types.RemoveAllDelegatedSignersRequest{
		Params:       envelopePayload,
		SubAccountID: fmtInt64(tc.State.SubAccountID),
		Nonce:        uint64(sw.Nonce),
		ExpiresAfter: sw.ExpiresAfter,
		Signature:    sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) RemoveDelegatedSignerWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload map[string]any,
	sw SignedWrite,
) (*types.RemoveDelegatedSignerResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.RemoveDelegatedSigner(ctx, &types.RemoveDelegatedSignerRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) ScheduleCancelWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload map[string]any,
	sw SignedWrite,
) (*types.ScheduleCancelResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.ScheduleCancel(ctx, &types.ScheduleCancelRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) TransferCollateralWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload map[string]any,
	sw SignedWrite,
) (*types.TransferCollateralResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.TransferCollateral(ctx, &types.TransferCollateralRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) UpdateLeverageWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload map[string]any,
	sw SignedWrite,
) (*types.UpdateLeverageResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.UpdateLeverage(ctx, &types.UpdateLeverageRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

func (c *TradeReadClient) WithdrawCollateralWithSignature(
	ctx context.Context,
	tc ToolContext,
	envelopePayload map[string]any,
	sw SignedWrite,
) (*types.WithdrawCollateralResponse, error) {
	if err := c.assertRESTAvailable(); err != nil {
		return nil, err
	}
	return c.rest.WithdrawCollateral(ctx, &types.WithdrawCollateralRequest{
		Params:        envelopePayload,
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: sw.WalletAddress,
		Nonce:         uint64(sw.Nonce),
		ExpiresAfter:  sw.ExpiresAfter,
		Signature:     sigToComponents(sw.Signature),
	})
}

// Cheaper gate for ext-wallet writes: we only need the REST
// transport, not the broker or the validator, so we don't call
// signWrite's richer precondition chain.
func (c *TradeReadClient) assertRESTAvailable() error {
	if c == nil || c.rest == nil {
		return ErrWriteUnavailable
	}
	return nil
}

// Shared gate + broker-sign + local-validate codepath. Returns the
// signature plus (nonce, expiresAfter) so callers can build the
// typed envelope themselves.
func (c *TradeReadClient) signWrite(
	tc ToolContext,
	action snx_lib_api_types.RequestAction,
	signPayload any,
) (snx_lib_auth.TradeSignature, int64, int64, error) {
	if c == nil || c.rest == nil {
		return snx_lib_auth.TradeSignature{}, 0, 0, ErrWriteUnavailable
	}
	if c.writer == nil {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf("%w: agent broker is disabled", ErrWriteUnavailable)
	}
	if c.validator == nil {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf("%w: trade-action validator is not wired", ErrWriteUnavailable)
	}
	if tc.State == nil || tc.State.SubAccountID <= 0 {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf("%w: session is not bound to a subaccount", ErrWriteUnavailable)
	}
	brokerSub := c.writer.SubAccountID()
	if brokerSub <= 0 {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf("%w: broker has not bound a subaccount yet", ErrWriteUnavailable)
	}
	if brokerSub != tc.State.SubAccountID {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf(
			"%w: session subAccountId=%d, broker subAccountId=%d",
			ErrBrokerSubAccountMismatch, tc.State.SubAccountID, brokerSub,
		)
	}

	nonce, expiresAfter := c.writer.AllocateNonce()
	sig, err := c.writer.SignTradeAction(
		tc.State.SubAccountID,
		nonce,
		expiresAfter,
		action,
		signPayload,
	)
	if err != nil {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf("sign %s: %w", string(action), err)
	}
	if err := c.validator.ValidateTradeAction(
		tc.State.WalletAddress,
		tc.State.SubAccountID,
		nonce,
		expiresAfter,
		action,
		signPayload,
		sig,
	); err != nil {
		return snx_lib_auth.TradeSignature{}, 0, 0, fmt.Errorf("validate %s: %w", string(action), err)
	}
	return sig, nonce, expiresAfter, nil
}

// Flattens the signature into the wire shape the REST envelope
// expects. Lives here rather than in types/ to avoid an import
// cycle on lib/auth.
func sigToComponents(sig snx_lib_auth.TradeSignature) types.SignatureComponents {
	return types.SignatureComponents{V: int(sig.V), R: sig.R, S: sig.S}
}

// Shared gate and signing path for authenticated reads.
// Binding checks happen before broker signing.
func (c *TradeReadClient) buildRequest(tc ToolContext, action snx_lib_api_types.RequestAction) (*types.SubAccountActionRequest, error) {
	if c == nil || c.rest == nil {
		return nil, ErrReadUnavailable
	}
	if tc.State == nil || tc.State.SubAccountID <= 0 {
		return nil, fmt.Errorf("%w: session is not bound to a subaccount", ErrReadUnavailable)
	}
	if c.broker == nil {
		return nil, fmt.Errorf("%w: agent broker is disabled", ErrReadUnavailable)
	}
	brokerSub := c.broker.SubAccountID()
	if brokerSub <= 0 {
		// Do not block hot read paths on discovery.
		return nil, fmt.Errorf("%w: broker has not bound a subaccount yet", ErrReadUnavailable)
	}
	if brokerSub != tc.State.SubAccountID {
		return nil, fmt.Errorf(
			"%w: session subAccountId=%d, broker subAccountId=%d",
			ErrBrokerSubAccountMismatch, tc.State.SubAccountID, brokerSub,
		)
	}

	sig, expiresAfter, err := c.broker.SignReadAction(tc.State.SubAccountID, action)
	if err != nil {
		return nil, fmt.Errorf("sign %s: %w", string(action), err)
	}

	// The envelope wallet must be the signer's, not the session
	// owner's: upstream recovers the address from the signature and
	// checks it before the delegate-permission lookup.
	wallet := c.broker.WalletAddress()

	return &types.SubAccountActionRequest{
		Params:        map[string]any{"action": string(action)},
		SubAccountID:  fmtInt64(tc.State.SubAccountID),
		WalletAddress: wallet,
		Nonce:         0,
		ExpiresAfter:  expiresAfter,
		Signature: types.SignatureComponents{
			V: int(sig.V),
			R: sig.R,
			S: sig.S,
		},
	}, nil
}

// Adds a caller-supplied filter map to the read envelope. Upstream
// reads params from the envelope, not from the signed payload, so
// filters can vary per call without re-signing.
func (c *TradeReadClient) buildRequestWithParams(
	tc ToolContext,
	action snx_lib_api_types.RequestAction,
	params map[string]any,
) (*types.SubAccountActionRequest, error) {
	req, err := c.buildRequest(tc, action)
	if err != nil {
		return nil, err
	}
	// Echo the action into params so older upstream builds that
	// dispatch from params["action"] still route correctly. Callers
	// can override by setting their own "action" key.
	if params == nil {
		params = make(map[string]any, 1)
	}
	if _, ok := params["action"]; !ok {
		params["action"] = string(action)
	}
	req.Params = params
	return req, nil
}

// Matches the string encoding upstream uses for subAccountId.
func fmtInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}
