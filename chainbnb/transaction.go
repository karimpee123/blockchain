package chainbnb

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CreateTransaction - Step 1: Backend create unsigned transaction
func (b *BNBChain) CreateTransaction(req TransactionRequest) (*CreateTransactionResponse, error) {
	// Validate addresses
	if !common.IsHexAddress(req.FromAddress) {
		return nil, fmt.Errorf("invalid from address")
	}
	if !common.IsHexAddress(req.ToAddress) {
		return nil, fmt.Errorf("invalid to address")
	}

	fromAddress := common.HexToAddress(req.FromAddress)
	toAddress := common.HexToAddress(req.ToAddress)

	// Parse amount
	amount := new(big.Int)
	amount, ok := amount.SetString(req.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}

	ctx := context.Background()

	// Get nonce
	nonce, err := b.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := b.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Gas limit for simple transfer
	gasLimit := uint64(21000)

	// Create unsigned transaction
	tx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, nil)

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	transactionID := fmt.Sprintf("bnb_txn_%d", time.Now().UnixNano())

	response := &CreateTransactionResponse{
		TransactionID:       transactionID,
		UnsignedTransaction: hex.EncodeToString(txBytes),
		Nonce:               nonce,
		GasPrice:            gasPrice.String(),
		GasLimit:            gasLimit,
	}

	return response, nil
}

// SendSignedTransaction - Step 3: Backend send signed transaction ke blockchain
func (b *BNBChain) SendSignedTransaction(req SignedTransactionRequest) (*TransactionResult, error) {
	// Decode signed transaction
	txBytes, err := hex.DecodeString(req.SignedTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signed transaction: %w", err)
	}

	// Unmarshal transaction
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	// Send transaction
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = b.client.SendTransaction(ctx, tx)

	result := &TransactionResult{
		TransactionID: req.TransactionID,
		Success:       err == nil,
	}

	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("Failed to send transaction: %v", err)
		return result, err
	}

	result.TxHash = tx.Hash().Hex()
	result.Status = "pending"
	result.Message = "Transaction sent successfully"
	result.ExplorerURL = b.GetExplorerURL(tx.Hash().Hex())

	return result, nil
}

// GetTransactionStatus - Check transaction status
func (b *BNBChain) GetTransactionStatus(txHash string) (*TransactionStatusResponse, error) {
	hash := common.HexToHash(txHash)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response := &TransactionStatusResponse{
		TxHash:      txHash,
		ExplorerURL: b.GetExplorerURL(txHash),
	}

	// Get transaction receipt
	receipt, err := b.client.TransactionReceipt(ctx, hash)
	if err != nil {
		response.Status = "not_found"
		return response, nil
	}

	// Check status
	if receipt.Status == types.ReceiptStatusSuccessful {
		response.Status = "confirmed"
	} else {
		response.Status = "failed"
		errMsg := "transaction reverted"
		response.Error = &errMsg
	}

	response.BlockNumber = receipt.BlockNumber.Uint64()
	response.GasUsed = receipt.GasUsed

	// Get block for timestamp
	block, err := b.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err == nil {
		blockTime := block.Time()
		response.BlockTime = &blockTime
	}

	// Get current block for confirmations
	currentBlock, err := b.client.BlockNumber(ctx)
	if err == nil {
		response.Confirmations = currentBlock - receipt.BlockNumber.Uint64()
	}

	return response, nil
}

// GetTransactionHistory - Get transaction history (requires database)
func (b *BNBChain) GetTransactionHistory(address string, limit int) ([]TransactionHistory, error) {
	// This would require database implementation
	return nil, fmt.Errorf("database not configured")
}
