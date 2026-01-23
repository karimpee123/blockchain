package chainbnb

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
)

type BNBChain struct {
	client  *ethclient.Client
	chainID int64
	network string // mainnet, testnet
}

type Config struct {
	RPCURL  string
	ChainID int64
	Network string
}

// NewBNBChain - Initialize BNB Chain
func NewBNBChain(config Config) *BNBChain {
	if config.Network == "" {
		config.Network = "testnet"
	}
	if config.ChainID == 0 {
		config.ChainID = 97 // BSC Testnet
	}

	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		log.Fatal(err)
	}

	return &BNBChain{
		client:  client,
		chainID: config.ChainID,
		network: config.Network,
	}
}

// GetExplorerURL - Generate explorer URL
func (b *BNBChain) GetExplorerURL(txHash string) string {
	baseURL := "https://bscscan.com/tx/"
	if b.network == "testnet" {
		baseURL = "https://testnet.bscscan.com/tx/"
	}
	return baseURL + txHash
}

// HealthCheck - Check connection to BNB Chain
func (b *BNBChain) HealthCheck() error {
	_, err := b.client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("BNB Chain health check failed: %w", err)
	}
	return nil
}
