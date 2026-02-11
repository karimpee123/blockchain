package main

import (
	"blockchain/solprogram"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

// Test users (Devnet)
var (
	// User 1 - Envelope Creator/Owner
	User1PrivateKey = solana.MustPrivateKeyFromBase58("3YMrwyXU2hNKDrUbxUUTBTr8HTSjLAiafWmGsmnUAVg8mMnH4osbPKEqiwkP2npstDA8uRzpUbDG1EZC2Pyvcur9")
	User1PublicKey  = solana.MustPublicKeyFromBase58("wFuFPgHsLt9t5HALqFQqbdM9WvyQstdKN8NQXB3GWeD")

	// User 2 - First Claimer
	User2PrivateKey = solana.MustPrivateKeyFromBase58("HheE1MM3ciGE5hBzbfXNNeW4W4QatfAkBZgee962CWENsQrWWagNemxb8hreYnxZa2AmS1fx9MSYnbKCXGDzemV")
	User2PublicKey  = solana.MustPublicKeyFromBase58("3YkzQC2PwFGvJr2GS7FDBopvG5tda4eXdq5pmwEbWeyd")

	// User 3 - Second Claimer
	User3PrivateKey = solana.MustPrivateKeyFromBase58("4MbCTDNAszFXV2ZUnkPni7oQJRs7DxbyJkGvfY2YNdtJcyG8QkXuW4MET62NQBNebMRqNVuTbuew3N1BoKs2ppn")
	User3PublicKey  = solana.MustPublicKeyFromBase58("9fru5gQYKd8PMS1qztZ9zLdTvVRQ11eF87PZYVUYVQsx")
)

func main() {
	fmt.Println("=== Solana USDC Envelope Program Demo ===\n")

	// =====================================================
	// TEST CONFIGURATION - Edit these flags to enable/disable tests
	// Run ONE test at a time to avoid state conflicts
	// =====================================================
	const (
		runInitUserState   = false // Initialize user state (required first time)
		runGroupFixed      = false // Create GroupFixed envelope (multiple claimers)
		runDirectFixed     = false // Create DirectFixed envelope (single claimer)
		runGetEnvelopeInfo = false // Get envelope info (requires envelope created)
		runClaim           = false // Claim envelope (requires User2 ATA + envelope exists)
		runWaitAndRefund   = false // Wait for expiry then refund
		runCheckTxStatus   = false // Check transaction status

		// NEW: Unsigned Transaction Demos
		runUnsignedInit   = false // Demo: Generate unsigned init_user_state transaction
		runUnsignedCreate = false // Demo: Generate unsigned create_envelope transaction
		runUnsignedClaim  = false // Demo: Generate unsigned claim transaction
		runUnsignedRefund = false // Demo: Generate unsigned refund transaction (after expiry)

		// Complete Flow Demo (create -> claim -> refund)
		runCompleteFlow = true // Demo: Complete unsigned transaction flow (create -> wait 2-3s -> claim -> wait 60s -> refund)
	)

	// Setup
	ctx := context.Background()

	// Create client (Devnet)
	client, err := solprogram.NewUSDCEnvelopeClient(
		solprogram.RPCURLDevnet,
		solprogram.WSURLDevnet,
		"devnet",
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("✅ Connected to Solana Devnet\n")
	fmt.Printf("Program ID: %s\n\n", client.GetProgramID().String())

	// Display users
	fmt.Println("📋 Test Users:")
	fmt.Printf("User 1 (Owner):   %s\n", User1PublicKey.String())
	fmt.Printf("User 2 (Claimer): %s\n", User2PublicKey.String())
	fmt.Printf("User 3 (Claimer): %s\n\n", User3PublicKey.String())

	// Variables to store created envelope IDs
	var groupFixedEnvelopeID uint64
	var directFixedEnvelopeID uint64
	var lastTxSignature string

	// Example 1: Initialize User State
	if runInitUserState {
		fmt.Println("--- Example 1: Initialize User State ---")
		demonstrateInitUserState(ctx, client)
	}

	// Example 2: Create GroupFixed Envelope
	if runGroupFixed {
		fmt.Println("\n--- Example 2: Create GroupFixed Envelope ---")
		groupFixedEnvelopeID = demonstrateCreateGroupFixed(ctx, client)

		// If claim is also enabled, wait extra time for RPC to sync
		if runClaim {
			fmt.Println("\n⏳ Waiting for RPC to sync envelope state before claiming...")
			time.Sleep(5 * time.Second)
		}
	}

	// Example 3: Create DirectFixed Envelope
	if runDirectFixed {
		fmt.Println("\n--- Example 3: Create DirectFixed Envelope ---")
		directFixedEnvelopeID, lastTxSignature = demonstrateCreateDirectFixed(ctx, client)

		// If GetEnvelopeInfo or refund is also enabled, ensure envelope exists before proceeding
		if runGetEnvelopeInfo || runWaitAndRefund {
			fmt.Println("\n⏳ Waiting for RPC to sync envelope state...")
			time.Sleep(5 * time.Second)
		}
	}

	// Example 4: Get Envelope Info
	if runGetEnvelopeInfo {
		fmt.Println("\n--- Example 4: Get Envelope Info ---")
		envelopeID := groupFixedEnvelopeID
		if envelopeID == 0 {
			envelopeID = directFixedEnvelopeID
		}
		if envelopeID == 0 {
			log.Fatal("❌ Error: No envelope created. Set runGroupFixed=true or runDirectFixed=true first!")
		}
		demonstrateGetEnvelopeInfo(ctx, client, envelopeID)
	}

	// Example 5: Claim Envelope
	if runClaim {
		fmt.Println("\n--- Example 5: Claim Envelope ---")
		if groupFixedEnvelopeID == 0 {
			log.Fatal("❌ Error: No GroupFixed envelope created. Set runGroupFixed=true first!")
		}

		// Verify envelope exists before claiming
		fmt.Printf("Verifying envelope %d exists...\n", groupFixedEnvelopeID)
		_, err := client.GetEnvelopeInfo(ctx, User1PublicKey, groupFixedEnvelopeID)
		if err != nil {
			fmt.Printf("⚠️  Envelope not ready yet. Waiting 5 more seconds...\n")
			time.Sleep(5 * time.Second)
			_, err = client.GetEnvelopeInfo(ctx, User1PublicKey, groupFixedEnvelopeID)
			if err != nil {
				log.Fatalf("❌ Error: Envelope %d not found: %v", groupFixedEnvelopeID, err)
			}
		}
		fmt.Println("✅ Envelope verified!")

		demonstrateClaim(ctx, client, groupFixedEnvelopeID)
	}

	// Example 6: Wait for Expiry and Refund
	if runWaitAndRefund {
		fmt.Println("\n--- Example 6: Test Refund After Expiry ---")
		if directFixedEnvelopeID == 0 {
			log.Fatal("❌ Error: No DirectFixed envelope created. Set runDirectFixed=true first!")
		}
		fmt.Println("Waiting 65 seconds for envelope to expire (60 seconds + buffer)...")
		time.Sleep(65 * time.Second)
		demonstrateRefund(ctx, client, directFixedEnvelopeID)
	}

	// Example 7: Check Transaction Status
	if runCheckTxStatus {
		fmt.Println("\n--- Example 7: Check Transaction Status ---")
		if lastTxSignature != "" {
			demonstrateTransactionStatus(ctx, client, lastTxSignature)
		} else {
			fmt.Println("⚠️  Skipped: No transaction signature available. Create an envelope first!")
		}
	}

	// =========================
	// UNSIGNED TRANSACTION DEMOS
	// =========================

	// Example 8: Unsigned Init User State
	if runUnsignedInit {
		fmt.Println("\n--- Example 8: Unsigned Transaction - Init User State ---")
		demonstrateUnsignedInitUserState(ctx, client)
	}

	// Example 9: Unsigned Create Envelope
	var unsignedEnvelopeID uint64
	if runUnsignedCreate {
		fmt.Println("\n--- Example 9: Unsigned Transaction - Create Envelope ---")
		unsignedEnvelopeID = demonstrateUnsignedCreateEnvelope(ctx, client)
	}

	// Example 10: Unsigned Claim
	if runUnsignedClaim {
		fmt.Println("\n--- Example 10: Unsigned Transaction - Claim ---")
		if unsignedEnvelopeID == 0 {
			log.Fatal("❌ Error: No envelope created via unsigned transaction. Set runUnsignedCreate=true first!")
		}
		fmt.Println("Waiting 5 seconds for envelope to be confirmed...")
		time.Sleep(5 * time.Second)
		demonstrateUnsignedClaim(ctx, client, unsignedEnvelopeID)
	}

	// Example 11: Unsigned Refund (after expiry)
	if runUnsignedRefund {
		fmt.Println("\n--- Example 11: Unsigned Transaction - Refund ---")
		if unsignedEnvelopeID == 0 {
			log.Fatal("❌ Error: No envelope created via unsigned transaction. Set runUnsignedCreate=true first!")
		}
		fmt.Println("Waiting 65 seconds for envelope to expire (60 seconds + buffer)...")
		time.Sleep(65 * time.Second)
		demonstrateUnsignedRefund(ctx, client, unsignedEnvelopeID)
	}

	// Example 12: Complete Flow Demo
	if runCompleteFlow {
		fmt.Println("\n--- Example 12: Complete Unsigned Transaction Flow ---")
		demonstrateCompleteFlow(ctx, client)
	}

	fmt.Println("\n=== Demo Complete ===")
}

// demonstrateInitUserState - Initialize user state
func demonstrateInitUserState(ctx context.Context, client *solprogram.USDCEnvelopeClient) {
	fmt.Printf("Initializing User 1 (Owner): %s\n", User1PublicKey.String())

	// Check if already initialized
	userState, err := client.GetUserState(ctx, User1PublicKey)
	if err == nil {
		fmt.Printf("✅ User 1 already initialized. Last Envelope ID: %d\n", userState.LastEnvelopeID)
		return
	}

	// Initialize user state
	fmt.Println("Initializing user state for User 1...")

	result, err := client.InitUserState(ctx, User1PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ User state initialized!\n")
	fmt.Printf("Signature: %s\n", result.Signature)
	fmt.Printf("Explorer: %s\n", result.ExplorerURL)

	// Wait for confirmation
	fmt.Println("Waiting for confirmation...")
	err = client.WaitForConfirmation(ctx, result.Signature, 30) // 30 second timeout
	if err != nil {
		fmt.Printf("❌ Confirmation failed: %v\n", err)
		return
	}
	fmt.Println("✅ Transaction confirmed!")

	// Verify initialization
	userState, err = client.GetUserState(ctx, User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error getting user state: %v\n", err)
		return
	}

	fmt.Printf("✅ User State verified! Last Envelope ID: %d\n", userState.LastEnvelopeID)
}

// demonstrateCreateGroupFixed - Create GroupFixed envelope
func demonstrateCreateGroupFixed(ctx context.Context, client *solprogram.USDCEnvelopeClient) uint64 {
	fmt.Printf("Creating envelope with User 1 (Owner): %s\n", User1PublicKey.String())

	// Get User1's USDC Associated Token Account
	userTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving ATA: %v\n", err)
		return 0
	}
	fmt.Printf("  User1 USDC Token Account: %s\n", userTokenAccount.String())

	// Create envelope parameters
	params := solprogram.CreateEnvelopeParams{
		EnvelopeType: solprogram.EnvelopeTypeData{
			Type:           solprogram.EnvelopeTypeGroupFixed,
			AllowedAddress: nil,
		},
		TotalAmount:   1_000_000, // 1 USDC
		TotalUsers:    2,         // User2 and User3 can claim
		ExpirySeconds: 60,        // Expires in 60 seconds
	}

	fmt.Printf("  Total Amount: %.2f USDC\n", float64(params.TotalAmount)/1_000_000)
	fmt.Printf("  Total Users: %d\n", params.TotalUsers)
	fmt.Printf("  Amount per User: %.2f USDC\n", float64(params.TotalAmount)/float64(params.TotalUsers)/1_000_000)
	fmt.Printf("  Expiry: %d seconds\n", params.ExpirySeconds)

	// Create and send transaction
	response, err := client.CreateEnvelope(ctx, User1PrivateKey, userTokenAccount, params)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		fmt.Println("\nNote: Make sure you have:")
		fmt.Println("  1. Initialized user state first")
		fmt.Println("  2. SOL for transaction fees")
		fmt.Println("  3. USDC tokens in User1's account")
		fmt.Println("  4. Created USDC token account (ATA) for User1")
		fmt.Println("\nCreate USDC ATA: spl-token create-account <USDC_MINT> --owner", User1PublicKey.String())
		return 0
	}

	fmt.Printf("\n✅ Envelope created successfully!\n")
	fmt.Printf("Envelope ID: %d\n", response.EnvelopeID)
	fmt.Printf("Envelope PDA: %s\n", response.EnvelopePDA.String())
	fmt.Printf("Vault PDA: %s\n", response.VaultPDA.String())
	fmt.Printf("Signature: %s\n", response.Signature)

	// Wait for confirmation
	fmt.Println("\nWaiting for confirmation...")
	err = client.WaitForConfirmation(ctx, response.Signature, 30)
	if err != nil {
		fmt.Printf("⚠️  Warning: Confirmation failed: %v\n", err)
	} else {
		fmt.Println("✅ Transaction confirmed!")
		// Give RPC time to update account state
		time.Sleep(3 * time.Second)
	}

	return response.EnvelopeID
}

// demonstrateCreateDirectFixed - Create DirectFixed envelope
func demonstrateCreateDirectFixed(ctx context.Context, client *solprogram.USDCEnvelopeClient) (uint64, string) {
	fmt.Printf("Creating DirectFixed envelope with User 1: %s\n", User1PublicKey.String())
	fmt.Printf("Only User 2 can claim: %s\n", User2PublicKey.String())

	// Get User1's USDC Associated Token Account
	userTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving ATA: %v\n", err)
		return 0, ""
	}

	params := solprogram.CreateEnvelopeParams{
		EnvelopeType: solprogram.EnvelopeTypeData{
			Type:           solprogram.EnvelopeTypeDirectFixed,
			AllowedAddress: &User2PublicKey,
		},
		TotalAmount:   500_000, // 0.5 USDC
		TotalUsers:    1,       // Only User2 can claim
		ExpirySeconds: 60,      // Expires in 60 seconds
	}

	fmt.Printf("  Amount: %.2f USDC\n", float64(params.TotalAmount)/1_000_000)
	fmt.Printf("  Expiry: %d seconds\n", params.ExpirySeconds)

	response, err := client.CreateEnvelope(ctx, User1PrivateKey, userTokenAccount, params)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return 0, ""
	}

	fmt.Printf("\n✅ DirectFixed envelope created!\n")
	fmt.Printf("Envelope ID: %d\n", response.EnvelopeID)
	fmt.Printf("Signature: %s\n", response.Signature)

	// Wait for confirmation
	fmt.Println("\nWaiting for confirmation...")
	err = client.WaitForConfirmation(ctx, response.Signature, 30)
	if err != nil {
		fmt.Printf("⚠️  Warning: Confirmation failed: %v\n", err)
	} else {
		fmt.Println("✅ Transaction confirmed!")
		// Give RPC time to update account state
		time.Sleep(3 * time.Second)
	}

	return response.EnvelopeID, response.Signature
}

// demonstrateGetEnvelopeInfo - Get envelope info
func demonstrateGetEnvelopeInfo(ctx context.Context, client *solprogram.USDCEnvelopeClient, envelopeID uint64) {
	fmt.Printf("Fetching envelope info for User 1: %s\n", User1PublicKey.String())
	fmt.Printf("Envelope ID: %d\n", envelopeID)

	info, err := client.GetEnvelopeInfo(ctx, User1PublicKey, envelopeID)
	if err != nil {
		// Retry once after 5 seconds
		fmt.Printf("⚠️  Envelope not ready yet. Waiting 5 more seconds...\n")
		time.Sleep(5 * time.Second)
		info, err = client.GetEnvelopeInfo(ctx, User1PublicKey, envelopeID)
		if err != nil {
			fmt.Printf("❌ Error: Envelope not found: %v\n", err)
			fmt.Println("Note: Make sure envelope was created successfully")
			return
		}
	}

	// Pretty print envelope info
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	fmt.Printf("\n✅ Envelope Info:\n%s\n", string(infoJSON))

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Type: %s\n", info.EnvelopeType)
	fmt.Printf("  Total Amount: %.6f USDC\n", float64(info.TotalAmount)/1_000_000)
	fmt.Printf("  Claimed: %d/%d users\n", info.ClaimedCount, info.TotalUsers)
	fmt.Printf("  Remaining: %.6f USDC\n", float64(info.RemainingAmount)/1_000_000)
	fmt.Printf("  Expired: %v\n", info.IsExpired)
	fmt.Printf("  Expiry Time: %s\n", info.ExpiryTime.Format(time.RFC3339))
}

// demonstrateClaim - Claim from envelope
func demonstrateClaim(ctx context.Context, client *solprogram.USDCEnvelopeClient, envelopeID uint64) {
	fmt.Printf("User 2 claiming from envelope: %s\n", User2PublicKey.String())

	// Get User2's USDC Associated Token Account
	claimerTokenAccount, err := client.GetUSDCTokenAddress(User2PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving ATA: %v\n", err)
		return
	}

	// Verify User2 has USDC token account
	fmt.Printf("Checking User2 USDC Token Account: %s\n", claimerTokenAccount.String())
	accountInfo, err := client.GetClient().GetAccountInfo(ctx, claimerTokenAccount)
	if err != nil || accountInfo == nil || accountInfo.Value == nil {
		fmt.Printf("\n❌ Error: User2 doesn't have USDC token account!\n")
		fmt.Printf("   ATA Address: %s\n", claimerTokenAccount.String())
		fmt.Printf("\n🔧 Create it with:\n")
		fmt.Printf("   spl-token create-account 4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU \\\n")
		fmt.Printf("     --owner %s \\\n", User2PublicKey.String())
		fmt.Printf("     --url devnet\n\n")
		return
	}
	fmt.Println("✅ User2 USDC Token Account verified!")

	params := solprogram.ClaimEnvelopeParams{
		EnvelopeID:          envelopeID,
		Owner:               User1PublicKey,
		Claimer:             User2PublicKey,
		ClaimerTokenAccount: claimerTokenAccount,
	}

	fmt.Printf("  Envelope ID: %d\n", envelopeID)
	fmt.Printf("  Owner: %s\n", User1PublicKey.String())

	response, err := client.ClaimEnvelope(ctx, User2PrivateKey, params)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		fmt.Println("\nPossible reasons:")
		fmt.Println("  1. Envelope doesn't exist yet")
		fmt.Println("  2. Already claimed")
		fmt.Println("  3. Not allowed (for DirectFixed)")
		fmt.Println("  4. Envelope expired")
		fmt.Println("  5. Quota full")
		fmt.Println("\nCreate USDC ATA: spl-token create-account <USDC_MINT> --owner", User2PublicKey.String())
		return
	}

	fmt.Printf("\n✅ Claimed successfully!\n")
	fmt.Printf("Signature: %s\n", response.Signature)

	// Wait for confirmation
	fmt.Println("\nWaiting for confirmation...")
	err = client.WaitForConfirmation(ctx, response.Signature, 30)
	if err != nil {
		fmt.Printf("⚠️  Warning: Confirmation failed: %v\n", err)
		return
	}
	fmt.Println("✅ Transaction confirmed!")

	// Check final status
	status, _ := client.GetTransactionStatus(ctx, response.Signature)
	if status != nil {
		fmt.Printf("Status: %s\n", status.Status)
	}
}

// demonstrateRefund - Refund expired envelope
func demonstrateRefund(ctx context.Context, client *solprogram.USDCEnvelopeClient, envelopeID uint64) {
	fmt.Printf("User 1 refunding expired envelope: %s\n", User1PublicKey.String())

	// Get User1's USDC Associated Token Account
	ownerTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving ATA: %v\n", err)
		return
	}

	fmt.Printf("  Envelope ID: %d\n", envelopeID)
	fmt.Printf("  Owner Token Account: %s\n", ownerTokenAccount.String())

	response, err := client.RefundEnvelope(ctx, User1PrivateKey, ownerTokenAccount, envelopeID)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		fmt.Println("\nPossible reasons:")
		fmt.Println("  1. Envelope doesn't exist")
		fmt.Println("  2. Not expired yet")
		fmt.Println("  3. Already refunded")
		fmt.Println("  4. Not the owner")
		return
	}

	fmt.Printf("\n✅ Refunded successfully!\n")
	fmt.Printf("Signature: %s\n", response.Signature)

	// Wait for confirmation
	fmt.Println("\nWaiting for confirmation...")
	err = client.WaitForConfirmation(ctx, response.Signature, 30)
	if err != nil {
		fmt.Printf("⚠️  Warning: Confirmation failed: %v\n", err)
		return
	}
	fmt.Println("✅ Transaction confirmed!")
}

// demonstrateTransactionStatus - Check transaction status
func demonstrateTransactionStatus(ctx context.Context, client *solprogram.USDCEnvelopeClient, signature string) {
	fmt.Printf("Checking transaction status...\n")
	fmt.Printf("Signature: %s\n", signature)

	result, err := client.GetTransactionStatus(ctx, signature)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Transaction Status:\n")
	fmt.Printf("  Status: %s\n", result.Status)
	if result.Error != nil {
		fmt.Printf("  Error: %s\n", *result.Error)
	}
	fmt.Printf("  Explorer: %s\n", result.ExplorerURL)

	// Status explanation
	switch result.Status {
	case solprogram.StatusFinalized:
		fmt.Println("  ✅ Transaction is finalized (permanent)")
	case solprogram.StatusConfirmed:
		fmt.Println("  ⏳ Transaction is confirmed (waiting for finalization)")
	case solprogram.StatusPending:
		fmt.Println("  ⏳ Transaction is pending")
	case solprogram.StatusFailed:
		fmt.Println("  ❌ Transaction failed")
	}
}

// =========================
// UNSIGNED TRANSACTION DEMOS
// =========================

// demonstrateUnsignedInitUserState - Example: Generate unsigned transaction for init user state
func demonstrateUnsignedInitUserState(ctx context.Context, client *solprogram.USDCEnvelopeClient) {
	fmt.Println("Generating unsigned transaction for init user state...")
	fmt.Printf("User: %s\n", User1PublicKey.String())

	// Step 1: Backend generates unsigned transaction
	response, err := client.GenerateUnsignedInitUserState(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error generating unsigned transaction: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Unsigned transaction generated!\n")
	fmt.Printf("Transaction ID: %s\n", response.TransactionID)
	fmt.Printf("Recent Blockhash: %s\n", response.RecentBlockhash)
	fmt.Printf("Unsigned Transaction (base64): %s...\n", response.UnsignedTransaction[:50])
	fmt.Printf("\n📤 Next step: Frontend will sign this transaction with user's private key\n")
	fmt.Printf("📥 Then: Submit signed transaction back to backend via SubmitSignedTransaction()\n")

	// Step 2: Frontend signs transaction (simulated here for demo)
	fmt.Println("\n--- Simulating Frontend Signing (FOR DEMO ONLY) ---")
	signedTx, err := signTransactionDemo(response.UnsignedTransaction, User1PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error signing: %v\n", err)
		return
	}

	// Step 3: Backend submits signed transaction
	fmt.Println("\n--- Submitting Signed Transaction ---")
	signedReq := solprogram.SignedTransactionRequest{
		TransactionID:     response.TransactionID,
		SignedTransaction: signedTx,
	}

	result, err := client.SubmitSignedTransaction(signedReq)
	if err != nil {
		fmt.Printf("❌ Error submitting: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Transaction submitted!\n")
	fmt.Printf("Signature: %s\n", result.Signature)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Explorer: %s\n", result.ExplorerURL)
}

// demonstrateUnsignedCreateEnvelope - Example: Generate unsigned transaction for create envelope
func demonstrateUnsignedCreateEnvelope(ctx context.Context, client *solprogram.USDCEnvelopeClient) uint64 {
	fmt.Println("Generating unsigned transaction for create envelope...")

	// Get user state
	userState, err := client.GetUserState(ctx, User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error getting user state: %v\n", err)
		fmt.Println("Run init_user_state first!")
		return 0
	}

	nextEnvelopeID := userState.LastEnvelopeID + 1

	// Get user's USDC token account
	userTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving token account: %v\n", err)
		return 0
	}

	// Create params for DirectFixed envelope
	params := solprogram.CreateEnvelopeParams{
		EnvelopeType: solprogram.EnvelopeTypeData{
			Type:           solprogram.EnvelopeTypeDirectFixed,
			AllowedAddress: &User2PublicKey,
		},
		TotalAmount:   1_000_000, // 1 USDC (6 decimals)
		TotalUsers:    1,
		ExpirySeconds: 60, // 60 seconds
	}

	fmt.Printf("  Next Envelope ID: %d\n", nextEnvelopeID)
	fmt.Printf("  Type: DirectFixed\n")
	fmt.Printf("  Allowed Address: %s\n", User2PublicKey.String())
	fmt.Printf("  Amount: %d (1 USDC)\n", params.TotalAmount)
	fmt.Printf("  Expiry: %d seconds\n", params.ExpirySeconds)

	// Step 1: Backend generates unsigned transaction
	response, err := client.GenerateUnsignedCreateEnvelope(
		User1PublicKey,
		userTokenAccount,
		params,
		nextEnvelopeID,
	)
	if err != nil {
		fmt.Printf("❌ Error generating unsigned transaction: %v\n", err)
		return 0
	}

	fmt.Printf("\n✅ Unsigned transaction generated!\n")
	fmt.Printf("Transaction ID: %s\n", response.TransactionID)
	fmt.Printf("Unsigned Transaction (base64): %s...\n", response.UnsignedTransaction[:50])

	// Step 2: Simulate signing
	fmt.Println("\n--- Simulating Frontend Signing ---")
	signedTx, err := signTransactionDemo(response.UnsignedTransaction, User1PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error signing: %v\n", err)
		return 0
	}

	// Step 3: Submit signed transaction
	fmt.Println("\n--- Submitting Signed Transaction ---")
	signedReq := solprogram.SignedTransactionRequest{
		TransactionID:     response.TransactionID,
		SignedTransaction: signedTx,
	}

	result, err := client.SubmitSignedTransaction(signedReq)
	if err != nil {
		fmt.Printf("❌ Error submitting: %v\n", err)
		return 0
	}

	fmt.Printf("\n✅ Envelope created!\n")
	fmt.Printf("Envelope ID: %d\n", nextEnvelopeID)
	fmt.Printf("Signature: %s\n", result.Signature)
	fmt.Printf("Explorer: %s\n", result.ExplorerURL)

	return nextEnvelopeID
}

// demonstrateUnsignedClaim - Example: Generate unsigned transaction for claim
func demonstrateUnsignedClaim(ctx context.Context, client *solprogram.USDCEnvelopeClient, envelopeID uint64) {
	fmt.Println("Generating unsigned transaction for claim...")
	fmt.Printf("Envelope ID: %d\n", envelopeID)
	fmt.Printf("Claimer: %s\n", User2PublicKey.String())

	// Get claimer's USDC token account
	claimerTokenAccount, err := client.GetUSDCTokenAddress(User2PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving token account: %v\n", err)
		return
	}

	params := solprogram.ClaimEnvelopeParams{
		Owner:               User1PublicKey,
		EnvelopeID:          envelopeID,
		Claimer:             User2PublicKey,
		ClaimerTokenAccount: claimerTokenAccount,
	}

	// Step 1: Backend generates unsigned transaction
	response, err := client.GenerateUnsignedClaim(params)
	if err != nil {
		fmt.Printf("❌ Error generating unsigned transaction: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Unsigned transaction generated!\n")
	fmt.Printf("Transaction ID: %s\n", response.TransactionID)

	// Step 2: Simulate signing by User2
	fmt.Println("\n--- Simulating Frontend Signing (User2) ---")
	signedTx, err := signTransactionDemo(response.UnsignedTransaction, User2PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error signing: %v\n", err)
		return
	}

	// Step 3: Submit signed transaction
	fmt.Println("\n--- Submitting Signed Transaction ---")
	signedReq := solprogram.SignedTransactionRequest{
		TransactionID:     response.TransactionID,
		SignedTransaction: signedTx,
	}

	result, err := client.SubmitSignedTransaction(signedReq)
	if err != nil {
		fmt.Printf("❌ Error submitting: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Claim successful!\n")
	fmt.Printf("Signature: %s\n", result.Signature)
	fmt.Printf("Explorer: %s\n", result.ExplorerURL)
}

// demonstrateUnsignedRefund - Example: Generate unsigned transaction for refund
func demonstrateUnsignedRefund(ctx context.Context, client *solprogram.USDCEnvelopeClient, envelopeID uint64) {
	fmt.Println("Generating unsigned transaction for refund...")
	fmt.Printf("Envelope ID: %d\n", envelopeID)

	// Get owner's USDC token account
	ownerTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving token account: %v\n", err)
		return
	}

	params := solprogram.RefundParams{
		Owner:             User1PublicKey,
		EnvelopeID:        envelopeID,
		OwnerTokenAccount: ownerTokenAccount,
	}

	// Step 1: Backend generates unsigned transaction
	response, err := client.GenerateUnsignedRefund(params)
	if err != nil {
		fmt.Printf("❌ Error generating unsigned transaction: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Unsigned transaction generated!\n")
	fmt.Printf("Transaction ID: %s\n", response.TransactionID)

	// Step 2: Simulate signing
	fmt.Println("\n--- Simulating Frontend Signing ---")
	signedTx, err := signTransactionDemo(response.UnsignedTransaction, User1PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error signing: %v\n", err)
		return
	}

	// Step 3: Submit signed transaction
	fmt.Println("\n--- Submitting Signed Transaction ---")
	signedReq := solprogram.SignedTransactionRequest{
		TransactionID:     response.TransactionID,
		SignedTransaction: signedTx,
	}

	result, err := client.SubmitSignedTransaction(signedReq)
	if err != nil {
		fmt.Printf("❌ Error submitting: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Refund successful!\n")
	fmt.Printf("Signature: %s\n", result.Signature)
	fmt.Printf("Explorer: %s\n", result.ExplorerURL)
}

// demonstrateCompleteFlow - Complete flow: create -> wait -> claim -> wait -> refund
func demonstrateCompleteFlow(ctx context.Context, client *solprogram.USDCEnvelopeClient) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🚀 COMPLETE UNSIGNED TRANSACTION FLOW DEMONSTRATION")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Flow: Create Envelope → Wait 2-3s → Claim → Wait 60s → Refund")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// ========================================
	// STEP 1: Create Envelope (Unsigned Transaction)
	// ========================================
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STEP 1: CREATE ENVELOPE (Unsigned Transaction Pattern)   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// Get user state
	userState, err := client.GetUserState(ctx, User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error getting user state: %v\n", err)
		fmt.Println("\n💡 TIP: Run with runInitUserState=true first to initialize user state")
		return
	}

	envelopeID := userState.LastEnvelopeID + 1

	// Get user's USDC token account
	userTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving token account: %v\n", err)
		return
	}

	// Create params for GroupFixed envelope (anyone can claim, up to 3 users)
	params := solprogram.CreateEnvelopeParams{
		EnvelopeType: solprogram.EnvelopeTypeData{
			Type:           solprogram.EnvelopeTypeGroupFixed,
			AllowedAddress: nil, // Anyone can claim
		},
		TotalAmount:   3_000_000, // 3 USDC total (6 decimals)
		TotalUsers:    3,         // Max 3 claimers, each gets 1 USDC
		ExpirySeconds: 60,        // 60 seconds
	}

	fmt.Printf("📋 Envelope Configuration:\n")
	fmt.Printf("   Envelope ID: %d\n", envelopeID)
	fmt.Printf("   Type: GroupFixed\n")
	fmt.Printf("   Owner: %s\n", User1PublicKey.String())
	fmt.Printf("   Total Users: %d (Anyone can claim)\n", params.TotalUsers)
	fmt.Printf("   Total Amount: %.2f USDC\n", float64(params.TotalAmount)/1_000_000)
	fmt.Printf("   Amount per User: %.2f USDC\n", float64(params.TotalAmount)/float64(params.TotalUsers)/1_000_000)
	fmt.Printf("   Expiry: %d seconds\n\n", params.ExpirySeconds)

	// Generate unsigned transaction (backend)
	fmt.Println("🔧 Backend: Generating unsigned transaction...")
	unsignedResp, err := client.GenerateUnsignedCreateEnvelope(
		User1PublicKey,
		userTokenAccount,
		params,
		envelopeID,
	)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Unsigned transaction generated\n")
	fmt.Printf("   Transaction ID: %s\n", unsignedResp.TransactionID)
	fmt.Printf("   Recent Blockhash: %s\n\n", unsignedResp.RecentBlockhash)

	// Sign transaction (frontend - simulated)
	fmt.Println("🔐 Frontend: Signing transaction with User1's private key...")
	signedTx, err := signTransactionDemo(unsignedResp.UnsignedTransaction, User1PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Transaction signed\n\n")

	// Submit signed transaction (backend)
	fmt.Println("📤 Backend: Submitting signed transaction...")
	signedReq := solprogram.SignedTransactionRequest{
		TransactionID:     unsignedResp.TransactionID,
		SignedTransaction: signedTx,
	}

	createResult, err := client.SubmitSignedTransaction(signedReq)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Envelope created successfully!\n")
	fmt.Printf("   Signature: %s\n", createResult.Signature)
	fmt.Printf("   Explorer: %s\n\n", createResult.ExplorerURL)

	// ========================================
	// STEP 2: Wait for confirmation
	// ========================================
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STEP 2: WAIT FOR CONFIRMATION (2-3 seconds)              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("⏳ Waiting 3 seconds for transaction to be confirmed...")
	time.Sleep(3 * time.Second)
	fmt.Println("✅ Wait complete\n")

	// ========================================
	// STEP 3: Claim Envelope (Unsigned Transaction)
	// ========================================
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STEP 3: CLAIM ENVELOPE (Unsigned Transaction Pattern)    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// Get claimer's USDC token account
	claimerTokenAccount, err := client.GetUSDCTokenAddress(User2PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving token account: %v\n", err)
		return
	}

	claimParams := solprogram.ClaimEnvelopeParams{
		Owner:               User1PublicKey,
		EnvelopeID:          envelopeID,
		Claimer:             User2PublicKey,
		ClaimerTokenAccount: claimerTokenAccount,
	}

	fmt.Printf("📋 Claim Configuration:\n")
	fmt.Printf("   Envelope ID: %d\n", envelopeID)
	fmt.Printf("   Claimer: %s\n", User2PublicKey.String())
	fmt.Printf("   Amount per Claim: %.2f USDC (1/%d share)\n\n", float64(params.TotalAmount)/float64(params.TotalUsers)/1_000_000, params.TotalUsers)

	// Generate unsigned claim transaction (backend)
	fmt.Println("🔧 Backend: Generating unsigned claim transaction...")
	unsignedClaimResp, err := client.GenerateUnsignedClaim(claimParams)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Unsigned claim transaction generated\n")
	fmt.Printf("   Transaction ID: %s\n\n", unsignedClaimResp.TransactionID)

	// Sign transaction (frontend - simulated with User2's key)
	fmt.Println("🔐 Frontend: Signing transaction with User2's private key...")
	signedClaimTx, err := signTransactionDemo(unsignedClaimResp.UnsignedTransaction, User2PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Transaction signed\n\n")

	// Submit signed claim transaction (backend)
	fmt.Println("📤 Backend: Submitting signed claim transaction...")
	signedClaimReq := solprogram.SignedTransactionRequest{
		TransactionID:     unsignedClaimResp.TransactionID,
		SignedTransaction: signedClaimTx,
	}

	claimResult, err := client.SubmitSignedTransaction(signedClaimReq)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Envelope claimed successfully!\n")
	fmt.Printf("   Signature: %s\n", claimResult.Signature)
	fmt.Printf("   Explorer: %s\n\n", claimResult.ExplorerURL)

	// ========================================
	// STEP 4: Wait for envelope to expire
	// ========================================
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STEP 4: WAIT FOR EXPIRY (60 seconds)                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("⏳ Waiting 60 seconds for envelope to expire...")
	fmt.Println("   (In production, you would check expiry time instead of fixed wait)")

	// Progress indicator
	for i := 1; i <= 60; i++ {
		time.Sleep(1 * time.Second)
		if i%10 == 0 {
			fmt.Printf("   ⏱️  %d seconds elapsed...\n", i)
		}
	}
	fmt.Println("✅ Envelope expired\n")

	// ========================================
	// STEP 5: Refund Envelope (Unsigned Transaction)
	// ========================================
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STEP 5: REFUND ENVELOPE (Unsigned Transaction Pattern)   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// Get owner's USDC token account
	ownerTokenAccount, err := client.GetUSDCTokenAddress(User1PublicKey)
	if err != nil {
		fmt.Printf("❌ Error deriving token account: %v\n", err)
		return
	}

	refundParams := solprogram.RefundParams{
		Owner:             User1PublicKey,
		EnvelopeID:        envelopeID,
		OwnerTokenAccount: ownerTokenAccount,
	}

	fmt.Printf("📋 Refund Configuration:\n")
	fmt.Printf("   Envelope ID: %d\n", envelopeID)
	fmt.Printf("   Owner: %s\n", User1PublicKey.String())
	fmt.Printf("   Refund to: %s\n\n", ownerTokenAccount.String())

	// Generate unsigned refund transaction (backend)
	fmt.Println("🔧 Backend: Generating unsigned refund transaction...")
	unsignedRefundResp, err := client.GenerateUnsignedRefund(refundParams)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Unsigned refund transaction generated\n")
	fmt.Printf("   Transaction ID: %s\n\n", unsignedRefundResp.TransactionID)

	// Sign transaction (frontend - simulated with User1's key)
	fmt.Println("🔐 Frontend: Signing transaction with User1's private key...")
	signedRefundTx, err := signTransactionDemo(unsignedRefundResp.UnsignedTransaction, User1PrivateKey)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Transaction signed\n\n")

	// Submit signed refund transaction (backend)
	fmt.Println("📤 Backend: Submitting signed refund transaction...")
	signedRefundReq := solprogram.SignedTransactionRequest{
		TransactionID:     unsignedRefundResp.TransactionID,
		SignedTransaction: signedRefundTx,
	}

	refundResult, err := client.SubmitSignedTransaction(signedRefundReq)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Envelope refunded successfully!\n")
	fmt.Printf("   Signature: %s\n", refundResult.Signature)
	fmt.Printf("   Explorer: %s\n\n", refundResult.ExplorerURL)

	// ========================================
	// SUMMARY
	// ========================================
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  ✅ COMPLETE FLOW FINISHED SUCCESSFULLY!                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("\n📊 Flow Summary:")
	fmt.Printf("   1. ✅ Created GroupFixed Envelope #%d (3 USDC total, 3 users max, 60s expiry)\n", envelopeID)
	fmt.Printf("   2. ✅ Waited 3 seconds for confirmation\n")
	fmt.Printf("   3. ✅ Claimed by User2 (received 1 USDC, 1/3 share)\n")
	fmt.Printf("   4. ✅ Waited 60 seconds for expiry\n")
	fmt.Printf("   5. ✅ Refunded remaining 2 USDC (2/3 unclaimed) to User1\n")
	fmt.Println("\n🔗 Transaction Links:")
	fmt.Printf("   Create:  %s\n", createResult.ExplorerURL)
	fmt.Printf("   Claim:   %s\n", claimResult.ExplorerURL)
	fmt.Printf("   Refund:  %s\n", refundResult.ExplorerURL)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// signTransactionDemo - Helper function to simulate frontend signing (FOR DEMO ONLY)
// In production, this ONLY happens on frontend with user's wallet, NEVER on backend!
func signTransactionDemo(unsignedTxBase64 string, privateKey solana.PrivateKey) (string, error) {
	txBytes, err := base64.StdEncoding.DecodeString(unsignedTxBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode transaction: %w", err)
	}

	decoder := bin.NewBinDecoder(txBytes)
	var tx solana.Transaction
	if err := tx.UnmarshalWithDecoder(decoder); err != nil {
		return "", fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if privateKey.PublicKey().Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	signedBytes, err := tx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to marshal signed transaction: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signedBytes), nil
}
