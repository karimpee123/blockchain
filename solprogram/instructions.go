package solprogram

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// InstructionDiscriminators
func getDiscriminator(name string) [8]byte {
	hash := sha256.Sum256([]byte(name))
	var disc [8]byte
	copy(disc[:], hash[:8])
	return disc
}

var (
	InitUserStateDisc = getDiscriminator("global:init_user_state")
	CreateDisc        = getDiscriminator("global:create")
	ClaimDisc         = getDiscriminator("global:claim")
	RefundDisc        = getDiscriminator("global:refund")
)

// DeriveUserStatePDA derives user_state PDA address
func DeriveUserStatePDA(programID, user solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{
			[]byte("user_state"),
			user.Bytes(),
		},
		programID,
	)
}

// DeriveEnvelopePDA derives envelope PDA address
func DeriveEnvelopePDA(programID, user solana.PublicKey, envelopeID uint64) (solana.PublicKey, uint8, error) {
	envelopeIDBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(envelopeIDBytes, envelopeID)

	return solana.FindProgramAddress(
		[][]byte{
			[]byte("envelope"),
			user.Bytes(),
			envelopeIDBytes,
		},
		programID,
	)
}

// CheckUserStateExists checks if user_state account exists
func CheckUserStateExists(rpcClient *rpc.Client, userStatePDA solana.PublicKey) (bool, uint64, error) {
	accountInfo, err := rpcClient.GetAccountInfo(context.Background(), userStatePDA)
	if err != nil {
		// Account doesn't exist
		return false, 0, nil
	}

	if accountInfo == nil || accountInfo.Value == nil {
		return false, 0, nil
	}

	// Parse last_envelope_id from account data
	// Layout: discriminator(8) + owner(32) + last_envelope_id(8)
	data := accountInfo.Value.Data.GetBinary()
	if len(data) < 48 {
		return false, 0, nil
	}

	lastEnvelopeID := binary.LittleEndian.Uint64(data[40:48])
	return true, lastEnvelopeID, nil
}

// BuildInitUserStateInstruction builds init_user_state instruction
func BuildInitUserStateInstruction(
	programID solana.PublicKey,
	user solana.PublicKey,
) (solana.Instruction, error) {
	userState, _, err := DeriveUserStatePDA(programID, user)
	if err != nil {
		return nil, err
	}

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(userState).WRITE(),
			solana.Meta(user).WRITE().SIGNER(),
			solana.Meta(solana.SystemProgramID),
		},
		InitUserStateDisc[:],
	), nil
}

// BuildCreateInstruction builds create envelope instruction (simplified - DirectFixed only)
func BuildCreateInstruction(
	programID solana.PublicKey,
	user solana.PublicKey,
	envelopeID uint64,
	amount uint64,
	expiryHours uint64,
) (solana.Instruction, error) {
	// Derive PDAs
	userState, _, _ := DeriveUserStatePDA(programID, user)
	envelope, _, _ := DeriveEnvelopePDA(programID, user, envelopeID)

	// Build instruction data
	// Format: discriminator(8) + envelope_type + expiry(8)
	data := make([]byte, 0, 64)
	data = append(data, CreateDisc[:]...)

	// DirectFixed type (variant 0)
	data = append(data, 0)               // variant index
	data = append(data, user.Bytes()...) // allowed_address (32 bytes)

	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, amount)
	data = append(data, amountBytes...) // amount (8 bytes)

	expiryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expiryBytes, expiryHours)
	data = append(data, expiryBytes...) // expiry_hours (8 bytes)

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(userState).WRITE(),
			solana.Meta(envelope).WRITE(),
			solana.Meta(user).WRITE().SIGNER(),
			solana.Meta(solana.SystemProgramID),
		},
		data,
	), nil
}

// BuildClaimInstruction builds claim instruction
func BuildClaimInstruction(
	programID solana.PublicKey,
	owner solana.PublicKey,
	claimer solana.PublicKey,
	envelopeID uint64,
) (solana.Instruction, error) {
	envelope, _, _ := DeriveEnvelopePDA(programID, owner, envelopeID)

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(envelope).WRITE(),
			solana.Meta(claimer).WRITE().SIGNER(),
		},
		ClaimDisc[:],
	), nil
}

// BuildRefundInstruction builds refund instruction
func BuildRefundInstruction(
	programID solana.PublicKey,
	owner solana.PublicKey,
	envelopeID uint64,
) (solana.Instruction, error) {
	envelope, _, _ := DeriveEnvelopePDA(programID, owner, envelopeID)

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(envelope).WRITE(),
			solana.Meta(owner).WRITE().SIGNER(),
		},
		RefundDisc[:],
	), nil
}

// BuildCreateDirectFixedInstruction builds DirectFixed envelope
func BuildCreateDirectFixedInstruction(
	programID solana.PublicKey,
	user solana.PublicKey,
	envelopeID uint64,
	allowedAddress string,
	amount uint64,
	expiryHours uint64,
) (solana.Instruction, error) {
	// Derive PDAs
	userState, _, _ := DeriveUserStatePDA(programID, user)
	envelope, _, _ := DeriveEnvelopePDA(programID, user, envelopeID)

	// Parse allowed address
	allowedPubkey := solana.MustPublicKeyFromBase58(allowedAddress)

	// Build instruction data
	// Format: discriminator(8) + variant(1) + allowed_address(32) + amount(8) + expiry(8)
	data := make([]byte, 0, 64)
	data = append(data, CreateDisc[:]...)         // discriminator (8 bytes)
	data = append(data, 0)                        // variant 0 = DirectFixed
	data = append(data, allowedPubkey.Bytes()...) // allowed_address (32 bytes)

	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, amount)
	data = append(data, amountBytes...) // amount (8 bytes)

	expiryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expiryBytes, expiryHours)
	data = append(data, expiryBytes...) // expiry_hours (8 bytes)

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(userState).WRITE(),
			solana.Meta(envelope).WRITE(),
			solana.Meta(user).WRITE().SIGNER(),
			solana.Meta(solana.SystemProgramID),
		},
		data,
	), nil
}

// BuildCreateGroupFixedInstruction builds GroupFixed envelope
func BuildCreateGroupFixedInstruction(
	programID solana.PublicKey,
	user solana.PublicKey,
	envelopeID uint64,
	totalUsers uint64,
	amountPerUser uint64,
	expiryHours uint64,
) (solana.Instruction, error) {
	userState, _, _ := DeriveUserStatePDA(programID, user)
	envelope, _, _ := DeriveEnvelopePDA(programID, user, envelopeID)

	// Build instruction data
	// Format: discriminator(8) + variant(1) + total_users(8) + amount_per_user(8) + expiry(8)
	data := make([]byte, 0, 40)
	data = append(data, CreateDisc[:]...) // discriminator (8 bytes)
	data = append(data, 1)                // variant 1 = GroupFixed

	totalUsersBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(totalUsersBytes, totalUsers)
	data = append(data, totalUsersBytes...) // total_users (8 bytes)

	amountPerUserBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountPerUserBytes, amountPerUser)
	data = append(data, amountPerUserBytes...) // amount_per_user (8 bytes)

	expiryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expiryBytes, expiryHours)
	data = append(data, expiryBytes...) // expiry_hours (8 bytes)

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(userState).WRITE(),
			solana.Meta(envelope).WRITE(),
			solana.Meta(user).WRITE().SIGNER(),
			solana.Meta(solana.SystemProgramID),
		},
		data,
	), nil
}

// BuildCreateGroupRandomInstruction builds GroupRandom envelope
func BuildCreateGroupRandomInstruction(
	programID solana.PublicKey,
	user solana.PublicKey,
	envelopeID uint64,
	totalAmount uint64,
	maxClaimers uint64,
	expiryHours uint64,
) (solana.Instruction, error) {
	userState, _, _ := DeriveUserStatePDA(programID, user)
	envelope, _, _ := DeriveEnvelopePDA(programID, user, envelopeID)

	// Build instruction data
	// Format: discriminator(8) + variant(1) + total_amount(8) + max_claimers(8) + expiry(8)
	data := make([]byte, 0, 40)
	data = append(data, CreateDisc[:]...) // discriminator (8 bytes)
	data = append(data, 2)                // variant 2 = GroupRandom

	totalAmountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(totalAmountBytes, totalAmount)
	data = append(data, totalAmountBytes...) // total_amount (8 bytes)

	maxClaimersBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(maxClaimersBytes, maxClaimers)
	data = append(data, maxClaimersBytes...) // max_claimers (8 bytes)

	expiryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expiryBytes, expiryHours)
	data = append(data, expiryBytes...) // expiry_hours (8 bytes)

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(userState).WRITE(),
			solana.Meta(envelope).WRITE(),
			solana.Meta(user).WRITE().SIGNER(),
			solana.Meta(solana.SystemProgramID),
		},
		data,
	), nil
}
