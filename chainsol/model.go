package src

import "time"

// CreateTransactionResponse - Response dari create transaction
type CreateTransactionResponse struct {
	TransactionID       string `json:"transaction_id"`
	UnsignedTransaction string `json:"unsigned_transaction"`
	RecentBlockhash     string `json:"recent_blockhash"`
}

// TransactionRequest - Request dari client untuk create transaction
type TransactionRequest struct {
	FromAddress string `json:"from_address" binding:"required" validate:"required"`
	ToAddress   string `json:"to_address" binding:"required" validate:"required"`
	Amount      uint64 `json:"amount" binding:"required" validate:"required,gt=0"`
}

// UnsignedTransactionResponse - Response unsigned transaction ke client
type UnsignedTransactionResponse struct {
	TransactionID   string `json:"transaction_id"`   // Unique ID untuk tracking
	Transaction     string `json:"transaction"`      // Base64 encoded unsigned tx
	RecentBlockhash string `json:"recent_blockhash"` // Blockhash yang digunakan
	FromAddress     string `json:"from_address"`
	ToAddress       string `json:"to_address"`
	Amount          uint64 `json:"amount"`
	ExpiresAt       int64  `json:"expires_at"` // Timestamp expiry (blockhash valid ~60s)
	Message         string `json:"message"`
}

// SignedTransactionRequest - Request signed transaction dari client
type SignedTransactionRequest struct {
	TransactionID     string `json:"transaction_id" binding:"required"`
	SignedTransaction string `json:"signed_transaction" binding:"required"` // Base64 encoded signed tx
}

// TransactionResult - Response final setelah send ke blockchain
type TransactionResult struct {
	TransactionID string `json:"transaction_id"`
	Signature     string `json:"signature"`
	Success       bool   `json:"success"`
	Status        string `json:"status"` // pending, confirmed, failed
	Message       string `json:"message"`
	ExplorerURL   string `json:"explorer_url,omitempty"`
}

// TransactionStatusRequest - Request untuk cek status
type TransactionStatusRequest struct {
	Signature string `json:"signature" binding:"required"`
}

// TransactionStatusResponse - Response status transaction
type TransactionStatusResponse struct {
	Signature     string  `json:"signature"`
	Status        string  `json:"status"` // confirmed, finalized, failed, not_found
	Confirmations uint64  `json:"confirmations"`
	Slot          uint64  `json:"slot"`
	BlockTime     *int64  `json:"block_time,omitempty"`
	Fee           uint64  `json:"fee"`
	Error         *string `json:"error,omitempty"`
	ExplorerURL   string  `json:"explorer_url"`
}

// ErrorResponse - Standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// TransactionHistory - Model untuk database (optional)
type TransactionHistory struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TransactionID   string     `gorm:"uniqueIndex;size:64" json:"transaction_id"`
	FromAddress     string     `gorm:"index;size:44" json:"from_address"`
	ToAddress       string     `gorm:"index;size:44" json:"to_address"`
	Amount          uint64     `json:"amount"`
	Signature       string     `gorm:"index;size:88" json:"signature"`
	Status          string     `gorm:"index;size:20" json:"status"`
	RecentBlockhash string     `gorm:"size:44" json:"recent_blockhash"`
	Fee             uint64     `json:"fee"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
}

func (TransactionHistory) TableName() string {
	return "transaction_histories"
}
