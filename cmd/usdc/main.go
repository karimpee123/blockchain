package main

import (
	"blockchain/solprogram"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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
		runInitUserState   = true  // Initialize user state (required first time)
		runGroupFixed      = false // Create GroupFixed envelope (multiple claimers)
		runDirectFixed     = true  // Create DirectFixed envelope (single claimer)
		runGetEnvelopeInfo = true  // Get envelope info (requires envelope created)
		runClaim           = false // Claim envelope (requires User2 ATA + envelope exists)
		runWaitAndRefund   = true  // Wait for expiry then refund
		runCheckTxStatus   = true  // Check transaction status
	)

	// Setup
	ctx := context.Background()

	// Create client (Devnet)
	client, err := solprogram.NewUSDCEnvelopeClient(
		solprogram.RPCURLDevnet,
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
