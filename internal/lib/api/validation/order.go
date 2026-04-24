package validation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-viper/mapstructure/v2"
	shopspring_decimal "github.com/shopspring/decimal"
	angols_slices "github.com/synesissoftware/ANGoLS/slices"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation/utils"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_runtime_deadmanswitch "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/deadmanswitch"
	snx_lib_snaxpot "github.com/Fenway-snx/synthetix-mcp/internal/lib/snaxpot"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// SourceMaxLength is the maximum allowed length for the source field.
// This matches the varchar(100) constraint in the database.
const SourceMaxLength = 100

// Action-type specific errors (kept local as they're validation-flow specific)

const (
	errmsgQuantityInvalid  = "quantity is invalid"
	errmsgQuantityMissing  = "quantity is missing"
	errmsgQuantityNegative = "quantity is negative"
	errmsgQuantityZero     = "quantity is zero"
)

var (
	errActionPayloadRequired                     = errors.New("action payload is required")
	errActionTypeMustBeAddDelegatedSigner        = errors.New("action type must be addDelegatedSigner")
	errActionTypeMustBeCancelAllOrders           = errors.New("action type must be 'cancelAllOrders'")
	errActionTypeMustBeCancelOrders              = errors.New("action type must be 'cancelOrders'")
	errActionTypeMustBeClearSnaxpotPreference    = errors.New("action type must be 'clearSnaxpotPreference'")
	errActionTypeMustBeCreateSubaccount          = errors.New("action type must be createSubaccount")
	errActionTypeMustBeModifyOrder               = errors.New("action type must be 'modifyOrder'")
	errActionTypeMustBeModifyOrderBatch          = errors.New("action type must be 'modifyOrderBatch'")
	errActionTypeMustBePlaceOrders               = errors.New("action type must be 'placeOrders'")
	errActionTypeMustBeRemoveAllDelegatedSigners = errors.New("action type must be removeAllDelegatedSigners")
	errActionTypeMustBeRemoveDelegatedSigner     = errors.New("action type must be removeDelegatedSigner")
	errActionTypeMustBeSaveSnaxpotTickets        = errors.New("action type must be 'saveSnaxpotTickets'")
	errActionTypeMustBeScheduleCancel            = errors.New("action type must be 'scheduleCancel'")
	errActionTypeMustBeSetSnaxpotPreference      = errors.New("action type must be 'setSnaxpotPreference'")
	errActionTypeMustBeUpdateLeverage            = errors.New("action type must be 'updateLeverage'")
	errActionTypeMustBeUpdateSubAccountName      = errors.New("action type must be 'updateSubAccountName'")
	errActionTypeMustBeWithdrawCollateral        = errors.New("action type must be 'withdrawCollateral'")
	errCanonicalDelegateAddress                  = errors.New("walletAddress must not include leading or trailing whitespace")
	errCanonicalDestination                      = errors.New("destination must not include leading or trailing whitespace")
	errClientOrderIdMustBeNonempty               = errors.New("clientOrderId: input must be nonempty")
	errDelegateAddressMustBeValidEthAddress      = errors.New("delegate address must be a valid Ethereum address")
	errDelegateIsRequired                        = errors.New("delegate is required")
	errExactlyOnePermissionRequired              = errors.New("exactly one permission must be specified")
	errInvalidPermissionValue                    = errors.New("invalid permission value: must be 'delegate', 'session', or 'trading'")
	errModifyOrderBatchEmpty                     = errors.New("modifyOrderBatch: orders array cannot be empty")
	errNameIsRequired                            = errors.New("name is required")
	errNameMustBeSafeAndHumanReadable            = errors.New("name must be a safe and human readable name")
	errOrderIdMustBePositiveInteger              = errors.New("orderId must be a positive integer")
	errOrderIdMustBeValidInteger                 = errors.New("orderId must be a valid integer")
	errOrderIdRequired                           = errors.New("orderId is required")
	errQuantityMustBePositiveValue               = errors.New("quantity must be a positive value")
	errQuantityMustBeValidDecimalNumber          = errors.New("quantity must be a valid decimal number")
	errScheduleCancelTimeoutSecondsRequired      = errors.New("timeoutSeconds is required")
	errSignatureRAndSFieldsRequired         = errors.New("signature r and s fields are required")
	errSnaxpotEntriesRequired               = errors.New("entries are required")
	errSnaxpotScopeInvalid                  = errors.New("scope must be 'currentEpoch' or 'persistent'")
	errSnaxpotSnaxBallInvalid              = errors.New("snaxBall must be between 1 and 5")
	errSnaxpotStandardBallsInvalid         = errors.New("balls must use five unique standard balls in ascending order")
	errSnaxpotTicketSerialRequired         = errors.New("ticketSerial is required")
	errSnaxpotTicketSerialValidNumber      = errors.New("ticketSerial must be a valid number")
)

// Holds the typed payload for a placeOrders action.
// All fields are represented using the normalized types expected by internal services.
type PlaceOrdersActionPayload struct {
	Action   RequestAction                        `mapstructure:"action" json:"action"`
	Orders   []snx_lib_api_json.PlaceOrderRequest `mapstructure:"orders" json:"orders"`
	Grouping GroupingValues                       `mapstructure:"grouping" json:"grouping"`
	// Symbol optional: when set, orders that omit symbol inherit this value (canonical form required).
	Symbol Symbol `mapstructure:"symbol" json:"symbol,omitempty"`
	Source string `mapstructure:"source" json:"source,omitempty"`
}

// Captures the cancelAllOrders action payload before validation.
type CancelAllOrdersActionPayload struct {
	Action  RequestAction `mapstructure:"action" json:"action"`
	Symbols []Symbol      `mapstructure:"symbols" json:"symbols"`
}

// Captures the cancelOrders action payload before validation.
type CancelOrdersActionPayload struct {
	Action        RequestAction  `mapstructure:"action" json:"action"`
	VenueOrderIds []VenueOrderId `mapstructure:"orderIds" json:"orderIds"`
}

// Captures the cancelOrders-by-cloid payload before validation.
type CancelOrdersByCloidActionPayload struct {
	Action         RequestAction   `mapstructure:"action" json:"action"`
	ClientOrderIds []ClientOrderId `mapstructure:"clientOrderIds" json:"clientOrderIds"`
}

// Captures the scheduleCancel action payload before validation.
type ScheduleCancelActionPayload struct {
	Action         RequestAction `mapstructure:"action" json:"action"`
	TimeoutSeconds *int64        `mapstructure:"timeoutSeconds" json:"timeoutSeconds"`
}

// Strict payload-only struct for modifyOrder action.
//
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type ModifyOrderActionPayload struct {
	Action       RequestAction `mapstructure:"action" json:"action"`
	VenueOrderId VenueOrderId  `mapstructure:"orderId" json:"orderId"`
	Price        *Price        `mapstructure:"price" json:"price,omitempty"`
	Quantity     *Quantity     `mapstructure:"quantity" json:"quantity,omitempty"`
	TriggerPrice *Price        `mapstructure:"triggerPrice" json:"triggerPrice,omitempty"`
}

// Captures the modifyOrder-by-cloid payload before validation.
type ModifyOrderByCloidActionPayload struct {
	Action        RequestAction `mapstructure:"action" json:"action"`
	ClientOrderId ClientOrderId `mapstructure:"clientOrderId" json:"clientOrderId"`
	Price         *Price        `mapstructure:"price" json:"price,omitempty"`
	Quantity      *Quantity     `mapstructure:"quantity" json:"quantity,omitempty"`
	TriggerPrice  *Price        `mapstructure:"triggerPrice" json:"triggerPrice,omitempty"`
}

// A strict payload-only struct for withdrawCollateral action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type WithdrawCollateralActionPayload struct {
	Action      RequestAction `mapstructure:"action" json:"action"`
	Symbol      Asset         `mapstructure:"symbol" json:"symbol"` // TODO: SNX-6098: rename to `AssetName`
	Amount      string        `mapstructure:"amount" json:"amount"`
	Destination WalletAddress `mapstructure:"destination" json:"destination"`
}

// A strict payload-only struct for addDelegatedSigner action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type AddDelegatedSignerActionPayload struct {
	Action          RequestAction `mapstructure:"action" json:"action"`
	DelegateAddress WalletAddress `mapstructure:"walletAddress" json:"walletAddress"`
	Permissions     []string      `mapstructure:"permissions" json:"permissions"`
	ExpiresAt       *int64        `mapstructure:"expiresAt" json:"expiresAt,omitempty"`
}

// A strict payload-only struct for removeDelegatedSigner action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type RemoveDelegatedSignerActionPayload struct {
	Action          RequestAction `mapstructure:"action" json:"action"`
	DelegateAddress WalletAddress `mapstructure:"walletAddress" json:"walletAddress"`
}

// A strict payload-only struct for removeAllDelegatedSigners action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type RemoveAllDelegatedSignersActionPayload struct {
	Action RequestAction `mapstructure:"action" json:"action"`
}

// A strict payload-only struct for updateLeverage action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type UpdateLeverageActionPayload struct {
	Action   RequestAction `mapstructure:"action" json:"action"`
	Symbol   Symbol        `mapstructure:"symbol" json:"symbol"`
	Leverage string        `mapstructure:"leverage" json:"leverage"`
}

// A strict payload-only struct for createSubaccount action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type CreateSubaccountActionPayload struct {
	Action RequestAction `mapstructure:"action" json:"action"`
	Name   string        `mapstructure:"name" json:"name"`
}

// A strict payload-only struct for updateSubAccountName action
// It intentionally excludes auth-related fields (nonce, signature, wallet, subaccount)
// to keep validation focused on the action payload only.
type UpdateSubAccountNameActionPayload struct {
	Action RequestAction `mapstructure:"action" json:"action"`
	Name   string        `mapstructure:"name" json:"name"`
}

type ClearSnaxpotPreferenceActionPayload struct {
	Action RequestAction `mapstructure:"action" json:"action"`
	Scope  string        `mapstructure:"scope" json:"scope"`
}

type SaveSnaxpotTicketsActionPayload struct {
	Action  RequestAction                       `mapstructure:"action" json:"action"`
	Entries []SnaxpotTicketMutationEntryPayload `mapstructure:"entries" json:"entries"`
}

type SetSnaxpotPreferenceActionPayload struct {
	Action   RequestAction `mapstructure:"action" json:"action"`
	Scope    string        `mapstructure:"scope" json:"scope"`
	SnaxBall int32         `mapstructure:"snaxBall" json:"snaxBall"`
}

type SnaxpotTicketMutationEntryPayload struct {
	Ball1        int32  `mapstructure:"ball1" json:"ball1"`
	Ball2        int32  `mapstructure:"ball2" json:"ball2"`
	Ball3        int32  `mapstructure:"ball3" json:"ball3"`
	Ball4        int32  `mapstructure:"ball4" json:"ball4"`
	Ball5        int32  `mapstructure:"ball5" json:"ball5"`
	SnaxBall     int32  `mapstructure:"snaxBall" json:"snaxBall"`
	TicketSerial string `mapstructure:"ticketSerial" json:"ticketSerial"`
}

// Represents a single order modification within a batch.
type ModifyOrderBatchItem struct {
	VenueOrderId VenueOrderId `mapstructure:"orderId" json:"orderId"`
	Price        *Price       `mapstructure:"price" json:"price,omitempty"`
	Quantity     *Quantity    `mapstructure:"quantity" json:"quantity,omitempty"`
	TriggerPrice *Price       `mapstructure:"triggerPrice" json:"triggerPrice,omitempty"`
}

// Holds the typed payload for a modifyOrderBatch action.
type ModifyOrderBatchActionPayload struct {
	Action RequestAction          `mapstructure:"action" json:"action"`
	Orders []ModifyOrderBatchItem `mapstructure:"orders" json:"orders"`
}

// Holds a validated single order modification.
type ValidatedModifyOrderBatchItem struct {
	VenueOrderId VenueOrderId
	Price        *Price
	Quantity     *Quantity
	TriggerPrice *Price
}

// Bundles a validated modifyOrderBatch payload.
type ValidatedModifyOrderBatchAction struct {
	Payload *ModifyOrderBatchActionPayload
	Orders  []ValidatedModifyOrderBatchItem
}

// Bundles a validated placeOrders payload.
type ValidatedPlaceOrdersAction struct {
	Payload *PlaceOrdersActionPayload
}

// Bundles a validated modifyOrder payload with parsed order ID.
type ValidatedModifyOrderAction struct {
	Payload      *ModifyOrderActionPayload
	VenueOrderId VenueOrderId
}

// Bundles a validated modifyOrder-by-cloid payload.
type ValidatedModifyOrderByCloidAction struct {
	ClientOrderId ClientOrderId
	Payload       *ModifyOrderByCloidActionPayload
}

// Bundles a validated cancelAllOrders payload with parsed IDs.
type ValidatedCancelAllOrdersAction struct {
	Payload *CancelAllOrdersActionPayload
	Symbols []Symbol
}

// Bundles a validated cancelOrders payload with parsed IDs.
type ValidatedCancelOrdersAction struct {
	Payload       *CancelOrdersActionPayload
	VenueOrderIds []VenueOrderId
}

// Bundles a validated cancelOrders-by-cloid payload.
type ValidatedCancelOrdersByCloidAction struct {
	ClientOrderIds []ClientOrderId
	Payload        *CancelOrdersByCloidActionPayload
}

// Bundles a validated scheduleCancel payload.
type ValidatedScheduleCancelAction struct {
	Payload        *ScheduleCancelActionPayload
	TimeoutSeconds int64
}

// Bundles a validated withdrawCollateral payload.
type ValidatedWithdrawCollateralAction struct {
	Payload *WithdrawCollateralActionPayload
}

// Bundles a validated addDelegatedSigner payload.
type ValidatedAddDelegatedSignerAction struct {
	Payload *AddDelegatedSignerActionPayload
}

// Bundles a validated removeDelegatedSigner payload.
type ValidatedRemoveDelegatedSignerAction struct {
	Payload *RemoveDelegatedSignerActionPayload
}

// Bundles a validated removeAllDelegatedSigners payload.
type ValidatedRemoveAllDelegatedSignersAction struct {
	Payload *RemoveAllDelegatedSignersActionPayload
}

// Bundles a validated updateLeverage payload.
type ValidatedUpdateLeverageAction struct {
	Payload *UpdateLeverageActionPayload
}

// Bundles a validated createSubaccount payload.
type ValidatedCreateSubaccountAction struct {
	Payload *CreateSubaccountActionPayload
}

// Bundles a validated updateSubAccountName payload.
type ValidatedUpdateSubAccountNameAction struct {
	Payload *UpdateSubAccountNameActionPayload
}

type ValidatedClearSnaxpotPreferenceAction struct {
	Payload *ClearSnaxpotPreferenceActionPayload
}

type ValidatedSaveSnaxpotTicketsAction struct {
	Payload *SaveSnaxpotTicketsActionPayload
}

type ValidatedSetSnaxpotPreferenceAction struct {
	Payload *SetSnaxpotPreferenceActionPayload
}

// This function required because mapstructure.Decode does not utilise in
// any way the unmarshaling of `VenueOrderId`.
//
// NOTE: determine and apply universally (to all API types) a mechanism that
// is applied in all contexts.
func validateVenueOrderId(venueOrderId VenueOrderId) (VenueOrderId, error) {

	s := string(venueOrderId)

	s = strings.TrimSpace(s)

	if s == "" {
		return VenueOrderId_None, errOrderIdRequired
	}

	// parse as signed integer to detect outlandishly large numbers that may
	// arise, for example, if a small positive number is subject to undue
	// decrementing/subtraction on the client side.
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return VenueOrderId_None, errOrderIdMustBeValidInteger
	}
	if i < 1 {
		return VenueOrderId_None, errOrderIdMustBePositiveInteger
	}

	return VenueOrderId(strconv.FormatInt(i, 10)), nil
}

// Checks if the given string is a valid Ethereum address
// with a 0x prefix and exactly 42 characters (0x + 40 hex chars).
func isValidEthereumAddress(addr WalletAddress) bool {
	return common.IsHexAddress(string(addr)) && len(addr) == 42 && addr[:2] == "0x"
}

// Validates that a numeric value is non-negative.
func ValidateNonNegative(value int, fieldName string) error {
	if value < 0 {
		return fmt.Errorf("%s must be non-negative", fieldName)
	}
	return nil
}

// Validates that a numeric value doesn't exceed a maximum limit.
func ValidateMaxLimit(value, maxLimit int, fieldName string) error {
	if value > maxLimit {
		return fmt.Errorf("%s cannot exceed %d", fieldName, maxLimit)
	}
	return nil
}

// Validates that a string value is in a set of allowed values. Pass context
// as a string for static labels, or as func() string to defer building the
// label until an invalid value is detected.
func ValidateEnum[T deferredValidationLabel](value string, allowedValues []string, context T) error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	var label string
	switch c := any(context).(type) {
	case string:
		label = c
	case func() string:
		label = c()
	}

	return fmt.Errorf("%s must be one of: %v", label, allowedValues)
}

// Validates the source field using client-side string validation.
// Empty string is treated as "not specified" and is valid.
// Returns the validated (trimmed) string and error if validation fails.
func ValidateSource(source string) (string, error) {
	// Empty string means "not specified" - this is valid
	if source == "" {
		return "", nil
	}

	// Use ValidateClientSideString to prevent SQL injection and other attacks
	// Trim whitespace - if it becomes empty after trimming, ValidateClientSideString
	// will return ("", nil) which is correct (empty means "not specified")
	validatedSource, err := snx_lib_api_validation_utils.ValidateClientSideString(
		source,
		SourceMaxLength,
		snx_lib_api_validation_utils.ClientSideOption_Trim,
	)
	if err != nil {
		return "", fmt.Errorf("invalid source attribute: %w", err)
	}

	return validatedSource, nil
}

// Validates timestamp range constraints
func ValidateTimestampRange(
	startTime, endTime Timestamp,
	maxDuration time.Duration,
	fieldName string,
) error {
	if startTime < 0 {
		return fmt.Errorf("%s startTime must be non-negative", fieldName)
	}
	if endTime < 0 {
		return fmt.Errorf("%s endTime must be non-negative", fieldName)
	}
	if startTime != 0 && endTime != 0 {
		if startTime > endTime {
			return fmt.Errorf("%s startTime must be before endTime", fieldName)
		}
		if endTime.Sub(startTime) > maxDuration {
			return fmt.Errorf("%s time range cannot exceed %d seconds", fieldName, int64(maxDuration.Seconds()))
		}
	}
	return nil
}

// Decodes a raw map payload into a typed placeOrders payload.
func DecodePlaceOrdersAction(action map[string]any) (*PlaceOrdersActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload PlaceOrdersActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid placeOrders payload: %w", err)
	}

	return &payload, nil
}

// Enforces placeOrders-specific constraints on a typed payload.
func ValidatePlaceOrdersAction(action *PlaceOrdersActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	if action.Action != "" && action.Action != "placeOrders" {
		return errActionTypeMustBePlaceOrders
	}

	if len(action.Orders) == 0 {
		return ErrOrdersArrayEmpty
	}
	if len(action.Orders) > MaxOrdersPerBatch {
		return fmt.Errorf("orders cannot exceed %d per request", MaxOrdersPerBatch)
	}

	if strings.TrimSpace(string(action.Symbol)) != "" {
		if err := ValidateCanonicalSymbol(action.Symbol, "symbol"); err != nil {
			return err
		}
	}

	for i, order := range action.Orders {
		if err := ValidateStringMaxLength(order.Side, MaxEnumFieldLength, func() string {
			return fmt.Sprintf("order %d: %s", i, API_WKS_side)
		}); err != nil {
			return err
		}
		if err := ValidateStringMaxLength(order.OrderType, MaxEnumFieldLength, func() string {
			return fmt.Sprintf("order %d: orderType", i)
		}); err != nil {
			return err
		}
		if err := ValidateStringMaxLength(order.Price, MaxDecimalStringLength, func() string {
			return fmt.Sprintf("order %d: price", i)
		}); err != nil {
			return err
		}
		if err := ValidateStringMaxLength(order.Quantity, MaxDecimalStringLength, func() string {
			return fmt.Sprintf("order %d: quantity", i)
		}); err != nil {
			return err
		}
		if err := ValidateStringMaxLength(order.TriggerPrice, MaxDecimalStringLength, func() string {
			return fmt.Sprintf("order %d: triggerPrice", i)
		}); err != nil {
			return err
		}

		// TODO: move this function to appropriate place
		validateOrderQuantity := func(i int, order *snx_lib_api_json.PlaceOrderRequest) error {

			if order.ClosePosition {
				return nil
			}

			if order.Quantity == Quantity_None {
				return fmt.Errorf("order %d: %s", i, errmsgQuantityMissing)
			}

			if qd, err := shopspring_decimal.NewFromString(string(order.Quantity)); err != nil {
				return fmt.Errorf("order %d: %s: %w", i, errmsgQuantityInvalid, err)
			} else {
				if qd.IsNegative() {
					return fmt.Errorf("order %d: %s", i, errmsgQuantityNegative)
				}
				if qd.IsZero() {
					return fmt.Errorf("order %d: %s", i, errmsgQuantityZero)
				}
			}

			return nil
		}
		if err := validateOrderQuantity(i, &order); err != nil {
			return err
		}

		if err := ValidateEnum(order.Side, []string{"buy", "sell"}, func() string {
			return fmt.Sprintf("order %d: %s", i, API_WKS_side)
		}); err != nil {
			return err
		}

		symbol := order.Symbol
		if symbol == Symbol_None {
			symbol = action.Symbol
		}
		if err := ValidateCanonicalSymbol(symbol, "symbol"); err != nil {
			return fmt.Errorf("order %d: %w", i, err)
		}
		action.Orders[i].Symbol = symbol

		if err := snx_lib_api_json.ValidateOrderTypeConstraints(order, i); err != nil {
			return err
		}

		if order.ExpiresAt != nil {
			if *order.ExpiresAt <= snx_lib_utils_time.Now().Unix() {
				return fmt.Errorf("order %d: expiresAt must be in the future", i)
			}
		}

		if order.OrderType == API_WKS_triggerTp || order.OrderType == API_WKS_triggerSl {
			tpt, err := snx_lib_api_types.APITriggerPriceTypeFromString(string(order.TriggerPriceType))
			if err != nil {
				return fmt.Errorf("order %d: %w", i, err)
			}
			action.Orders[i].TriggerPriceType = tpt
		}

		if order.ClientOrderId != ClientOrderId_Empty {
			if validatedCLOID, err := snx_lib_api_types.ValidateClientOrderId(order.ClientOrderId); err != nil {
				context := fmt.Sprintf("order %d: %s", i, API_WKS_clientOrderId)

				return fmt.Errorf("%s: %w", context, err)
			} else {
				action.Orders[i].ClientOrderId = validatedCLOID
			}
		}
	}

	if action.Grouping != "" {
		switch action.Grouping {
		case
			GroupingValues_na,
			GroupingValues_normalTpsl,
			GroupingValues_positionsTpsl,
			GroupingValues_twap:

		default:
			return ErrInvalidGrouping
		}
	}

	validatedSource, err := ValidateSource(action.Source)
	if err != nil {
		return err
	}
	action.Source = validatedSource

	return nil
}

// Decodes a raw map payload into a typed cancelAllOrders payload.
func DecodeCancelAllOrdersAction(action map[string]any) (*CancelAllOrdersActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload CancelAllOrdersActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid cancelAllOrders payload: %w", err)
	}

	return &payload, nil
}

// Decodes a raw map payload into a typed cancelOrders payload.
func DecodeCancelOrdersAction(action map[string]any) (*CancelOrdersActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload CancelOrdersActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid cancelOrders payload: %w", err)
	}

	return &payload, nil
}

// Decodes a raw map payload into a typed cancelOrders-by-cloid payload.
func DecodeCancelOrdersByCloidAction(action map[string]any) (*CancelOrdersByCloidActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload CancelOrdersByCloidActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid cancelOrders payload: %w", err)
	}

	return &payload, nil
}

// Decodes a raw map payload into a typed scheduleCancel payload.
func DecodeScheduleCancelAction(action map[string]any) (*ScheduleCancelActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload ScheduleCancelActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid scheduleCancel payload: %w", err)
	}

	return &payload, nil
}

// Decodes a raw map payload into a typed modifyOrder payload.
func DecodeModifyOrderAction(action map[string]any) (*ModifyOrderActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload ModifyOrderActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid modifyOrder payload: %w", err)
	}

	return &payload, nil
}

// Decodes a raw map payload into a typed modifyOrder-by-cloid payload.
func DecodeModifyOrderByCloidAction(action map[string]any) (*ModifyOrderByCloidActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload ModifyOrderByCloidActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid modifyOrder payload: %w", err)
	}

	return &payload, nil
}

// Validates cancelAllOrders payload and populates parsed IDs.
// Symbols must be non-empty. To cancel all orders across all markets, pass ["*"].
// The wildcard ["*"] must be the only element; mixing "*" with other symbols is invalid.
func ValidateCancelAllOrdersAction(action *CancelAllOrdersActionPayload) ([]Symbol, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	if action.Action != "" && action.Action != "cancelAllOrders" {
		return nil, errActionTypeMustBeCancelAllOrders
	}

	if len(action.Symbols) == 0 {
		return nil, ErrSymbolsMustBeNonempty
	}

	// Wildcard: ["*"] means cancel all. It must be the sole element.
	if len(action.Symbols) == 1 && action.Symbols[0] == "*" {
		return action.Symbols, nil
	}

	// Validate canonical symbol format and reject "*" mixed with others.
	for i, symbol := range action.Symbols {
		if strings.TrimSpace(string(symbol)) == "" {
			return nil, fmt.Errorf("symbols[%d]: symbol cannot be empty or whitespace", i)
		}
		if symbol == "*" {
			return nil, fmt.Errorf("symbols[%d]: wildcard \"*\" must be the only element in symbols", i)
		}
		if err := ValidateCanonicalSymbol(symbol, func() string {
			return fmt.Sprintf("symbols[%d]", i)
		}); err != nil {
			return nil, err
		}
	}

	return action.Symbols, nil
}

// Validates cancelOrders payload and populates parsed IDs.
func ValidateCancelOrdersAction(action *CancelOrdersActionPayload) ([]VenueOrderId, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	if action.Action != "" && action.Action != "cancelOrders" {
		return nil, errActionTypeMustBeCancelOrders
	}

	if len(action.VenueOrderIds) == 0 {
		return nil, ErrOrderIdsMustBeNonempty
	}

	// NOTE: we do not need to to validate the order ids themselves because
	// that is now taken care of by `OrderId#UnmarshalJSON()`, but we do so in
	// order to satisfy the unit-tests.

	if sanitised, err := angols_slices.CollectSlice(action.VenueOrderIds, func(index int, input_item *VenueOrderId) (
		VenueOrderId,
		error,
	) {
		return validateVenueOrderId(*input_item)
	}); err != nil {

		return nil, err
	} else {
		return sanitised, nil
	}
}

// Validates cancelOrders-by-cloid payload and returns parsed IDs.
func ValidateCancelOrdersByCloidAction(action *CancelOrdersByCloidActionPayload) ([]ClientOrderId, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	if action.Action != "" && action.Action != "cancelOrders" {
		return nil, errActionTypeMustBeCancelOrders
	}

	if len(action.ClientOrderIds) == 0 {
		return nil, ErrOrderIdsMustBeNonempty
	}

	clientOrderIds := make([]ClientOrderId, 0, len(action.ClientOrderIds))
	for i, clientOrderId := range action.ClientOrderIds {
		if clientOrderId == ClientOrderId_Empty {
			return nil, fmt.Errorf("clientOrderIds[%d]: %w", i, errClientOrderIdMustBeNonempty)
		}

		validatedCLOID, err := snx_lib_api_types.ValidateClientOrderId(clientOrderId)
		if err != nil {
			return nil, fmt.Errorf("clientOrderIds[%d]: %w", i, err)
		}
		clientOrderIds = append(clientOrderIds, validatedCLOID)
	}

	return clientOrderIds, nil
}

// Validates a scheduleCancel payload and returns the parsed timeout.
func ValidateScheduleCancelAction(action *ScheduleCancelActionPayload) (int64, error) {
	if action == nil {
		return 0, ErrActionPayloadRequired
	}

	if action.Action != "" && action.Action != "scheduleCancel" {
		return 0, errActionTypeMustBeScheduleCancel
	}

	if action.TimeoutSeconds == nil {
		return 0, errScheduleCancelTimeoutSecondsRequired
	}

	bounds, err := snx_lib_runtime_deadmanswitch.LoadTimeoutBounds()
	if err != nil {
		return 0, err
	}

	timeoutSeconds := *action.TimeoutSeconds
	if timeoutSeconds < 0 {
		return 0, fmt.Errorf("timeoutSeconds must be 0 or at least %d", bounds.MinTimeoutSeconds)
	}
	if timeoutSeconds > bounds.MaxTimeoutSeconds {
		return 0, fmt.Errorf("timeoutSeconds must be less than or equal to %d", bounds.MaxTimeoutSeconds)
	}
	if timeoutSeconds != 0 && timeoutSeconds < bounds.MinTimeoutSeconds {
		return 0, fmt.Errorf("timeoutSeconds must be 0 or at least %d", bounds.MinTimeoutSeconds)
	}

	return timeoutSeconds, nil
}

// Validates only the action payload for modifyOrder and returns the parsed
// order ID. Pass errorPrefix as "" for no prefix, a string for a static
// prefix, or func() string to defer building the prefix until an error is
// returned (e.g. modifyOrderBatch order index).
func validateModifyOrderFields[T deferredValidationLabel](
	price *Price,
	quantity *Quantity,
	triggerPrice *Price,
	errorPrefix T,
) error {
	wrapError := func(err error) error {
		prefix := deferredValidationLabelString(errorPrefix)
		if prefix == "" {
			return err
		}

		return fmt.Errorf("%s%w", prefix, err)
	}

	if price == nil && quantity == nil && triggerPrice == nil {
		return wrapError(ErrInvalidModifyOrderPayload)
	}

	if price != nil {
		if err := ValidateStringMaxLength(*price, MaxDecimalStringLength, "price"); err != nil {
			return wrapError(err)
		}

		priceValue, err := shopspring_decimal.NewFromString(string(*price))
		if err != nil {
			return wrapError(ErrPriceMustBeValidDecimal)
		}
		if priceValue.LessThanOrEqual(shopspring_decimal.Zero) {
			return wrapError(ErrPriceMustBePositive)
		}
	}

	if quantity != nil {
		if err := ValidateStringMaxLength(*quantity, MaxDecimalStringLength, "quantity"); err != nil {
			return wrapError(err)
		}

		quantityValue, err := shopspring_decimal.NewFromString(string(*quantity))
		if err != nil {
			return wrapError(errQuantityMustBeValidDecimalNumber)
		}
		if quantityValue.LessThanOrEqual(shopspring_decimal.Zero) {
			return wrapError(errQuantityMustBePositiveValue)
		}
	}

	if triggerPrice != nil {
		if err := ValidateStringMaxLength(*triggerPrice, MaxDecimalStringLength, "triggerPrice"); err != nil {
			return wrapError(err)
		}

		triggerPriceValue, err := shopspring_decimal.NewFromString(string(*triggerPrice))
		if err != nil {
			return wrapError(ErrPriceMustBeValidDecimal)
		}
		if triggerPriceValue.LessThanOrEqual(shopspring_decimal.Zero) {
			return wrapError(ErrPriceMustBePositive)
		}
	}

	return nil
}

func ValidateModifyOrderAction(action *ModifyOrderActionPayload) (VenueOrderId, error) {
	if action == nil {
		return VenueOrderId_None, ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "modifyOrder" {
		return VenueOrderId_None, errActionTypeMustBeModifyOrder
	}

	// NOTE: we do not need to to validate the order id itself because
	// that is now taken care of by `OrderId#UnmarshalJSON()`, but we do so in
	// order to satisfy the unit-tests.

	var venueOrderId VenueOrderId
	var err error
	if venueOrderId, err = validateVenueOrderId(action.VenueOrderId); err != nil {
		return VenueOrderId_None, err
	}

	if err := validateModifyOrderFields(action.Price, action.Quantity, action.TriggerPrice, ""); err != nil {
		return VenueOrderId_None, err
	}

	return venueOrderId, nil
}

// Validates modifyOrder-by-cloid payload and returns the parsed client order ID.
func ValidateModifyOrderByCloidAction(action *ModifyOrderByCloidActionPayload) (ClientOrderId, error) {
	if action == nil {
		return ClientOrderId_Empty, ErrActionPayloadRequired
	}

	if action.Action != "modifyOrder" {
		return ClientOrderId_Empty, errActionTypeMustBeModifyOrder
	}

	if action.ClientOrderId == ClientOrderId_Empty {
		return ClientOrderId_Empty, errClientOrderIdMustBeNonempty
	}

	validatedCLOID, err := snx_lib_api_types.ValidateClientOrderId(action.ClientOrderId)
	if err != nil {
		return ClientOrderId_Empty, fmt.Errorf("clientOrderId: %w", err)
	}

	if err := validateModifyOrderFields(action.Price, action.Quantity, action.TriggerPrice, ""); err != nil {
		return ClientOrderId_Empty, err
	}

	return validatedCLOID, nil
}

// Validates and wraps a placeOrders payload.
func NewValidatedPlaceOrdersAction(payload *PlaceOrdersActionPayload) (*ValidatedPlaceOrdersAction, error) {
	if err := ValidatePlaceOrdersAction(payload); err != nil {
		return nil, err
	}

	return &ValidatedPlaceOrdersAction{Payload: payload}, nil
}

// Validates and wraps a modifyOrder payload.
func NewValidatedModifyOrderAction(payload *ModifyOrderActionPayload) (*ValidatedModifyOrderAction, error) {
	venueOrderId, err := ValidateModifyOrderAction(payload)
	if err != nil {
		return nil, err
	}

	payload.VenueOrderId = venueOrderId

	return &ValidatedModifyOrderAction{Payload: payload, VenueOrderId: venueOrderId}, nil
}

// Validates and wraps a modifyOrder-by-cloid payload.
func NewValidatedModifyOrderByCloidAction(payload *ModifyOrderByCloidActionPayload) (*ValidatedModifyOrderByCloidAction, error) {
	clientOrderId, err := ValidateModifyOrderByCloidAction(payload)
	if err != nil {
		return nil, err
	}

	payload.ClientOrderId = clientOrderId

	return &ValidatedModifyOrderByCloidAction{ClientOrderId: clientOrderId, Payload: payload}, nil
}

const maxModifyOrderBatchSize = 10

// Validates a modifyOrderBatch payload.
// Each order is validated using the same rules as single modifyOrder.
//
// TODO: remove this if there is not intent to support batch order modification
func ValidateModifyOrderBatchAction(action *ModifyOrderBatchActionPayload) ([]ValidatedModifyOrderBatchItem, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	if action.Action != "" && action.Action != "modifyOrderBatch" {
		return nil, errActionTypeMustBeModifyOrderBatch
	}

	if len(action.Orders) == 0 {
		return nil, errModifyOrderBatchEmpty
	}

	if len(action.Orders) > maxModifyOrderBatchSize {
		return nil, fmt.Errorf("modifyOrderBatch: exceeds maximum batch size of %d", maxModifyOrderBatchSize)
	}

	validatedOrders := make([]ValidatedModifyOrderBatchItem, 0, len(action.Orders))
	for i, order := range action.Orders {
		venueOrderId, err := validateVenueOrderId(order.VenueOrderId)
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}

		if err := validateModifyOrderFields(
			order.Price,
			order.Quantity,
			order.TriggerPrice,
			func() string {
				return fmt.Sprintf("order %d: ", i)
			},
		); err != nil {
			return nil, err
		}

		validatedOrders = append(validatedOrders, ValidatedModifyOrderBatchItem{
			VenueOrderId: venueOrderId,
			Price:        order.Price,
			Quantity:     order.Quantity,
			TriggerPrice: order.TriggerPrice,
		})
	}

	return validatedOrders, nil
}

// Validates and wraps a cancelAllOrders payload.
func NewValidatedCancelAllOrdersAction(payload *CancelAllOrdersActionPayload) (*ValidatedCancelAllOrdersAction, error) {
	symbols, err := ValidateCancelAllOrdersAction(payload)
	if err != nil {
		return nil, err
	}
	return &ValidatedCancelAllOrdersAction{
		Payload: payload,
		Symbols: symbols,
	}, nil
}

// Validates and wraps a cancelOrders payload.
func NewValidatedCancelOrdersAction(payload *CancelOrdersActionPayload) (*ValidatedCancelOrdersAction, error) {
	venueOrderIds, err := ValidateCancelOrdersAction(payload)
	if err != nil {
		return nil, err
	}
	return &ValidatedCancelOrdersAction{Payload: payload, VenueOrderIds: venueOrderIds}, nil
}

// Validates and wraps a cancelOrders-by-cloid payload.
func NewValidatedCancelOrdersByCloidAction(payload *CancelOrdersByCloidActionPayload) (*ValidatedCancelOrdersByCloidAction, error) {
	clientOrderIds, err := ValidateCancelOrdersByCloidAction(payload)
	if err != nil {
		return nil, err
	}

	payload.ClientOrderIds = clientOrderIds

	return &ValidatedCancelOrdersByCloidAction{ClientOrderIds: clientOrderIds, Payload: payload}, nil
}

// Validates and wraps a scheduleCancel payload.
func NewValidatedScheduleCancelAction(payload *ScheduleCancelActionPayload) (*ValidatedScheduleCancelAction, error) {
	timeoutSeconds, err := ValidateScheduleCancelAction(payload)
	if err != nil {
		return nil, err
	}
	return &ValidatedScheduleCancelAction{Payload: payload, TimeoutSeconds: timeoutSeconds}, nil
}

// Decodes a raw map payload into a typed withdrawCollateral payload.
func DecodeWithdrawCollateralAction(action map[string]any) (*WithdrawCollateralActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload WithdrawCollateralActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid withdrawCollateral payload: %w", err)
	}

	return &payload, nil
}

// Validates withdrawCollateral payload.
func ValidateWithdrawCollateralAction(action *WithdrawCollateralActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "withdrawCollateral" {
		return errActionTypeMustBeWithdrawCollateral
	}

	// Reject symbols that only become valid after trimming or uppercasing.
	if err := ValidateCanonicalCollateralSymbol(action.Symbol, "symbol"); err != nil {
		return err
	}

	// Validate amount is provided and positive
	if action.Amount == "" {
		return ErrAmountInvalid
	}
	if err := ValidateStringMaxLength(action.Amount, MaxDecimalStringLength, "amount"); err != nil {
		return err
	}

	amount, err := shopspring_decimal.NewFromString(action.Amount)
	if err != nil {
		return ErrAmountMustBeValidDecimal
	}

	if !amount.IsPositive() {
		return ErrAmountInvalid
	}

	// Validate destination address
	if action.Destination == "" {
		return ErrDestinationRequired
	}
	if strings.TrimSpace(string(action.Destination)) != string(action.Destination) {
		return errCanonicalDestination
	}
	if err := ValidateStringMaxLength(action.Destination, MaxEthAddressLength, "destination"); err != nil {
		return err
	}

	// Validate destination is a valid Ethereum address with 0x prefix
	if !isValidEthereumAddress(action.Destination) {
		return ErrDestinationInvalidAddress
	}

	return nil
}

// Validates and wraps a withdrawCollateral payload.
func NewValidatedWithdrawCollateralAction(payload *WithdrawCollateralActionPayload) (*ValidatedWithdrawCollateralAction, error) {
	if err := ValidateWithdrawCollateralAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedWithdrawCollateralAction{Payload: payload}, nil
}

// Decodes a raw map payload into a typed addDelegatedSigner payload.
func DecodeAddDelegatedSignerAction(action map[string]any) (*AddDelegatedSignerActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload AddDelegatedSignerActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid addDelegatedSigner payload: %w", err)
	}

	return &payload, nil
}

// Validates addDelegatedSigner payload.
func ValidateAddDelegatedSignerAction(action *AddDelegatedSignerActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "addDelegatedSigner" {
		return errActionTypeMustBeAddDelegatedSigner
	}

	// Validate delegate address is provided and valid
	if action.DelegateAddress == "" {
		return errDelegateIsRequired
	}
	if strings.TrimSpace(string(action.DelegateAddress)) != string(action.DelegateAddress) {
		return errCanonicalDelegateAddress
	}
	if err := ValidateStringMaxLength(action.DelegateAddress, MaxEthAddressLength, "walletAddress"); err != nil {
		return err
	}
	if !isValidEthereumAddress(action.DelegateAddress) {
		return errDelegateAddressMustBeValidEthAddress
	}

	// Validate exactly one permission (required for EIP-712 string[] type)
	if len(action.Permissions) != 1 {
		return errExactlyOnePermissionRequired
	}
	if err := ValidateStringMaxLength(action.Permissions[0], MaxEnumFieldLength, "permissions[0]"); err != nil {
		return err
	}

	// Validate permission value
	if !snx_lib_core.IsValidDelegationPermission(snx_lib_core.DelegationPermission(action.Permissions[0])) {
		return errInvalidPermissionValue
	}

	return nil
}

// Validates and wraps an addDelegatedSigner payload.
func NewValidatedAddDelegatedSignerAction(payload *AddDelegatedSignerActionPayload) (*ValidatedAddDelegatedSignerAction, error) {
	if err := ValidateAddDelegatedSignerAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedAddDelegatedSignerAction{Payload: payload}, nil
}

// Decodes a raw map payload into a typed removeDelegatedSigner payload.
func DecodeRemoveDelegatedSignerAction(action map[string]any) (*RemoveDelegatedSignerActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload RemoveDelegatedSignerActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid removeDelegatedSigner payload: %w", err)
	}

	return &payload, nil
}

// Validates removeDelegatedSigner payload.
func ValidateRemoveDelegatedSignerAction(action *RemoveDelegatedSignerActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "removeDelegatedSigner" {
		return errActionTypeMustBeRemoveDelegatedSigner
	}

	// Validate delegate address is provided and valid
	if action.DelegateAddress == "" {
		return errDelegateIsRequired
	}
	if strings.TrimSpace(string(action.DelegateAddress)) != string(action.DelegateAddress) {
		return errCanonicalDelegateAddress
	}
	if err := ValidateStringMaxLength(action.DelegateAddress, MaxEthAddressLength, "walletAddress"); err != nil {
		return err
	}
	if !isValidEthereumAddress(action.DelegateAddress) {
		return errDelegateAddressMustBeValidEthAddress
	}

	return nil
}

// Validates and wraps a removeDelegatedSigner payload.
func NewValidatedRemoveDelegatedSignerAction(payload *RemoveDelegatedSignerActionPayload) (*ValidatedRemoveDelegatedSignerAction, error) {
	if err := ValidateRemoveDelegatedSignerAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedRemoveDelegatedSignerAction{Payload: payload}, nil
}

// Decodes a raw map payload into a typed removeAllDelegatedSigners payload.
func DecodeRemoveAllDelegatedSignersAction(action map[string]any) (*RemoveAllDelegatedSignersActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload RemoveAllDelegatedSignersActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid removeAllDelegatedSigners payload: %w", err)
	}

	return &payload, nil
}

// Validates removeAllDelegatedSigners payload.
func ValidateRemoveAllDelegatedSignersAction(action *RemoveAllDelegatedSignersActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "removeAllDelegatedSigners" {
		return errActionTypeMustBeRemoveAllDelegatedSigners
	}

	return nil
}

// Validates and wraps a removeAllDelegatedSigners payload.
func NewValidatedRemoveAllDelegatedSignersAction(payload *RemoveAllDelegatedSignersActionPayload) (*ValidatedRemoveAllDelegatedSignersAction, error) {
	if err := ValidateRemoveAllDelegatedSignersAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedRemoveAllDelegatedSignersAction{Payload: payload}, nil
}

// Decodes a raw map payload into a typed updateLeverage payload.
func DecodeUpdateLeverageAction(action map[string]any) (*UpdateLeverageActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload UpdateLeverageActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid updateLeverage payload: %w", err)
	}

	return &payload, nil
}

// Validates updateLeverage payload.
func ValidateUpdateLeverageAction(action *UpdateLeverageActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "updateLeverage" {
		return errActionTypeMustBeUpdateLeverage
	}

	// Validate symbol is provided
	if action.Symbol == "" {
		return ErrSymbolRequired
	}
	if err := ValidateCanonicalSymbol(action.Symbol, "symbol"); err != nil {
		return err
	}

	// Validate leverage is a valid positive integer
	if action.Leverage == "" {
		return ErrLeverageMustBePositive
	}
	if err := ValidateStringMaxLength(action.Leverage, MaxDecimalStringLength, "leverage"); err != nil {
		return err
	}

	leverage, err := strconv.ParseUint(action.Leverage, 10, 32)
	if err != nil {
		return ErrLeverageMustBePositive
	}

	if leverage < 1 {
		return ErrLeverageMustBePositive
	}

	return nil
}

// Validates and wraps an updateLeverage payload.
func NewValidatedUpdateLeverageAction(payload *UpdateLeverageActionPayload) (*ValidatedUpdateLeverageAction, error) {
	if err := ValidateUpdateLeverageAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedUpdateLeverageAction{Payload: payload}, nil
}

// Decodes a raw map payload into a typed createSubaccount payload.
func DecodeCreateSubaccountAction(action map[string]any) (*CreateSubaccountActionPayload, error) {
	if action == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload CreateSubaccountActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid createSubaccount payload: %w", err)
	}

	return &payload, nil
}

// Validates createSubaccount payload.
// The name field is optional - an empty name is valid.
func ValidateCreateSubaccountAction(action *CreateSubaccountActionPayload) error {
	if action == nil {
		return ErrActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "createSubaccount" {
		return errActionTypeMustBeCreateSubaccount
	}

	if err := ValidateCanonicalClientSideString(action.Name, 50, "name"); err != nil {
		return err
	}

	// Name is optional, so no validation needed for empty names
	return nil
}

// Validates and wraps a createSubaccount payload.
func NewValidatedCreateSubaccountAction(payload *CreateSubaccountActionPayload) (*ValidatedCreateSubaccountAction, error) {
	if err := ValidateCreateSubaccountAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedCreateSubaccountAction{Payload: payload}, nil
}

// Decodes a raw map payload into a typed updateSubAccountName payload.
func DecodeUpdateSubAccountNameAction(action map[string]any) (*UpdateSubAccountNameActionPayload, error) {
	if action == nil {
		return nil, errActionPayloadRequired
	}

	var payload UpdateSubAccountNameActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid updateSubAccountName payload: %w", err)
	}

	return &payload, nil
}

// Validates updateSubAccountName payload.
func ValidateUpdateSubAccountNameAction(action *UpdateSubAccountNameActionPayload) error {
	if action == nil {
		return errActionPayloadRequired
	}

	// Validate action type
	if action.Action != "" && action.Action != "updateSubAccountName" {
		return errActionTypeMustBeUpdateSubAccountName
	}

	// Validate name is provided and valid
	if action.Name == "" {
		return errNameIsRequired
	}

	_, err := snx_lib_api_validation_utils.ValidateClientSideString(action.Name, 50, snx_lib_api_validation_utils.ClientSideOption_Trim)
	if err != nil {
		return errNameMustBeSafeAndHumanReadable
	}

	return nil
}

// Validates and wraps an updateSubAccountName payload.
func NewValidatedUpdateSubAccountNameAction(payload *UpdateSubAccountNameActionPayload) (*ValidatedUpdateSubAccountNameAction, error) {
	if err := ValidateUpdateSubAccountNameAction(payload); err != nil {
		return nil, err
	}
	return &ValidatedUpdateSubAccountNameAction{Payload: payload}, nil
}

func DecodeClearSnaxpotPreferenceAction(action map[string]any) (*ClearSnaxpotPreferenceActionPayload, error) {
	if action == nil {
		return nil, errActionPayloadRequired
	}

	var payload ClearSnaxpotPreferenceActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid clearSnaxpotPreference payload: %w", err)
	}

	return &payload, nil
}

func ValidateClearSnaxpotPreferenceAction(payload *ClearSnaxpotPreferenceActionPayload) error {
	if payload == nil {
		return errActionPayloadRequired
	}

	if payload.Action != "" && payload.Action != "clearSnaxpotPreference" {
		return errActionTypeMustBeClearSnaxpotPreference
	}

	switch payload.Scope {
	case "currentEpoch", "persistent":
	default:
		return errSnaxpotScopeInvalid
	}

	return nil
}

func NewValidatedClearSnaxpotPreferenceAction(payload *ClearSnaxpotPreferenceActionPayload) (*ValidatedClearSnaxpotPreferenceAction, error) {
	if err := ValidateClearSnaxpotPreferenceAction(payload); err != nil {
		return nil, err
	}

	return &ValidatedClearSnaxpotPreferenceAction{Payload: payload}, nil
}

func DecodeSaveSnaxpotTicketsAction(action map[string]any) (*SaveSnaxpotTicketsActionPayload, error) {
	if action == nil {
		return nil, errActionPayloadRequired
	}

	var payload SaveSnaxpotTicketsActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid saveSnaxpotTickets payload: %w", err)
	}

	return &payload, nil
}

func ValidateSaveSnaxpotTicketsAction(payload *SaveSnaxpotTicketsActionPayload) error {
	if payload == nil {
		return errActionPayloadRequired
	}

	if payload.Action != "" && payload.Action != "saveSnaxpotTickets" {
		return errActionTypeMustBeSaveSnaxpotTickets
	}

	if len(payload.Entries) == 0 {
		return errSnaxpotEntriesRequired
	}

	for i, entry := range payload.Entries {
		if err := validateSnaxpotTicketMutationEntry(i, entry); err != nil {
			return err
		}
	}

	return nil
}

func NewValidatedSaveSnaxpotTicketsAction(payload *SaveSnaxpotTicketsActionPayload) (*ValidatedSaveSnaxpotTicketsAction, error) {
	if err := ValidateSaveSnaxpotTicketsAction(payload); err != nil {
		return nil, err
	}

	return &ValidatedSaveSnaxpotTicketsAction{Payload: payload}, nil
}

func DecodeSetSnaxpotPreferenceAction(action map[string]any) (*SetSnaxpotPreferenceActionPayload, error) {
	if action == nil {
		return nil, errActionPayloadRequired
	}

	var payload SetSnaxpotPreferenceActionPayload
	if err := mapstructure.Decode(action, &payload); err != nil {
		return nil, fmt.Errorf("invalid setSnaxpotPreference payload: %w", err)
	}

	return &payload, nil
}

func ValidateSetSnaxpotPreferenceAction(payload *SetSnaxpotPreferenceActionPayload) error {
	if payload == nil {
		return errActionPayloadRequired
	}

	if payload.Action != "" && payload.Action != "setSnaxpotPreference" {
		return errActionTypeMustBeSetSnaxpotPreference
	}

	switch payload.Scope {
	case "currentEpoch", "persistent":
	default:
		return errSnaxpotScopeInvalid
	}

	if !snx_lib_snaxpot.ValidSnaxBall(int(payload.SnaxBall)) {
		return errSnaxpotSnaxBallInvalid
	}

	return nil
}

func NewValidatedSetSnaxpotPreferenceAction(payload *SetSnaxpotPreferenceActionPayload) (*ValidatedSetSnaxpotPreferenceAction, error) {
	if err := ValidateSetSnaxpotPreferenceAction(payload); err != nil {
		return nil, err
	}

	return &ValidatedSetSnaxpotPreferenceAction{Payload: payload}, nil
}

func validateSnaxpotTicketMutationEntry(
	index int,
	entry SnaxpotTicketMutationEntryPayload,
) error {
	if strings.TrimSpace(entry.TicketSerial) == "" {
		return fmt.Errorf("entries[%d]: %w", index, errSnaxpotTicketSerialRequired)
	}

	if _, err := strconv.ParseUint(entry.TicketSerial, 10, 64); err != nil {
		return fmt.Errorf("entries[%d]: %w", index, errSnaxpotTicketSerialValidNumber)
	}

	standardBalls := []int{
		int(entry.Ball1),
		int(entry.Ball2),
		int(entry.Ball3),
		int(entry.Ball4),
		int(entry.Ball5),
	}
	if !snx_lib_snaxpot.ValidStandardBalls(standardBalls) {
		return fmt.Errorf("entries[%d]: %w", index, errSnaxpotStandardBallsInvalid)
	}

	if !snx_lib_snaxpot.ValidSnaxBall(int(entry.SnaxBall)) {
		return fmt.Errorf("entries[%d]: %w", index, errSnaxpotSnaxBallInvalid)
	}

	return nil
}
