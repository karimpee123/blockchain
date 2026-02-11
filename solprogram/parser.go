package solprogram

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
)

// parseUserStateData - Parse user state account data
func parseUserStateData(data []byte) (*UserState, error) {
	if len(data) < 48 { // 8 (discriminator) + 32 (pubkey) + 8 (u64)
		return nil, fmt.Errorf("invalid user state data length: %d", len(data))
	}

	// Skip 8-byte discriminator
	offset := 8

	// Parse owner (32 bytes)
	owner := solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	// Parse last_envelope_id (8 bytes, little-endian)
	lastEnvelopeID := binary.LittleEndian.Uint64(data[offset : offset+8])

	return &UserState{
		Owner:          owner,
		LastEnvelopeID: lastEnvelopeID,
	}, nil
}

// parseEnvelopeData - Parse envelope account data
func parseEnvelopeData(data []byte) (*EnvelopeInfo, error) {
	if len(data) < 120 { // Minimum size
		return nil, fmt.Errorf("invalid envelope data length: %d", len(data))
	}

	// Skip 8-byte discriminator
	offset := 8

	// Parse owner (32 bytes)
	owner := solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	// Parse envelope_id (8 bytes)
	envelopeID := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse envelope_type (1 byte discriminator + optional 32 bytes for DirectFixed)
	envelopeTypeDiscriminator := data[offset]
	offset += 1

	var envelopeTypeName string
	var allowedAddress *string

	switch envelopeTypeDiscriminator {
	case 0: // DirectFixed
		allowed := solana.PublicKeyFromBytes(data[offset : offset+32])
		allowedStr := allowed.String()
		allowedAddress = &allowedStr
		envelopeTypeName = "DirectFixed"
		offset += 32
	case 1: // GroupFixed
		envelopeTypeName = "GroupFixed"
	case 2: // GroupRandom
		envelopeTypeName = "GroupRandom"
	default:
		return nil, fmt.Errorf("unknown envelope type: %d", envelopeTypeDiscriminator)
	}

	// Align to 8-byte boundary if needed (Rust alignment)
	// For non-DirectFixed, we may need to skip padding
	if envelopeTypeDiscriminator != 0 {
		// Skip padding to align (33 bytes needs 7 bytes padding to reach 40)
		offset += 39 // Skip to reach consistent offset
	}

	// Parse total_amount (8 bytes)
	totalAmount := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse total_users (8 bytes)
	totalUsers := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse withdrawn_amount (8 bytes)
	withdrawnAmount := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse claimed_count (8 bytes)
	claimedCount := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse expiry (8 bytes, i64 timestamp)
	expiryTimestamp := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Parse is_cancelled (1 byte bool)
	isCancelled := data[offset] != 0

	// Calculate remaining amount
	remainingAmount := totalAmount - withdrawnAmount

	// Convert expiry to time.Time
	expiryTime := time.Unix(expiryTimestamp, 0)
	isExpired := time.Now().Unix() >= expiryTimestamp

	return &EnvelopeInfo{
		Owner:           owner,
		EnvelopeID:      envelopeID,
		EnvelopeType:    envelopeTypeName,
		AllowedAddress:  allowedAddress,
		TotalAmount:     totalAmount,
		TotalUsers:      totalUsers,
		WithdrawnAmount: withdrawnAmount,
		ClaimedCount:    claimedCount,
		RemainingAmount: remainingAmount,
		IsCancelled:     isCancelled,
		ExpiryTime:      expiryTime,
		IsExpired:       isExpired,
	}, nil
}

// parseClaimRecordData - Parse claim record account data
func parseClaimRecordData(data []byte) (*ClaimRecord, error) {
	if len(data) < 64 { // 8 + 32 + 8 + 8 + 8
		return nil, fmt.Errorf("invalid claim record data length: %d", len(data))
	}

	// Skip 8-byte discriminator
	offset := 8

	// Parse claimer (32 bytes)
	claimer := solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	// Parse envelope_id (8 bytes)
	envelopeID := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse amount (8 bytes)
	amount := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse claimed_at (8 bytes, i64 timestamp)
	claimedAt := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))

	return &ClaimRecord{
		Claimer:    claimer,
		EnvelopeID: envelopeID,
		Amount:     amount,
		ClaimedAt:  claimedAt,
	}, nil
}
