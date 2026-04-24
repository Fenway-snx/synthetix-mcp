package pricing

import (
	"encoding/json"
	"fmt"
	"sync"

	nats_api "github.com/nats-io/nats.go"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

type priceTypeToPriceMap map[snx_lib_core.PriceType]snx_lib_api_types.Price
type marketNameToPriceTypeToPriceMap map[snx_lib_core.MarketName]priceTypeToPriceMap

// Thread-safe in-memory cache of the latest prices, populated by a NATS
// subscription to the pricing service's batch broadcast. Intended for use
// by any service that needs current mark/index/last prices without reaching
// into Redis.
type PriceCache struct {
	logger snx_lib_logging.Logger
	mu     sync.RWMutex
	prices marketNameToPriceTypeToPriceMap // market-name → feed → price (decimal string)
	sub    *nats_api.Subscription
}

// Subscribes to the prices.batch NATS subject and begins populating the
// cache. Returns an error if the subscription fails. Call Stop() to
// unsubscribe.
func NewPriceCache(logger snx_lib_logging.Logger, nc *nats_api.Conn) (*PriceCache, error) {
	pc := &PriceCache{
		logger: logger,
		prices: make(marketNameToPriceTypeToPriceMap),
	}

	subject := snx_lib_db_nats.PriceBatchUpdate.String()
	sub, err := nc.Subscribe(subject, pc.handleBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}
	pc.sub = sub

	logger.Info("price cache subscribed", "subject", subject)

	return pc, nil
}

func (pc *PriceCache) handleBatch(msg *nats_api.Msg) {
	var batch snx_lib_core.PriceBatchUpdate
	if err := json.Unmarshal(msg.Data, &batch); err != nil {
		pc.logger.Error("failed to unmarshal price batch", "error", err)
		return
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	for name, elem := range batch.MarkPrices {
		pc.setLocked(name, snx_lib_core.PriceType_mark, elem.CurrentPrice)
	}
	for name, elem := range batch.IndexPrices {
		pc.setLocked(name, snx_lib_core.PriceType_index, elem.CurrentPrice)
	}
	for name, elem := range batch.LastPrices {
		pc.setLocked(name, snx_lib_core.PriceType_last, elem.CurrentPrice)
	}
}

func (pc *PriceCache) setLocked(
	marketName snx_lib_core.MarketName,
	priceType snx_lib_core.PriceType,
	price snx_lib_core.Price,
) {
	if pc.prices[marketName] == nil {
		pc.prices[marketName] = make(priceTypeToPriceMap)
	}

	pc.prices[marketName][priceType] = snx_lib_api_types.PriceFromStringUnvalidated(price.String())
}

// Returns the latest price string for the given market and feed, or ""
// if no price has been received yet.
func (pc *PriceCache) GetPrice(
	marketName snx_lib_core.MarketName,
	priceType snx_lib_core.PriceType,
) snx_lib_api_types.Price {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if feeds, ok := pc.prices[marketName]; ok {
		return feeds[priceType]
	}

	return snx_lib_api_types.Price_None
}

// Unsubscribes from the NATS subject.
func (pc *PriceCache) Stop() {
	if pc.sub != nil {
		_ = pc.sub.Unsubscribe()
	}
}
