package chain

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Contract represents a smart contract with its address and ABI
type Contract struct {
	Address common.Address
	ABI     abi.ABI
}

// For now, we'll define minimal ABIs for the methods we need
// In production, these should be loaded from JSON files or generated from Solidity

// CoreProxy ABI - minimal definition for getAccountOwner
const CoreProxyABI = `[
	{
		"inputs": [
			{
				"internalType": "uint128",
				"name": "accountId",
				"type": "uint128"
			}
		],
		"name": "getAccountOwner",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

// PerpsMarketProxy ABI - placeholder for perps market functions
const PerpsMarketProxyABI = `[
	{
		"inputs": [],
		"name": "name",
		"outputs": [
			{
				"internalType": "string",
				"name": "",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

// ParseABI parses an ABI string and returns the ABI object
func ParseABI(abiStr string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(abiStr))
}

// GetContractABI returns the ABI for a specific contract type
func GetContractABI(contractName string) (string, error) {
	switch contractName {
	case "CoreProxy":
		return CoreProxyABI, nil
	case "PerpsMarketProxy":
		return PerpsMarketProxyABI, nil
	default:
		return "", fmt.Errorf("unknown contract: %s", contractName)
	}
}

// LoadContract creates a Contract instance with parsed ABI
func LoadContract(address common.Address, abiStr string) (*Contract, error) {
	parsedABI, err := ParseABI(abiStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &Contract{
		Address: address,
		ABI:     parsedABI,
	}, nil
}
