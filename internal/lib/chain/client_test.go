package chain

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func Test_ChainConfig(t *testing.T) {
	// Test getting chain config
	config, err := GetChainConfig(SepoliaChainID)
	if err != nil {
		t.Fatalf("Failed to get chain config: %v", err)
	}

	if config.Name != "sepolia" {
		t.Errorf("Expected chain name 'sepolia', got '%s'", config.Name)
	}

	// Test unsupported chain
	_, err = GetChainConfig(999999)
	if err == nil {
		t.Error("Expected error for unsupported chain, got nil")
	}
}

func Test_ContractABI(t *testing.T) {
	// Test getting CoreProxy ABI
	abi, err := GetContractABI("CoreProxy")
	if err != nil {
		t.Fatalf("Failed to get CoreProxy ABI: %v", err)
	}

	if abi == "" {
		t.Error("CoreProxy ABI is empty")
	}

	// Test parsing ABI
	parsedABI, err := ParseABI(abi)
	if err != nil {
		t.Fatalf("Failed to parse CoreProxy ABI: %v", err)
	}

	// Check if getAccountOwner method exists
	method := parsedABI.Methods["getAccountOwner"]
	if method.Name != "getAccountOwner" {
		t.Error("getAccountOwner method not found in ABI")
	}
}

func Test_LoadContract(t *testing.T) {
	// Test loading a contract
	testAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	abi, _ := GetContractABI("CoreProxy")

	contract, err := LoadContract(testAddr, abi)
	if err != nil {
		t.Fatalf("Failed to load contract: %v", err)
	}

	if contract.Address != testAddr {
		t.Errorf("Contract address mismatch: expected %s, got '%s'", testAddr.Hex(), contract.Address.Hex())
	}
}

func Test_ChainClientManager(t *testing.T) {
	// Skip if environment variables are not set
	if os.Getenv("RPC_URL_11155111") == "" || os.Getenv("PRIVATE_KEY_11155111") == "" {
		t.Skip("Skipping chain client test: RPC_URL_11155111 and PRIVATE_KEY_11155111 not set")
	}

	manager := NewChainClientManager()

	// Test getting chain client
	client, err := manager.GetChainClient(SepoliaChainID)
	if err != nil {
		t.Fatalf("Failed to get chain client: %v", err)
	}
	defer client.Close()

	// Verify client properties
	if client.ChainID != SepoliaChainID {
		t.Errorf("Expected chain ID %d, got %d", SepoliaChainID, client.ChainID)
	}

	if client.PublicClient == nil {
		t.Error("Public client is nil")
	}

	if client.Auth == nil {
		t.Error("Auth is nil")
	}

	// Test getting the same client again (should return cached instance)
	client2, err := manager.GetChainClient(SepoliaChainID)
	if err != nil {
		t.Fatalf("Failed to get cached chain client: %v", err)
	}

	if client != client2 {
		t.Error("Expected cached client instance, got new instance")
	}
}
