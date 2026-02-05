package eip712

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// VerifyOrderSignature verifies that the signature was produced by the expected maker.
func VerifyOrderSignature(domain DomainSeparator, order OrderData, signature []byte) (bool, error) {
	if len(signature) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(signature))
	}

	structHash := HashOrder(order)
	digest := HashTypedData(domain.Hash(), structHash)

	// Adjust v value: Ethereum uses 27/28, go-ethereum's crypto.Ecrecover expects 0/1
	sig := make([]byte, 65)
	copy(sig, signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	pubKey, err := crypto.Ecrecover(digest.Bytes(), sig)
	if err != nil {
		return false, fmt.Errorf("ecrecover failed: %w", err)
	}

	recoveredPub, err := crypto.UnmarshalPubkey(pubKey)
	if err != nil {
		return false, fmt.Errorf("unmarshal pubkey failed: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*recoveredPub)
	return recoveredAddr == order.Maker, nil
}

// RecoverSigner recovers the signer address from a digest and signature.
func RecoverSigner(digest common.Hash, signature []byte) (common.Address, error) {
	if len(signature) != 65 {
		return common.Address{}, fmt.Errorf("invalid signature length: %d", len(signature))
	}

	sig := make([]byte, 65)
	copy(sig, signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	pubKey, err := crypto.Ecrecover(digest.Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	recoveredPub, err := crypto.UnmarshalPubkey(pubKey)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*recoveredPub), nil
}
