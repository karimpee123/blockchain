package solprogram

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"net/http"

	"github.com/gagliardetto/solana-go"
)

// Request types
type CreateEnvelopeRequest struct {
	UserAddress string `json:"user_address"`
	Amount      uint64 `json:"amount"` // in lamports
	ExpiryHours uint64 `json:"expiry_hours"`
}

type ClaimEnvelopeRequest struct {
	OwnerAddress   string `json:"owner_address"`
	ClaimerAddress string `json:"claimer_address"`
	EnvelopeID     uint64 `json:"envelope_id"`
}

type RefundEnvelopeRequest struct {
	OwnerAddress string `json:"owner_address"`
	EnvelopeID   uint64 `json:"envelope_id"`
}

type SendTransactionRequest struct {
	SignedTransaction string `json:"signed_transaction"`
}

// Response type
type Response struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	UnsignedTx     string `json:"unsigned_tx,omitempty"`
	TransactionSig string `json:"transaction_sig,omitempty"`
	EnvelopeID     uint64 `json:"envelope_id,omitempty"`
}

// HandleCreateEnvelope handles create envelope request (with auto-init)
func (c *Client) HandleCreateEnvelope(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateEnvelopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	user := solana.MustPublicKeyFromBase58(req.UserAddress)
	userStatePDA, _, _ := DeriveUserStatePDA(c.ProgramID, user)

	// Check if user_state exists
	exists, lastEnvelopeID, err := CheckUserStateExists(c.RPC, userStatePDA)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	instructions := []solana.Instruction{}

	// Add init_user_state instruction if needed
	if !exists {
		initInstruction, err := BuildInitUserStateInstruction(c.ProgramID, user)
		if err != nil {
			json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
			return
		}
		instructions = append(instructions, initInstruction)
		lastEnvelopeID = 0 // Start from 0 for new user
	}

	// Calculate next envelope ID
	nextEnvelopeID := lastEnvelopeID + 1

	// Add create envelope instruction
	createInstruction, err := BuildCreateInstruction(
		c.ProgramID,
		user,
		nextEnvelopeID,
		req.Amount,
		req.ExpiryHours,
	)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}
	instructions = append(instructions, createInstruction)

	// Create transaction with all instructions
	unsignedTx, err := c.CreateTransactionWithInstructions(instructions, user)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	message := fmt.Sprintf("Envelope #%d transaction created", nextEnvelopeID)
	if !exists {
		message += " (including user init)"
	}

	json.NewEncoder(w).Encode(Response{
		Success:    true,
		Message:    message + ". Sign on client side.",
		UnsignedTx: unsignedTx,
		EnvelopeID: nextEnvelopeID,
	})
}

// HandleClaimEnvelope handles claim envelope request
func (c *Client) HandleClaimEnvelope(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ClaimEnvelopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	owner := solana.MustPublicKeyFromBase58(req.OwnerAddress)
	claimer := solana.MustPublicKeyFromBase58(req.ClaimerAddress)

	instruction, err := BuildClaimInstruction(c.ProgramID, owner, claimer, req.EnvelopeID)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	unsignedTx, err := c.CreateTransaction(instruction, claimer)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success:    true,
		Message:    fmt.Sprintf("Claim envelope #%d transaction created. Sign on client side.", req.EnvelopeID),
		UnsignedTx: unsignedTx,
	})
}

// HandleRefundEnvelope handles refund envelope request
func (c *Client) HandleRefundEnvelope(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req RefundEnvelopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	owner := solana.MustPublicKeyFromBase58(req.OwnerAddress)

	instruction, err := BuildRefundInstruction(c.ProgramID, owner, req.EnvelopeID)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	unsignedTx, err := c.CreateTransaction(instruction, owner)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success:    true,
		Message:    fmt.Sprintf("Refund envelope #%d transaction created. Sign on client side.", req.EnvelopeID),
		UnsignedTx: unsignedTx,
	})
}

// HandleSendTransaction handles signed transaction submission
func (c *Client) HandleSendTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SendTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	sig, err := c.SendTransaction(req.SignedTransaction)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success:        true,
		Message:        "Transaction sent successfully",
		TransactionSig: sig,
	})
}

// ------------------------------ CLIENT SIDE ------------------------------ //

type SignTransactionRequest struct {
	UnsignedTransaction string `json:"unsigned_transaction"`
	PrivateKey          string `json:"private_key"` // Base58 encoded
}

type SignTransactionResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	SignedTransaction string `json:"signed_transaction"`
}

// HandleSignTransaction signs transaction on backend (⚠️ TESTING ONLY!)
func (c *Client) HandleSignTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SignTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(SignTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// ⚠️ WARNING: Never do this in production!
	// Parse private key
	privateKey, err := solana.PrivateKeyFromBase58(req.PrivateKey)
	if err != nil {
		json.NewEncoder(w).Encode(SignTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid private key: %v", err),
		})
		return
	}

	// Decode unsigned transaction
	txBytes, err := base64.StdEncoding.DecodeString(req.UnsignedTransaction)
	if err != nil {
		json.NewEncoder(w).Encode(SignTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to decode transaction: %v", err),
		})
		return
	}

	// Parse transaction
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		json.NewEncoder(w).Encode(SignTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse transaction: %v", err),
		})
		return
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if privateKey.PublicKey().Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		json.NewEncoder(w).Encode(SignTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to sign transaction: %v", err),
		})
		return
	}

	// Serialize signed transaction
	signedTxBytes, err := tx.MarshalBinary()
	if err != nil {
		json.NewEncoder(w).Encode(SignTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to serialize signed transaction: %v", err),
		})
		return
	}

	signedTxBase64 := base64.StdEncoding.EncodeToString(signedTxBytes)

	json.NewEncoder(w).Encode(SignTransactionResponse{
		Success:           true,
		Message:           "Transaction signed successfully",
		SignedTransaction: signedTxBase64,
	})
}
