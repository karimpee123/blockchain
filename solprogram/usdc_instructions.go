package solprogram

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// getAnchorDiscriminator - Generate Anchor instruction discriminator
// Anchor uses: sha256("global:<method_name>")[:8]
func getAnchorDiscriminator(methodName string) []byte {
	hash := sha256.Sum256([]byte("global:" + methodName))
	return hash[:8]
}

// Anchor instruction discriminators
var (
	DiscriminatorInitUserState = getAnchorDiscriminator("init_user_state")
	DiscriminatorCreate        = getAnchorDiscriminator("create")
	DiscriminatorClaim         = getAnchorDiscriminator("claim")
	DiscriminatorRefund        = getAnchorDiscriminator("refund")
	DiscriminatorCancel        = getAnchorDiscriminator("cancel")
	DiscriminatorClose         = getAnchorDiscriminator("close")
)

// BuildInitUserStateInstruction - Build init_user_state instruction
func (c *USDCEnvelopeClient) BuildInitUserStateInstruction(user solana.PublicKey) (solana.Instruction, error) {
	userStatePDA, _, err := c.DeriveUserStatePDA(user)
	if err != nil {
		return nil, err
	}

	// Use Anchor discriminator (8 bytes)
	data := DiscriminatorInitUserState

	accounts := []*solana.AccountMeta{
		solana.Meta(userStatePDA).WRITE(),
		solana.Meta(user).SIGNER().WRITE(),
		solana.Meta(SystemProgramID),
	}

	return solana.NewInstruction(
		c.programID,
		accounts,
		data,
	), nil
}

// BuildCreateEnvelopeInstruction - Build create envelope instruction
func (c *USDCEnvelopeClient) BuildCreateEnvelopeInstruction(
	user solana.PublicKey,
	userTokenAccount solana.PublicKey,
	params CreateEnvelopeParams,
	nextEnvelopeID uint64,
) (solana.Instruction, error) {
	// Derive PDAs
	userStatePDA, _, err := c.DeriveUserStatePDA(user)
	if err != nil {
		return nil, err
	}

	envelopePDA, _, err := c.DeriveEnvelopePDA(user, nextEnvelopeID)
	if err != nil {
		return nil, err
	}

	vaultPDA, _, err := c.DeriveEnvelopeVaultPDA(user, nextEnvelopeID)
	if err != nil {
		return nil, err
	}

	// Build instruction data: discriminator (8 bytes) + envelope_type + amounts + expiry
	data := make([]byte, 0, 8+1+32+8+8+8)
	// Add Anchor discriminator
	data = append(data, DiscriminatorCreate...)

	// Envelope type (1 byte)
	data = append(data, uint8(params.EnvelopeType.Type))

	// If DirectFixed, add allowed address (32 bytes)
	if params.EnvelopeType.Type == EnvelopeTypeDirectFixed {
		if params.EnvelopeType.AllowedAddress == nil {
			return nil, fmt.Errorf("allowed_address required for DirectFixed")
		}
		data = append(data, params.EnvelopeType.AllowedAddress.Bytes()...)
	}

	// Total amount (8 bytes)
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, params.TotalAmount)
	data = append(data, amountBytes...)

	// Total users (8 bytes)
	usersBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(usersBytes, params.TotalUsers)
	data = append(data, usersBytes...)

	// Expiry seconds (8 bytes)
	expiryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expiryBytes, params.ExpirySeconds)
	data = append(data, expiryBytes...)

	accounts := []*solana.AccountMeta{
		solana.Meta(userStatePDA).WRITE(),
		solana.Meta(envelopePDA).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(userTokenAccount).WRITE(),
		solana.Meta(c.usdcMint),
		solana.Meta(user).SIGNER().WRITE(),
		solana.Meta(TokenProgramID),
		solana.Meta(SystemProgramID),
	}

	return solana.NewInstruction(
		c.programID,
		accounts,
		data,
	), nil
}

// BuildClaimInstruction - Build claim instruction
func (c *USDCEnvelopeClient) BuildClaimInstruction(
	params ClaimEnvelopeParams,
) (solana.Instruction, error) {
	// Derive PDAs
	envelopePDA, _, err := c.DeriveEnvelopePDA(params.Owner, params.EnvelopeID)
	if err != nil {
		return nil, err
	}

	vaultPDA, _, err := c.DeriveEnvelopeVaultPDA(params.Owner, params.EnvelopeID)
	if err != nil {
		return nil, err
	}

	claimRecordPDA, _, err := c.DeriveClaimRecordPDA(envelopePDA, params.Claimer)
	if err != nil {
		return nil, err
	}

	// Build instruction data - only discriminator for claim
	data := DiscriminatorClaim

	// Account order MUST match Rust program's Claim struct:
	// 1. envelope, 2. envelope_vault, 3. claimer_token_account,
	// 4. claim_record, 5. claimer, 6. token_program, 7. system_program
	accounts := []*solana.AccountMeta{
		solana.Meta(envelopePDA).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(params.ClaimerTokenAccount).WRITE(),
		solana.Meta(claimRecordPDA).WRITE(),
		solana.Meta(params.Claimer).SIGNER().WRITE(),
		solana.Meta(TokenProgramID),
		solana.Meta(SystemProgramID),
	}

	return solana.NewInstruction(
		c.programID,
		accounts,
		data,
	), nil
}

// BuildRefundInstruction - Build refund instruction
func (c *USDCEnvelopeClient) BuildRefundInstruction(
	params RefundParams,
) (solana.Instruction, error) {
	// Derive PDAs
	envelopePDA, _, err := c.DeriveEnvelopePDA(params.Owner, params.EnvelopeID)
	if err != nil {
		return nil, err
	}

	vaultPDA, _, err := c.DeriveEnvelopeVaultPDA(params.Owner, params.EnvelopeID)
	if err != nil {
		return nil, err
	}

	// Build instruction data - only discriminator for refund
	data := DiscriminatorRefund

	// Account order MUST match Rust program's Refund struct:
	// 1. envelope, 2. envelope_vault, 3. owner_token_account,
	// 4. owner, 5. token_program, 6. system_program
	accounts := []*solana.AccountMeta{
		solana.Meta(envelopePDA).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(params.OwnerTokenAccount).WRITE(),
		solana.Meta(params.Owner).SIGNER().WRITE(),
		solana.Meta(TokenProgramID),
		solana.Meta(SystemProgramID),
	}

	return solana.NewInstruction(
		c.programID,
		accounts,
		data,
	), nil
}

// BuildCancelInstruction - Build cancel instruction
func (c *USDCEnvelopeClient) BuildCancelInstruction(
	owner solana.PublicKey,
	envelopeID uint64,
) (solana.Instruction, error) {
	// Derive PDAs
	userStatePDA, _, err := c.DeriveUserStatePDA(owner)
	if err != nil {
		return nil, err
	}

	envelopePDA, _, err := c.DeriveEnvelopePDA(owner, envelopeID)
	if err != nil {
		return nil, err
	}

	// Build instruction data - only discriminator for cancel
	data := DiscriminatorCancel

	accounts := []*solana.AccountMeta{
		solana.Meta(envelopePDA).WRITE(),
		solana.Meta(userStatePDA).WRITE(),
		solana.Meta(owner).SIGNER().WRITE(),
	}

	return solana.NewInstruction(
		c.programID,
		accounts,
		data,
	), nil
}

// BuildCloseEnvelopeInstruction - Build close envelope instruction
func (c *USDCEnvelopeClient) BuildCloseEnvelopeInstruction(
	owner solana.PublicKey,
	envelopeID uint64,
) (solana.Instruction, error) {
	// Derive PDAs
	envelopePDA, _, err := c.DeriveEnvelopePDA(owner, envelopeID)
	if err != nil {
		return nil, err
	}

	// Build instruction data - only discriminator for close
	data := DiscriminatorClose

	accounts := []*solana.AccountMeta{
		solana.Meta(envelopePDA).WRITE(),
		solana.Meta(owner).SIGNER().WRITE(),
	}

	return solana.NewInstruction(
		c.programID,
		accounts,
		data,
	), nil
}
