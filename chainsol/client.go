package chainsol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

// HandleSignTransaction - Function for CLIENT SIDE
// Private key will NEVER SEND to backend side
// Reference/example and TESTING PURPOSE ONLY
func (p *SolChain) HandleSignTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UnsignedTransaction string `json:"unsigned_transaction"`
		PrivateKey          string `json:"private_key"` // BASE58 encoded private key
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Decode transaction
	txBytes, err := base64.StdEncoding.DecodeString(req.UnsignedTransaction)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to decode transaction: %v", err), http.StatusBadRequest)
		return
	}
	// Parse private key - WARNING: INSECURE!
	privateKey, err := solana.PrivateKeyFromBase58(req.PrivateKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid private key: %v", err), http.StatusBadRequest)
		return
	}
	// Unmarshal transaction using decoder
	decoder := bin.NewBinDecoder(txBytes)
	var tx solana.Transaction
	if err := tx.UnmarshalWithDecoder(decoder); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal transaction: %v", err), http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("failed to sign transaction: %v", err), http.StatusInternalServerError)
		return
	}
	// Serialize signed transaction
	signedTxBytes, err := tx.MarshalBinary()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to serialize: %v", err), http.StatusInternalServerError)
		return
	}
	response := map[string]string{
		"signed_transaction": base64.StdEncoding.EncodeToString(signedTxBytes),
		"warning":            "⚠️ TESTING ONLY - Never send private keys in production!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
