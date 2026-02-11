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
	RequestTypeDirectFixed EnvelopeTypeRequest = "direct_fixed"
	RequestTypeGroupFixed  EnvelopeTypeRequest = "group_fixed"
	RequestTypeGroupRandom EnvelopeTypeRequest = "group_random"
)

// CreateEnvelopeRequest with envelope types
type CreateEnvelopeRequest struct {
	UserAddress    string              `json:"user_address"`
	EnvelopeType   EnvelopeTypeRequest `json:"envelope_type"`
	TotalAmount    uint64              `json:"total_amount"`
	TotalUsers     uint64              `json:"total_users"`
	ExpiryHours    uint64              `json:"expiry_hours"`
	AllowedAddress *string             `json:"allowed_address,omitempty"`
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

	// Validate required fields
	if req.EnvelopeType == "" {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "envelope_type is required",
		})
		return
	}

	if req.TotalAmount == 0 {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "total_amount must be greater than 0",
		})
		return
	}

	if req.TotalUsers == 0 {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "total_users must be greater than 0",
		})
		return
	}

	// Validate DirectFixed
	if req.EnvelopeType == RequestTypeDirectFixed {
		if req.AllowedAddress == nil || *req.AllowedAddress == "" {
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "DirectFixed requires allowed_address",
			})
			return
		}
		if req.TotalUsers != 1 {
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "DirectFixed must have total_users = 1",
			})
			return
		}
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

	// Build create instruction (UNIFIED)
	var createInstruction solana.Instruction

	switch req.EnvelopeType {
	case RequestTypeDirectFixed:
		createInstruction, err = BuildCreateEnvelopeInstruction(
			c.ProgramID,
			user,
			nextEnvelopeID,
			RequestTypeDirectFixed,
			req.TotalAmount,
			req.TotalUsers,
			req.ExpiryHours,
			req.AllowedAddress, // Only for DirectFixed
		)

	case RequestTypeGroupFixed:
		createInstruction, err = BuildCreateEnvelopeInstruction(
			c.ProgramID,
			user,
			nextEnvelopeID,
			RequestTypeGroupFixed,
			req.TotalAmount,
			req.TotalUsers,
			req.ExpiryHours,
			nil, // No allowed_address
		)

	case RequestTypeGroupRandom:
		createInstruction, err = BuildCreateEnvelopeInstruction(
			c.ProgramID,
			user,
			nextEnvelopeID,
			RequestTypeGroupRandom,
			req.TotalAmount,
			req.TotalUsers,
			req.ExpiryHours,
			nil, // No allowed_address
		)

	default:
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: fmt.Sprintf("Invalid envelope_type: %s", req.EnvelopeType),
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

	message := fmt.Sprintf("%s envelope #%d created (%.9f SOL, %d users)",
		req.EnvelopeType, nextEnvelopeID, float64(req.TotalAmount)/1e9, req.TotalUsers)
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
