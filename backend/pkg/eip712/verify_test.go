package eip712

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestHashOrder(t *testing.T) {
	order := OrderData{
		Maker:      common.HexToAddress("0x1234567890123456789012345678901234567890"),
		TokenSell:  common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		TokenBuy:   common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		AmountSell: big.NewInt(1000),
		AmountBuy:  big.NewInt(500),
		Expiry:     big.NewInt(1700000000),
		Nonce:      big.NewInt(0),
		Salt:       big.NewInt(12345),
	}

	hash := HashOrder(order)
	if hash == (common.Hash{}) {
		t.Fatal("hash should not be zero")
	}
}

func TestVerifyOrderSignature(t *testing.T) {
	// Generate test key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	maker := crypto.PubkeyToAddress(privateKey.PublicKey)

	domain := DomainSeparator{
		Name:              "NexusOrderBook",
		Version:           "1",
		ChainID:           big.NewInt(31337),
		VerifyingContract: common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
	}

	order := OrderData{
		Maker:      maker,
		TokenSell:  common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		TokenBuy:   common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		AmountSell: big.NewInt(1000),
		AmountBuy:  big.NewInt(500),
		Expiry:     big.NewInt(1700000000),
		Nonce:      big.NewInt(0),
		Salt:       big.NewInt(12345),
	}

	// Create signature
	structHash := HashOrder(order)
	digest := HashTypedData(domain.Hash(), structHash)
	sig, err := crypto.Sign(digest.Bytes(), privateKey)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	// Adjust v (Ethereum convention: 27 or 28)
	if sig[64] < 27 {
		sig[64] += 27
	}

	valid, err := VerifyOrderSignature(domain, order, sig)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}
}

func TestVerifyOrderSignature_WrongMaker(t *testing.T) {
	privateKey, _ := crypto.GenerateKey()
	wrongKey, _ := crypto.GenerateKey()
	wrongMaker := crypto.PubkeyToAddress(wrongKey.PublicKey)

	domain := NewDomainSeparator(big.NewInt(31337), common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"))

	order := OrderData{
		Maker:      wrongMaker, // Different from signer
		TokenSell:  common.HexToAddress("0xaa"),
		TokenBuy:   common.HexToAddress("0xbb"),
		AmountSell: big.NewInt(1000),
		AmountBuy:  big.NewInt(500),
		Expiry:     big.NewInt(1700000000),
		Nonce:      big.NewInt(0),
		Salt:       big.NewInt(12345),
	}

	structHash := HashOrder(order)
	digest := HashTypedData(domain.Hash(), structHash)
	sig, _ := crypto.Sign(digest.Bytes(), privateKey)
	if sig[64] < 27 {
		sig[64] += 27
	}

	valid, err := VerifyOrderSignature(domain, order, sig)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if valid {
		t.Fatal("signature should NOT be valid (wrong maker)")
	}
}
