package solprogram

import (
	"context"
	"encoding/base64"
	"fmt"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Client wraps Solana RPC client
type Client struct {
	RPC       *rpc.Client
	ProgramID solana.PublicKey
}

// NewClient creates new Solana program client
func NewClient(rpcURL string, programID string) (*Client, error) {
	rpcClient := rpc.New(rpcURL)

	programPubkey, err := solana.PublicKeyFromBase58(programID)
	if err != nil {
		return nil, fmt.Errorf("invalid program ID: %w", err)
	}

	return &Client{
		RPC:       rpcClient,
		ProgramID: programPubkey,
	}, nil
}

// CreateTransaction creates unsigned transaction for single instruction
func (c *Client) CreateTransaction(
	instruction solana.Instruction,
	payer solana.PublicKey,
) (string, error) {
	return c.CreateTransactionWithInstructions([]solana.Instruction{instruction}, payer)
}

// CreateTransactionWithInstructions creates unsigned transaction for multiple instructions
func (c *Client) CreateTransactionWithInstructions(
	instructions []solana.Instruction,
	payer solana.PublicKey,
) (string, error) {
	ctx := context.Background()
	recent, err := c.RPC.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(payer),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Serialize to base64
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to serialize: %w", err)
	}

	return base64.StdEncoding.EncodeToString(txBytes), nil
}

// SendTransaction sends signed transaction
func (c *Client) SendTransaction(signedTxBase64 string) (string, error) {
	// Decode
	txBytes, err := base64.StdEncoding.DecodeString(signedTxBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode: %w", err)
	}

	// Parse transaction
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		return "", fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Send
	sig, err := c.RPC.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("failed to send: %w", err)
	}

	return sig.String(), nil
}
