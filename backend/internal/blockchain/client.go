package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	EthClient  *ethclient.Client
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
	ChainID    *big.Int
	Contract   common.Address
}

func NewClient(rpcURL string, privateKeyHex string, chainID int64, contractAddr string) (*Client, error) {
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Verify chain ID
	networkChainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}
	if networkChainID.Int64() != chainID {
		return nil, fmt.Errorf("chain ID mismatch: expected %d, got %d", chainID, networkChainID.Int64())
	}

	return &Client{
		EthClient:  ethClient,
		PrivateKey: privateKey,
		Address:    address,
		ChainID:    big.NewInt(chainID),
		Contract:   common.HexToAddress(contractAddr),
	}, nil
}
