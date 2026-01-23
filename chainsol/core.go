package src

import (
	"context"
	"log"

	"gorm.io/gorm"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

type SolChain struct {
	http    *rpc.Client
	ws      *ws.Client
	db      *gorm.DB
	network string // mainnet, devnet, testnet
}

type Config struct {
	RPCURL  string
	WSURL   string
	Network string
}

// NewSolChain - Initialize Solana
func NewSolChain(config Config) *SolChain {
	if config.Network == "" {
		config.Network = "mainnet"
	}
	http := rpc.New(config.RPCURL)
	wss, err := ws.Connect(context.TODO(), config.WSURL)
	if err != nil {
		log.Fatal(err)
	}

	return &SolChain{
		http:    http,
		ws:      wss,
		network: config.Network,
	}
}

// GetExplorerURL - Generate explorer URL
func (p *SolChain) GetExplorerURL(signature string) string {
	baseURL := "https://explorer.solana.com/tx/"
	switch p.network {
	case "devnet":
		return baseURL + signature + "?cluster=devnet"
	case "testnet":
		return baseURL + signature + "?cluster=testnet"
	default:
		return baseURL + signature
	}
}

// Health check
func (p *SolChain) HealthCheck() error {
	_, err := p.http.GetHealth(context.Background())
	return err
}
