package solprogram

import "github.com/gagliardetto/solana-go"

// Program IDs
const (
	// USDC Program ID (dari declare_id di program Solana)
	USDCProgramID = "5DXoYSQxaJzQ1W4LqSq2nWZ12PvFsb4FHo4xWgSrchVH"

	// SOL Program ID (dari declare_id di program Solana)
	SOLProgramID = "8sVfWmonJAzAQnS4nYcxv3GBSs4rDpvmniRrApwrh1QK"

	// USDC Mint Address (Devnet)
	USDCMintDevnet = "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"

	// USDC Mint Address (Mainnet)
	USDCMintMainnet = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

// PDA Seeds
var (
	SeedUserState     = []byte("user_state")
	SeedEnvelope      = []byte("envelope")
	SeedEnvelopeVault = []byte("envelope_vault")
	SeedClaim         = []byte("claim")
)

// Limits
const (
	// Max amount per envelope: 100 USDC
	MaxCreateAmountUSDC = 100_000_000 // 100 USDC (6 decimals)

	// Min amount per user: 0.01 USDC
	MinAmountPerUserUSDC = 10_000 // 0.01 USDC

	// Max amount per envelope: 10 SOL
	MaxCreateAmountSOL = 10_000_000_000 // 10 SOL (9 decimals)

	// Min amount per user: 0.01 SOL
	MinAmountPerUserSOL = 10_000_000 // 0.01 SOL
)

// System Program IDs
var (
	SystemProgramID       = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	TokenProgramID        = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	AssociatedTokenProgID = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	SysVarRentID          = solana.MustPublicKeyFromBase58("SysvarRent111111111111111111111111111111111")
)

// Explorer URLs
const (
	ExplorerURLDevnet  = "https://explorer.solana.com/tx/%s?cluster=devnet"
	ExplorerURLMainnet = "https://explorer.solana.com/tx/%s"
)

// RPC URLs
const (
	RPCURLDevnet    = "https://api.devnet.solana.com"
	RPCURLMainnet   = "https://api.mainnet-beta.solana.com"
	RPCURLLocalhost = "http://localhost:8899"
)
