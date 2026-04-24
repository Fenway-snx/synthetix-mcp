package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-viper/mapstructure/v2"
	go_redis "github.com/redis/go-redis/v9"
	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/vmihailenco/msgpack/v5"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

const candlesAggregateScript = `
if ARGV[1] == 'init' then return end
local windowStart = tonumber(ARGV[1])
local windowEnd = tonumber(ARGV[2])
local klineSize = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])

local klines = {}
local curKline

local lastTs = 0
local lastPrice = 0

local function rollKline(startTime, match)
	if curKline ~= nil then
		-- add the last price to the weighted average and then divide by the time period
		curKline['weightedAvgPrice'] = math.floor(
			(curKline['weightedAvgPrice'] + lastPrice * (curKline['closeTime'] + 1 - lastTs)) /
			((curKline['closeTime'] + 1 - curKline['openTime']))
		)
		curKline['priceChange'] = curKline['lastPrice'] - curKline['openPrice']

        if curKline['openPrice'] > 0 then
            curKline['priceChangePercent'] = curKline['priceChange'] / curKline['openPrice'] * 100
        end

		-- remove table keys and return values in order
		klines[#klines + 1] = {
			tostring(curKline['priceChange']),
			tostring(curKline['priceChangePercent']),
			tostring(curKline['weightedAvgPrice']),
			tostring(curKline['openPrice']),
			tostring(curKline['highPrice']),
			tostring(curKline['lowPrice']),
			tostring(curKline['lastPrice']),
			tostring(curKline['volume']),
			tostring(curKline['quoteVolume']),
			tostring(curKline['openTime']),
			tostring(curKline['closeTime']),
			tostring(curKline['firstId']),
			tostring(curKline['lastId']),
			tostring(curKline['count'])
		}
	end

	curKline = {
		priceChange = 0,
		priceChangePercent = 0,
		weightedAvgPrice = 0,
		openPrice = 0,
		highPrice = 0,
		lowPrice = 0,
		lastPrice = 0,
		volume = 0,
		quoteVolume = 0,
		openTime = startTime,
		closeTime = startTime + klineSize - 1,
		firstId = 0,
		lastId = 0,
		count = 0
	}

	if match then
        local price = match[2]
        curKline['openPrice'] = price
        curKline['lastPrice'] = price
        curKline['highPrice'] = price
        curKline['lowPrice'] = price
        if lastPrice == 0 then
            lastPrice = price or 0
        end
	end

	lastTs = curKline['openTime']
end

local firstDayId = math.floor(windowStart / 86400000) - 1
local lastDayId = math.floor(windowEnd / 86400000) + 1

local function updateKlineStats(updateData, currentTs)
    curKline['lowPrice'] = math.min(updateData[3], curKline['lowPrice'])
    curKline['highPrice'] = math.max(updateData[4], curKline['highPrice'])
    curKline['lastPrice'] = updateData[2]
    curKline['weightedAvgPrice'] = curKline['weightedAvgPrice'] + lastPrice * (currentTs - lastTs)
end

for i = firstDayId, lastDayId, 1 do
    local updates = redis.call('lrange', KEYS[1] .. ':' .. i, 0, -1)
    for _, update in ipairs(updates) do
        local updateData = cmsgpack.unpack(update)
        local currentTs = updateData[1] * 1000
        if currentTs >= windowEnd then
            break
        end

        if (not curKline and currentTs >= windowStart) or
            (curKline and currentTs > curKline['closeTime']) then

            -- we need to get the window that contains this kline. unfortunately this math is a bit funky because of the windowStart offset consideation
            local windowStartAdjustment = currentTs % klineSize - windowStart % klineSize
            if windowStartAdjustment < 0 then
                windowStartAdjustment = windowStartAdjustment + klineSize
            end

            rollKline(currentTs - windowStartAdjustment, updateData)

			if #klines + 1 >= limit then
				break
			end
        end

        if curKline then
            if currentTs >= curKline['closeTime'] then
                break
            end

            updateKlineStats(updateData, currentTs)
        end

        lastPrice = updateData[2]
        lastTs = currentTs + 999
    end

	if #klines + 1 >= limit then
		break
	end
end

rollKline(windowEnd, nil)

return klines
`

// PriceRepository handles price-related data in Redis
type PriceRepository struct {
	rc *snx_lib_db_redis.SnxClient
}

func NewPriceRepository(rc *snx_lib_db_redis.SnxClient) *PriceRepository {
	return &PriceRepository{
		rc: rc,
	}
}

func DecodePriceData(data []byte) (PriceData, error) {
	var rawPriceData RawPriceData
	if err := msgpack.Unmarshal(data, &rawPriceData); err != nil {
		return PriceData{}, fmt.Errorf("failed to unmarshal price data: %w", err)
	}
	// for lua compatibility we store the price as a number. so divide by 8 here to put it in system form
	parsedPrice, err := shopspring_decimal.NewFromString(rawPriceData.Price)
	if err != nil {
		return PriceData{}, fmt.Errorf("failed to parse price: %w", err)
	}
	priceData := PriceData{
		PublishTime: uint64(rawPriceData.PublishTime),
		Price:       parsedPrice,
	}
	return priceData, nil
}

// GetPriceHistory retrieves recent price history for a symbol and feed
func (r *PriceRepository) GetPriceHistory(
	ctx context.Context,
	symbol string,
	feed PriceType,
	limit int,
) ([]PriceData, error) {
	key := fmt.Sprintf("prices:%s:%s", symbol, feed)

	values, err := r.rc.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get price history: %w", err)
	}

	prices := make([]PriceData, 0, len(values))
	for _, value := range values {
		data, err := DecodePriceData([]byte(value))
		if err != nil {
			return nil, fmt.Errorf("failed to decode price data: %w", err)
		}
		prices = append(prices, data)
	}

	return prices, nil
}

func (r *PriceRepository) SubscribeToPriceUpdates(ctx context.Context, channels ...string) (*go_redis.PubSub, error) {
	pubsub := r.rc.PSubscribe(ctx, channels...)

	return pubsub, nil
}

type PriceData struct {
	PublishTime uint64
	Price       shopspring_decimal.Decimal
}

type RawPriceData struct {
	_           struct{} `msgpack:",as_array"`
	PublishTime int64
	Price       string
}

// Candle represents a single OHLCV candle
type Candle struct {
	OpenTime       uint64                     `json:"t"` // Open time in milliseconds
	Open           shopspring_decimal.Decimal `json:"o"` // Open price
	High           shopspring_decimal.Decimal `json:"h"` // High price
	Low            shopspring_decimal.Decimal `json:"l"` // Low price
	Close          shopspring_decimal.Decimal `json:"c"` // Close price
	Volume         shopspring_decimal.Decimal `json:"v"` // Volume
	CloseTime      uint64                     `json:"T"` // Close time in milliseconds
	NumberOfTrades uint64                     `json:"n"` // Number of trades
}

// postprocessCandle converts a raw candle to the format expected by the client
func postprocessCandle(candle []string) Candle {
	// Ensure candle has enough elements
	if len(candle) < 14 {
		// Return empty candle if not enough data
		return Candle{}
	}

	// Extract values with safe type assertions
	openTime := candle[9]
	openPrice := candle[3]
	highPrice := candle[4]
	lowPrice := candle[5]
	closePrice := candle[6]
	closeTime := candle[10]

	// Helper function to safely convert to string
	openTimeInt, err := strconv.ParseUint(openTime, 10, 64)
	if err != nil {
		return Candle{}
	}
	closeTimeInt, err := strconv.ParseUint(closeTime, 10, 64)
	if err != nil {
		return Candle{}
	}

	openPriceDecimal, err := shopspring_decimal.NewFromString(openPrice)
	if err != nil {
		return Candle{}
	}
	highPriceDecimal, err := shopspring_decimal.NewFromString(highPrice)
	if err != nil {
		return Candle{}
	}
	lowPriceDecimal, err := shopspring_decimal.NewFromString(lowPrice)
	if err != nil {
		return Candle{}
	}
	closePriceDecimal, err := shopspring_decimal.NewFromString(closePrice)
	if err != nil {
		return Candle{}
	}

	return Candle{
		OpenTime:       openTimeInt,                             // Candle open time
		Open:           openPriceDecimal,                        // Open price
		High:           highPriceDecimal,                        // High price
		Low:            lowPriceDecimal,                         // Low price
		Close:          closePriceDecimal,                       // Close price
		Volume:         shopspring_decimal.NewFromInt(int64(0)), // Volume
		CloseTime:      closeTimeInt,                            // Candle close time
		NumberOfTrades: 0,                                       // Number of trades
	}
}

func (r *PriceRepository) GetCandles(
	ctx context.Context,
	symbol string,
	feed PriceType,
	startTime int64,
	endTime int64,
	intervalMs int64,
	limit int,
) ([]Candle, error) {
	key := fmt.Sprintf("prices:%s:%s", symbol, feed)

	// TODO: use evalscript instead of eval for better preprocessing
	values, err := r.rc.Eval(ctx, candlesAggregateScript, []string{key}, startTime, endTime, intervalMs, limit).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	var valuesAsArray [][]string
	ok := mapstructure.Decode(values, &valuesAsArray)
	if ok != nil {
		return nil, fmt.Errorf("failed to convert values to [][]string: %w", ok)
	}

	candles := make([]Candle, 0, len(valuesAsArray))
	for _, value := range valuesAsArray {
		candles = append(candles, postprocessCandle(value))
	}

	return candles, nil
}
