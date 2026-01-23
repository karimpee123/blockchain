package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gagliardetto/solana-go/rpc"

	src "test/chainsol"
)

func main() {
	// Initialize Solana client with Alchemy
	solChain := src.NewSolChain(src.Config{
		RPCURL:  rpc.DevNet_RPC,
		WSURL:   rpc.DevNet_WS,
		Network: rpc.DevNet.Name,
	})

	// Health check
	if err := solChain.HealthCheck(); err != nil {
		log.Fatalf("Solana health check failed: %v", err)
	}

	// Setup routes
	http.HandleFunc("/api/v1/transaction/create", solChain.HandleCreateTransaction)
	http.HandleFunc("/api/v1/transaction/sign", solChain.HandleSignTransaction)
	http.HandleFunc("/api/v1/transaction/send", solChain.HandleSendTransaction)
	http.HandleFunc("/api/v1/transaction/status", solChain.HandleGetTransactionStatus)
	http.HandleFunc("/api/v1/transaction/history", solChain.HandleGetTransactionHistory)

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
