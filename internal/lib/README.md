# V4 Offchain Services Docker Compose

This Docker Compose configuration provides a complete setup for running all V4 Offchain services (10 microservices) along with their dependencies.

## Services Included

### Core Infrastructure
1. **NATS Server** (with JetStream enabled) - Message broker for service communication
2. **Redis Cluster** - Data storage and caching (standalone and cluster modes)
3. **PostgreSQL** - Database for Market Config and Subaccount services
4. **NATS Box** - Utility container for NATS management commands

### Application Services
5. **API Gateway** - REST API gateway for client requests
6. **Trading Service** - gRPC service for order management and margin calculations
7. **Matching Service** - High-performance order matching engine
8. **Market Data Service** - gRPC API for orderbook snapshots
9. **Pricing Service** - Price feed aggregation from multiple sources
10. **WebSocket Service** - Real-time bidirectional communication
11. **Market Config Service** - REST API for market configuration management
12. **Subaccount Service** - gRPC service for account management
13. **Relayer Service** - Ethereum L1 event listening and publishing
14. **Sync Bridge Service** - Blockchain synchronization

## Prerequisites

- Docker and Docker Compose installed
- Go 1.24.5+ installed (for local development)
- Access to Ethereum RPC endpoints (for Relayer service)
- The project uses Go workspace configuration for all services

## Usage

### Building with Private Dependencies

Start all services using Docker Compose:

```bash
# Start all services
docker compose up

# Start in background
docker compose up -d
```

### Stop all services

```bash
docker compose down
```

### View logs

```bash
# All services
docker compose logs -f

# Specific services
docker compose logs -f api-gateway
docker compose logs -f trading-service
docker compose logs -f websocket-service
docker compose logs -f pricing-service
```

### Rebuild services after code changes

```bash
# Rebuild specific service
docker compose build trading-service
docker compose up -d trading-service

# Rebuild all services
docker compose build
docker compose up -d
```

## Service Endpoints

### HTTP/REST Services
- **API Gateway**: http://localhost:8080
- **Market Config Service**: http://localhost:8081
- **WebSocket Service**: http://localhost:8090
- **Pricing Service Metrics**: http://localhost:9000/metrics

### gRPC Services
- **Trading Service**: localhost:50051
- **Matching Service**: localhost:50052
- **Market Data Service**: localhost:50053
- **Subaccount Service**: localhost:50054

### Infrastructure
- **NATS**: localhost:4222
- **NATS Monitoring**: http://localhost:8222
- **Redis**: localhost:6379
- **PostgreSQL**: localhost:5432

## Environment Variables

The services are configured with default environment variables in the compose.yml file. You can override these by creating a `.env` file in the same directory.

Key environment prefixes by service:
- `SNXAPI_` - API Gateway
- `SNXTRADING_` - Trading Service
- `SNXMATCHING_` - Matching Service
- `SNXMARKETDATA_` - Market Data Service
- `SNXPRICING_` - Pricing Service
- `SNXMARKETCONFIG_` - Market Config Service
- `SNXSUBACCOUNT_` - Subaccount Service
- `SNXRELAYER_` - Relayer Service

## Network

All services are connected via the `v4-offchain-network` bridge network, allowing them to communicate using service names as hostnames.

## Volumes

Persistent data is stored in Docker volumes:

- `nats-data` - NATS JetStream data
- `redis-cluster-data` - Redis cluster data
- `postgres-data` - PostgreSQL database data
- `matching-data` - Matching service snapshots

## NATS Management

You can use the NATS Box container to run NATS CLI commands:

```bash
# Access NATS Box shell
docker exec -it v4-offchain-nats-box sh

# Inside the container, you can run NATS commands
nats -s nats://nats:4222 stream ls
nats -s nats://nats:4222 consumer ls snx-v1-ORDERS
```

## Troubleshooting

1. **Services fail to start**: Check that all required ports are free on your host
2. **Connection issues**: Ensure all services are healthy using `docker-compose ps`
3. **Build failures**: Make sure you have the correct directory structure and all source files
4. **Private dependency errors**: Run `./build.sh` instead of `docker-compose up` to ensure dependencies are vendored
