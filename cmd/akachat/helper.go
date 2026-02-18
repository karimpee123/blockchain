package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/uuid"
)

const (
	NetSOL = "solana"
	NetBSC = "bsc"

	ActionCreate = "create"
	ActionClaim  = "claim"
	ActionRefund = "refund"

	// Token mint addresses (devnet)
	USDCMintDevnet = "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"

	// Token program IDs
	TokenProgramID           = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	AssociatedTokenProgramID = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
)

var (
	network                    Network
	userA, userB, userC, userD User
	httpClient                 = &http.Client{Timeout: 10 * time.Second}
	baseURL                    = "http://localhost:10011"
	rpcClient                  *rpc.Client
)

func initAll(chain string) {
	initUser()
	initNetwork(chain)
}

func initNetwork(chain string) {
	switch chain {
	case NetSOL:
		network = Network{Name: "solana", Symbol: "SOL"}
		// Initialize RPC client for devnet
		rpcClient = rpc.New(rpc.DevNet_RPC)
	case NetBSC:
		network = Network{Name: "bsc", Symbol: "BNB"}
	}
}

func initUser() {
	// User A: lampu (ID dari token: 7237260465)
	userA = User{
		ID:         "7237260465",
		Address:    "3YkzQC2PwFGvJr2GS7FDBopvG5tda4eXdq5pmwEbWeyd",
		PrivateKey: "HheE1MM3ciGE5hBzbfXNNeW4W4QatfAkBZgee962CWENsQrWWagNemxb8hreYnxZa2AmS1fx9MSYnbKCXGDzemV",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiI3MjM3MjYwNDY1IiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3ODcyNjkyNywibmJmIjoxNzcwOTUwODY3LCJpYXQiOjE3NzA5NTA5Mjd9.4ldyPENcydA9CZsr_5ZARLtPtXT8ZL323lcQlfhhhr8",
	}
	// User B: meja (ID dari token: 4138007321)
	userB = User{
		ID:         "4138007321",
		Address:    "wFuFPgHsLt9t5HALqFQqbdM9WvyQstdKN8NQXB3GWeD",
		PrivateKey: "3YMrwyXU2hNKDrUbxUUTBTr8HTSjLAiafWmGsmnUAVg8mMnH4osbPKEqiwkP2npstDA8uRzpUbDG1EZC2Pyvcur9",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiI0MTM4MDA3MzIxIiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3ODcyNjk3MywibmJmIjoxNzcwOTUwOTEzLCJpYXQiOjE3NzA5NTA5NzN9.pWiG2W_Of__DPriEl3Wq-ZHcjYYasdRtbVsjIb-ssi4",
	}
	// User C: lampu backup (ID dari token: 5834654941)
	userC = User{
		ID:         "5834654941",
		Address:    "12sYN8fo8Gxu526YnLxq76Xgt8bFXLuc6y21EEfna3qs",
		PrivateKey: "4LGCUbyTSa44aZshiHdgG8iDwV43wrNjhRhqsihGH7TJ9hYunnynTp2uCDjhHDL4sa1FnJHsyBsbCCiX6AQFpgED",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiI1ODM0NjU0OTQxIiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3ODcyNzEwNiwibmJmIjoxNzcwOTUxMDQ2LCJpYXQiOjE3NzA5NTExMDZ9.SqHLR_jPoYW57_Xos769b2ty_6NptlZsuHJ0myu_dI4",
	}
	// User D: pensil (ID dari token: 2698615014)
	userD = User{
		ID:         "2698615014",
		Address:    "9fru5gQYKd8PMS1qztZ9zLdTvVRQ11eF87PZYVUYVQsx",
		PrivateKey: "4MbCTDNAszFXV2ZUnkPni7oQJRs7DxbyJkGvfY2YNdtJcyG8QkXuW4MET62NQBNebMRqNVuTbuew3N1BoKs2ppn",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiIyNjk4NjE1MDE0IiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3ODcyNzE0NiwibmJmIjoxNzcwOTUxMDg2LCJpYXQiOjE3NzA5NTExNDZ9._KWH6hVx6SRNALkzsrSUfssydrGSAgontJ_cvhvmxiw",
	}
}

func doPost[T any](url string, body any, token string) (*APIResponse[T], error) {
	rawReq, _ := json.MarshalIndent(body, "", "  ")
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(rawReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("operationID", newOperationID())
	req.Header.Set("token", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(rawResp))
	}

	var result APIResponse[T]
	if err := json.Unmarshal(rawResp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func newOperationID() string {
	return uuid.NewString()
}

// waitForConfirmation waits for transaction to be confirmed and returns claim amount
func waitForConfirmation(txHash string, token string) (float64, error) {
	if rpcClient == nil {
		return 0, fmt.Errorf("RPC client not initialized")
	}

	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return 0, fmt.Errorf("invalid signature: %v", err)
	}

	ctx := context.Background()
	maxRetries := 20 // 20 seconds max - confirmed should be fast
	for i := 0; i < maxRetries; i++ {
		// Use confirmed commitment only - finalized takes too long
		out, err := rpcClient.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64,
			Commitment:                     rpc.CommitmentConfirmed,
			MaxSupportedTransactionVersion: func() *uint64 { v := uint64(0); return &v }(),
		})

		if err == nil && out != nil && out.Meta != nil {
			// Transaction found and confirmed
			if out.Meta.Err == nil {
				// Transaction successful, try to extract amount
				amount := extractClaimAmount(out, token)
				if amount > 0 {
					return amount, nil
				}
				// Transaction successful but can't extract amount
				// After 10 retries with confirmed status, accept that we can't get the amount
				if i >= 10 {
					return -1, nil // -1 indicates tx confirmed but amount indeterminate
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return 0, fmt.Errorf("transaction confirmation timeout")
}

// extractClaimAmount extracts the claim amount from transaction
func extractClaimAmount(tx *rpc.GetTransactionResult, token string) float64 {
	if tx.Meta == nil {
		return 0
	}

	// PRIORITY 1: Try to extract from program logs (most reliable for lucky envelopes)
	if tx.Meta.LogMessages != nil && len(tx.Meta.LogMessages) > 0 {
		_, amount := extractPayoutFromLogs(tx.Meta.LogMessages)
		if amount > 0 {
			// Convert based on token decimals
			if token == "SOL" {
				return float64(amount) / 1e9
			} else {
				// USDC/USDT have 6 decimals
				return float64(amount) / 1e6
			}
		}
	}

	// PRIORITY 2: Fallback to balance detection if logs don't have the info

	// For SOL: calculate balance difference (in lamports)
	if token == "SOL" {
		// Find the account with positive balance change (recipient)
		if tx.Meta.PreBalances != nil && tx.Meta.PostBalances != nil {
			var maxPositiveDiff int64 = 0
			// Check all accounts and find the largest positive balance change
			for i := 0; i < len(tx.Meta.PostBalances) && i < len(tx.Meta.PreBalances); i++ {
				diff := int64(tx.Meta.PostBalances[i]) - int64(tx.Meta.PreBalances[i])
				// Look for positive balance change (received SOL)
				// Use very low threshold to catch even small amounts
				if diff > 1000 && diff > maxPositiveDiff { // More than 1000 lamports (0.000001 SOL)
					maxPositiveDiff = diff
				}
			}
			if maxPositiveDiff > 0 {
				return float64(maxPositiveDiff) / 1e9 // Convert lamports to SOL
			}
		}
	} else {
		// For SPL tokens (USDC/USDT): check token balance changes
		if tx.Meta.PostTokenBalances != nil {
			// Create a map of pre-balances for quick lookup
			preBalanceMap := make(map[uint16]float64)
			if tx.Meta.PreTokenBalances != nil {
				for _, preBalance := range tx.Meta.PreTokenBalances {
					if preBalance.UiTokenAmount.UiAmount != nil {
						preBalanceMap[preBalance.AccountIndex] = *preBalance.UiTokenAmount.UiAmount
					}
				}
			}

			var maxPositiveDiff float64 = 0
			// Find the largest positive balance change
			for _, postBalance := range tx.Meta.PostTokenBalances {
				if postBalance.UiTokenAmount.UiAmount != nil {
					postAmount := *postBalance.UiTokenAmount.UiAmount
					preAmount := preBalanceMap[postBalance.AccountIndex]
					diff := postAmount - preAmount

					// Look for positive balance change (received tokens)
					// Very low threshold to catch small lucky amounts
					if diff > 0.000001 && diff > maxPositiveDiff {
						maxPositiveDiff = diff
					}
				}
			}
			if maxPositiveDiff > 0 {
				return maxPositiveDiff
			}
		}
	}

	return 0
}

// containsIgnoreCase checks if str contains substr (case-insensitive)
func containsIgnoreCase(str, substr string) bool {
	str = strings.ToLower(str)
	substr = strings.ToLower(substr)
	return strings.Contains(str, substr)
}

// getBalance returns SOL balance in lamports for the given address
func getBalance(address string) (uint64, error) {
	if rpcClient == nil {
		return 0, fmt.Errorf("RPC client not initialized")
	}

	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return 0, fmt.Errorf("invalid address: %v", err)
	}

	ctx := context.Background()
	balance, err := rpcClient.GetBalance(ctx, pubKey, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %v", err)
	}

	return balance.Value, nil
}

// deriveAssociatedTokenAddress derives the ATA address for a wallet and mint
func deriveAssociatedTokenAddress(walletAddress, mintAddress string) (solana.PublicKey, error) {
	wallet, err := solana.PublicKeyFromBase58(walletAddress)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("invalid wallet address: %v", err)
	}
	mint, err := solana.PublicKeyFromBase58(mintAddress)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("invalid mint address: %v", err)
	}
	ata, _, err := solana.FindAssociatedTokenAddress(wallet, mint)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to derive ATA: %v", err)
	}
	return ata, nil
}

// getTokenBalance returns token balance for the given address and mint (in base units)
func getTokenBalance(address, mintAddress string) (uint64, error) {
	if rpcClient == nil {
		return 0, fmt.Errorf("RPC client not initialized")
	}

	// Derive ATA
	ata, err := deriveAssociatedTokenAddress(address, mintAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to derive ATA: %v", err)
	}

	ctx := context.Background()

	// Get token account balance
	result, err := rpcClient.GetTokenAccountBalance(ctx, ata, rpc.CommitmentConfirmed)
	if err != nil {
		// Account might not exist yet (0 balance)
		return 0, nil
	}

	if result == nil || result.Value == nil || result.Value.Amount == "" {
		return 0, nil
	}

	// Parse amount string to uint64
	var amount uint64
	_, err = fmt.Sscanf(result.Value.Amount, "%d", &amount)
	if err != nil {
		return 0, fmt.Errorf("failed to parse token amount: %v", err)
	}

	return amount, nil
}

// getTransactionDetails returns fee, rent-exempt, SOL balance changes, and token balance changes from transaction
func getTransactionDetails(txHash string, userAddress string, tokenMint string) (fee uint64, rentExempt uint64, balanceBefore uint64, balanceAfter uint64, tokenBalanceBefore float64, tokenBalanceAfter float64, err error) {
	if rpcClient == nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("RPC client not initialized")
	}

	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("invalid signature: %v", err)
	}

	ctx := context.Background()
	// Wait a bit for transaction to be confirmed
	time.Sleep(2 * time.Second)

	out, err := rpcClient.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: func() *uint64 { v := uint64(0); return &v }(),
	})

	if err != nil || out == nil || out.Meta == nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("transaction not found or not confirmed yet")
	}

	fee = out.Meta.Fee

	// Get user's balance from transaction metadata
	// Find user's account index
	userPubkey, err := solana.PublicKeyFromBase58(userAddress)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("invalid user address: %v", err)
	}

	userAccountIndex := -1
	if out.Transaction != nil {
		tx, err := out.Transaction.GetTransaction()
		if err == nil && tx != nil {
			for i, account := range tx.Message.AccountKeys {
				if account.Equals(userPubkey) {
					userAccountIndex = i
					break
				}
			}
		}
	}

	// Get balances from metadata
	if userAccountIndex >= 0 && userAccountIndex < len(out.Meta.PreBalances) && userAccountIndex < len(out.Meta.PostBalances) {
		balanceBefore = out.Meta.PreBalances[userAccountIndex]
		balanceAfter = out.Meta.PostBalances[userAccountIndex]
	}

	// Calculate rent-exempt for account creation only (not including envelope value)
	// Rent-exempt accounts are small (<10M lamports = 0.01 SOL typically ~0.001-0.003 SOL)
	rentExempt = 0
	if out.Meta.PreBalances != nil && out.Meta.PostBalances != nil && len(out.Meta.PreBalances) > 0 {
		for i := 0; i < len(out.Meta.PostBalances) && i < len(out.Meta.PreBalances); i++ {
			// If an account went from 0 to >0, that's a new account
			if out.Meta.PreBalances[i] == 0 && out.Meta.PostBalances[i] > 0 {
				// Only count as rent-exempt if balance is small (< 10M lamports = 0.01 SOL)
				// Larger amounts are envelope value + rent-exempt
				if out.Meta.PostBalances[i] < 10_000_000 {
					rentExempt += out.Meta.PostBalances[i]
				} else {
					// For envelope accounts (large balance), estimate rent-exempt as ~2M lamports
					rentExempt += 2_039_280 // Typical rent-exempt for program accounts
				}
			}
		}
	}

	// Extract token balance changes if tokenMint is provided
	tokenBalanceBefore = 0
	tokenBalanceAfter = 0
	if tokenMint != "" && tokenMint != "SOL" {
		mintPubkey, err := solana.PublicKeyFromBase58(tokenMint)
		if err == nil {
			// Derive user's ATA
			ata, _, _ := solana.FindAssociatedTokenAddress(userPubkey, mintPubkey)

			// Find ATA in account keys
			ataIndex := -1
			if out.Transaction != nil {
				tx, err := out.Transaction.GetTransaction()
				if err == nil && tx != nil {
					for i, account := range tx.Message.AccountKeys {
						if account.Equals(ata) {
							ataIndex = i
							break
						}
					}
				}
			}

			// Extract token balances from PreTokenBalances and PostTokenBalances
			if ataIndex >= 0 {
				if out.Meta.PreTokenBalances != nil {
					for _, tokenBalance := range out.Meta.PreTokenBalances {
						if tokenBalance.AccountIndex == uint16(ataIndex) && tokenBalance.UiTokenAmount.UiAmount != nil {
							tokenBalanceBefore = *tokenBalance.UiTokenAmount.UiAmount
							break
						}
					}
				}

				if out.Meta.PostTokenBalances != nil {
					for _, tokenBalance := range out.Meta.PostTokenBalances {
						if tokenBalance.AccountIndex == uint16(ataIndex) && tokenBalance.UiTokenAmount.UiAmount != nil {
							tokenBalanceAfter = *tokenBalance.UiTokenAmount.UiAmount
							break
						}
					}
				}
			}
		}
	}

	return fee, rentExempt, balanceBefore, balanceAfter, tokenBalanceBefore, tokenBalanceAfter, nil
}
