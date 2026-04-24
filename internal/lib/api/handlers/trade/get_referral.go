package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getReferral
*/

// Referral represents a referral relationship
type Referral struct {
	ReferralID     string    `json:"referralId"`     // Unique referral identifier
	ReferrerID     string    `json:"referrerId"`     // ID of the user who made the referral
	ReferredID     string    `json:"referredId"`     // ID of the user who was referred
	Status         string    `json:"status"`         // Referral status (ACTIVE, EXPIRED, etc.)
	CommissionRate string    `json:"commissionRate"` // Commission rate for the referral
	TotalEarnings  string    `json:"totalEarnings"`  // Total earnings from this referral
	CreatedAt      Timestamp `json:"createdAt"`      // Referral creation timestamp
}

// Referrals represents the response for referral data
type Referrals []Referral

// generateMockReferral creates mock referral data
func generateMockReferral() Referrals {
	return Referrals{
		{
			ReferralID:     "ref123",
			ReferrerID:     "user456",
			ReferredID:     "user789",
			Status:         "ACTIVE",
			CommissionRate: "0.1",
			TotalEarnings:  "125.50",
			CreatedAt:      1704067200000,
		},
		{
			ReferralID:     "ref124",
			ReferrerID:     "user456",
			ReferredID:     "user101",
			Status:         "ACTIVE",
			CommissionRate: "0.1",
			TotalEarnings:  "89.25",
			CreatedAt:      1704067200000,
		},
		{
			ReferralID:     "ref125",
			ReferrerID:     "user456",
			ReferredID:     "user102",
			Status:         "EXPIRED",
			CommissionRate: "0.1",
			TotalEarnings:  "0.00",
			CreatedAt:      1704067200000,
		},
	}
}

// Handler for "getReferral".
//
//dd:span
func Handle_getReferral(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	/*var req TradeRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any]("", snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}*/

	referrals := generateMockReferral()
	ctx.Logger.Info("Generated mock referral data", "count", len(referrals))

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, referrals)
}
