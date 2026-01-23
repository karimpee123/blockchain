package chainbnb

import "time"

// CreateTransactionResponse - Response dari create transaction
type CreateTransactionResponse struct {
	TransactionID       string `json:"transaction_id"`
	UnsignedTransaction string `json:"unsigned_transaction"`
	Nonce               uint64 `json:"nonce"`
	GasPrice            string `json:"gas_price"`
	GasLimit            uint64 `json:"gas_limit"`
}

// TransactionRequest - Request dari client untuk create transaction
type TransactionRequest struct {
	FromAddress string `json:"from_address" binding:"required"`
	ToAddress   string `json:"to_address" binding:"required"`
	Amount      string `json:"amount" binding:"required"` // in wei or BNB
}

// SignedTransactionRequest - Request signed transaction dari client
type SignedTransactionRequest struct {
	TransactionID     string `json:"transaction_id" binding:"required"`
	SignedTransaction string `json:"signed_transaction" binding:"required"` // Hex encoded signed tx
}

// TransactionResult - Response final setelah send ke blockchain
type TransactionResult struct {
	TransactionID string `json:"transaction_id"`
	TxHash        string `json:"tx_hash"`
	Success       bool   `json:"success"`
	Status        string `json:"status"` // pending, confirmed, failed
	Message       string `json:"message"`
	ExplorerURL   string `json:"explorer_url,omitempty"`
}

// TransactionStatusRequest - Request untuk cek status
type TransactionStatusRequest struct {
	TxHash string `json:"tx_hash" binding:"required"`
}

// TransactionStatusResponse - Response status transaction
type TransactionStatusResponse struct {
	TxHash        string  `json:"tx_hash"`
	Status        string  `json:"status"` // pending, confirmed, failed, not_found
	Confirmations uint64  `json:"confirmations"`
	BlockNumber   uint64  `json:"block_number"`
	BlockTime     *uint64 `json:"block_time,omitempty"`
	GasUsed       uint64  `json:"gas_used"`
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
	ID            uint       `gorm:"primaryKey" json:"id"`
	TransactionID string     `gorm:"uniqueIndex;size:64" json:"transaction_id"`
	FromAddress   string     `gorm:"index;size:42" json:"from_address"`
	ToAddress     string     `gorm:"index;size:42" json:"to_address"`
	Amount        string     `json:"amount"`
	TxHash        string     `gorm:"index;size:66" json:"tx_hash"`
	Status        string     `gorm:"index;size:20" json:"status"`
	Nonce         uint64     `json:"nonce"`
	GasUsed       uint64     `json:"gas_used"`
	GasPrice      string     `json:"gas_price"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
}

func (TransactionHistory) TableName() string {
	return "bnb_transaction_histories"
}
