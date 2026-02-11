package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

type TransactionLog struct {
	BlockTime int64  `json:"blockTime"`
	Slot      uint64 `json:"slot"`
	Meta      struct {
		Fee          uint64   `json:"fee"`
		LogMessages  []string `json:"logMessages"`
		PostBalances []uint64 `json:"postBalances"`
		PreBalances  []uint64 `json:"preBalances"`
	} `json:"meta"`
}

func GetSignature(signature string) {
	resp, err := GetTransaction(signature)
	if err != nil {
		panic(err)
	}
	if resp == nil {
		fmt.Println("tx is nil")
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	var txLog TransactionLog
	err = json.Unmarshal(respBytes, &txLog)
	if err != nil {
		log.Fatal(err)
	}
	if len(txLog.Meta.PostBalances) == 0 || len(txLog.Meta.PreBalances) == 0 {
		log.Fatal("PostBalances atau PreBalances kosong")
	}
	diff := txLog.Meta.PostBalances[0] - txLog.Meta.PreBalances[0]
	amountSol := convertFromLampToSol(diff)

	fmt.Println("BlockTime:", txLog.BlockTime)
	fmt.Println("Slot:", txLog.Slot)
	fmt.Println("Fee:", txLog.Meta.Fee)
	fmt.Println("PostBalances:", txLog.Meta.PostBalances[0])
	fmt.Println("PreBalances:", txLog.Meta.PreBalances[0])
	fmt.Println("PostBalances - PreBalances:", amountSol)

	logs := txLog.Meta.LogMessages
	action, payout := extractPayoutFromLogs(logs)
	payoutSol := convertFromLampToSol(payout)
	fmt.Printf("%s Amount: %.2f SOL\n", action, payoutSol)
}

func GetTransaction(signature string) (interface{}, error) {
	url := "https://api.devnet.solana.com"

	reqBody := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getTransaction",
		Params: []interface{}{
			signature,
			map[string]interface{}{
				"encoding":                       "json",
				"maxSupportedTransactionVersion": 0,
			},
		},
	}

	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %v", rpcResp.Error)
	}

	return rpcResp.Result, nil
}

func extractPayoutFromLogs(logs []string) (string, uint64) {
	for _, log := range logs {
		if strings.Contains(log, "Claim success:") && strings.Contains(log, "Amount=") {
			parts := strings.Split(log, "Amount=")
			if len(parts) >= 2 {
				amountPart := parts[1]
				amountStr := strings.TrimSpace(strings.Split(amountPart, ",")[0])
				if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
					return "Claim", amount
				}
			}
		}

		if strings.Contains(log, "Refund success:") && strings.Contains(log, "Amount=") {
			parts := strings.Split(log, "Amount=")
			if len(parts) >= 2 {
				amountPart := parts[1]
				amountStr := strings.TrimSpace(strings.Split(amountPart, ",")[0])
				if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
					return "Refund", amount
				}
			}
		}

		if strings.Contains(log, "Claim amount:") {
			parts := strings.Split(log, "Claim amount:")
			if len(parts) >= 2 {
				amountStr := strings.TrimSpace(parts[1])
				if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
					return "Claim", amount
				}
			}
		}
	}
	return "none", 0
}

func convertFromLampToSol(amount uint64) float64 {
	return float64(amount) / 1_000_000_000
}

// ------------------------------ CLIENT SIDE ------------------------------ //
func clientSign(unsignedTx string, key string) (*string, error) {
	privateKey, err := solana.PrivateKeyFromBase58(key)
	if err != nil {
		return nil, err
	}
	txBytes, err := base64.StdEncoding.DecodeString(unsignedTx)
	if err != nil {
		return nil, err
	}
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if privateKey.PublicKey().Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	signedTxBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	signedTxBase64 := base64.StdEncoding.EncodeToString(signedTxBytes)
	return &signedTxBase64, nil
}
