# Solana USDC Envelope - Go Integration Guide

Complete guide untuk mengintegrasikan Solana USDC Envelope Program dengan backend Go.

## 📁 File Structure

```
blockchain/
├── solprogram/
│   ├── constants.go              # Program IDs & constants
│   ├── types.go                  # Data models
│   ├── usdc_client.go           # Client & PDA helpers
│   ├── usdc_instructions.go     # Instruction builders
│   ├── service.go               # Service methods
│   └── parser.go                # Data parsers
└── cmd/usdc/
    └── main.go                  # Example usage
```

## 🚀 Quick Start Example

```go
package main

import (
    "blockchain/solprogram"
    "context"
    "log"
)

func main() {
    ctx := context.Background()
    
    // 1. Create client
    client, err := solprogram.NewUSDCEnvelopeClient(
        solprogram.RPCURLDevnet,
        "devnet",
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. Init user (first time)
    privateKey := // your private key
    _, err = client.InitUserState(ctx, privateKey)
    
    // 3. Create envelope
    params := solprogram.CreateEnvelopeParams{
        EnvelopeType: solprogram.EnvelopeTypeData{
            Type: solprogram.EnvelopeTypeGroupFixed,
        },
        TotalAmount: 10_000_000,  // 10 USDC
        TotalUsers:  5,
        ExpiryHours: 24,
    }
    
    response, err := client.CreateEnvelope(ctx, privateKey, userTokenAcc, params)
    log.Printf("Created envelope %d: %s", response.EnvelopeID, response.Signature)
}
```

## 📚 Complete Documentation

Lihat [solprogram/README.md](../../solprogram/README.md) untuk dokumentasi lengkap.

## 🔥 Key Features

- ✅ Create 3 types of envelopes (DirectFixed, GroupFixed, GroupRandom)
- ✅ Claim from envelopes
- ✅ Refund after expiry
- ✅ Client-side signing support  
- ✅ Transaction status tracking
- ✅ PDA derivation helpers
- ✅ Account data parsers

## 💡 Production Checklist

- [ ] Store private keys securely (not in code!)
- [ ] Use environment variables for RPC URLs
- [ ] Implement proper error handling
- [ ] Add retry logic for RPC calls
- [ ] Monitor transaction confirmations
- [ ] Set up logging & monitoring
- [ ] Test on devnet before mainnet
- [ ] Audit smart contracts

## 🛠️ Run Demo

```bash
cd cmd/usdc
go run main.go
```
