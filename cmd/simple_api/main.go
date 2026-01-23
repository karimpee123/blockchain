package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gagliardetto/solana-go/rpc"

	"blockchain/chainbnb"
	"blockchain/chainsol"
)

func main() {
	// Initialize Sol client
	solChain := chainsol.NewSolChain(chainsol.Config{
		RPCURL:  rpc.DevNet_RPC,
		WSURL:   rpc.DevNet_WS,
		Network: rpc.DevNet.Name,
	})

	// Initialize BNB Chain client
	bnbChain := chainbnb.NewBNBChain(chainbnb.Config{
		RPCURL:  "https://data-seed-prebsc-1-s1.binance.org:8545/",
		ChainID: 97,
		Network: "testnet",
	})

	// Health checks
	if err := solChain.HealthCheck(); err != nil {
		log.Fatalf("❌ Solana health check failed: %v", err)
	}
	if err := bnbChain.HealthCheck(); err != nil {
		log.Fatalf("❌ BNB Chain health check failed: %v", err)
	}

	// Solana routes
	http.HandleFunc("/api/v1/sol/transaction/create", solChain.HandleCreateTransaction)
	http.HandleFunc("/api/v1/sol/transaction/sign", solChain.HandleSignTransaction)
	http.HandleFunc("/api/v1/sol/transaction/send", solChain.HandleSendTransaction)
	http.HandleFunc("/api/v1/sol/transaction/status", solChain.HandleGetTransactionStatus)
	http.HandleFunc("/api/v1/sol/transaction/history", solChain.HandleGetTransactionHistory)

	// BNB routes
	http.HandleFunc("/api/v1/bnb/transaction/create", bnbChain.HandleCreateTransaction)
	http.HandleFunc("/api/v1/bnb/transaction/sign", bnbChain.HandleSignTransaction)
	http.HandleFunc("/api/v1/bnb/transaction/send", bnbChain.HandleSendTransaction)
	http.HandleFunc("/api/v1/bnb/transaction/status", bnbChain.HandleGetTransactionStatus)
	http.HandleFunc("/api/v1/bnb/transaction/history", bnbChain.HandleGetTransactionHistory)

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Simple API Server starting on port %s", port)
	log.Printf("✅ Solana DevNet connected")
	log.Printf("✅ BNB Testnet connected")
	log.Printf("📡 Endpoints:")
	log.Printf("   - SOL: /api/v1/sol/*")
	log.Printf("   - BNB: /api/v1/bnb/*")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
