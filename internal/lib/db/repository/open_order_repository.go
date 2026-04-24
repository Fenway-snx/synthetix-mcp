package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// OpenOrderRepository handles open order database operations
type OpenOrderRepository struct {
	db *gorm.DB
}

// NewOpenOrderRepository creates a new open order repository
func NewOpenOrderRepository(db *gorm.DB) *OpenOrderRepository {
	return &OpenOrderRepository{
		db: db,
	}
}

func (r *OpenOrderRepository) CreateOpenOrder(ctx context.Context, order *postgrestypes.OpenOrder) error {
	return r.CreateOpenOrderDB(ctx, r.db, order)
}

// CreateOpenOrderDB persists an open order using db (caller may pass a transaction handle).
func (r *OpenOrderRepository) CreateOpenOrderDB(ctx context.Context, db *gorm.DB, order *postgrestypes.OpenOrder) error {
	return db.WithContext(ctx).Create(order).Error
}

func (r *OpenOrderRepository) UpdateOpenOrder(ctx context.Context, order *postgrestypes.OpenOrder) error {
	return r.UpdateOpenOrderDB(ctx, r.db, order)
}

// UpdateOpenOrderDB updates an open order using db (caller may pass a transaction handle).
func (r *OpenOrderRepository) UpdateOpenOrderDB(ctx context.Context, db *gorm.DB, order *postgrestypes.OpenOrder) error {
	result := db.WithContext(ctx).
		Model(postgrestypes.OpenOrder{}).
		Where(postgrestypes.OpenOrder{
			VenueOrderId: order.VenueOrderId,
		}).
		Select("*").
		Updates(order)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// UpsertOpenOrderTwapDB upserts open_order_twap using db (caller may pass a transaction handle).
func (r *OpenOrderRepository) UpsertOpenOrderTwapDB(ctx context.Context, db *gorm.DB, row *postgrestypes.OpenOrderTwap) error {
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "order_id"}},
		UpdateAll: true,
	}).Create(row).Error
}

// GetOpenOrderTwapsByOrderIDs returns TWAP rows keyed by order_id.
func (r *OpenOrderRepository) GetOpenOrderTwapsByOrderIDs(
	ctx context.Context,
	orderIDs []uint64,
) (map[uint64]*postgrestypes.OpenOrderTwap, error) {
	if len(orderIDs) == 0 {
		return map[uint64]*postgrestypes.OpenOrderTwap{}, nil
	}

	var rows []postgrestypes.OpenOrderTwap
	if err := r.db.WithContext(ctx).
		Where("order_id IN ?", orderIDs).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("get open_order_twap by order ids: %w", err)
	}

	out := make(map[uint64]*postgrestypes.OpenOrderTwap, len(rows))
	for i := range rows {
		out[rows[i].OrderID] = &rows[i]
	}
	return out, nil
}

// DeleteOpenOrder deletes an open order (typically when canceled or fully filled)
func (r *OpenOrderRepository) DeleteOpenOrder(ctx context.Context, venueOrderId VenueOrderId) error {
	result := r.db.WithContext(ctx).
		Where("order_id = ?", venueOrderId).
		Delete(&postgrestypes.OpenOrder{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete open order: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("open order not found for order ID %d", venueOrderId)
	}

	return nil
}

// Retrieves open orders for a subaccount with optional client order ID,
// symbol, and time filtering, plus pagination.
func (r *OpenOrderRepository) GetOpenOrdersBySubAccountWithSymbol(
	ctx context.Context,
	subAccountID snx_lib_core.SubAccountId,
	clientOrderID snx_lib_core.ClientOrderId,
	symbol *string,
	startTime int64,
	endTime int64,
	offset int,
	limit int,
) ([]*postgrestypes.OpenOrder, error) {
	var openOrders []*postgrestypes.OpenOrder

	query := r.db.WithContext(ctx).
		Model(&postgrestypes.OpenOrder{}).
		Where("sub_account_id = ?", subAccountID)

	if clientOrderID != snx_lib_core.ClientOrderId_Empty {
		query = query.Where("client_order_id = ?", string(clientOrderID))
	}

	// Apply symbol filter if provided
	if symbol != nil && *symbol != "" {
		query = query.Where("symbol = ?", *symbol)
	}

	// Apply time filters if provided
	if startTime > 0 {
		startTimeVal := time.Unix(startTime, 0)
		query = query.Where("updated_at >= ?", startTimeVal)
	}
	if endTime > 0 {
		endTimeVal := time.Unix(endTime, 0)
		query = query.Where("updated_at <= ?", endTimeVal)
	}

	// Get paginated results
	if err := query.Order("order_id DESC").
		Offset(offset).
		Limit(limit).
		Find(&openOrders).Error; err != nil {
		return nil, fmt.Errorf("failed to get open orders by subaccount with symbol: %w", err)
	}

	return openOrders, nil
}

// Returns open orders with non-null expires_at at or before the given cutoff.
func (r *OpenOrderRepository) GetExpiredOpenOrders(
	ctx context.Context,
	cutoff time.Time,
	offset int,
	limit int,
) ([]*postgrestypes.OpenOrder, int64, error) {
	var openOrders []*postgrestypes.OpenOrder
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).
		Model(&postgrestypes.OpenOrder{}).
		Where("expires_at IS NOT NULL AND expires_at <= ?", cutoff)

	if err := baseQuery.Session(&gorm.Session{}).Count(&totalCount).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count expired open orders: %w", err)
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("expires_at ASC, order_id ASC").
		Offset(offset).
		Limit(limit).
		Find(&openOrders).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get expired open orders: %w", err)
	}

	return openOrders, totalCount, nil
}
