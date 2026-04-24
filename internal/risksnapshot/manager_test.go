package risksnapshot

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type fakeHydrationClient struct {
	openOrdersCalls    int
	openOrdersResponse [][]HydrationOrder
	positionsCalls     int
	positionsResponse  [][]HydrationPosition
}

func (f *fakeHydrationClient) GetOpenOrders(ctx context.Context, subAccountID int64, limit, offset int) ([]HydrationOrder, error) {
	f.openOrdersCalls++
	if len(f.openOrdersResponse) == 0 {
		return nil, nil
	}
	index := f.openOrdersCalls - 1
	if index >= len(f.openOrdersResponse) {
		index = len(f.openOrdersResponse) - 1
	}
	return f.openOrdersResponse[index], nil
}

func (f *fakeHydrationClient) GetPositions(ctx context.Context, subAccountID int64, limit, offset int) ([]HydrationPosition, error) {
	f.positionsCalls++
	if len(f.positionsResponse) == 0 {
		return nil, nil
	}
	index := f.positionsCalls - 1
	if index >= len(f.positionsResponse) {
		index = len(f.positionsResponse) - 1
	}
	return f.positionsResponse[index], nil
}

func TestEnsureHydratedRefreshesExpiredSnapshot(t *testing.T) {
	client := &fakeHydrationClient{
		openOrdersResponse: [][]HydrationOrder{nil, nil},
		positionsResponse: [][]HydrationPosition{
			nil,
			{{
				Quantity: "3",
				Side:     "long",
				Symbol:   "BTC-USDT",
			}},
		},
	}
	manager := NewManager(client)
	manager.maxSnapshotAge = time.Minute

	initial, err := manager.EnsureHydrated(context.Background(), "session-a", 77)
	if err != nil {
		t.Fatalf("initial ensure hydrated failed: %v", err)
	}
	if !initial.SignedPosition("BTC-USDT").Equal(decimal.Zero) {
		t.Fatalf("expected empty initial position, got %s", initial.SignedPosition("BTC-USDT").String())
	}

	manager.mu.Lock()
	manager.entries[77].snapshot.refreshedAtMs = time.Now().Add(-10 * time.Minute).UnixMilli()
	manager.mu.Unlock()

	refreshed, err := manager.EnsureHydrated(context.Background(), "session-a", 77)
	if err != nil {
		t.Fatalf("refresh ensure hydrated failed: %v", err)
	}
	if !refreshed.SignedPosition("BTC-USDT").Equal(decimal.RequireFromString("3")) {
		t.Fatalf("expected refreshed position quantity 3, got %s", refreshed.SignedPosition("BTC-USDT").String())
	}
	if client.positionsCalls != 2 {
		t.Fatalf("expected expired snapshot to refresh positions, got %d calls", client.positionsCalls)
	}
	if client.openOrdersCalls != 2 {
		t.Fatalf("expected expired snapshot to refresh open orders, got %d calls", client.openOrdersCalls)
	}
}
