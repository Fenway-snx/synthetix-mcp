package chain

import (
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/joho/godotenv"
)

// Example demonstrates how to use the chain client
func Example() {
	// Load environment variables from .env file (optional)
	_ = godotenv.Load()

	// Create a chain client manager
	manager := NewChainClientManager()

	// Initialize client for Sepolia testnet
	chainID := int64(SepoliaChainID)

	// Ensure environment variables are set:
	// RPC_URL_11155111=https://sepolia.infura.io/v3/YOUR_INFURA_KEY
	// PRIVATE_KEY_11155111=your_private_key_without_0x_prefix

	// Get chain client
	client, err := manager.GetChainClient(chainID)
	if err != nil {
		log.Fatalf("Failed to get chain client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Connected to chain %d\n", chainID)
	fmt.Printf("Account address: %s\n", client.Account.Hex())

	// Load contracts (you need to provide the actual contract addresses)
	coreProxyAddr := "0x..."   // Replace with actual CoreProxy address
	perpsMarketAddr := "0x..." // Replace with actual PerpsMarketProxy address

	err = client.LoadContracts(coreProxyAddr, perpsMarketAddr)
	if err != nil {
		log.Fatalf("Failed to load contracts: %v", err)
	}

	// Example: Get account owner
	accountID := big.NewInt(1234) // Replace with actual account ID
	owner, err := client.GetAccountOwner(accountID)
	if err != nil {
		log.Printf("Failed to get account owner: %v", err)
	} else {
		fmt.Printf("Account %s owner: %s\n", accountID.String(), owner.Hex())
	}

	// Example: Direct contract call
	var result any
	err = client.CallContract("CoreProxy", "getAccountOwner", &result, accountID)
	if err != nil {
		log.Printf("Failed to call contract: %v", err)
	}
}

// ExampleWithEnvFile shows how to set up environment variables
func ExampleWithEnvFile() {
	// Create a .env file in your project root with:
	/*
		RPC_URL_11155111=https://sepolia.infura.io/v3/YOUR_INFURA_KEY
		PRIVATE_KEY_11155111=your_private_key_without_0x_prefix

		# For multiple chains:
		RPC_URL_1=https://mainnet.infura.io/v3/YOUR_INFURA_KEY
		PRIVATE_KEY_1=your_mainnet_private_key_without_0x_prefix
	*/

	// Then load it:
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Now you can use the chain client as shown above
}

// GetEnvOrDefault gets an environment variable or returns a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
