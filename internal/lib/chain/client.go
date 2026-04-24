package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

var (
	errFailedToCastPublicKeyToECDSA = errors.New("failed to cast public key to ECDSA")
)

// ChainClient represents a client for interacting with a specific blockchain
type ChainClient struct {
	ChainID      int64
	PublicClient *ethclient.Client
	Auth         *bind.TransactOpts
	PrivateKey   *ecdsa.PrivateKey
	Account      common.Address
	Contracts    map[string]*Contract
	mutex        sync.RWMutex
}

// ChainClientManager manages multiple chain clients
type ChainClientManager struct {
	clients map[int64]*ChainClient
	mutex   sync.RWMutex
}

// NewChainClientManager creates a new chain client manager
func NewChainClientManager() *ChainClientManager {
	return &ChainClientManager{
		clients: make(map[int64]*ChainClient),
	}
}

// GetChainClient returns a chain client for the specified chain ID
func (m *ChainClientManager) GetChainClient(chainID int64) (*ChainClient, error) {
	m.mutex.RLock()
	client, exists := m.clients[chainID]
	m.mutex.RUnlock()

	if exists {
		return client, nil
	}

	// Create new client
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if client, exists = m.clients[chainID]; exists {
		return client, nil
	}

	// Get chain config
	chainConfig, err := GetChainConfig(chainID)
	if err != nil {
		return nil, err
	}

	// Create client
	client, err = createChainClient(chainConfig)
	if err != nil {
		return nil, err
	}

	m.clients[chainID] = client
	return client, nil
}

// createChainClient creates a new chain client for the given configuration
func createChainClient(config *ChainConfig) (*ChainClient, error) {
	// Get RPC URL from environment
	rpcURL := os.Getenv(GetRPCURL(config.ChainID))
	if rpcURL == "" {
		return nil, fmt.Errorf("no RPC_URL_%d environment variable set", config.ChainID)
	}

	// Get private key from environment
	privateKeyHex := os.Getenv(GetPrivateKey(config.ChainID))
	if privateKeyHex == "" {
		return nil, fmt.Errorf("no PRIVATE_KEY_%d environment variable set", config.ChainID)
	}

	// Connect to the Ethereum client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Ethereum client")
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse private key")
	}

	// Get account address from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errFailedToCastPublicKeyToECDSA
	}
	account := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Create auth transactor
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, ChainIDToBigInt(config.ChainID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transactor")
	}

	return &ChainClient{
		ChainID:      config.ChainID,
		PublicClient: client,
		Auth:         auth,
		PrivateKey:   privateKey,
		Account:      account,
		Contracts:    make(map[string]*Contract),
	}, nil
}

// LoadContracts loads the standard contracts (CoreProxy and PerpsMarketProxy)
func (c *ChainClient) LoadContracts(coreProxyAddr, perpsMarketAddr string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Load CoreProxy
	if coreProxyAddr != "" {
		coreABI, err := GetContractABI("CoreProxy")
		if err != nil {
			return err
		}
		coreContract, err := LoadContract(common.HexToAddress(coreProxyAddr), coreABI)
		if err != nil {
			return errors.Wrap(err, "failed to load CoreProxy")
		}
		c.Contracts["CoreProxy"] = coreContract
	}

	// Load PerpsMarketProxy
	if perpsMarketAddr != "" {
		perpsABI, err := GetContractABI("PerpsMarketProxy")
		if err != nil {
			return err
		}
		perpsContract, err := LoadContract(common.HexToAddress(perpsMarketAddr), perpsABI)
		if err != nil {
			return errors.Wrap(err, "failed to load PerpsMarketProxy")
		}
		c.Contracts["PerpsMarketProxy"] = perpsContract
	}

	return nil
}

// GetContract returns a loaded contract by name
func (c *ChainClient) GetContract(name string) (*Contract, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	contract, exists := c.Contracts[name]
	if !exists {
		return nil, fmt.Errorf("contract %s not loaded", name)
	}
	return contract, nil
}

// CallContract makes a read-only call to a smart contract
func (c *ChainClient) CallContract(contractName string, method string, result any, args ...any) error {
	contract, err := c.GetContract(contractName)
	if err != nil {
		return err
	}

	// Pack the method call
	data, err := contract.ABI.Pack(method, args...)
	if err != nil {
		return errors.Wrap(err, "failed to pack method call")
	}

	// Make the call
	msg := ethereum.CallMsg{
		To:   &contract.Address,
		Data: data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	output, err := c.PublicClient.CallContract(ctx, msg, nil)
	if err != nil {
		return errors.Wrap(err, "failed to call contract")
	}

	// Unpack the result
	err = contract.ABI.UnpackIntoInterface(result, method, output)
	if err != nil {
		return errors.Wrap(err, "failed to unpack result")
	}

	return nil
}

// GetAccountOwner calls the getAccountOwner method on the CoreProxy contract
func (c *ChainClient) GetAccountOwner(accountID *big.Int) (common.Address, error) {
	var owner common.Address
	err := c.CallContract("CoreProxy", "getAccountOwner", &owner, accountID)
	if err != nil {
		return common.Address{}, err
	}
	return owner, nil
}

// Close closes the client connection
func (c *ChainClient) Close() {
	if c.PublicClient != nil {
		c.PublicClient.Close()
	}
}

// InitializeChainClients initializes clients for all configured chains
func (m *ChainClientManager) InitializeChainClients(chainIDs []int64) error {
	for _, chainID := range chainIDs {
		_, err := m.GetChainClient(chainID)
		if err != nil {
			return errors.Wrapf(err, "failed to initialize client for chain %d", chainID)
		}
	}
	return nil
}
