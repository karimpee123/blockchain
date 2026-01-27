package solprogram

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

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

// BuildCreateEnvelopeInstruction builds create envelope instruction (simplified - DirectFixed only)
func BuildCreateEnvelopeInstruction(
	programID solana.PublicKey,
	user solana.PublicKey,
	envelopeID uint64,
	envelopeType EnvelopeTypeRequest,
	totalAmount uint64,
	totalUsers uint64,
	expiryHours uint64,
	allowedAddress *string,
) (solana.Instruction, error) {
	userStatePDA, _, _ := DeriveUserStatePDA(programID, user)
	envelopePDA, _, _ := DeriveEnvelopePDA(programID, user, envelopeID)

	// ✅ Discriminator dari IDL
	discriminator := []byte{24, 30, 200, 40, 5, 28, 7, 119}

	// Build envelope_type enum
	var envelopeTypeData []byte

	switch envelopeType {
	case EnvelopeTypeDirectFixed:
		if allowedAddress == nil {
			return nil, fmt.Errorf("allowed_address required for DirectFixed")
		}
		allowedPubkey := solana.MustPublicKeyFromBase58(*allowedAddress)

		// Enum: variant (1 byte) + data (32 bytes)
		envelopeTypeData = make([]byte, 33)
		envelopeTypeData[0] = 0 // DirectFixed = variant 0
		copy(envelopeTypeData[1:33], allowedPubkey.Bytes())

	case EnvelopeTypeGroupFixed:
		envelopeTypeData = []byte{1} // GroupFixed = variant 1

	case EnvelopeTypeGroupRandom:
		envelopeTypeData = []byte{2} // GroupRandom = variant 2

	default:
		return nil, fmt.Errorf("invalid envelope type: %s", envelopeType)
	}

	// Serialize instruction data
	instructionData := make([]byte, 0)

	// 1. Discriminator (8 bytes)
	instructionData = append(instructionData, discriminator...)

	// 2. EnvelopeType (1 or 33 bytes)
	instructionData = append(instructionData, envelopeTypeData...)

	// 3. total_amount (8 bytes LE)
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, totalAmount)
	instructionData = append(instructionData, amountBytes...)

	// 4. total_users (8 bytes LE)
	usersBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(usersBytes, totalUsers)
	instructionData = append(instructionData, usersBytes...)

	// 5. expiry_hours (8 bytes LE)
	expiryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expiryBytes, expiryHours)
	instructionData = append(instructionData, expiryBytes...)

	// ✅ DEBUG LOGGING
	fmt.Printf("\n=== CREATE INSTRUCTION DEBUG ===\n")
	fmt.Printf("Program ID: %s\n", programID)
	fmt.Printf("User: %s\n", user)
	fmt.Printf("User State PDA: %s\n", userStatePDA)
	fmt.Printf("Envelope PDA: %s\n", envelopePDA)
	fmt.Printf("Envelope ID: %d\n", envelopeID)
	fmt.Printf("Envelope Type: %s\n", envelopeType)
	fmt.Printf("Total Amount: %d\n", totalAmount)
	fmt.Printf("Total Users: %d\n", totalUsers)
	fmt.Printf("Expiry Hours: %d\n", expiryHours)
	fmt.Printf("\nInstruction Data (%d bytes):\n", len(instructionData))
	fmt.Printf("  Discriminator: %v\n", discriminator)
	fmt.Printf("  EnvelopeType: %v (len=%d)\n", envelopeTypeData, len(envelopeTypeData))
	fmt.Printf("  TotalAmount: %v\n", amountBytes)
	fmt.Printf("  TotalUsers: %v\n", usersBytes)
	fmt.Printf("  ExpiryHours: %v\n", expiryBytes)
	fmt.Printf("  Full hex: %x\n", instructionData)
	fmt.Printf("================================\n\n")

	return solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(userStatePDA).WRITE(),
			solana.Meta(envelopePDA).WRITE(),
			solana.Meta(user).WRITE().SIGNER(),
			solana.Meta(solana.SystemProgramID),
		},
		instructionData,
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
