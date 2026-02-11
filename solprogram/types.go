package solprogram

import (
	"time"

	"github.com/gagliardetto/solana-go"
)

// EnvelopeType - Tipe envelope yang tersedia
type EnvelopeType uint8

const (
	EnvelopeTypeDirectFixed EnvelopeType = 0
	EnvelopeTypeGroupFixed  EnvelopeType = 1
	EnvelopeTypeGroupRandom EnvelopeType = 2
)

// TokenType - Tipe token yang didukung
type TokenType string

const (
	TokenTypeUSDC TokenType = "USDC"
	TokenTypeSOL  TokenType = "SOL"
)

// DirectFixedEnvelope - Envelope untuk 1 user spesifik
type DirectFixedEnvelope struct {
	AllowedAddress solana.PublicKey
}

// EnvelopeTypeData - Data envelope type dengan allowed address untuk DirectFixed
type EnvelopeTypeData struct {
	Type           EnvelopeType
	AllowedAddress *solana.PublicKey // Only for DirectFixed
}

// UserState - State untuk tracking envelope IDs per user
type UserState struct {
	Owner          solana.PublicKey
	LastEnvelopeID uint64
}

// EnvelopeAccount - Main envelope account structure
type EnvelopeAccount struct {
	Owner           solana.PublicKey
	EnvelopeID      uint64
	EnvelopeType    EnvelopeTypeData
	TotalAmount     uint64
	TotalUsers      uint64
	WithdrawnAmount uint64
	ClaimedCount    uint64
	Expiry          int64
	IsCancelled     bool
}

// ClaimRecord - Record untuk track siapa sudah claim
type ClaimRecord struct {
	Claimer    solana.PublicKey
	EnvelopeID uint64
	Amount     uint64
	ClaimedAt  int64
}

// CreateEnvelopeParams - Parameters untuk create envelope
type CreateEnvelopeParams struct {
	EnvelopeType   EnvelopeTypeData
	TotalAmount    uint64
	TotalUsers     uint64
	ExpirySeconds  uint64
	AllowedAddress *solana.PublicKey // Optional: hanya untuk DirectFixed
}

// CreateEnvelopeResponse - Response setelah create envelope
type CreateEnvelopeResponse struct {
	EnvelopeID          uint64           `json:"envelope_id"`
	EnvelopePDA         solana.PublicKey `json:"envelope_pda"`
	VaultPDA            solana.PublicKey `json:"vault_pda"`
	Signature           string           `json:"signature"`
	UnsignedTransaction string           `json:"unsigned_transaction,omitempty"`
	Message             string           `json:"message"`
}

// ClaimEnvelopeParams - Parameters untuk claim envelope
type ClaimEnvelopeParams struct {
	EnvelopeID          uint64
	Owner               solana.PublicKey
	Claimer             solana.PublicKey
	ClaimerTokenAccount solana.PublicKey
}

// ClaimEnvelopeResponse - Response setelah claim
type ClaimEnvelopeResponse struct {
	EnvelopeID          uint64 `json:"envelope_id"`
	ClaimedAmount       uint64 `json:"claimed_amount"`
	Signature           string `json:"signature"`
	UnsignedTransaction string `json:"unsigned_transaction,omitempty"`
	Message             string `json:"message"`
}

// RefundParams - Parameters untuk refund
type RefundParams struct {
	EnvelopeID        uint64
	Owner             solana.PublicKey
	OwnerTokenAccount solana.PublicKey
}

// RefundResponse - Response setelah refund
type RefundResponse struct {
	EnvelopeID          uint64 `json:"envelope_id"`
	RefundedAmount      uint64 `json:"refunded_amount"`
	Signature           string `json:"signature"`
	UnsignedTransaction string `json:"unsigned_transaction,omitempty"`
	Message             string `json:"message"`
}

// EnvelopeInfo - Info lengkap tentang envelope
type EnvelopeInfo struct {
	Owner           solana.PublicKey `json:"owner"`
	EnvelopeID      uint64           `json:"envelope_id"`
	EnvelopeType    string           `json:"envelope_type"`
	AllowedAddress  *string          `json:"allowed_address,omitempty"`
	TotalAmount     uint64           `json:"total_amount"`
	TotalUsers      uint64           `json:"total_users"`
	WithdrawnAmount uint64           `json:"withdrawn_amount"`
	ClaimedCount    uint64           `json:"claimed_count"`
	RemainingAmount uint64           `json:"remaining_amount"`
	IsCancelled     bool             `json:"is_cancelled"`
	ExpiryTime      time.Time        `json:"expiry_time"`
	IsExpired       bool             `json:"is_expired"`
}

// TransactionStatus - Status transaksi
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusConfirmed TransactionStatus = "confirmed"
	StatusFinalized TransactionStatus = "finalized"
	StatusFailed    TransactionStatus = "failed"
)

// TransactionResult - Hasil transaksi
type TransactionResult struct {
	Signature   string            `json:"signature"`
	Status      TransactionStatus `json:"status"`
	Error       *string           `json:"error,omitempty"`
	ExplorerURL string            `json:"explorer_url"`
}
