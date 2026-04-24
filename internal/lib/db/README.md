# Database Library

This shared library provides database access for all services in the v4-offchain system.

## Architecture

```
lib/db/
├── redis/          # Redis client wrapper
├── models/         # Shared data models
└── repository/     # Repository pattern implementations
```

## Design Principles

1. **Fast Reads**: API service reads directly from Redis for sub-millisecond latency
2. **Write Through Cache**: Trading service writes to Redis immediately after state changes
3. **Shared Models**: All services use the same data structures
4. **Atomic Updates**: Critical operations use Redis transactions

## Data Flow

```
Order Request → API → Trading Service → Redis → API (for reads)
                           ↓
                      State Updates
                           ↓
                    Margin Calculations
```

## Redis Data Structure

```
account:{id}                    # Full account data (JSON)
account:{id}:positions          # Positions array (JSON)
account:{id}:orders             # Open orders array (JSON)
account:{id}:margin             # Margin info (JSON)
```

## Usage

### In Trading Service (Write)

```go
import (
    "github.com/Synthetixio/v4-offchain-v3/lib/db/redis"
    "github.com/Synthetixio/v4-offchain-v3/lib/db/repository"
    "github.com/Synthetixio/v4-offchain-v3/lib/db/models"
)

// Initialize
redisClient, _ := redis.NewClient(redis.Config{
    Addr: "localhost:6379",
})
accountRepo := repository.NewAccountRepository(redisClient)

// Update account after order fill
account := &models.Account{
    ID: 12345,
    Positions: []models.Position{
        {
            ID:            1,
            MarketID:      1,
            Symbol:        "BTC-USD",
            Size:          50000000, // 0.5 BTC
            AvgEntryPrice: 4500000000000,
        },
    },
    // ... other fields
}

err := accountRepo.SaveAccount(ctx, account)
```

### In API Service (Read)

```go
// Get full account
account, err := accountRepo.GetAccount(ctx, 12345)

// Get only positions (faster)
positions, err := accountRepo.GetPositions(ctx, 12345)

// Get only open orders (faster)
orders, err := accountRepo.GetOpenOrders(ctx, 12345)
```

## Performance Considerations

1. **Separate Keys**: Positions and orders are stored separately for faster partial reads
2. **No Joins**: All data is denormalized for speed
3. **JSON Storage**: Simple serialization, fast parsing
4. **Connection Pooling**: Redis client manages connection pool

## Future Enhancements

1. **PostgreSQL**: Add repository for historical data
2. **Caching**: Add local in-memory cache with TTL
3. **Pub/Sub**: Real-time updates via Redis pub/sub
4. **Compression**: Compress large account data
