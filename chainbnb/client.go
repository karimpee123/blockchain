package chainbnb

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// HandleSignTransaction - Function for CLIENT SIDE
// Private key will NEVER SEND to backend side
// Reference/example and TESTING PURPOSE ONLY
func (b *BNBChain) HandleSignTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UnsignedTransaction string `json:"unsigned_transaction"`
		PrivateKey          string `json:"private_key"` // Hex encoded private key (without 0x)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Decode transaction
	txBytes, err := hex.DecodeString(req.UnsignedTransaction)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to decode transaction: %v", err), http.StatusBadRequest)
		return
	}

	// Parse transaction
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal transaction: %v", err), http.StatusBadRequest)
		return
	}

	// Parse private key - WARNING: INSECURE!
	privateKey, err := crypto.HexToECDSA(req.PrivateKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid private key: %v", err), http.StatusBadRequest)
		return
	}

	// Sign transaction
	signer := types.NewEIP155Signer(big.NewInt(b.chainID))
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Serialize signed transaction
	signedTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to serialize: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"signed_transaction": hex.EncodeToString(signedTxBytes),
		"tx_hash":            signedTx.Hash().Hex(),
		"warning":            "⚠️ TESTING ONLY - Never send private keys in production!",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPublicKeyFromPrivateKey - Helper untuk mendapatkan address dari private key
func GetPublicKeyFromPrivateKey(privateKeyHex string) (string, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address.Hex(), nil
}
