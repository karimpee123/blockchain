package main

import (
	"github.com/gagliardetto/solana-go/rpc"
	"log"
	"net/http"
	"os"

	"blockchain/solprogram"
)

func main() {
	programID := os.Getenv("PROGRAM_ID")
	if programID == "" {
		programID = "8sVfWmonJAzAQnS4nYcxv3GBSs4rDpvmniRrApwrh1QK"
	}

	client, err := solprogram.NewClient(rpc.DevNet_RPC, programID)
	if err != nil {
		log.Fatal(err)
	}

	// Routes
	http.HandleFunc("/api/create-envelope", client.HandleCreateEnvelope)
	http.HandleFunc("/api/claim-envelope", client.HandleClaimEnvelope)
	http.HandleFunc("/api/refund-envelope", client.HandleRefundEnvelope)
	http.HandleFunc("/api/sign-transaction", client.HandleSignTransaction) // ⚠️ TESTING ONLY
	http.HandleFunc("/api/send-transaction", client.HandleSendTransaction)

	// Health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	port := "8081"
	log.Printf("🚀 SPL API running on :%s", port)
	log.Printf("📦 Program ID: %s", programID)
	log.Printf("📡 Endpoints:")
	log.Printf("   POST /api/create-envelope")
	log.Printf("   POST /api/claim-envelope")
	log.Printf("   POST /api/refund-envelope")
	log.Printf("   POST /api/sign-transaction   ⚠️  TESTING ONLY")
	log.Printf("   POST /api/send-transaction")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
