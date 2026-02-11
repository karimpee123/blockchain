package solprogram

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// USDCEnvelopeClient - Client untuk interact dengan USDC envelope program
type USDCEnvelopeClient struct {
	rpcClient *rpc.Client
	wsClient  *ws.Client
	programID solana.PublicKey
	usdcMint  solana.PublicKey
	network   string // "devnet", "mainnet", "localhost"
}

// NewUSDCEnvelopeClient - Create new USDC envelope client
func NewUSDCEnvelopeClient(rpcURL string, wsURL string, network string) (*USDCEnvelopeClient, error) {
	client := rpc.New(rpcURL)

	// Connect to WebSocket for transaction confirmation
	wsClient, err := ws.Connect(context.Background(), wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	programID, err := solana.PublicKeyFromBase58(USDCProgramID)
	if err != nil {
		return nil, fmt.Errorf("invalid program ID: %w", err)
	}

	// Select USDC mint based on network
	var usdcMintAddr string
	if network == "mainnet" {
		usdcMintAddr = USDCMintMainnet
	} else {
		usdcMintAddr = USDCMintDevnet
	}

	usdcMint, err := solana.PublicKeyFromBase58(usdcMintAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid USDC mint: %w", err)
	}

	return &USDCEnvelopeClient{
		rpcClient: client,
		wsClient:  wsClient,
		programID: programID,
		usdcMint:  usdcMint,
		network:   network,
	}, nil
}

// GetClient - Get RPC client
func (c *USDCEnvelopeClient) GetClient() *rpc.Client {
	return c.rpcClient
}

// GetProgramID - Get program ID
func (c *USDCEnvelopeClient) GetProgramID() solana.PublicKey {
	return c.programID
}

// GetUSDCMint - Get USDC mint address
func (c *USDCEnvelopeClient) GetUSDCMint() solana.PublicKey {
	return c.usdcMint
}

// DeriveUserStatePDA - Derive user state PDA
func (c *USDCEnvelopeClient) DeriveUserStatePDA(userPubkey solana.PublicKey) (solana.PublicKey, uint8, error) {
	pda, bump, err := solana.FindProgramAddress(
		[][]byte{
			SeedUserState,
			userPubkey.Bytes(),
		},
		c.programID,
	)
	if err != nil {
		return solana.PublicKey{}, 0, fmt.Errorf("failed to derive user state PDA: %w", err)
	}
	return pda, bump, nil
}

// DeriveEnvelopePDA - Derive envelope PDA
func (c *USDCEnvelopeClient) DeriveEnvelopePDA(owner solana.PublicKey, envelopeID uint64) (solana.PublicKey, uint8, error) {
	envelopeIDBytes := uint64ToBytes(envelopeID)

	pda, bump, err := solana.FindProgramAddress(
		[][]byte{
			SeedEnvelope,
			owner.Bytes(),
			envelopeIDBytes,
		},
		c.programID,
	)
	if err != nil {
		return solana.PublicKey{}, 0, fmt.Errorf("failed to derive envelope PDA: %w", err)
	}
	return pda, bump, nil
}

// DeriveEnvelopeVaultPDA - Derive envelope vault PDA (untuk hold USDC)
func (c *USDCEnvelopeClient) DeriveEnvelopeVaultPDA(owner solana.PublicKey, envelopeID uint64) (solana.PublicKey, uint8, error) {
	envelopeIDBytes := uint64ToBytes(envelopeID)

	pda, bump, err := solana.FindProgramAddress(
		[][]byte{
			SeedEnvelopeVault,
			owner.Bytes(),
			envelopeIDBytes,
		},
		c.programID,
	)
	if err != nil {
		return solana.PublicKey{}, 0, fmt.Errorf("failed to derive envelope vault PDA: %w", err)
	}
	return pda, bump, nil
}

// DeriveClaimRecordPDA - Derive claim record PDA
func (c *USDCEnvelopeClient) DeriveClaimRecordPDA(envelopePDA solana.PublicKey, claimer solana.PublicKey) (solana.PublicKey, uint8, error) {
	pda, bump, err := solana.FindProgramAddress(
		[][]byte{
			SeedClaim,
			envelopePDA.Bytes(),
			claimer.Bytes(),
		},
		c.programID,
	)
	if err != nil {
		return solana.PublicKey{}, 0, fmt.Errorf("failed to derive claim record PDA: %w", err)
	}
	return pda, bump, nil
}

// GetAssociatedTokenAddress - Derive Associated Token Account address for a wallet and mint
func (c *USDCEnvelopeClient) GetAssociatedTokenAddress(wallet solana.PublicKey, mint solana.PublicKey) (solana.PublicKey, error) {
	ata, _, err := solana.FindProgramAddress(
		[][]byte{
			wallet.Bytes(),
			TokenProgramID.Bytes(),
			mint.Bytes(),
		},
		AssociatedTokenProgID,
	)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to derive ATA: %w", err)
	}
	return ata, nil
}

// GetUSDCTokenAddress - Get USDC Associated Token Account for a wallet
func (c *USDCEnvelopeClient) GetUSDCTokenAddress(wallet solana.PublicKey) (solana.PublicKey, error) {
	return c.GetAssociatedTokenAddress(wallet, c.usdcMint)
}

// GetUserState - Fetch user state from blockchain
func (c *USDCEnvelopeClient) GetUserState(ctx context.Context, userPubkey solana.PublicKey) (*UserState, error) {
	userStatePDA, _, err := c.DeriveUserStatePDA(userPubkey)
	if err != nil {
		return nil, err
	}

	accountInfo, err := c.rpcClient.GetAccountInfo(ctx, userStatePDA)
	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	if accountInfo.Value == nil {
		return nil, fmt.Errorf("user state not found - need to initialize first")
	}

	// Parse account data
	userState, err := parseUserStateData(accountInfo.Value.Data.GetBinary())
	if err != nil {
		return nil, fmt.Errorf("failed to parse user state: %w", err)
	}

	return userState, nil
}

// GetEnvelopeInfo - Fetch envelope info from blockchain
func (c *USDCEnvelopeClient) GetEnvelopeInfo(ctx context.Context, owner solana.PublicKey, envelopeID uint64) (*EnvelopeInfo, error) {
	envelopePDA, _, err := c.DeriveEnvelopePDA(owner, envelopeID)
	if err != nil {
		return nil, err
	}

	accountInfo, err := c.rpcClient.GetAccountInfo(ctx, envelopePDA)
	if err != nil {
		return nil, fmt.Errorf("failed to get envelope info: %w", err)
	}

	if accountInfo.Value == nil {
		return nil, fmt.Errorf("envelope not found")
	}

	// Parse account data
	envelope, err := parseEnvelopeData(accountInfo.Value.Data.GetBinary())
	if err != nil {
		return nil, fmt.Errorf("failed to parse envelope: %w", err)
	}

	return envelope, nil
}

// GetTransactionStatus - Check transaction status
func (c *USDCEnvelopeClient) GetTransactionStatus(ctx context.Context, signature string) (*TransactionResult, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	status, err := c.rpcClient.GetSignatureStatuses(ctx, true, sig)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature status: %w", err)
	}

	if status == nil || len(status.Value) == 0 || status.Value[0] == nil {
		return &TransactionResult{
			Signature:   signature,
			Status:      StatusPending,
			ExplorerURL: c.getExplorerURL(signature),
		}, nil
	}

	txStatus := status.Value[0]
	result := &TransactionResult{
		Signature:   signature,
		ExplorerURL: c.getExplorerURL(signature),
	}

	if txStatus.Err != nil {
		errMsg := fmt.Sprintf("%v", txStatus.Err)
		result.Status = StatusFailed
		result.Error = &errMsg
	} else if txStatus.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
		result.Status = StatusFinalized
	} else if txStatus.ConfirmationStatus == rpc.ConfirmationStatusConfirmed {
		result.Status = StatusConfirmed
	} else {
		result.Status = StatusPending
	}

	return result, nil
}

// getExplorerURL - Generate explorer URL
func (c *USDCEnvelopeClient) getExplorerURL(signature string) string {
	if c.network == "mainnet" {
		return fmt.Sprintf(ExplorerURLMainnet, signature)
	}
	return fmt.Sprintf(ExplorerURLDevnet, signature)
}

// Helper function to convert uint64 to little-endian bytes
func uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(n)
	b[1] = byte(n >> 8)
	b[2] = byte(n >> 16)
	b[3] = byte(n >> 24)
	b[4] = byte(n >> 32)
	b[5] = byte(n >> 40)
	b[6] = byte(n >> 48)
	b[7] = byte(n >> 56)
	return b
}

// =========================
// UNSIGNED TRANSACTION FUNCTIONS
// =========================

// UnsignedTransactionResponse - Response for unsigned transaction
type UnsignedTransactionResponse struct {
	TransactionID       string `json:"transaction_id"`
	UnsignedTransaction string `json:"unsigned_transaction"` // base64 encoded
	RecentBlockhash     string `json:"recent_blockhash"`
	Message             string `json:"message,omitempty"`
}

// SignedTransactionRequest - Request to send signed transaction
type SignedTransactionRequest struct {
	TransactionID     string `json:"transaction_id"`
	SignedTransaction string `json:"signed_transaction"` // base64 encoded
}

// GenerateUnsignedInitUserState - Generate unsigned transaction for init_user_state
func (c *USDCEnvelopeClient) GenerateUnsignedInitUserState(user solana.PublicKey) (*UnsignedTransactionResponse, error) {
	// Build instruction
	instruction, err := c.BuildInitUserStateInstruction(user)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get recent blockhash
	ctx := context.Background()
	recent, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Build transaction WITHOUT signatures
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(user),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	transactionID := fmt.Sprintf("usdc_init_%d", time.Now().UnixNano())

	return &UnsignedTransactionResponse{
		TransactionID:       transactionID,
		UnsignedTransaction: base64.StdEncoding.EncodeToString(txBytes),
		RecentBlockhash:     recent.Value.Blockhash.String(),
		Message:             "Transaction ready to be signed by user",
	}, nil
}

// GenerateUnsignedCreateEnvelope - Generate unsigned transaction for create envelope
func (c *USDCEnvelopeClient) GenerateUnsignedCreateEnvelope(
	user solana.PublicKey,
	userTokenAccount solana.PublicKey,
	params CreateEnvelopeParams,
	nextEnvelopeID uint64,
) (*UnsignedTransactionResponse, error) {
	// Build instruction
	instruction, err := c.BuildCreateEnvelopeInstruction(user, userTokenAccount, params, nextEnvelopeID)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get recent blockhash
	ctx := context.Background()
	recent, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Build transaction WITHOUT signatures
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(user),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	transactionID := fmt.Sprintf("usdc_create_%d", time.Now().UnixNano())

	return &UnsignedTransactionResponse{
		TransactionID:       transactionID,
		UnsignedTransaction: base64.StdEncoding.EncodeToString(txBytes),
		RecentBlockhash:     recent.Value.Blockhash.String(),
		Message:             "Transaction ready to be signed by user",
	}, nil
}

// GenerateUnsignedClaim - Generate unsigned transaction for claim
func (c *USDCEnvelopeClient) GenerateUnsignedClaim(
	params ClaimEnvelopeParams,
) (*UnsignedTransactionResponse, error) {
	// Build instruction
	instruction, err := c.BuildClaimInstruction(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get recent blockhash
	ctx := context.Background()
	recent, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Build transaction WITHOUT signatures
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(params.Claimer),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	transactionID := fmt.Sprintf("usdc_claim_%d", time.Now().UnixNano())

	return &UnsignedTransactionResponse{
		TransactionID:       transactionID,
		UnsignedTransaction: base64.StdEncoding.EncodeToString(txBytes),
		RecentBlockhash:     recent.Value.Blockhash.String(),
		Message:             "Transaction ready to be signed by user",
	}, nil
}

// GenerateUnsignedRefund - Generate unsigned transaction for refund
func (c *USDCEnvelopeClient) GenerateUnsignedRefund(
	params RefundParams,
) (*UnsignedTransactionResponse, error) {
	// Build instruction
	instruction, err := c.BuildRefundInstruction(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build instruction: %w", err)
	}

	// Get recent blockhash
	ctx := context.Background()
	recent, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Build transaction WITHOUT signatures
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(params.Owner),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	transactionID := fmt.Sprintf("usdc_refund_%d", time.Now().UnixNano())

	return &UnsignedTransactionResponse{
		TransactionID:       transactionID,
		UnsignedTransaction: base64.StdEncoding.EncodeToString(txBytes),
		RecentBlockhash:     recent.Value.Blockhash.String(),
		Message:             "Transaction ready to be signed by user",
	}, nil
}

// SubmitSignedTransaction - Send signed transaction to blockchain
// Note: This is a convenience wrapper for unsigned transaction flow
func (c *USDCEnvelopeClient) SubmitSignedTransaction(req SignedTransactionRequest) (*TransactionResult, error) {
	// Decode signed transaction
	txBytes, err := base64.StdEncoding.DecodeString(req.SignedTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signed transaction: %w", err)
	}

	// Unmarshal transaction
	decoder := bin.NewBinDecoder(txBytes)
	var tx solana.Transaction
	if err := tx.UnmarshalWithDecoder(decoder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	// Validate transaction has signature
	if len(tx.Signatures) == 0 {
		return nil, fmt.Errorf("transaction is not signed")
	}

	// Send transaction to Solana
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sig, err := confirm.SendAndConfirmTransaction(
		ctx,
		c.rpcClient,
		c.wsClient,
		&tx,
	)

	if err != nil {
		return &TransactionResult{
			Signature:   "",
			Status:      StatusFailed,
			Error:       stringPtr(fmt.Sprintf("Failed to send transaction: %v", err)),
			ExplorerURL: "",
		}, err
	}

	signature := sig.String()

	return &TransactionResult{
		Signature:   signature,
		Status:      StatusFinalized,
		Error:       nil,
		ExplorerURL: c.getExplorerURL(signature),
	}, nil
}

// stringPtr - helper to get string pointer
func stringPtr(s string) *string {
	return &s
}
