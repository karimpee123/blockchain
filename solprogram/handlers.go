package solprogram

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

// EnvelopeTypeRequest enum
type EnvelopeTypeRequest string

const (
	EnvelopeTypeDirectFixed EnvelopeTypeRequest = "direct_fixed"
	EnvelopeTypeGroupFixed  EnvelopeTypeRequest = "group_fixed"
	EnvelopeTypeGroupRandom EnvelopeTypeRequest = "group_random"
)

// CreateEnvelopeRequest with envelope types
type CreateEnvelopeRequest struct {
	UserAddress    string              `json:"user_address"`
	EnvelopeType   EnvelopeTypeRequest `json:"envelope_type"`
	ExpiryHours    uint64              `json:"expiry_hours"`
	AllowedAddress *string             `json:"allowed_address,omitempty"` // For DirectFixed
	Amount         *uint64             `json:"amount,omitempty"`          // For DirectFixed
	TotalUsers     *uint64             `json:"total_users,omitempty"`     // For GroupFixed
	AmountPerUser  *uint64             `json:"amount_per_user,omitempty"` // For GroupFixed
	TotalAmount    *uint64             `json:"total_amount,omitempty"`    // For GroupRandom
	MaxClaimers    *uint64             `json:"max_claimers,omitempty"`    // For GroupRandom
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
	Success        bool     `json:"success"`
	Message        string   `json:"message,omitempty"`
	UnsignedTx     string   `json:"unsigned_tx,omitempty"`
	TransactionSig string   `json:"transaction_sig,omitempty"`
	EnvelopeID     uint64   `json:"envelope_id,omitempty"`
	ErrorCode      *int     `json:"error_code,omitempty"`
	ProgramLogs    []string `json:"program_logs,omitempty"`
}

// HandleCreateEnvelope handles create envelope request (with auto-init)
func (c *Client) HandleCreateEnvelope(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateEnvelopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Validate envelope type
	if req.EnvelopeType == "" {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "envelope_type is required",
		})
		return
	}

	user := solana.MustPublicKeyFromBase58(req.UserAddress)
	userStatePDA, _, _ := DeriveUserStatePDA(c.ProgramID, user)

	// Check if user_state exists
	exists, lastEnvelopeID, err := CheckUserStateExists(c.RPC, userStatePDA)
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Failed to check user state: %v", err),
		})
		return
	}

	instructions := []solana.Instruction{}

	// Add init_user_state if needed
	if !exists {
		initInstruction, err := BuildInitUserStateInstruction(c.ProgramID, user)
		if err != nil {
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: fmt.Sprintf("Failed to build init instruction: %v", err),
			})
			return
		}
		instructions = append(instructions, initInstruction)
		lastEnvelopeID = 0
	}

	// Calculate next envelope ID
	nextEnvelopeID := lastEnvelopeID + 1

	// Build create instruction based on envelope type
	var createInstruction solana.Instruction
	var totalAmount uint64

	switch req.EnvelopeType {
	case EnvelopeTypeDirectFixed:
		if req.AllowedAddress == nil || req.Amount == nil {
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "DirectFixed requires: allowed_address, amount",
			})
			return
		}
		createInstruction, err = BuildCreateDirectFixedInstruction(
			c.ProgramID,
			user,
			nextEnvelopeID,
			*req.AllowedAddress,
			*req.Amount,
			req.ExpiryHours,
		)
		totalAmount = *req.Amount

	case EnvelopeTypeGroupFixed:
		if req.TotalUsers == nil || req.AmountPerUser == nil {
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "GroupFixed requires: total_users, amount_per_user",
			})
			return
		}
		createInstruction, err = BuildCreateGroupFixedInstruction(
			c.ProgramID,
			user,
			nextEnvelopeID,
			*req.TotalUsers,
			*req.AmountPerUser,
			req.ExpiryHours,
		)
		totalAmount = *req.TotalUsers * *req.AmountPerUser

	case EnvelopeTypeGroupRandom:
		if req.TotalAmount == nil || req.MaxClaimers == nil {
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "GroupRandom requires: total_amount, max_claimers",
			})
			return
		}
		createInstruction, err = BuildCreateGroupRandomInstruction(
			c.ProgramID,
			user,
			nextEnvelopeID,
			*req.TotalAmount,
			*req.MaxClaimers,
			req.ExpiryHours,
		)
		totalAmount = *req.TotalAmount

	default:
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Invalid envelope_type: %s. Must be: direct_fixed, group_fixed, or group_random", req.EnvelopeType),
		})
		return
	}

	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Failed to build create instruction: %v", err),
		})
		return
	}

	instructions = append(instructions, createInstruction)

	// Create unsigned transaction
	unsignedTx, err := c.CreateTransactionWithInstructions(instructions, user)
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Failed to create transaction: %v", err),
		})
		return
	}

	message := fmt.Sprintf("%s envelope #%d created (%.3f SOL)",
		req.EnvelopeType, nextEnvelopeID, float64(totalAmount)/1e9)
	if !exists {
		message += " (including user init)"
	}

	json.NewEncoder(w).Encode(Response{
		Success:    true,
		Message:    message,
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
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Send transaction with detailed result
	result, err := c.SendTransaction(req.SignedTransaction)
	if err != nil {
		// Parse error to user-friendly message
		friendlyError := ParseSolanaError(err)

		response := Response{
			Success: false,
			Message: friendlyError,
		}

		// Add error code if available
		if result != nil && result.ErrorCode != nil {
			response.ErrorCode = result.ErrorCode
		}

		// Add program logs if available
		if result != nil && len(result.ProgramLogs) > 0 {
			response.ProgramLogs = result.ProgramLogs
		}

		// Special handling for BlockhashNotFound
		errStr := err.Error()
		if strings.Contains(errStr, "BlockhashNotFound") ||
			strings.Contains(errStr, "Blockhash not found") {
			response.Message = "Transaction expired. Please request a new unsigned transaction and try again."
			response.ErrorCode = nil // No custom error code for this
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success:        true,
		Message:        "Transaction sent successfully",
		TransactionSig: result.Signature,
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
