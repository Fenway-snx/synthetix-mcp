package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type CreateDelegationRequest1 struct {
	WalletAddress WalletAddress `json:"walletAddress"`
	Permissions   []string      `json:"permissions"`
	ExpiresAt     *int64        `json:"expiresAt,omitempty"`
}

type CreateDelegationRequest2 struct {
	WalletAddress WalletAddress `json:"walletAddress"`
	Permissions   []string      `json:"permissions"`
	ExpiresAt     int64         `json:"expiresAt,omitempty"`
}

type CreateDelegationRequest3 struct {
	WalletAddress WalletAddress `json:"walletAddress"`
	Permissions   []string      `json:"permissions"`
	ExpiresAt     *Timestamp    `json:"expiresAt,omitempty"`
}

func Test_CreateDelegationRequest1_MARSHALING(t *testing.T) {

	t.Run("Marshal", func(t *testing.T) {

		{
			req := CreateDelegationRequest1{}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":null}`, s)
		}

		{
			req := CreateDelegationRequest1{
				Permissions: []string{},
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":[]}`, s)
		}

		{
			req := CreateDelegationRequest1{
				Permissions: []string{
					"abc",
					"def",
				},
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":["abc","def"]}`, s)
		}

		{
			expiresAt := new(int64)

			*expiresAt = 1234

			req := CreateDelegationRequest1{
				Permissions: []string{},
				ExpiresAt:   expiresAt,
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":[],"expiresAt":1234}`, s)
		}
	})

	t.Run("Unmarshal", func(t *testing.T) {

		{
			input := `{"walletAddress":"","permissions":null}`

			var req CreateDelegationRequest1

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Nil(t, req.Permissions)
			assert.Nil(t, req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":[]}`

			var req CreateDelegationRequest1

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{}, req.Permissions)
			assert.Nil(t, req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":["abc","def"]}`

			var req CreateDelegationRequest1

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{
				"abc",
				"def",
			}, req.Permissions)
			assert.Nil(t, req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":[],"expiresAt":1234}`

			var req CreateDelegationRequest1

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{}, req.Permissions)
			assert.Equal(t, int64(1234), *req.ExpiresAt)
		}
	})
}

func Test_CreateDelegationRequest2_MARSHALING(t *testing.T) {

	t.Run("Marshal", func(t *testing.T) {

		{
			req := CreateDelegationRequest2{}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":null}`, s)
		}

		{
			req := CreateDelegationRequest2{
				Permissions: []string{},
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":[]}`, s)
		}

		{
			req := CreateDelegationRequest2{
				Permissions: []string{
					"abc",
					"def",
				},
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":["abc","def"]}`, s)
		}

		{
			expiresAt := int64(1234)

			req := CreateDelegationRequest2{
				Permissions: []string{},
				ExpiresAt:   expiresAt,
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":[],"expiresAt":1234}`, s)
		}
	})

	t.Run("Unmarshal", func(t *testing.T) {

		{
			input := `{"walletAddress":"","permissions":null}`

			var req CreateDelegationRequest2

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Nil(t, req.Permissions)
			assert.Equal(t, int64(0), req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":[]}`

			var req CreateDelegationRequest2

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{}, req.Permissions)
			assert.Equal(t, int64(0), req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":["abc","def"]}`

			var req CreateDelegationRequest2

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{
				"abc",
				"def",
			}, req.Permissions)
			assert.Equal(t, int64(0), req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":[],"expiresAt":1234}`

			var req CreateDelegationRequest2

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{}, req.Permissions)
			assert.Equal(t, int64(1234), req.ExpiresAt)
		}
	})
}

func Test_CreateDelegationRequest3_MARSHALING(t *testing.T) {

	t.Run("Marshal", func(t *testing.T) {

		{
			req := CreateDelegationRequest3{}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":null}`, s)
		}

		{
			req := CreateDelegationRequest3{
				Permissions: []string{},
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":[]}`, s)
		}

		{
			req := CreateDelegationRequest3{
				Permissions: []string{
					"abc",
					"def",
				},
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":["abc","def"]}`, s)
		}

		{
			expiresAt := new(Timestamp)

			*expiresAt = Timestamp(1234)

			req := CreateDelegationRequest3{
				Permissions: []string{},
				ExpiresAt:   expiresAt,
			}

			bytes, err := json.Marshal(req)

			require.Nil(t, err, "expected `err` to be `nil`")

			s := string(bytes)

			assert.Equal(t, `{"walletAddress":"","permissions":[],"expiresAt":1234}`, s)
		}
	})

	t.Run("Unmarshal", func(t *testing.T) {

		{
			input := `{"walletAddress":"","permissions":null}`

			var req CreateDelegationRequest3

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Nil(t, req.Permissions)
			assert.Nil(t, req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":[]}`

			var req CreateDelegationRequest3

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{}, req.Permissions)
			assert.Nil(t, req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":["abc","def"]}`

			var req CreateDelegationRequest3

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{
				"abc",
				"def",
			}, req.Permissions)
			assert.Nil(t, req.ExpiresAt)
		}

		{
			input := `{"walletAddress":"","permissions":[],"expiresAt":1234}`

			var req CreateDelegationRequest3

			err := json.Unmarshal([]byte(input), &req)

			require.Nil(t, err, "expected `err` to be `nil`")

			assert.Equal(t, WalletAddress(""), req.WalletAddress)
			assert.Equal(t, []string{}, req.Permissions)
			assert.Equal(t, Timestamp(1234), *req.ExpiresAt)
		}
	})
}

// mockDelegationClient mocks the SubaccountServiceClient for getDelegationsForDelegate tests
type mockDelegationClient struct {
	v4grpc.SubaccountServiceClient

	// GetDelegationsForDelegate mock
	getDelegationsForDelegateResponse *v4grpc.GetDelegationsForDelegateResponse
	getDelegationsForDelegateErr      error
	capturedDelegateAddress           string

	// VerifySubaccountAuthorization mock
	verifyAuthResponse *v4grpc.VerifySubaccountAuthorizationResponse
	verifyAuthErr      error
	capturedAuthReq    *v4grpc.VerifySubaccountAuthorizationRequest
}

func (m *mockDelegationClient) GetDelegationsForDelegate(ctx context.Context, in *v4grpc.GetDelegationsForDelegateRequest, opts ...grpc.CallOption) (*v4grpc.GetDelegationsForDelegateResponse, error) {
	m.capturedDelegateAddress = in.DelegateAddress
	return m.getDelegationsForDelegateResponse, m.getDelegationsForDelegateErr
}

func (m *mockDelegationClient) VerifySubaccountAuthorization(ctx context.Context, in *v4grpc.VerifySubaccountAuthorizationRequest, opts ...grpc.CallOption) (*v4grpc.VerifySubaccountAuthorizationResponse, error) {
	m.capturedAuthReq = in
	return m.verifyAuthResponse, m.verifyAuthErr
}

func createDelegationTestContext(
	walletAddress string,
	subAccountId snx_lib_core.SubAccountId,
	client v4grpc.SubaccountServiceClient,
) TradeContext {
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil, nil,
		client,
		nil, nil, nil,
		"req-id",
		snx_lib_api_handlers_types.ClientRequestId("test-client-req"),
		snx_lib_api_handlers_types.WalletAddress(walletAddress),
		subAccountId,
	)
}

func Test_Handle_getDelegationsForDelegate(t *testing.T) {
	const (
		callerAddress = "0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa"
		owningAddress = "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
		subAccountId  = snx_lib_core.SubAccountId(12345)
	)

	defaultDelegationsResponse := &v4grpc.GetDelegationsForDelegateResponse{
		TimestampMs: 1000,
		TimestampUs: 1000000,
		Delegations: []*v4grpc.DelegationWithOwnerInfo{
			{
				SubAccountId:    99,
				OwnerAddress:    "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
				AccountName:     "Test Account",
				AccountValue:    "1234.56",
				Permissions:     []string{"TRADE"},
				DelegateAddress: callerAddress,
			},
		},
	}

	t.Run("owningAddress omitted uses ctx.WalletAddress", func(t *testing.T) {
		mock := &mockDelegationClient{
			getDelegationsForDelegateResponse: defaultDelegationsResponse,
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, callerAddress, mock.capturedDelegateAddress)
		assert.Nil(t, mock.capturedAuthReq, "Should not call VerifySubaccountAuthorization")
	})

	t.Run("owningAddress same as caller skips auth check", func(t *testing.T) {
		mock := &mockDelegationClient{
			getDelegationsForDelegateResponse: defaultDelegationsResponse,
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{
			"owningAddress": callerAddress,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, callerAddress, mock.capturedDelegateAddress)
		assert.Nil(t, mock.capturedAuthReq, "Should not call VerifySubaccountAuthorization when same address")
	})

	t.Run("owningAddress same as caller case-insensitive", func(t *testing.T) {
		mock := &mockDelegationClient{
			getDelegationsForDelegateResponse: defaultDelegationsResponse,
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{
			"owningAddress": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		assert.Nil(t, mock.capturedAuthReq, "Should not call VerifySubaccountAuthorization for case-insensitive match")
	})

	t.Run("valid owningAddress with authorized caller", func(t *testing.T) {
		mock := &mockDelegationClient{
			getDelegationsForDelegateResponse: defaultDelegationsResponse,
			verifyAuthResponse: &v4grpc.VerifySubaccountAuthorizationResponse{
				IsAuthorized:      true,
				AuthorizationType: v4grpc.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{
			"owningAddress": owningAddress,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, owningAddress, mock.capturedDelegateAddress, "Should query for owningAddress instead of caller")

		// Verify the auth check was made correctly
		require.NotNil(t, mock.capturedAuthReq)
		assert.Equal(t, int64(subAccountId), mock.capturedAuthReq.SubAccountId)
		assert.Equal(t, owningAddress, mock.capturedAuthReq.Address)
		assert.Equal(t, []string{string(snx_lib_core.DelegationPermissionTrading)}, mock.capturedAuthReq.Permissions)
	})

	t.Run("valid owningAddress with unauthorized caller returns 403", func(t *testing.T) {
		mock := &mockDelegationClient{
			verifyAuthResponse: &v4grpc.VerifySubaccountAuthorizationResponse{
				IsAuthorized:      false,
				AuthorizationType: v4grpc.AuthorizationType_AUTHORIZATION_TYPE_NONE,
			},
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{
			"owningAddress": owningAddress,
		})

		require.Equal(t, HTTPStatusCode_403_Forbidden, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeForbidden, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Not authorized")
		assert.Empty(t, mock.capturedDelegateAddress, "Should not call GetDelegationsForDelegate")
	})

	t.Run("invalid owningAddress format returns 400", func(t *testing.T) {
		mock := &mockDelegationClient{}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{
			"owningAddress": "not-a-valid-address",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Invalid owningAddress format")
		assert.Nil(t, mock.capturedAuthReq, "Should not call VerifySubaccountAuthorization")
		assert.Empty(t, mock.capturedDelegateAddress, "Should not call GetDelegationsForDelegate")
	})

	t.Run("VerifySubaccountAuthorization gRPC error returns error response", func(t *testing.T) {
		mock := &mockDelegationClient{
			verifyAuthErr: status.Error(codes.Internal, "service unavailable"),
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{
			"owningAddress": owningAddress,
		})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.Empty(t, mock.capturedDelegateAddress, "Should not call GetDelegationsForDelegate")
	})

	t.Run("GetDelegationsForDelegate gRPC error returns error response", func(t *testing.T) {
		mock := &mockDelegationClient{
			getDelegationsForDelegateErr: status.Error(codes.Internal, "service unavailable"),
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("response contains correct delegated accounts data", func(t *testing.T) {
		mock := &mockDelegationClient{
			getDelegationsForDelegateResponse: defaultDelegationsResponse,
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)

		data, ok := resp.Response.(DelegatedAccountsList)
		require.True(t, ok, "Response should be DelegatedAccountsList")
		require.Len(t, data.DelegatedAccounts, 1)

		account := data.DelegatedAccounts[0]
		assert.Equal(t, SubAccountId("99"), account.SubAccountId)
		assert.Equal(t, WalletAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"), account.OwnerAddress)
		assert.Equal(t, "Test Account", account.AccountName)
		assert.Equal(t, "1234.56", account.AccountValue)
		assert.Equal(t, []string{"TRADE"}, account.Permissions)
	})

	t.Run("accountValue defaults to zero string when not populated", func(t *testing.T) {
		emptyValueResponse := &v4grpc.GetDelegationsForDelegateResponse{
			TimestampMs: 1000,
			TimestampUs: 1000000,
			Delegations: []*v4grpc.DelegationWithOwnerInfo{
				{
					SubAccountId:    99,
					OwnerAddress:    "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
					AccountName:     "Test Account",
					Permissions:     []string{"TRADE"},
					DelegateAddress: callerAddress,
				},
			},
		}

		mock := &mockDelegationClient{
			getDelegationsForDelegateResponse: emptyValueResponse,
		}

		ctx := createDelegationTestContext(callerAddress, subAccountId, mock)

		statusCode, resp := Handle_getDelegationsForDelegate(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)

		data, ok := resp.Response.(DelegatedAccountsList)
		require.True(t, ok, "Response should be DelegatedAccountsList")
		require.Len(t, data.DelegatedAccounts, 1)

		account := data.DelegatedAccounts[0]
		assert.Equal(t, "", account.AccountValue, "AccountValue should be empty string when not populated in gRPC response")
	})
}

// mockSubaccountClientWithNormalizedPerms wraps MockSubaccountServiceClient and
// returns server-normalized permissions in CreateDelegation responses instead of
// echoing the request permissions.
type mockSubaccountClientWithNormalizedPerms struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	lastSignerAddress     string
	normalizedPermissions []string
}

func (m *mockSubaccountClientWithNormalizedPerms) CreateDelegation(ctx context.Context, req *v4grpc.CreateDelegationRequest, opts ...grpc.CallOption) (*v4grpc.CreateDelegationResponse, error) {
	m.lastSignerAddress = req.SignerAddress
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.CreateDelegationResponse{
		TimestampMs:     timestamp_ms,
		TimestampUs:     timestamp_us,
		Id:              1,
		SubAccountId:    req.SubAccountId,
		DelegateAddress: req.DelegateAddress,
		Permissions:     m.normalizedPermissions,
		ExpiresAt:       req.ExpiresAt,
		CreatedAt:       timestamppb.New(snx_lib_utils_time.Now()),
	}, nil
}

func Test_Handle_addDelegatedSigner_ReturnsServerPermissions(t *testing.T) {
	mockClient := &mockSubaccountClientWithNormalizedPerms{
		MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		normalizedPermissions:       []string{"session"},
	}

	ctx := snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil, nil,
		mockClient,
		nil, nil, nil,
		"test-request-1",
		snx_lib_api_handlers_types.ClientRequestId("test-client-req"),
		snx_lib_api_handlers_types.WalletAddress("0x1234567890123456789012345678901234567890"),
		snx_lib_core.SubAccountId(100),
	)
	validated, err := snx_lib_api_validation.NewValidatedAddDelegatedSignerAction(&snx_lib_api_validation.AddDelegatedSignerActionPayload{
		Action:          "addDelegatedSigner",
		DelegateAddress: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		Permissions:     []string{"trading"},
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	ctx = ctx.WithAction("addDelegatedSigner", validated)

	params := map[string]any{
		"WalletAddress": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		"Permissions":   []string{"trading"},
	}

	statusCode, resp := Handle_addDelegatedSigner(ctx, params)

	require.Equal(t, HTTPStatusCode_200_OK, statusCode)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)

	responseBytes, err := json.Marshal(resp.Response)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var signer DelegatedSigner
	err = json.Unmarshal(responseBytes, &signer)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, []string{"session"}, signer.Permissions, "response should contain server-normalized permissions, not request echo")
	assert.Equal(t, WalletAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"), signer.WalletAddress)

	// Verify signer address was propagated from context
	assert.Equal(t, "0x1234567890123456789012345678901234567890", mockClient.lastSignerAddress, "SignerAddress should be propagated from ctx.WalletAddress")
}
