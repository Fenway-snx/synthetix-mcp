# Info Endpoint Documentation

This document describes the available endpoints under `/v1/info`.

## getExchangeStatus

Get whether the exchange is accepting normal API traffic.

This action is designed for integrators to safely poll during logical
maintenance (services are still listening but may reject trading and other
public actions). During maintenance, `getExchangeStatus` remains available
while other public actions return HTTP 503.

### REST bindings

- `GET /v1/exchange/status` (API service)
- `GET /v1/ws/exchange/status` (WebSocket service — reports the WS instance's own halt state)

### Response payload (APIResponse.response)

```json
{
  "accepting_orders": true,
  "exchange_status": "RUNNING",
  "code": "",
  "message": "OK",
  "timestamp_ms": 1704067200000
}
```

### Fields

- `accepting_orders`: `true` if normal traffic is accepted.
- `exchange_status`: `RUNNING` or `MAINTENANCE`.
- `code`: `SERVICE_DRAINING` during maintenance; `STATUS_DEGRADED` when Redis is unavailable (best-effort status).
- `message`: human-readable, non-leaking status.
- `timestamp_ms`: server time in milliseconds (UTC).

## getLastTrades

Get recent trades for a specified market from all users.

### Request Format
```json
{
  "type": "getLastTrades",
  "symbol": "BTC-USDT",
  "limit": 20,
  "offset": 0
}
```

### Parameters
- `symbol` (required): Trading pair symbol (e.g., "BTC-USDT")
- `limit` (optional): Number of trades to return. Default: 50, Maximum: 100
- `offset` (optional): Pagination offset for retrieving additional pages. Default: 0

### Response Format
```json
{
  "status": "ok",
  "response": {
    "trades": [
      {
        "tradeId": "123456789",
        "symbol": "BTC-USDT",
        "side": "buy",
        "price": "50000.50",
        "quantity": "0.1",
        "timestamp": 1704067200500,
        "isMaker": false
      },
      {
        "tradeId": "123456788",
        "symbol": "BTC-USDT",
        "side": "sell",
        "price": "50000.25",
        "quantity": "0.05",
        "timestamp": 1704067199800,
        "isMaker": true
      }
    ]
  }
}
```

### Response Fields
- `trades`: Array of recent trades for the specified market
  - `tradeId`: Unique trade identifier (string)
  - `symbol`: Trading pair symbol (string)
  - `side`: Trade side - "buy" or "sell" (string)
  - `price`: Execution price (string)
  - `quantity`: Executed quantity (string)
  - `timestamp`: Trade execution time in milliseconds (int64)
  - `isMaker`: Whether this trade was executed by a maker order (boolean)

### Error Responses
- `400 Bad Request`: Invalid parameters (missing symbol, invalid limit/offset)
- `500 Internal Server Error`: Service unavailable or internal error

### Usage Notes
- Returns recent trades from all users for the specified market (public trade feed)
- Trades are sorted by timestamp (most recent first)
- Maximum limit of 100 trades per request for performance
- Use `offset` parameter for pagination to retrieve additional pages

### Pagination Example
```javascript
// Get first 50 trades
{
  "type": "getLastTrades",
  "symbol": "BTC-USDT",
  "limit": 50,
  "offset": 0
}

// Get next 50 trades
{
  "type": "getLastTrades",
  "symbol": "BTC-USDT",
  "limit": 50,
  "offset": 50
}
```

## getOrderbook

Get current orderbook depth for a trading pair.

### Request Format
```json
{
  "type": "getOrderbook",
  "symbol": "BTC-USDT",
  "limit": 20
}
```

### Parameters
- `symbol` (required): Trading pair symbol (e.g., "BTC-USDT")
- `limit` (optional): Number of price levels to return. Valid values: 5, 10, 20, 50, 100, 500, 1000. Default: 500

### Response Format
```json
{
  "status": "ok",
  "response": {
    "bids": [
      ["45000.00", "1.5"],
      ["44999.50", "2.0"]
    ],
    "asks": [
      ["45001.00", "1.2"],
      ["45001.50", "3.1"]
    ]
  }
}
```

### Response Fields
- `bids`: Array of [price, quantity] arrays for buy orders, sorted by price descending
- `asks`: Array of [price, quantity] arrays for sell orders, sorted by price ascending
