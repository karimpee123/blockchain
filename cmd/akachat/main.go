package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// EnvelopeInfo stores envelope metadata for tracking claims
type EnvelopeInfo struct {
	EnvType        string
	Token          string
	AmountPerClaim int64
	TotalClaims    int
}

var envelopeRegistry = make(map[int64]EnvelopeInfo)

func createTransfer(payload PayloadTransferCreate, from User, flag bool) (envID int64) {
	if !flag {
		log.Println("Skipping creation of tranfer")
		return
	}
	fmt.Println("#============ CREATE TRANSFER START ============#")
	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/transfer/request_unsigned_create", payload, from.Token)
	if err != nil {
		log.Fatal(err)
		return
	}
	if unsignedResp.ErrCode > 0 {
		log.Fatal("Failed to create transfer: ", unsignedResp.ErrMsg)
	}
	signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, userA.PrivateKey)
	if err != nil {
		log.Fatal(err)
		return
	}
	payloadSignedTx := PayloadSignedTx{
		RawTransaction: *signedTx,
		TxHash:         "",
		Chain:          "solana",
		CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
		Action:         "create",
	}
	signedResp, err := doPost[SignedTxResult](baseURL+"/v2/transfer/process_signed_transaction", payloadSignedTx, from.Token)
	if err != nil {
		log.Fatal(err)
		return
	}
	if signedResp.ErrCode != 0 {
		log.Printf("Business logic error: %d - %s\n", signedResp.ErrCode, signedResp.ErrMsg)
		return
	}
	fmt.Println("Create Transfer TX Hash:", signedResp.Data.TxHash)
	fmt.Println("Transfer ID:", signedResp.Data.TransferID)
	fmt.Printf("%+v\n", signedResp)
	fmt.Println("#============ CREATE Transfer DONE ============#")

	return
}

func claimTransfer(payload PayloadTransferClaim, claimer User, flag bool) {
	if !flag {
		log.Println("Skipping claiming transfer")
		return
	}
	fmt.Println("#============ CLAIM Transfer START ============#")
	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/transfer/request_unsigned_claim", payload, claimer.Token)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("%+v\n", unsignedResp)
	//signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, claimer.PrivateKey)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//payloadSignedTx := PayloadSignedTx{
	//	RawTransaction: *signedTx,
	//	TxHash:         "",
	//	Chain:          "solana",
	//	CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
	//	Action:         "claim",
	//}
	//signedResp, err := doPost[SignedTxResult](baseURL+"/v2/transfer/process_signed_transaction", payloadSignedTx, claimer.Token)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//if signedResp.ErrCode != 0 {
	//	log.Printf("Business logic error: %d - %s\n", signedResp.ErrCode, signedResp.ErrMsg)
	//	return
	//}
	//fmt.Println("Claim TX Hash:", signedResp.Data.TxHash)
	//fmt.Printf("%+v\n", signedResp)
	//fmt.Println("#============ CLAIM Transfer DONE ============#")
}

func createEnvelope(payload PayloadCreate, from User) (envID int64) {
	// Get balance before create
	balanceBefore, err := getBalance(from.Address)
	if err != nil {
		log.Printf("⚠️ Failed to get balance before: %v\n", err)
		balanceBefore = 0
	}

	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/envelope/request_unsigned_create", payload, from.Token)
		if err != nil {
			log.Printf("❌ API Error (attempt %d/%d): %v\n", attempt, maxRetries, err)
			if attempt < maxRetries {
				waitTime := time.Duration(attempt) * time.Second
				fmt.Printf("  ⏳ Retrying in %d seconds...\n", attempt)
				time.Sleep(waitTime)
				continue
			}
			return
		}
		if unsignedResp.ErrCode > 0 {
			log.Printf("❌ Create Failed [Code:%d] (attempt %d/%d): %s\n", unsignedResp.ErrCode, attempt, maxRetries, unsignedResp.ErrMsg)
			if attempt < maxRetries {
				waitTime := time.Duration(attempt) * time.Second
				fmt.Printf("  ⏳ Retrying in %d seconds...\n", attempt)
				time.Sleep(waitTime)
				continue
			}
			return
		}

		signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, from.PrivateKey)
		if err != nil {
			log.Printf("❌ Sign Error: %v\n", err)
			return
		}

		payloadSignedTx := PayloadSignedTx{
			RawTransaction: *signedTx,
			TxHash:         "",
			Chain:          network.Name,
			CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
			Action:         ActionCreate,
		}
		signedResp, err := doPost[SignedTxResult](baseURL+"/v2/envelope/process_signed_transaction", payloadSignedTx, from.Token)
		if err != nil {
			log.Printf("❌ Process Tx Error (attempt %d/%d): %v\n", attempt, maxRetries, err)
			if attempt < maxRetries {
				waitTime := time.Duration(attempt) * time.Second
				fmt.Printf("  ⏳ Retrying in %d seconds...\n", attempt)
				time.Sleep(waitTime)
				continue
			}
			return
		}
		if signedResp.ErrCode != 0 {
			// Check if it's a seeds constraint error
			isSeedError := false
			if signedResp.ErrMsg != "" {
				// Check for "seeds constraint" or "2006" error
				isSeedError = containsIgnoreCase(signedResp.ErrMsg, "seeds constraint") ||
					containsIgnoreCase(signedResp.ErrMsg, "2006")
			}

			if isSeedError && attempt < maxRetries {
				log.Printf("❌ Seeds Constraint Error (attempt %d/%d): Backend envelope ID counter not synced\n", attempt, maxRetries)
				// Exponentially increase wait time for seed errors
				// Attempt 1: wait 5s, Attempt 2: wait 10s, Attempt 3: wait 15s, etc.
				waitTime := time.Duration(attempt*5) * time.Second
				fmt.Printf("  ⏳ Waiting %d seconds for backend to sync...\n", attempt*5)
				time.Sleep(waitTime)
				continue
			}

			log.Printf("❌ CREATE ERROR [Code:%d]: %s\n", signedResp.ErrCode, signedResp.ErrMsg)
			return
		}

		// Success! Print create info
		envID = signedResp.Data.EnvelopeID
		var decimals int64 = 1_000_000 // Default for SPL tokens (6 decimals)
		if payload.Token == "SOL" {
			decimals = 1_000_000_000 // 9 decimals for SOL
		}
		// Store envelope info for later claim tracking
		envelopeRegistry[envID] = EnvelopeInfo{
			EnvType:        payload.EnvelopeType,
			Token:          payload.Token,
			AmountPerClaim: int64(payload.AmountPerClaimOrPot),
			TotalClaims:    payload.TotalClaims,
		}

		// Determine token mint for SPL tokens
		var tokenMint string
		if payload.Token == "USDC" {
			tokenMint = USDCMintDevnet
		}

		// Get transaction details from metadata (more accurate than polling RPC)
		fee, rentExempt, txBalanceBefore, txBalanceAfter, tokenBalanceBefore, tokenBalanceAfter, err := getTransactionDetails(signedResp.Data.TxHash, from.Address, tokenMint)
		if err != nil {
			log.Printf("⚠️ Failed to get transaction details: %v\n", err)
			fee = 0
			rentExempt = 0
			txBalanceBefore = balanceBefore
			txBalanceAfter = balanceBefore // fallback
			tokenBalanceBefore = 0
			tokenBalanceAfter = 0
		}

		// Use transaction metadata balances (more accurate)
		if txBalanceBefore > 0 {
			balanceBefore = txBalanceBefore
			balanceAfter := txBalanceAfter

			// Print detailed create information
			fmt.Println("\nCreate")
			balanceBeforeSol := float64(balanceBefore) / 1e9
			fmt.Printf("- User : Owner=%s, balance before create envelope = %.9f SOL", from.Address, balanceBeforeSol)

			// Show token balance for SPL tokens
			if payload.Token != "SOL" && tokenBalanceBefore > 0 {
				fmt.Printf(", %s balance = %.6f %s", payload.Token, tokenBalanceBefore, payload.Token)
			}
			fmt.Println()

			amountDisplay := float64(payload.AmountPerClaimOrPot) / float64(decimals)
			totalDisplay := float64(payload.Value) / float64(decimals)
			fmt.Printf("- Envelope:  Type=%s, Token=%s, Amount=%.2f %s, claimer=%d, total=%.2f EnvID=%d,  TxHash: %s\n",
				payload.EnvelopeType, payload.Token, amountDisplay, payload.Token, payload.TotalClaims, totalDisplay, envID, signedResp.Data.TxHash)

			balanceAfterSol := float64(balanceAfter) / 1e9
			feeSol := float64(fee) / 1e9
			rentExemptSol := float64(rentExempt) / 1e9
			fmt.Printf("- Balance owner after create envelope = %.9f SOL, Fee = %.9f SOL, rent-exempt = %.9f SOL", balanceAfterSol, feeSol, rentExemptSol)

			// Show token balance after for SPL tokens
			if payload.Token != "SOL" && tokenBalanceAfter > 0 {
				fmt.Printf(", %s balance after = %.6f %s", payload.Token, tokenBalanceAfter, payload.Token)
			}
			fmt.Println()
		} else {
			// Fallback to old method
			balanceAfter, _ := getBalance(from.Address)
			fmt.Println("\nCreate")
			balanceBeforeSol := float64(balanceBefore) / 1e9
			fmt.Printf("- User : Owner=%s, balance before create envelope = %.9f SOL\n", from.Address, balanceBeforeSol)

			amountDisplay := float64(payload.AmountPerClaimOrPot) / float64(decimals)
			totalDisplay := float64(payload.Value) / float64(decimals)
			fmt.Printf("- Envelope:  Type=%s, Token=%s, Amount=%.2f %s, claimer=%d, total=%.2f EnvID=%d,  TxHash: %s\n",
				payload.EnvelopeType, payload.Token, amountDisplay, payload.Token, payload.TotalClaims, totalDisplay, envID, signedResp.Data.TxHash)

			balanceAfterSol := float64(balanceAfter) / 1e9
			feeSol := float64(fee) / 1e9
			rentExemptSol := float64(rentExempt) / 1e9
			fmt.Printf("- Balance owner after create envelope = %.9f SOL, Fee = %.9f SOL, rent-exempt = %.9f SOL\n",
				balanceAfterSol, feeSol, rentExemptSol)
		}

		return
	}

	log.Printf("❌ Failed to create envelope after %d attempts\n", maxRetries)
	return
}

func claimEnvelope(payload PayloadClaim, claimer User) {
	// Get balance before claim
	balanceBefore, err := getBalance(claimer.Address)
	if err != nil {
		log.Printf("⚠️ Failed to get balance before: %v\n", err)
		balanceBefore = 0
	}

	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/envelope/request_unsigned_claim", payload, claimer.Token)
	if err != nil {
		log.Printf("❌ Claim API Error: %v\n", err)
		return
	}
	if unsignedResp.ErrCode > 0 {
		log.Printf("❌ Claim Failed [Code:%d]: %s\n", unsignedResp.ErrCode, unsignedResp.ErrMsg)
		return
	}

	signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, claimer.PrivateKey)
	if err != nil {
		log.Printf("❌ Claim Sign Error: %v\n", err)
		return
	}

	payloadSignedTx := PayloadSignedTx{
		RawTransaction: *signedTx,
		TxHash:         "",
		Chain:          network.Name,
		CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
		Action:         ActionClaim,
	}
	signedResp, err := doPost[SignedTxResult](baseURL+"/v2/envelope/process_signed_transaction", payloadSignedTx, claimer.Token)
	if err != nil {
		log.Printf("❌ Claim Process Error: %v\n", err)
		return
	}
	if signedResp.ErrCode != 0 {
		log.Printf("❌ CLAIM ERROR [Code:%d]: %s\n", signedResp.ErrCode, signedResp.ErrMsg)
		return
	}

	// Get envelope info for amount display
	envInfo, hasInfo := envelopeRegistry[int64(payload.EnvelopeID)]
	if !hasInfo {
		fmt.Printf("- Claimed: Claimer=%s, EnvID=%d\n", claimer.Address, payload.EnvelopeID)
		fmt.Printf("  TxHash: %s\n", signedResp.Data.TxHash)
		return
	}

	// Determine decimals based on token
	var decimals int64 = 1_000_000
	if envInfo.Token == "SOL" {
		decimals = 1_000_000_000
	}

	// Determine token mint for SPL tokens
	var tokenMint string
	if envInfo.Token == "USDC" {
		tokenMint = USDCMintDevnet
	}

	// Get transaction details from metadata
	fee, rentExempt, txBalanceBefore, txBalanceAfter, tokenBalanceBefore, tokenBalanceAfter, txErr := getTransactionDetails(signedResp.Data.TxHash, claimer.Address, tokenMint)
	if txErr != nil {
		log.Printf("⚠️ Failed to get transaction details: %v\n", txErr)
		fee = 0
		rentExempt = 0
		txBalanceBefore = balanceBefore
		txBalanceAfter = balanceBefore
		tokenBalanceBefore = 0
		tokenBalanceAfter = 0
	}

	// Use transaction metadata balances
	if txBalanceBefore > 0 {
		balanceBefore = txBalanceBefore
	}
	balanceAfter := txBalanceAfter
	if balanceAfter == 0 {
		balanceAfter, _ = getBalance(claimer.Address)
	}

	// Print detailed claim information
	fmt.Println("\nClaim")
	balanceBeforeSol := float64(balanceBefore) / 1e9
	fmt.Printf("- Claimer=%s,  balance before claim envelope = %.9f SOL", claimer.Address, balanceBeforeSol)

	// Show token balance for SPL tokens
	if envInfo.Token != "SOL" && tokenBalanceBefore > 0 {
		fmt.Printf(", %s balance = %.6f %s", envInfo.Token, tokenBalanceBefore, envInfo.Token)
	}
	fmt.Println()

	// For fixed or single envelope, we know the exact amount
	if envInfo.EnvType == "fixed" || envInfo.EnvType == "single" {
		amount := float64(envInfo.AmountPerClaim) / float64(decimals)
		// Format with appropriate decimals (strip trailing zeros visually)
		amountStr := fmt.Sprintf("%.9f", amount)
		// Trim trailing zeros and decimal point if needed
		amountStr = strings.TrimRight(strings.TrimRight(amountStr, "0"), ".")
		fmt.Printf("- envelope: amount=%s %s from EnvID=%d,  TxHash: %s\n",
			amountStr, envInfo.Token, payload.EnvelopeID, signedResp.Data.TxHash)

		balanceAfterSol := float64(balanceAfter) / 1e9
		feeSol := float64(fee) / 1e9
		rentExemptSol := float64(rentExempt) / 1e9
		fmt.Printf("- balance claimer after claim envelope = %.9f SOL, fee = %.9f SOL, rent-exempt = %.9f SOL", balanceAfterSol, feeSol, rentExemptSol)

		// Show token balance after for SPL tokens
		if envInfo.Token != "SOL" && tokenBalanceAfter > 0 {
			fmt.Printf(", %s balance after = %.6f %s", envInfo.Token, tokenBalanceAfter, envInfo.Token)
		}
		fmt.Println()
	} else {
		// For lucky envelope, wait for confirmation to get actual amount
		fmt.Print("   Waiting for confirmation...")
		claimAmount, err := waitForConfirmation(signedResp.Data.TxHash, envInfo.Token)
		if err != nil || claimAmount == 0 {
			fmt.Printf(" ⚠️\n")
			totalPot := float64(envInfo.AmountPerClaim) / float64(decimals)
			fmt.Printf("- envelope: amount=random from pot %.2f %s, EnvID=%d,  TxHash: %s\n",
				totalPot, envInfo.Token, payload.EnvelopeID, signedResp.Data.TxHash)
		} else if claimAmount == -1 {
			// Transaction confirmed but amount indeterminate
			fmt.Printf(" ✓\n")
			totalPot := float64(envInfo.AmountPerClaim) / float64(decimals)
			fmt.Printf("- envelope: amount=??? (random from pot %.2f %s), EnvID=%d,  TxHash: %s\n",
				totalPot, envInfo.Token, payload.EnvelopeID, signedResp.Data.TxHash)
		} else {
			fmt.Printf(" OK\n")
			fmt.Printf("- envelope: amount=%.4f %s from EnvID=%d,  TxHash: %s\n",
				claimAmount, envInfo.Token, payload.EnvelopeID, signedResp.Data.TxHash)
		}

		balanceAfterSol := float64(balanceAfter) / 1e9
		feeSol := float64(fee) / 1e9
		rentExemptSol := float64(rentExempt) / 1e9
		fmt.Printf("- balance claimer after claim envelope = %.9f SOL, fee = %.9f SOL, rent-exempt = %.9f SOL", balanceAfterSol, feeSol, rentExemptSol)

		// Show token balance after for SPL tokens
		if envInfo.Token != "SOL" && tokenBalanceAfter > 0 {
			fmt.Printf(", %s balance after = %.6f %s", envInfo.Token, tokenBalanceAfter, envInfo.Token)
		}
		fmt.Println()
	}
}

func refundEnvelope(payload PayloadRefund, owner User) {
	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/envelope/request_unsigned_refund", payload, owner.Token)
	if err != nil {
		log.Printf("❌ Refund API Error: %v", err)
		return
	}
	if unsignedResp.ErrCode != 0 {
		log.Printf("❌ Refund Failed [Code:%d]: %s\n", unsignedResp.ErrCode, unsignedResp.ErrMsg)
		return
	}

	signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, owner.PrivateKey)
	if err != nil {
		log.Printf("❌ Refund Sign Error: %v", err)
		return
	}

	payloadSignedTx := PayloadSignedTx{
		RawTransaction: *signedTx,
		TxHash:         "",
		Chain:          network.Name,
		CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
		Action:         ActionRefund,
	}
	signedResp, err := doPost[SignedTxResult](baseURL+"/v2/envelope/process_signed_transaction", payloadSignedTx, owner.Token)
	if err != nil {
		log.Printf("❌ Refund Process Error: %v", err)
		return
	}
	if signedResp.ErrCode != 0 {
		log.Printf("❌ REFUND ERROR [Code:%d]: %s\n", signedResp.ErrCode, signedResp.ErrMsg)
		return
	}

	fmt.Printf("✅ Refunded EnvID=%d TxHash=%s\n", payload.EnvelopeID, signedResp.Data.TxHash)
}

func transfer() {
	amount := 1_000_000
	value := 1_000_000
	createFlag := false
	payloadCreate := PayloadTransferCreate{
		Token:    "SOL",
		Amount:   amount,
		Value:    value,
		Chain:    "solana",
		Remarks:  "waktu setempat",
		ToUserID: userB.ID,
		Expiry:   24,
	}
	_ = createTransfer(payloadCreate, userA, createFlag)

	transferID := 18
	claimFlag := true
	claimUser := userB
	payloadClaim := PayloadTransferClaim{
		Chain:      "solana",
		TransferID: transferID,
	}
	claimTransfer(payloadClaim, claimUser, claimFlag)
}

// createEnvelopeTyped creates envelope with specific token and type
func createEnvelopeTyped(token string, envType string, totalClaims int, amountPerClaim int64) int64 {
	var totalValue int64
	if envType == "lucky" {
		// For lucky type, amountPerClaim is actually the total pot
		totalValue = amountPerClaim
	} else {
		totalValue = amountPerClaim * int64(totalClaims)
	}

	payloadCreate := PayloadCreate{
		EnvelopeType:        envType,
		Token:               token,
		TotalClaims:         totalClaims,
		AmountPerClaimOrPot: int(amountPerClaim),
		Value:               int(totalValue),
		Chain:               "solana",
		GroupID:             "441250605",
		Remarks:             fmt.Sprintf("Test %s %s", token, envType),
		ThemeID:             1,
		ToUserID:            userB.ID,
		UserID:              userA.ID,
	}
	return createEnvelope(payloadCreate, userA)
}

// SOL test functions (9 decimals, 0.1 SOL = 100_000_000)
func createSOLFixed() int64 {
	return createEnvelopeTyped("SOL", "fixed", 3, 10_000_000)
}

func createSOLLucky() int64 {
	return createEnvelopeTyped("SOL", "lucky", 3, 300_000_000)
}

func createSOLSingle() int64 {
	return createEnvelopeTyped("SOL", "single", 1, 100_000_000)
}

// USDC test functions (6 decimals, 0.1 USDC = 100_000)
func createUSDCFixed() int64 {
	return createEnvelopeTyped("USDC", "fixed", 3, 100_000)
}

func createUSDCLucky() int64 {
	return createEnvelopeTyped("USDC", "lucky", 3, 300_000)
}

func createUSDCSingle() int64 {
	return createEnvelopeTyped("USDC", "single", 1, 100_000)
}

func claim(envelopeID int64) {
	claimUser := userB
	payloadClaim := PayloadClaim{
		Chain:          network.Name,
		UserID:         claimUser.ID,
		GroupID:        "123",
		EnvelopeID:     int(envelopeID),
		ConversationID: "123",
		Seq:            123,
		Status:         "",
	}
	claimEnvelope(payloadClaim, claimUser)
}

func claimWithUser(envelopeID int64, claimer User) {
	payloadClaim := PayloadClaim{
		Chain:          network.Name,
		UserID:         claimer.ID,
		GroupID:        "123",
		EnvelopeID:     int(envelopeID),
		ConversationID: "123",
		Seq:            123,
		Status:         "",
	}
	claimEnvelope(payloadClaim, claimer)
}

func refund(envID, envChainID int64) {
	refundUser := userA
	payloadClaim := PayloadRefund{
		UserID:          refundUser.ID,
		EnvelopeID:      int(envID),
		EnvelopeChainID: int(envChainID),
		Chain:           network.Name,
		AddressUser:     refundUser.Address,
	}
	refundEnvelope(payloadClaim, refundUser)
}

func main() {

	native_single := false
	native_fixed := false
	native_lucky := false

	token_single := false
	token_fixed := false
	token_lucky := true
	// Initialize for Solana network
	initAll(NetSOL)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("ENVELOPE TEST SUITE - 6 Tests (SOL & USDC)")
	fmt.Println(strings.Repeat("=", 50))

	// ========== SOL TESTS (3) ==========

	// TEST 1: SOL Fixed
	if native_fixed {
		fmt.Println("\n[TEST 1/6] SOL - Fixed (3 claims)")
		fmt.Println(strings.Repeat("-", 50))
		envIDSOLFixed := createSOLFixed()
		if envIDSOLFixed > 0 {
			time.Sleep(2 * time.Second)
			claimWithUser(envIDSOLFixed, userB)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDSOLFixed, userC)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDSOLFixed, userD)
		}
	}

	// TEST 2: SOL Lucky
	if native_lucky {
		fmt.Println("\n[TEST 2/6] SOL - Lucky (3 claims, random)")
		fmt.Println(strings.Repeat("-", 50))
		envIDSOLLucky := createSOLLucky()
		if envIDSOLLucky > 0 {
			time.Sleep(2 * time.Second)
			claimWithUser(envIDSOLLucky, userB)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDSOLLucky, userC)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDSOLLucky, userD)
		}
	}

	// TEST 3: SOL Single
	if native_single {
		fmt.Println("\n[TEST 3/6] SOL - Single (direct)")
		fmt.Println(strings.Repeat("-", 50))
		envIDSOLSingle := createSOLSingle()
		if envIDSOLSingle > 0 {
			time.Sleep(2 * time.Second)
			claim(envIDSOLSingle)
		}
	}

	// ========== USDC TESTS (3) ==========

	// TEST 4: USDC Fixed
	if token_fixed {
		fmt.Println("\n[TEST 4/6] USDC - Fixed (3 claims)")
		fmt.Println(strings.Repeat("-", 50))
		envIDUSDCFixed := createUSDCFixed()
		if envIDUSDCFixed > 0 {
			time.Sleep(2 * time.Second)
			claimWithUser(envIDUSDCFixed, userB)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDUSDCFixed, userC)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDUSDCFixed, userD)
		}
	}

	// TEST 5: USDC Lucky
	if token_lucky {
		fmt.Println("\n[TEST 5/6] USDC - Lucky (3 claims, random)")
		fmt.Println(strings.Repeat("-", 50))
		envIDUSDCLucky := createUSDCLucky()
		if envIDUSDCLucky > 0 {
			time.Sleep(2 * time.Second)
			claimWithUser(envIDUSDCLucky, userB)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDUSDCLucky, userC)
			time.Sleep(1 * time.Second)
			claimWithUser(envIDUSDCLucky, userD)
		}
	}

	// TEST 6: USDC Single
	if token_single {
		fmt.Println("\n[TEST 6/6] USDC - Single (direct)")
		fmt.Println(strings.Repeat("-", 50))
		envIDUSDCSingle := createUSDCSingle()
		if envIDUSDCSingle > 0 {
			time.Sleep(2 * time.Second)
			claim(envIDUSDCSingle)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("ALL TESTS COMPLETED")
	fmt.Println(strings.Repeat("=", 50))
}
