package info

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

const (
	testWalletOwner        = snx_lib_api_types.WalletAddress("0x1111111111111111111111111111111111111111")
	testWalletNobody       = snx_lib_api_types.WalletAddress("0x2222222222222222222222222222222222222222")
	testWalletAnyone       = snx_lib_api_types.WalletAddress("0x3333333333333333333333333333333333333333")
	testWalletOwnerOnly    = snx_lib_api_types.WalletAddress("0x4444444444444444444444444444444444444444")
	testWalletDelegateOnly = snx_lib_api_types.WalletAddress("0x5555555555555555555555555555555555555555")
)

// mockDelegationFailingClient succeeds on ListSubaccounts but fails on GetDelegationsForDelegate.
type mockDelegationFailingClient struct {
	*snx_lib_authtest.MockSubaccountServiceClient
}

func (m *mockDelegationFailingClient) GetDelegationsForDelegate(ctx context.Context, req *v4grpc.GetDelegationsForDelegateRequest, opts ...grpc.CallOption) (*v4grpc.GetDelegationsForDelegateResponse, error) {
	return nil, assert.AnError
}

func newInfoContext(client v4grpc.SubaccountServiceClient) InfoContext {
	return InfoContext{
		ContextCommon: ContextCommon{
			Context:          context.Background(),
			Logger:           snx_lib_logging_doubles.NewStubLogger(),
			SubaccountClient: client,
		},
		ClientRequestId: "test-request",
	}
}

func Test_Handle_getSubAccountIds_Legacy(t *testing.T) {
	t.Run("returns owned IDs as flat array", func(t *testing.T) {
		mock := snx_lib_authtest.NewMockSubaccountServiceClient()
		mock.AddMockAccount(testWalletOwner, 12345)
		mock.AddMockAccount(testWalletOwner, 12346)

		status, response := Handle_getSubAccountIds(newInfoContext(mock), map[string]any{
			"walletAddress": testWalletOwner,
		})

		assert.Equal(t, http.StatusOK, int(status))
		assert.Equal(t, "ok", response.Status)

		ids, ok := response.Response.([]SubAccountId)
		assert.True(t, ok, "Response should be []SubAccountId, but is in fact %T", response.Response)
		assert.Equal(t, []SubAccountId{"12345", "12346"}, ids)
	})

	t.Run("no accounts returns empty array", func(t *testing.T) {
		mock := snx_lib_authtest.NewMockSubaccountServiceClient()

		status, response := Handle_getSubAccountIds(newInfoContext(mock), map[string]any{
			"walletAddress": testWalletNobody,
		})

		assert.Equal(t, http.StatusOK, int(status))
		ids, ok := response.Response.([]SubAccountId)
		assert.True(t, ok)
		assert.Equal(t, []SubAccountId{}, ids)
	})

	t.Run("missing walletAddress returns 400", func(t *testing.T) {
		status, response := Handle_getSubAccountIds(
			newInfoContext(snx_lib_authtest.NewMockSubaccountServiceClient()),
			map[string]any{},
		)
		assert.Equal(t, http.StatusBadRequest, int(status))
		assert.Equal(t, "error", response.Status)
	})

	t.Run("ListSubaccounts failure returns 500", func(t *testing.T) {
		status, response := Handle_getSubAccountIds(
			newInfoContext(snx_lib_authtest.NewMockFailingSubaccountServiceClient()),
			map[string]any{"walletAddress": testWalletAnyone},
		)
		assert.Equal(t, http.StatusInternalServerError, int(status))
		assert.Equal(t, "error", response.Status)
	})
}

func Test_Handle_getSubAccountIds_WithDelegations(t *testing.T) {
	tests := []struct {
		name                   string
		params                 map[string]any
		setupMock              func() v4grpc.SubaccountServiceClient
		expectedStatus         int
		expectedResponseStatus string
		expectedOwnedIds       []SubAccountId
		expectedDelegatedIds   []SubAccountId
		expectError            bool
	}{
		{
			name:   "owned and delegated IDs returned correctly",
			params: map[string]any{"walletAddress": testWalletOwner, "includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				mock := snx_lib_authtest.NewMockSubaccountServiceClient()
				mock.AddMockAccount(testWalletOwner, 12345)
				mock.AddMockAccount(testWalletOwner, 12346)
				mock.AddMockDelegationForDelegate(testWalletOwner, 50)
				mock.AddMockDelegationForDelegate(testWalletOwner, 78)
				return mock
			},
			expectedStatus:         http.StatusOK,
			expectedResponseStatus: "ok",
			expectedOwnedIds:       []SubAccountId{"12345", "12346"},
			expectedDelegatedIds:   []SubAccountId{"50", "78"},
		},
		{
			name:   "no delegations returns empty delegatedSubAccountIds",
			params: map[string]any{"walletAddress": testWalletOwnerOnly, "includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				mock := snx_lib_authtest.NewMockSubaccountServiceClient()
				mock.AddMockAccount(testWalletOwnerOnly, 100)
				return mock
			},
			expectedStatus:         http.StatusOK,
			expectedResponseStatus: "ok",
			expectedOwnedIds:       []SubAccountId{"100"},
			expectedDelegatedIds:   []SubAccountId{},
		},
		{
			name:   "no owned accounts returns empty subAccountIds",
			params: map[string]any{"walletAddress": testWalletDelegateOnly, "includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				mock := snx_lib_authtest.NewMockSubaccountServiceClient()
				mock.AddMockDelegationForDelegate(testWalletDelegateOnly, 200)
				return mock
			},
			expectedStatus:         http.StatusOK,
			expectedResponseStatus: "ok",
			expectedOwnedIds:       []SubAccountId{},
			expectedDelegatedIds:   []SubAccountId{"200"},
		},
		{
			name:   "both empty",
			params: map[string]any{"walletAddress": testWalletNobody, "includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				return snx_lib_authtest.NewMockSubaccountServiceClient()
			},
			expectedStatus:         http.StatusOK,
			expectedResponseStatus: "ok",
			expectedOwnedIds:       []SubAccountId{},
			expectedDelegatedIds:   []SubAccountId{},
		},
		{
			name:   "ListSubaccounts failure returns 500",
			params: map[string]any{"walletAddress": testWalletAnyone, "includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				return snx_lib_authtest.NewMockFailingSubaccountServiceClient()
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:   "GetDelegationsForDelegate failure returns 500",
			params: map[string]any{"walletAddress": testWalletAnyone, "includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				return &mockDelegationFailingClient{
					MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:   "missing walletAddress returns 400",
			params: map[string]any{"includeDelegations": true},
			setupMock: func() v4grpc.SubaccountServiceClient {
				return snx_lib_authtest.NewMockSubaccountServiceClient()
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, response := Handle_getSubAccountIds(newInfoContext(tt.setupMock()), tt.params)

			assert.Equal(t, tt.expectedStatus, int(status))
			assert.NotNil(t, response)

			if tt.expectError {
				assert.Equal(t, "error", response.Status)
				return
			}

			assert.Equal(t, tt.expectedResponseStatus, response.Status)

			result, ok := response.Response.(SubAccountIdsWithDelegationsResponse)
			assert.True(t, ok, "Response should be SubAccountIdsWithDelegationsResponse")
			assert.Equal(t, tt.expectedOwnedIds, result.SubAccountIds)
			assert.Equal(t, tt.expectedDelegatedIds, result.DelegatedSubAccountIds)
		})
	}
}
