package chain

import (
	"fmt"
	"math/big"
)

// Chain ID constants for supported networks
const (
	MainnetChainID = 1
	SepoliaChainID = 11155111
	// Add more chain IDs as needed
)

// ChainConfig represents configuration for a specific blockchain
type ChainConfig struct {
	ChainID              int64  `json:"chainId"`
	Name                 string `json:"name"`
	PackageName          string `json:"packageName"`
	PerpsPackageContract string `json:"perpsPackageContract"`
}

// SupportedChains contains all supported chain configurations
var SupportedChains = map[int64]*ChainConfig{
	SepoliaChainID: {
		ChainID:              SepoliaChainID,
		Name:                 "sepolia",
		PackageName:          "synthetix-omnibus",
		PerpsPackageContract: "snx_v4_onchain.PerpsMarketProxy",
	},
	// Add more chains as needed
}

// GetChainConfig returns the configuration for a specific chain ID
func GetChainConfig(chainID int64) (*ChainConfig, error) {
	config, exists := SupportedChains[chainID]
	if !exists {
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
	return config, nil
}

// GetRPCURL returns the RPC URL for a specific chain from environment
func GetRPCURL(chainID int64) string {
	return fmt.Sprintf("RPC_URL_%d", chainID)
}

// GetPrivateKey returns the private key environment variable name for a specific chain
func GetPrivateKey(chainID int64) string {
	return fmt.Sprintf("PRIVATE_KEY_%d", chainID)
}

// ChainIDToBigInt converts chain ID to big.Int
func ChainIDToBigInt(chainID int64) *big.Int {
	return big.NewInt(chainID)
}
