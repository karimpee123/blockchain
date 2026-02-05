package main

import (
	"fmt"
	"log"
)

func createEnvelope(payload PayloadCreate, from User, flag bool) (envID int64) {
	if !flag {
		log.Println("Skipping creation of envelope")
		return
	}
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
		Chain:          "solana",
		CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
		Action:         "create",
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

func claimEnvelope(payload PayloadClaim, claimer User, flag bool) {
	if !flag {
		log.Println("Skipping claiming envelope")
		return
	}
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
		Chain:          "solana",
		CacheKey:       unsignedResp.Data.UnsignedTx.CacheKey,
		Action:         "claim",
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

func main() {
	//createEnvelope()
	initUser()
	claimer := 1
	envType := "single"
	amount := 1_000_000
	value := 1_000_000
	createFlag := true
	if claimer == 1 && envType == "fixed" {
		value = amount * claimer
	}
	payloadCreate := PayloadCreate{
		EnvelopeType:        envType,
		Token:               "SOL",
		TotalClaims:         claimer,
		AmountPerClaimOrPot: amount,
		Value:               value,
		Chain:               "solana",
		GroupID:             "441250605",
		Remarks:             "pakai script",
		ThemeID:             1,
		ToUserID:            userB.ID,
		UserID:              userA.ID,
	}
	_ = createEnvelope(payloadCreate, userA, createFlag)

	//envelopeID := int(envId - 1)
	envelopeID := 163
	claimFlag := false
	claimUser := userB
	payloadClaim := PayloadClaim{
		Chain:          "solana",
		UserID:         claimUser.ID,
		GroupID:        "123",
		EnvelopeID:     envelopeID,
		ConversationID: "123",
		Seq:            123,
		Status:         "",
	}
	claimEnvelope(payloadClaim, claimUser, claimFlag)
}
