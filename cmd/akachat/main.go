package main

import (
	"fmt"
	"log"
)

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
	fmt.Println("#============ CREATE ENVELOPE START ============#")
	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/envelope/request_unsigned_create", payload, from.Token)
	if err != nil {
		log.Fatal(err)
		return
	}
	if unsignedResp.ErrCode > 0 {
		log.Fatal("Failed to create envelope: ", unsignedResp.ErrMsg)
	}
	signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, userA.PrivateKey)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
		return
	}
	if signedResp.ErrCode != 0 {
		log.Printf("Business logic error: %d - %s\n", signedResp.ErrCode, signedResp.ErrMsg)
		return
	}
	fmt.Println("Create Envelope TX Hash:", signedResp.Data.TxHash)
	fmt.Println("Envelope ID:", signedResp.Data.EnvelopeID)
	fmt.Printf("%+v\n", signedResp)
	fmt.Println("#============ CREATE ENVELOPE DONE ============#")

	envID = signedResp.Data.EnvelopeID
	return
}

func claimEnvelope(payload PayloadClaim, claimer User) {
	fmt.Println("#============ CLAIM ENVELOPE START ============#")
	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/envelope/request_unsigned_claim", payload, claimer.Token)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("%+v\n", payload)
	signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, claimer.PrivateKey)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
		return
	}
	if signedResp.ErrCode != 0 {
		log.Printf("Business logic error: %d - %s\n", signedResp.ErrCode, signedResp.ErrMsg)
		return
	}
	fmt.Println("Claim TX Hash:", signedResp.Data.TxHash)
	fmt.Printf("%+v\n", signedResp)
	fmt.Println("#============ CLAIM ENVELOPE DONE ============#")
}

func refundEnvelope(payload PayloadRefund, owner User) {
	fmt.Println("#============ Refund ENVELOPE START ============#")
	unsignedResp, err := doPost[UnsignedTxData](baseURL+"/v2/envelope/request_unsigned_refund", payload, owner.Token)
	if err != nil {
		log.Fatal(err)
		return
	}
	if unsignedResp.ErrCode != 0 {
		fmt.Printf("Business logic error: %d - %s\n", unsignedResp.ErrCode, unsignedResp.ErrMsg)
	}
	signedTx, err := clientSign(unsignedResp.Data.UnsignedTx.Data, owner.PrivateKey)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
		return
	}
	if signedResp.ErrCode != 0 {
		log.Printf("Business logic error: %d - %s\n", signedResp.ErrCode, signedResp.ErrMsg)
		return
	}
	fmt.Println("Refund TX Hash:", signedResp.Data.TxHash)
	fmt.Printf("%+v\n", signedResp)
	fmt.Println("#============ Refund ENVELOPE DONE ============#")
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

func create() int64 {
	claimer := 2
	envType := "fixed"
	amount := 1_000_000_000
	value := 2_000_000_000
	payloadCreate := PayloadCreate{
		EnvelopeType:        envType,
		Token:               network.Symbol,
		TotalClaims:         claimer,
		AmountPerClaimOrPot: amount,
		Value:               value,
		Chain:               network.Name,
		GroupID:             "441250605",
		Remarks:             "mulai dari awal",
		ThemeID:             1,
		ToUserID:            userB.ID,
		UserID:              userA.ID,
	}
	return createEnvelope(payloadCreate, userA)
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
	GetSignature("2gbXPrAFfuUA3u3bkHxDfoVTjp6CwcB6qom2njbmqJc5EhyMxdqmLLw6N1jHE15W4w62FNofgirbn8tMvfKdsX7j")
	//initAll(NetSOL)
	//envID := create()
	//fmt.Println(envID)
	//time.Sleep(2 * time.Second)
	//claim(envID)
	//time.Sleep(60 * time.Second)

	//envID := int64(223)
	//refund(envID, envID+17)
}
