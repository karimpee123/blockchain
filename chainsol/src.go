package src

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandleCreateTransaction - POST /api/v1/transaction/create
func (p *SolChain) HandleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.FromAddress == "" || req.ToAddress == "" || req.Amount == 0 {
		respondError(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	response, err := p.CreateTransaction(req)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, response, http.StatusOK)
}

// HandleSendTransaction - POST /api/v1/transaction/send
func (p *SolChain) HandleSendTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req SignedTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.SignedTransaction == "" || req.TransactionID == "" {
		respondError(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	result, err := p.SendSignedTransaction(req)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, result, http.StatusOK)
}

// HandleGetTransactionStatus - GET /api/v1/transaction/status?signature=xxx
func (p *SolChain) HandleGetTransactionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	signature := r.URL.Query().Get("signature")
	if signature == "" {
		respondError(w, "signature parameter required", http.StatusBadRequest)
		return
	}
	result, err := p.GetTransactionStatus(signature)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, result, http.StatusOK)
}

// HandleGetTransactionHistory - GET /api/v1/transaction/history?address=xxx&limit=10
func (p *SolChain) HandleGetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	address := r.URL.Query().Get("address")
	if address == "" {
		respondError(w, "address parameter required", http.StatusBadRequest)
		return
	}
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit > 100 {
		limit = 100
	}
	histories, err := p.GetTransactionHistory(address, limit)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, histories, http.StatusOK)
}

// Helper functions
func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, status int) {
	respondJSON(w, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Code:    status,
	}, status)
}
