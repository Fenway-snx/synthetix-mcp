package utils

import (
	"github.com/bwmarrin/snowflake"
)

type IdGenerator interface {
	NewID() int64
}

type snowflakeIdGenerator struct {
	node *snowflake.Node
}

func newSnowflakeIdGenerator(nodeID int64) IdGenerator {
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		panic(err)
	}
	return &snowflakeIdGenerator{node: node}
}

func (g *snowflakeIdGenerator) NewID() int64 {
	return g.node.Generate().Int64()
}

// IdGeneratorFactory holds pre-initialized per-module IdGenerator instances.
// Each field uses a distinct snowflake node ID to guarantee globally unique IDs
// across modules.
type IdGeneratorFactory struct {
	FundingHistory IdGenerator
	Liquidation    IdGenerator
	Order          IdGenerator
	OrderHistory   IdGenerator
	Position       IdGenerator
	SubAccount     IdGenerator
	TradeHistory   IdGenerator
	Transfer       IdGenerator
	Withdrawal     IdGenerator
}

// Creates a factory with all generators pre-initialized using distinct snowflake node IDs.
func NewIdGeneratorFactory() *IdGeneratorFactory {
	return &IdGeneratorFactory{
		Liquidation:    newSnowflakeIdGenerator(1),
		Order:          newSnowflakeIdGenerator(2),
		OrderHistory:   newSnowflakeIdGenerator(3),
		Position:       newSnowflakeIdGenerator(4),
		SubAccount:     newSnowflakeIdGenerator(5),
		TradeHistory:   newSnowflakeIdGenerator(6),
		Transfer:       newSnowflakeIdGenerator(7),
		Withdrawal:     newSnowflakeIdGenerator(8),
		FundingHistory: newSnowflakeIdGenerator(9),
	}
}

func (f *IdGeneratorFactory) NewFundingHistoryID() int64 { return f.FundingHistory.NewID() }
func (f *IdGeneratorFactory) NewLiquidationID() int64    { return f.Liquidation.NewID() }
func (f *IdGeneratorFactory) NewOrderID() int64        { return f.Order.NewID() }
func (f *IdGeneratorFactory) NewOrderHistoryID() int64 { return f.OrderHistory.NewID() }
func (f *IdGeneratorFactory) NewPositionID() int64     { return f.Position.NewID() }
func (f *IdGeneratorFactory) NewSubAccountID() int64   { return f.SubAccount.NewID() }
func (f *IdGeneratorFactory) NewTradeHistoryID() int64 { return f.TradeHistory.NewID() }
func (f *IdGeneratorFactory) NewTransferID() int64     { return f.Transfer.NewID() }
func (f *IdGeneratorFactory) NewWithdrawalID() int64   { return f.Withdrawal.NewID() }
