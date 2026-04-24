package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

type WithdrawalHistoryRepository interface {
	// Creates a new withdrawal history in an append only manner grouped by the offchain id
	Create(ctx context.Context, withdrawal WithdrawalHistory, tx *gorm.DB) error
	// Get a withdrawal history by the offchain or onchain id.
	GetById(ctx context.Context, id any, tx *gorm.DB) (WithdrawalHistory, error)
	// Get all withdrawals history filtered by the given params, grouped by offchain id and created_at
	// using the withdrawal history view
	List(ctx context.Context, params ListWithdrawalsParams) ([]WithdrawalHistory, error)
}

type GormWithdrawalHistoryRepository struct {
	logger snx_lib_logging.Logger
	db     *gorm.DB
}

func NewWithdrawalHistoryRepository(
	logger snx_lib_logging.Logger,
	db *gorm.DB,
) WithdrawalHistoryRepository {
	return &GormWithdrawalHistoryRepository{
		logger: logger,
		db:     db,
	}
}

// Creates (or replaces) a view that gets withdrawals by the offchain id
// with their respective latest info. Because of the append-only nature of the
// withdrawal history, multiple records can exist for the same offchain_withdraw_id.
// This view returns only the most recent record for each offchain_withdraw_id.
func SetupWithdrawalHistoryView(db *gorm.DB) error {
	query := `
		CREATE OR REPLACE VIEW withdrawal_history_view AS
		SELECT DISTINCT ON (offchain_withdrawal_id) *
		FROM withdrawal_history
		ORDER BY offchain_withdrawal_id, created_at DESC;
	`
	return db.Exec(query).Error
}

func (w *GormWithdrawalHistoryRepository) Create(ctx context.Context, withdraw WithdrawalHistory, tx *gorm.DB) error {
	db := w.db
	if tx != nil {
		db = tx
	}

	// Reset id to ensure a new record is always inserted (append-only pattern)
	withdraw.Id = 0

	return db.WithContext(ctx).
		Model(&WithdrawalHistory{}).
		Create(&withdraw).
		Error
}

func (w *GormWithdrawalHistoryRepository) GetById(ctx context.Context, id any, tx *gorm.DB) (WithdrawalHistory, error) {
	db := w.db
	if tx != nil {
		db = tx
	}

	var withdraw WithdrawalHistory
	query := db.WithContext(ctx).Model(&WithdrawalHistory{})

	switch v := id.(type) {
	case OffchainWithdrawalId:
		query = query.Where(&WithdrawalHistory{
			OffchainId: v,
		})
	case OnchainWithdrawalId:
		query = query.Where(&WithdrawalHistory{
			OnchainId: &v,
		})
	case *OnchainWithdrawalId:
		query = query.Where(&WithdrawalHistory{
			OnchainId: v,
		})
	default:
		return withdraw, fmt.Errorf("unsupported withdraw id type: %T", id)
	}

	err := query.
		Order("created_at DESC").
		First(&withdraw).
		Error

	return withdraw, err
}

type ListWithdrawalsParams struct {
	From         time.Time
	Limit        int64
	OffchainId   OffchainWithdrawalId
	OnchainId    OnchainWithdrawalId
	OrderFilter  OrderFilter
	Status       []WithdrawalStatus
	SubAccountId SubAccountId
	To           time.Time
}

func (w *GormWithdrawalHistoryRepository) List(ctx context.Context, params ListWithdrawalsParams) ([]WithdrawalHistory, error) {

	query := w.db.WithContext(ctx).
		Model(&WithdrawalHistory_View{})

	if params.OffchainId > 0 {
		query = query.Where(&WithdrawalHistory_View{
			WithdrawalHistory: WithdrawalHistory{
				OffchainId: params.OffchainId,
			},
		})
	}

	if params.OnchainId > 0 {
		query = query.Where(&WithdrawalHistory_View{
			WithdrawalHistory: WithdrawalHistory{
				OnchainId: &params.OnchainId,
			},
		})
	}

	if len(params.Status) > 0 {
		query = query.Where("status IN ?", params.Status)
	}

	if params.SubAccountId > 0 {
		query = query.Where(&WithdrawalHistory_View{
			WithdrawalHistory: WithdrawalHistory{
				SubAccountId: params.SubAccountId,
			},
		})
	}

	// just checking if it is zero values not actually checking if the range is correct
	if !params.From.IsZero() {
		query = query.Where("created_at >= ?", params.From)
	}

	if !params.To.IsZero() {
		query = query.Where("created_at <= ?", params.To)
	}

	if params.OrderFilter != "" {

		switch params.OrderFilter {
		case OrderFilter_ASC:
			query = query.Order(CreatedAt_ASC)
		default:
			query = query.Order(CreatedAt_DESC)
		}
	}

	if params.Limit > 0 {
		query = query.Limit(int(params.Limit))
	}

	var withdraws []WithdrawalHistory
	err := query.Find(&withdraws).Error

	return withdraws, err
}
