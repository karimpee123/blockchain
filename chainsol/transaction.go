package chainsol

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
)

// CreateTransaction - Step 1: Backend create unsigned transaction
func (p *SolChain) CreateTransaction(req TransactionRequest) (*CreateTransactionResponse, error) {
	// Validate addresses
	accountFrom, err := solana.PublicKeyFromBase58(req.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}
	accountTo, err := solana.PublicKeyFromBase58(req.ToAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}
	// Get recent block hash
	ctx := context.Background()
	recent, err := p.http.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}
	// Create transfer instruction
	instruction := system.NewTransferInstruction(
		req.Amount,
		accountFrom,
		accountTo,
	).Build()

	// Build transaction WITHOUT signatures
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(accountFrom),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	// Serialize the FULL transaction (with empty signatures)
	// This is important - we serialize the whole tx structure
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}
	transactionID := fmt.Sprintf("txn_%d", time.Now().UnixNano())

	response := &CreateTransactionResponse{
		TransactionID:       transactionID,
		UnsignedTransaction: base64.StdEncoding.EncodeToString(txBytes),
		RecentBlockhash:     recent.Value.Blockhash.String(),
	}
	return response, nil
}

// SendSignedTransaction - Step 3: Backend send signed transaction ke blockchain
func (p *SolChain) SendSignedTransaction(req SignedTransactionRequest) (*TransactionResult, error) {
	// Decode signed transaction
	txBytes, err := base64.StdEncoding.DecodeString(req.SignedTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signed transaction: %w", err)
	}
	// Unmarshal transaction using decoder
	decoder := bin.NewBinDecoder(txBytes)
	var tx solana.Transaction
	if err := tx.UnmarshalWithDecoder(decoder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	// Validate transaction has signature
	if len(tx.Signatures) == 0 {
		return nil, fmt.Errorf("transaction is not signed")
	}
	// Send transaction to Solana via Alchemy
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sig, err := confirm.SendAndConfirmTransaction(
		ctx,
		p.http,
		p.ws,
		&tx,
	)
	result := &TransactionResult{
		TransactionID: req.TransactionID,
		Success:       err == nil,
	}
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("Failed to send transaction: %v", err)
		return result, err
	}
	result.Signature = sig.String()
	result.Status = "pending"
	result.Message = "Transaction sent successfully"
	result.ExplorerURL = p.GetExplorerURL(sig.String())
	return result, nil
}

// GetTransactionStatus - Check transaction status
func (p *SolChain) GetTransactionStatus(signature string) (*TransactionStatusResponse, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get transaction details
	result, err := p.http.GetTransaction(
		ctx,
		sig,
		&rpc.GetTransactionOpts{
			Encoding:   solana.EncodingBase64,
			Commitment: rpc.CommitmentConfirmed,
		},
	)
	response := &TransactionStatusResponse{
		Signature:   signature,
		ExplorerURL: p.GetExplorerURL(signature),
	}
	if err != nil {
		response.Status = "not_found"
		return response, nil
	}

	// Parse result
	if result != nil {
		if result.Meta != nil {
			if result.Meta.Err != nil {
				errMsg := fmt.Sprintf("%v", result.Meta.Err)
				response.Status = "failed"
				response.Error = &errMsg
			} else {
				response.Status = "confirmed"
			}
			response.Fee = result.Meta.Fee
		}
		response.Slot = result.Slot
		// Convert UnixTimeSeconds to int64
		if result.BlockTime != nil {
			blockTime := int64(*result.BlockTime)
			response.BlockTime = &blockTime
		}
		// Get confirmations
		currentSlot, err := p.http.GetSlot(ctx, rpc.CommitmentFinalized)
		if err == nil {
			response.Confirmations = currentSlot - result.Slot
		}
	}
	return response, nil
}

// GetTransactionHistory - Get transaction history from database
func (p *SolChain) GetTransactionHistory(address string, limit int) ([]TransactionHistory, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	var histories []TransactionHistory
	err := p.db.Where("from_address = ? OR to_address = ?", address, address).
		Order("created_at DESC").
		Limit(limit).
		Find(&histories).Error

	return histories, err
}
