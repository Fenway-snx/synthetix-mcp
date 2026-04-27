package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/synthetixio/synthetix-go/types"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	"github.com/Fenway-snx/synthetix-mcp/internal/agentbroker"
)

const maxDeadManTimeoutSeconds = int64(24 * 60 * 60)

var deadManSwitchCache sync.Map

type lifecycleSignatureInput struct {
	ExpiresAfter int64               `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Must match preview_trade_signature."`
	Nonce        int64               `json:"nonce" jsonschema:"Unique nonce for this action. Must match preview_trade_signature."`
	Signature    tradeSignatureInput `json:"signature" jsonschema:"EIP-712 signature for this action."`
}

type updateLeverageInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	Symbol       string    `json:"symbol" jsonschema:"Market symbol, e.g. BTC-USDT. Short symbols such as BTC are normalized to BTC-USDT."`
	Leverage     string    `json:"leverage" jsonschema:"Target leverage as a positive decimal string."`
	lifecycleSignatureInput
}

type withdrawCollateralInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	Symbol       string    `json:"symbol" jsonschema:"Collateral symbol, e.g. USDC."`
	Amount       string    `json:"amount" jsonschema:"Amount to withdraw as a decimal string."`
	Destination  string    `json:"destination" jsonschema:"Destination EVM address."`
	lifecycleSignatureInput
}

type transferCollateralInput struct {
	SubAccountID   FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional source subaccount ID. Omit to use the authenticated subaccount."`
	ToSubAccountID FlexInt64 `json:"toSubAccountId" jsonschema:"Destination subaccount ID."`
	Symbol         string    `json:"symbol" jsonschema:"Collateral symbol, e.g. USDC."`
	Amount         string    `json:"amount" jsonschema:"Amount to transfer as a decimal string."`
	lifecycleSignatureInput
}

type deadManSwitchInput struct {
	SubAccountID   FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	TimeoutSeconds int64     `json:"timeoutSeconds" jsonschema:"Seconds before all open orders are cancelled if no refresh arrives. Must be between 1 and 86400."`
	lifecycleSignatureInput
}

type delegatedSignerInput struct {
	SubAccountID    FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	DelegateAddress string    `json:"delegateAddress" jsonschema:"Delegate EVM wallet address."`
	Permissions     []string  `json:"permissions" jsonschema:"Delegated permissions, e.g. trading."`
	ExpiresAt       int64     `json:"expiresAt,omitempty" jsonschema:"Optional UNIX timestamp in seconds. Omit or set 0 for no expiry."`
	lifecycleSignatureInput
}

type removeDelegatedSignerInput struct {
	SubAccountID    FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	DelegateAddress string    `json:"delegateAddress" jsonschema:"Delegate EVM wallet address to remove."`
	lifecycleSignatureInput
}

type removeAllDelegatedSignersInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	lifecycleSignatureInput
}

type quickUpdateLeverageInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the broker subaccount."`
	Symbol       string    `json:"symbol" jsonschema:"Market symbol, e.g. BTC-USDT."`
	Leverage     string    `json:"leverage" jsonschema:"Target leverage as a positive decimal string."`
}

type quickWithdrawCollateralInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the broker subaccount."`
	Symbol       string    `json:"symbol" jsonschema:"Collateral symbol, e.g. USDC."`
	Amount       string    `json:"amount" jsonschema:"Amount to withdraw as a decimal string."`
	Destination  string    `json:"destination" jsonschema:"Destination EVM address."`
}

type quickTransferCollateralInput struct {
	SubAccountID   FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional source subaccount ID. Omit to use the broker subaccount."`
	ToSubAccountID FlexInt64 `json:"toSubAccountId" jsonschema:"Destination subaccount ID."`
	Symbol         string    `json:"symbol" jsonschema:"Collateral symbol, e.g. USDC."`
	Amount         string    `json:"amount" jsonschema:"Amount to transfer as a decimal string."`
}

type quickDeadManSwitchInput struct {
	SubAccountID   FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the broker subaccount."`
	TimeoutSeconds int64     `json:"timeoutSeconds" jsonschema:"Seconds before all open orders are cancelled if no refresh arrives. Must be between 1 and 86400."`
}

type lifecycleRawOutput struct {
	Meta   responseMeta `json:"_meta"`
	Result any          `json:"result"`
}

type deadManStatusOutput struct {
	Meta              responseMeta `json:"_meta"`
	Armed             bool         `json:"armed"`
	LastTimeoutSeconds int64        `json:"lastTimeoutSeconds,omitempty"`
	NotKnownToServer   bool        `json:"notKnownToServer"`
	SubAccountID       int64       `json:"subAccountId,string"`
}

type deadManSwitchState struct {
	SubAccountID    int64
	TimeoutSeconds  int64
}

func RegisterLifecycleTools(
	server *mcp.Server,
	deps *ToolDeps,
	authenticator tradeActionAuthenticator,
	tradeReads *TradeReadClient,
) {
	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_update_leverage",
		Description: "Advanced external-wallet path: update leverage with a caller-provided EIP-712 signature. Use preview_trade_signature action=updateLeverage before calling signed_update_leverage.",
	}, func(in updateLeverageInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input updateLeverageInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildUpdateLeveragePayload(input.Symbol, input.Leverage)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "updateLeverage", validated, input.lifecycleSignatureInput, func(sw SignedWrite) (any, error) {
				return tradeReads.UpdateLeverageWithSignature(ctx, tc, envelope, sw)
			})
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_withdraw_collateral",
		Description: "Advanced external-wallet path: withdraw collateral with a caller-provided EIP-712 signature. Use preview_trade_signature action=withdrawCollateral before calling signed_withdraw_collateral.",
	}, func(in withdrawCollateralInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input withdrawCollateralInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildWithdrawCollateralPayload(input.Symbol, input.Amount, input.Destination)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "withdrawCollateral", validated, input.lifecycleSignatureInput, func(sw SignedWrite) (any, error) {
				return tradeReads.WithdrawCollateralWithSignature(ctx, tc, envelope, sw)
			})
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_transfer_collateral",
		Description: "Advanced external-wallet path: transfer collateral with a caller-provided EIP-712 signature. Use preview_trade_signature action=transferCollateral before calling signed_transfer_collateral.",
	}, func(in transferCollateralInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input transferCollateralInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildTransferCollateralPayload(input.ToSubAccountID.Int64(), input.Symbol, input.Amount)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "transferCollateral", validated, input.lifecycleSignatureInput, func(sw SignedWrite) (any, error) {
				return tradeReads.TransferCollateralWithSignature(ctx, tc, envelope, sw)
			})
			return lifecycleResult(tc, result, err)
		})

	registerDeadManSwitchTools(server, deps, authenticator, tradeReads)
	registerDelegatedSignerTools(server, deps, authenticator, tradeReads)
}

func RegisterLifecycleBrokerTools(
	server *mcp.Server,
	deps *ToolDeps,
	broker *agentbroker.Broker,
	authenticator QuickAuthenticator,
	tradeReads *TradeReadClient,
) {
	if broker == nil || tradeReads == nil {
		return
	}
	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "update_leverage",
		Description: "Canonical self-hosted broker path: update leverage in one broker-signed call.",
	}, func(in quickUpdateLeverageInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickUpdateLeverageInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildUpdateLeveragePayload(input.Symbol, input.Leverage)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := tradeReads.UpdateLeverage(ctx, tc, validated, envelope)
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "withdraw_collateral",
		Description: "Canonical self-hosted broker path: withdraw collateral in one broker-signed call.",
	}, func(in quickWithdrawCollateralInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickWithdrawCollateralInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildWithdrawCollateralPayload(input.Symbol, input.Amount, input.Destination)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := tradeReads.WithdrawCollateral(ctx, tc, validated, envelope)
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "transfer_collateral",
		Description: "Canonical self-hosted broker path: transfer collateral between subaccounts in one broker-signed call.",
	}, func(in quickTransferCollateralInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickTransferCollateralInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildTransferCollateralPayload(input.ToSubAccountID.Int64(), input.Symbol, input.Amount)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := tradeReads.TransferCollateral(ctx, tc, validated, envelope)
			return lifecycleResult(tc, result, err)
		})

	registerQuickDeadManTools(server, deps, broker, authenticator, tradeReads)
}

func registerDeadManSwitchTools(
	server *mcp.Server,
	deps *ToolDeps,
	authenticator tradeActionAuthenticator,
	tradeReads *TradeReadClient,
) {
	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_arm_dead_man_switch",
		Description: "Advanced external-wallet path: arm the Synthetix dead-man switch with a caller-provided EIP-712 signature. Use preview_trade_signature action=scheduleCancel before calling signed_arm_dead_man_switch.",
	}, func(in deadManSwitchInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input deadManSwitchInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			result, err := submitSignedDeadMan(ctx, tc, authenticator, tradeReads, input.TimeoutSeconds, input.lifecycleSignatureInput)
			if err == nil {
				storeDeadManState(tc, input.TimeoutSeconds)
			}
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_disarm_dead_man_switch",
		Description: "Advanced external-wallet path: clear the Synthetix dead-man switch with a caller-provided EIP-712 signature. Use preview_trade_signature action=scheduleCancel with timeoutSeconds=0 before calling signed_disarm_dead_man_switch.",
	}, func(in lifecycleSignatureInput) *int64 { return nil },
		func(ctx context.Context, tc ToolContext, input lifecycleSignatureInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			result, err := submitSignedDeadMan(ctx, tc, authenticator, tradeReads, 0, input)
			if err == nil {
				storeDeadManState(tc, 0)
			}
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_dead_man_switch_status",
		Description: "Return the MCP server's last-known dead-man switch state for this session. The exchange does not expose a dedicated status read, so this reports local state since this MCP process started.",
	}, noSubAccount, func(_ context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, deadManStatusOutput, error) {
		return nil, deadManStatus(tc), nil
	})
}

func registerQuickDeadManTools(
	server *mcp.Server,
	deps *ToolDeps,
	broker *agentbroker.Broker,
	authenticator QuickAuthenticator,
	tradeReads *TradeReadClient,
) {
	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "arm_dead_man_switch",
		Description: "Canonical self-hosted broker path: arm the Synthetix dead-man switch in one broker-signed call.",
	}, func(in quickDeadManSwitchInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickDeadManSwitchInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildScheduleCancelPayload(input.TimeoutSeconds)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := tradeReads.ScheduleCancel(ctx, tc, validated, envelope)
			if err == nil {
				storeDeadManState(tc, input.TimeoutSeconds)
			}
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "disarm_dead_man_switch",
		Description: "Canonical self-hosted broker path: clear the Synthetix dead-man switch in one broker-signed call.",
	}, noSubAccount, func(ctx context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, lifecycleRawOutput, error) {
		validated, envelope, err := buildScheduleCancelPayload(0)
		if err != nil {
			return toolErrorResponse[lifecycleRawOutput](err)
		}
		result, err := tradeReads.ScheduleCancel(ctx, tc, validated, envelope)
		if err == nil {
			storeDeadManState(tc, 0)
		}
		return lifecycleResult(tc, result, err)
	})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "keep_alive",
		Description: "Canonical self-hosted broker path: refresh the last armed dead-man-switch timeout for the broker session.",
	}, noSubAccount, func(ctx context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, lifecycleRawOutput, error) {
		state := deadManStatus(tc)
		if !state.Armed || state.LastTimeoutSeconds <= 0 {
			return toolErrorResponse[lifecycleRawOutput](fmt.Errorf("dead-man switch is not armed for this MCP session"))
		}
		validated, envelope, err := buildScheduleCancelPayload(state.LastTimeoutSeconds)
		if err != nil {
			return toolErrorResponse[lifecycleRawOutput](err)
		}
		result, err := tradeReads.ScheduleCancel(ctx, tc, validated, envelope)
		return lifecycleResult(tc, result, err)
	})
}

func registerDelegatedSignerTools(
	server *mcp.Server,
	deps *ToolDeps,
	authenticator tradeActionAuthenticator,
	tradeReads *TradeReadClient,
) {
	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_add_delegated_signer",
		Description: "Advanced external-wallet path: add a delegated signer with a caller-provided EIP-712 signature. Use preview_trade_signature action=addDelegatedSigner before calling signed_add_delegated_signer.",
	}, func(in delegatedSignerInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input delegatedSignerInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildAddDelegatedSignerPayload(input.DelegateAddress, input.Permissions, input.ExpiresAt)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "addDelegatedSigner", validated, input.lifecycleSignatureInput, func(sw SignedWrite) (any, error) {
				return tradeReads.AddDelegatedSignerWithSignature(ctx, tc, envelope, sw)
			})
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_remove_delegated_signer",
		Description: "Advanced external-wallet path: remove one delegated signer with a caller-provided EIP-712 signature. Use preview_trade_signature action=removeDelegatedSigner before calling signed_remove_delegated_signer.",
	}, func(in removeDelegatedSignerInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input removeDelegatedSignerInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildRemoveDelegatedSignerPayload(input.DelegateAddress)
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "removeDelegatedSigner", validated, input.lifecycleSignatureInput, func(sw SignedWrite) (any, error) {
				return tradeReads.RemoveDelegatedSignerWithSignature(ctx, tc, envelope, sw)
			})
			return lifecycleResult(tc, result, err)
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_remove_all_delegated_signers",
		Description: "Advanced external-wallet path: remove all delegated signers with a caller-provided EIP-712 signature. Use preview_trade_signature action=removeAllDelegatedSigners before calling signed_remove_all_delegated_signers.",
	}, func(in removeAllDelegatedSignersInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input removeAllDelegatedSignersInput) (*mcp.CallToolResult, lifecycleRawOutput, error) {
			validated, envelope, err := buildRemoveAllDelegatedSignersPayload()
			if err != nil {
				return toolErrorResponse[lifecycleRawOutput](err)
			}
			result, err := submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "removeAllDelegatedSigners", validated, input.lifecycleSignatureInput, func(sw SignedWrite) (any, error) {
				return tradeReads.RemoveAllDelegatedSignersWithSignature(ctx, tc, envelope, sw)
			})
			return lifecycleResult(tc, result, err)
		})
}

func submitSignedDeadMan(
	ctx context.Context,
	tc ToolContext,
	authenticator tradeActionAuthenticator,
	tradeReads *TradeReadClient,
	timeoutSeconds int64,
	input lifecycleSignatureInput,
) (any, error) {
	validated, envelope, err := buildScheduleCancelPayload(timeoutSeconds)
	if err != nil {
		return nil, err
	}
	return submitSignedLifecycle(ctx, tc, authenticator, tradeReads, "scheduleCancel", validated, input, func(sw SignedWrite) (any, error) {
		return tradeReads.ScheduleCancelWithSignature(ctx, tc, envelope, sw)
	})
}

func submitSignedLifecycle(
	_ context.Context,
	tc ToolContext,
	authenticator tradeActionAuthenticator,
	tradeReads *TradeReadClient,
	action string,
	signPayload any,
	input lifecycleSignatureInput,
	submit func(SignedWrite) (any, error),
) (any, error) {
	if tradeReads == nil {
		return nil, ErrWriteUnavailable
	}
	sig := mapSignature(input.Signature)
	if err := authenticator.ValidateTradeAction(tc.State.WalletAddress, tc.State.SubAccountID, input.Nonce, input.ExpiresAfter, snx_lib_api_types.RequestAction(action), signPayload, sig); err != nil {
		return nil, err
	}
	return submit(SignedWrite{
		WalletAddress: tc.State.WalletAddress,
		Nonce:         input.Nonce,
		ExpiresAfter:  input.ExpiresAfter,
		Signature:     sig,
	})
}

func lifecycleResult(tc ToolContext, result any, err error) (*mcp.CallToolResult, lifecycleRawOutput, error) {
	if err != nil {
		return toolErrorResponse[lifecycleRawOutput](err)
	}
	return nil, lifecycleRawOutput{
		Meta:   newResponseMeta(authModeForState(tc.State)),
		Result: result,
	}, nil
}

func buildUpdateLeveragePayload(symbol string, leverage string) (*validation.ValidatedUpdateLeverageAction, map[string]any, error) {
	payload := &validation.UpdateLeverageActionPayload{
		Action:   "updateLeverage",
		Symbol:   validation.Symbol(normalizeSymbol(symbol)),
		Leverage: leverage,
	}
	validated, err := validation.NewValidatedUpdateLeverageAction(payload)
	return validated, map[string]any{"action": "updateLeverage", "symbol": normalizeSymbol(symbol), "leverage": leverage}, err
}

func buildWithdrawCollateralPayload(symbol string, amount string, destination string) (*validation.ValidatedWithdrawCollateralAction, map[string]any, error) {
	asset := strings.ToUpper(strings.TrimSpace(symbol))
	payload := &validation.WithdrawCollateralActionPayload{
		Action:      "withdrawCollateral",
		Symbol:      validation.Asset(asset),
		Amount:      amount,
		Destination: validation.WalletAddress(destination),
	}
	validated, err := validation.NewValidatedWithdrawCollateralAction(payload)
	return validated, map[string]any{"action": "withdrawCollateral", "symbol": asset, "amount": amount, "destination": destination}, err
}

func buildTransferCollateralPayload(toSubAccountID int64, symbol string, amount string) (*validation.ValidatedTransferCollateralAction, map[string]any, error) {
	if toSubAccountID <= 0 {
		return nil, nil, fmt.Errorf("toSubAccountId is required and must be positive")
	}
	asset := strings.ToUpper(strings.TrimSpace(symbol))
	to := fmt.Sprintf("%d", toSubAccountID)
	payload := &validation.TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     to,
		Symbol: validation.Asset(asset),
		Amount: amount,
	}
	validated, err := validation.NewValidatedTransferCollateralAction(payload)
	return validated, map[string]any{"action": "transferCollateral", "to": to, "symbol": asset, "amount": amount}, err
}

func buildScheduleCancelPayload(timeoutSeconds int64) (*validation.ValidatedScheduleCancelAction, map[string]any, error) {
	if timeoutSeconds > maxDeadManTimeoutSeconds {
		return nil, nil, fmt.Errorf("timeoutSeconds must be less than or equal to %d", maxDeadManTimeoutSeconds)
	}
	timeout := timeoutSeconds
	payload := &validation.ScheduleCancelActionPayload{
		Action:         "scheduleCancel",
		TimeoutSeconds: &timeout,
	}
	validated, err := validation.NewValidatedScheduleCancelAction(payload)
	return validated, map[string]any{"action": "scheduleCancel", "timeoutSeconds": timeoutSeconds}, err
}

func buildAddDelegatedSignerPayload(delegate string, permissions []string, expiresAt int64) (*validation.ValidatedAddDelegatedSignerAction, types.AddDelegatedSignerAction, error) {
	expiresAtPtr := &expiresAt
	if expiresAt == 0 {
		expiresAtPtr = nil
	}
	payload := &validation.AddDelegatedSignerActionPayload{
		Action:          "addDelegatedSigner",
		DelegateAddress: validation.WalletAddress(delegate),
		Permissions:     permissions,
		ExpiresAt:       expiresAtPtr,
	}
	validated, err := validation.NewValidatedAddDelegatedSignerAction(payload)
	return validated, types.AddDelegatedSignerAction{
		Action:        "addDelegatedSigner",
		WalletAddress: delegate,
		Permissions:   permissions,
		ExpiresAt:     expiresAt,
	}, err
}

func buildRemoveDelegatedSignerPayload(delegate string) (*validation.ValidatedRemoveDelegatedSignerAction, map[string]any, error) {
	payload := &validation.RemoveDelegatedSignerActionPayload{
		Action:          "removeDelegatedSigner",
		DelegateAddress: validation.WalletAddress(delegate),
	}
	validated, err := validation.NewValidatedRemoveDelegatedSignerAction(payload)
	return validated, map[string]any{"action": "removeDelegatedSigner", "walletAddress": delegate}, err
}

func buildRemoveAllDelegatedSignersPayload() (*validation.ValidatedRemoveAllDelegatedSignersAction, types.RemoveAllDelegatedSignersAction, error) {
	payload := &validation.RemoveAllDelegatedSignersActionPayload{Action: "removeAllDelegatedSigners"}
	validated, err := validation.NewValidatedRemoveAllDelegatedSignersAction(payload)
	return validated, types.RemoveAllDelegatedSignersAction{Action: "removeAllDelegatedSigners"}, err
}

func storeDeadManState(tc ToolContext, timeoutSeconds int64) {
	if tc.SessionID == "" || tc.State == nil {
		return
	}
	deadManSwitchCache.Store(tc.SessionID, deadManSwitchState{
		SubAccountID:    tc.State.SubAccountID,
		TimeoutSeconds:  timeoutSeconds,
	})
}

func deadManStatus(tc ToolContext) deadManStatusOutput {
	out := deadManStatusOutput{
		Meta:            newResponseMeta(authModeForState(tc.State)),
		NotKnownToServer: true,
	}
	if tc.State != nil {
		out.SubAccountID = tc.State.SubAccountID
	}
	if raw, ok := deadManSwitchCache.Load(tc.SessionID); ok {
		if state, ok := raw.(deadManSwitchState); ok {
			out.Armed = state.TimeoutSeconds > 0
			out.LastTimeoutSeconds = state.TimeoutSeconds
			out.NotKnownToServer = false
			out.SubAccountID = state.SubAccountID
		}
	}
	return out
}

