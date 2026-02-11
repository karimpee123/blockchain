package solprogram

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// InitUserState - Initialize user state (first time only)
func (c *USDCEnvelopeClient) InitUserState(ctx context.Context, userPrivateKey solana.PrivateKey) (*TransactionResult, error) {
	user := userPrivateKey.PublicKey()

	// Check if already initialized
	_, err := c.GetUserState(ctx, user)
	if err == nil {
		return nil, fmt.Errorf("user state already initialized")
	}

	// Build instruction
	instruction, err := c.BuildInitUserStateInstruction(user)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get latest blockhash
	latestBlockhash, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		latestBlockhash.Value.Blockhash,
		solana.TransactionPayer(user),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if userPrivateKey.PublicKey().Equals(key) {
			return &userPrivateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return &TransactionResult{
		Signature:   sig.String(),
		Status:      StatusPending,
		ExplorerURL: c.getExplorerURL(sig.String()),
	}, nil
}

// WaitForConfirmation - Wait for transaction confirmation with timeout
func (c *USDCEnvelopeClient) WaitForConfirmation(ctx context.Context, signature string, timeoutSeconds int) error {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	maxRetries := timeoutSeconds / 2 // Poll every 2 seconds
	for i := 0; i < maxRetries; i++ {
		status, err := c.rpcClient.GetSignatureStatuses(ctx, true, sig)
		if err == nil && status != nil && len(status.Value) > 0 && status.Value[0] != nil {
			txStatus := status.Value[0]

			// Check if confirmed or finalized
			if txStatus.ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
				txStatus.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				if txStatus.Err != nil {
					return fmt.Errorf("transaction failed: %v", txStatus.Err)
				}
				return nil // Success!
			}

			// Check if failed
			if txStatus.Err != nil {
				return fmt.Errorf("transaction failed: %v", txStatus.Err)
			}
		}

		// Wait 2 seconds before retry
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for confirmation after %d seconds", timeoutSeconds)
}

// CreateEnvelope - Create new envelope
func (c *USDCEnvelopeClient) CreateEnvelope(
	ctx context.Context,
	userPrivateKey solana.PrivateKey,
	userTokenAccount solana.PublicKey,
	params CreateEnvelopeParams,
) (*CreateEnvelopeResponse, error) {
	user := userPrivateKey.PublicKey()

	// Get user state to get next envelope ID
	userState, err := c.GetUserState(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("user state not initialized: %w", err)
	}

	nextEnvelopeID := userState.LastEnvelopeID + 1

	// Build instruction
	instruction, err := c.BuildCreateEnvelopeInstruction(user, userTokenAccount, params, nextEnvelopeID)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get latest blockhash
	latestBlockhash, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		latestBlockhash.Value.Blockhash,
		solana.TransactionPayer(user),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if userPrivateKey.PublicKey().Equals(key) {
			return &userPrivateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Derive PDAs for response
	envelopePDA, _, _ := c.DeriveEnvelopePDA(user, nextEnvelopeID)
	vaultPDA, _, _ := c.DeriveEnvelopeVaultPDA(user, nextEnvelopeID)

	return &CreateEnvelopeResponse{
		EnvelopeID:  nextEnvelopeID,
		EnvelopePDA: envelopePDA,
		VaultPDA:    vaultPDA,
		Signature:   sig.String(),
		Message:     "Envelope created successfully",
	}, nil
}

// CreateUnsignedEnvelope - Create unsigned transaction for client-side signing
func (c *USDCEnvelopeClient) CreateUnsignedEnvelope(
	ctx context.Context,
	user solana.PublicKey,
	userTokenAccount solana.PublicKey,
	params CreateEnvelopeParams,
) (*CreateEnvelopeResponse, error) {
	// Get user state to get next envelope ID
	userState, err := c.GetUserState(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("user state not initialized: %w", err)
	}

	nextEnvelopeID := userState.LastEnvelopeID + 1

	// Build instruction
	instruction, err := c.BuildCreateEnvelopeInstruction(user, userTokenAccount, params, nextEnvelopeID)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get latest blockhash
	latestBlockhash, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		latestBlockhash.Value.Blockhash,
		solana.TransactionPayer(user),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Serialize unsigned transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Derive PDAs for response
	envelopePDA, _, _ := c.DeriveEnvelopePDA(user, nextEnvelopeID)
	vaultPDA, _, _ := c.DeriveEnvelopeVaultPDA(user, nextEnvelopeID)

	return &CreateEnvelopeResponse{
		EnvelopeID:          nextEnvelopeID,
		EnvelopePDA:         envelopePDA,
		VaultPDA:            vaultPDA,
		UnsignedTransaction: base64.StdEncoding.EncodeToString(txBytes),
		Message:             "Unsigned transaction created - sign on client side",
	}, nil
}

// ClaimEnvelope - Claim from envelope
func (c *USDCEnvelopeClient) ClaimEnvelope(
	ctx context.Context,
	claimerPrivateKey solana.PrivateKey,
	params ClaimEnvelopeParams,
) (*ClaimEnvelopeResponse, error) {
	claimer := claimerPrivateKey.PublicKey()

	// Set claimer in params if not already set
	if params.Claimer.IsZero() {
		params.Claimer = claimer
	}

	// Build instruction
	instruction, err := c.BuildClaimInstruction(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get latest blockhash
	latestBlockhash, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		latestBlockhash.Value.Blockhash,
		solana.TransactionPayer(claimer),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if claimerPrivateKey.PublicKey().Equals(key) {
			return &claimerPrivateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return &ClaimEnvelopeResponse{
		EnvelopeID: params.EnvelopeID,
		Signature:  sig.String(),
		Message:    "Claim successful",
	}, nil
}

// RefundEnvelope - Refund unclaimed USDC after expiry
func (c *USDCEnvelopeClient) RefundEnvelope(
	ctx context.Context,
	ownerPrivateKey solana.PrivateKey,
	ownerTokenAccount solana.PublicKey,
	envelopeID uint64,
) (*RefundResponse, error) {
	owner := ownerPrivateKey.PublicKey()

	params := RefundParams{
		EnvelopeID:        envelopeID,
		Owner:             owner,
		OwnerTokenAccount: ownerTokenAccount,
	}

	// Build instruction
	instruction, err := c.BuildRefundInstruction(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get latest blockhash
	latestBlockhash, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		latestBlockhash.Value.Blockhash,
		solana.TransactionPayer(owner),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if ownerPrivateKey.PublicKey().Equals(key) {
			return &ownerPrivateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return &RefundResponse{
		EnvelopeID: envelopeID,
		Signature:  sig.String(),
		Message:    "Refund successful",
	}, nil
}

// SendSignedTransaction - Send signed transaction from client
func (c *USDCEnvelopeClient) SendSignedTransaction(ctx context.Context, signedTxBase64 string) (string, error) {
	// Decode transaction
	txBytes, err := base64.StdEncoding.DecodeString(signedTxBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode transaction: %w", err)
	}

	// Parse transaction
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig.String(), nil
}
