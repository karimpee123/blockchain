package solprogram

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Client wraps Sol RPC client
type Client struct {
	RPC       *rpc.Client
	ProgramID solana.PublicKey
}

// SendTransactionResult contains transaction result and parsed error
type SendTransactionResult struct {
	Signature   string
	ErrorCode   *int
	ProgramLogs []string
}

// NewClient creates new Sol program client
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
func (c *Client) SendTransaction(signedTxBase64 string) (*SendTransactionResult, error) {
	// Decode
	txBytes, err := base64.StdEncoding.DecodeString(signedTxBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	// Parse transaction
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Send
	sig, err := c.RPC.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Printf("=== RAW ERROR ===\n%+v\n=================\n", err)

		// Parse error for additional context
		result := &SendTransactionResult{
			ProgramLogs: ExtractLogMessages(err),
		}
		errStr := err.Error()

		// Pattern 1: "Custom": 6002
		if matches := regexp.MustCompile(`"Custom":\s*(\d+)`).FindStringSubmatch(errStr); len(matches) > 1 {
			if code, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				result.ErrorCode = &code
				fmt.Printf("Extracted error code (decimal): %d\n", code)
			}
		}

		// Pattern 2: custom program error: 0x1772
		if result.ErrorCode == nil {
			if matches := regexp.MustCompile(`custom program error: 0x([0-9a-fA-F]+)`).FindStringSubmatch(errStr); len(matches) > 1 {
				if code, parseErr := strconv.ParseInt(matches[1], 16, 64); parseErr == nil {
					intCode := int(code)
					result.ErrorCode = &intCode
					fmt.Printf("Extracted error code (hex): 0x%s = %d\n", matches[1], intCode)
				}
			}
		}

		return result, fmt.Errorf("failed to send: %w", err)
	}
	return &SendTransactionResult{
		Signature: sig.String(),
	}, nil
}

// SendTransactionSimple : Legacy function for backward compatibility (if needed)
func (c *Client) SendTransactionSimple(signedTxBase64 string) (string, error) {
	result, err := c.SendTransaction(signedTxBase64)
	if err != nil {
		return "", err
	}
	return result.Signature, nil
}

// SendTransactionWithRetry sends transaction with automatic retry on blockhash expiry
func (c *Client) SendTransactionWithRetry(signedTxBase64 string, maxRetries int) (*SendTransactionResult, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := c.SendTransaction(signedTxBase64)

		if err == nil {
			return result, nil
		}

		// Check if error is BlockhashNotFound
		if strings.Contains(err.Error(), "BlockhashNotFound") ||
			strings.Contains(err.Error(), "Blockhash not found") {
			if attempt < maxRetries {
				fmt.Printf("⚠️  Blockhash expired (attempt %d/%d). Cannot retry with signed transaction.\n", attempt, maxRetries)
				return result, fmt.Errorf("blockhash expired: %w", err)
			}
		}

		// For other errors, return immediately
		return result, err
	}

	return nil, fmt.Errorf("max retries exceeded")
}
