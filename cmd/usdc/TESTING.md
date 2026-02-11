# TESTING GUIDE

## Cara Pakai

1. **Edit main.go** - Set flag test yang mau dijalankan:
```go
const (
    runInitUserState    = true   // Set true untuk init user state
    runGroupFixed       = false  // Set true untuk test GroupFixed
    runDirectFixed      = false  // Set true untuk test DirectFixed
    runGetEnvelopeInfo  = false  
    runClaim            = false  
    runWaitAndRefund    = false  
    runCheckTxStatus    = false  
)
```

2. **Build & Run**:
```bash
go build ./cmd/usdc/... && ./cmd/usdc/usdc
```

## Test Flow (Satu-Satu)

### Step 1: Initialize User State
```go
runInitUserState = true  // hanya ini yang true
```
Jalankan pertama kali untuk init user state.

### Step 2: Test GroupFixed
```go
runInitUserState = false
runGroupFixed    = true  // hanya ini yang true
```
Creates envelope dengan multiple claimers (User2, User3 bisa claim).

### Step 3: Test DirectFixed  
```go
runGroupFixed  = false
runDirectFixed = true  // hanya ini yang true
```
Creates envelope untuk single claimer (hanya User2 yang bisa).

### Step 4: Get Envelope Info
```go
runDirectFixed     = false
runGetEnvelopeInfo = true  // hanya ini yang true
```
Edit `demonstrateGetEnvelopeInfo()` untuk set envelope ID yang mau dicek.

### Step 5: Claim Envelope
**PENTING**: User2 harus punya USDC token account!
```bash
spl-token create-account 4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU \
  --owner 3YkzQC2PwFGvJr2GS7FDBopvG5tda4eXdq5pmwEbWeyd --url devnet
```

```go
runGetEnvelopeInfo = false
runClaim           = true  // hanya ini yang true
```
Edit `demonstrateClaim()` untuk set envelope ID yang mau di-claim.

### Step 6: Test Refund
```go
runClaim         = false
runWaitAndRefund = true  // hanya ini yang true
```
⚠️ Akan wait 60 detik untuk envelope expire!

## Tips

- **Jalankan SATU test per execution** untuk hindari conflict
- **Track envelope IDs** - increment setiap kali create
- Kalau ada error state, tunggu 3-5 detik lalu coba lagi
- User1 sudah init dengan Last Envelope ID: 1

## Envelope Types

**GroupFixed**: Multiple users, dibagi rata
**DirectFixed**: Single user, ada allowed_address
**GroupRandom**: Random amount (belum ada di demo)
