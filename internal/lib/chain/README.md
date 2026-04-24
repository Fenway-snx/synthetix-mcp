# Chain Package

The `chain` package provides a Go client for interacting with Ethereum smart contracts, specifically designed for the Synthetix V4 offchain system.

## Features

- Multi-chain support with chain-specific configurations
- Environment-based configuration for RPC URLs and private keys
- Thread-safe client management
- Contract ABI management and method calling
- Built-in support for CoreProxy and PerpsMarketProxy contracts

## Installation

Ensure you have the required dependencies in your `go.mod`:

```go
require (
    github.com/ethereum/go-ethereum v1.13.5
    github.com/joho/godotenv v1.5.1
    github.com/pkg/errors v0.9.1
)
```

## Configuration

### Environment Variables

The package expects the following environment variables for each chain:

- `RPC_URL_<CHAIN_ID>`: The RPC endpoint URL for the chain
- `PRIVATE_KEY_<CHAIN_ID>`: The private key (without 0x prefix) for signing transactions

Example for Sepolia testnet (chain ID 11155111):

```bash
RPC_URL_11155111=https://sepolia.infura.io/v3/YOUR_INFURA_KEY
PRIVATE_KEY_11155111=your_private_key_without_0x_prefix
```

### Using .env File

You can use a `.env` file for local development:

```env
# Sepolia testnet
RPC_URL_11155111=https://sepolia.infura.io/v3/YOUR_INFURA_KEY
PRIVATE_KEY_11155111=your_private_key_without_0x_prefix

# Mainnet (example)
RPC_URL_1=https://mainnet.infura.io/v3/YOUR_INFURA_KEY
PRIVATE_KEY_1=your_mainnet_private_key_without_0x_prefix
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "log"
    "math/big"

    "github.com/Synthetixio/v4-offchain-lib/chain"
    "github.com/joho/godotenv"
)

func main() {
    // Load environment variables
    _ = godotenv.Load()

    // Create chain client manager
    manager := chain.NewChainClientManager()

    // Get client for Sepolia testnet
    client, err := manager.GetChainClient(11155111)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Load contracts
    err = client.LoadContracts(
        "0x76490713314fCEC173f44e99346F54c6e92a8E42", // CoreProxy address
        "0x75c43165ea38cB857C45216a37C5405A7656673c", // PerpsMarketProxy address
    )
    if err != nil {
        log.Fatal(err)
    }

    // Call getAccountOwner
    accountID := big.NewInt(1234)
    owner, err := client.GetAccountOwner(accountID)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Account %s owner: %s\n", accountID.String(), owner.Hex())
}
```

### Adding New Chains

To add support for a new chain, update the `SupportedChains` map in `config.go`:

```go
var SupportedChains = map[int64]*ChainConfig{
    11155111: {
        ChainID:             11155111,
        Name:                "sepolia",
        PackageName:         "synthetix-omnibus",
        PerpsPackageContract: "snx_v4_onchain.PerpsMarketProxy",
    },
    // Add your new chain here
    1: {
        ChainID:             1,
        Name:                "mainnet",
        PackageName:         "synthetix-omnibus",
        PerpsPackageContract: "snx_v4_onchain.PerpsMarketProxy",
    },
}
```

### Adding New Contract Methods

To add new contract methods:

1. Update the contract ABI in `contracts.go`
2. Add a convenience method in `client.go` (optional)

Example:

```go
// In contracts.go - add to CoreProxyABI
const CoreProxyABI = `[
    // ... existing methods ...
    {
        "inputs": [{"internalType": "uint128", "name": "accountId", "type": "uint128"}],
        "name": "getAccountBalance",
        "outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
        "stateMutability": "view",
        "type": "function"
    }
]`

// In client.go - add convenience method
func (c *ChainClient) GetAccountBalance(accountID *big.Int) (*big.Int, error) {
    var balance *big.Int
    err := c.CallContract("CoreProxy", "getAccountBalance", &balance, accountID)
    if err != nil {
        return nil, err
    }
    return balance, nil
}
```

## Contract Addresses

You'll need to manually provide contract addresses for each chain. These can be obtained from:

- Synthetix deployment documentation
- Cannon deployment artifacts
- Chain explorers

## Error Handling

The package uses wrapped errors for better debugging:

```go
client, err := manager.GetChainClient(11155111)
if err != nil {
    // Error will include context about what failed
    log.Fatalf("Failed to get chain client: %v", err)
}
```

## Thread Safety

The `ChainClientManager` and `ChainClient` are thread-safe and can be used concurrently from multiple goroutines.

## TODO

- [ ] Add support for loading contract addresses from Cannon
- [ ] Add more contract methods as needed
- [ ] Add transaction sending capabilities
- [ ] Add event listening support
- [ ] Add WebSocket support for real-time updates
