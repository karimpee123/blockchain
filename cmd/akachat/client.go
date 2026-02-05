package main

import (
	"encoding/base64"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

// ------------------------------ CLIENT SIDE ------------------------------ //
func clientSign(unsignedTx string, key string) (*string, error) {
	privateKey, err := solana.PrivateKeyFromBase58(key)
	if err != nil {
		return nil, err
	}
	txBytes, err := base64.StdEncoding.DecodeString(unsignedTx)
	if err != nil {
		return nil, err
	}
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if privateKey.PublicKey().Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	signedTxBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	signedTxBase64 := base64.StdEncoding.EncodeToString(signedTxBytes)
	return &signedTxBase64, nil
}
