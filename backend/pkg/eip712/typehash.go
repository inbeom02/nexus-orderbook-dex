package eip712

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var OrderTypeHash = crypto.Keccak256Hash([]byte(
	"Order(address maker,address tokenSell,address tokenBuy,uint256 amountSell,uint256 amountBuy,uint256 expiry,uint256 nonce,uint256 salt)",
))

type DomainSeparator struct {
	Name              string
	Version           string
	ChainID           *big.Int
	VerifyingContract common.Address
}

func NewDomainSeparator(chainID *big.Int, contractAddr common.Address) DomainSeparator {
	return DomainSeparator{
		Name:              "NexusOrderBook",
		Version:           "1",
		ChainID:           chainID,
		VerifyingContract: contractAddr,
	}
}

func (d DomainSeparator) Hash() common.Hash {
	typeHash := crypto.Keccak256Hash([]byte(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)",
	))
	return crypto.Keccak256Hash(
		typeHash.Bytes(),
		crypto.Keccak256Hash([]byte(d.Name)).Bytes(),
		crypto.Keccak256Hash([]byte(d.Version)).Bytes(),
		common.LeftPadBytes(d.ChainID.Bytes(), 32),
		common.LeftPadBytes(d.VerifyingContract.Bytes(), 32),
	)
}

type OrderData struct {
	Maker      common.Address
	TokenSell  common.Address
	TokenBuy   common.Address
	AmountSell *big.Int
	AmountBuy  *big.Int
	Expiry     *big.Int
	Nonce      *big.Int
	Salt       *big.Int
}

func HashOrder(order OrderData) common.Hash {
	return crypto.Keccak256Hash(
		OrderTypeHash.Bytes(),
		common.LeftPadBytes(order.Maker.Bytes(), 32),
		common.LeftPadBytes(order.TokenSell.Bytes(), 32),
		common.LeftPadBytes(order.TokenBuy.Bytes(), 32),
		common.LeftPadBytes(order.AmountSell.Bytes(), 32),
		common.LeftPadBytes(order.AmountBuy.Bytes(), 32),
		common.LeftPadBytes(order.Expiry.Bytes(), 32),
		common.LeftPadBytes(order.Nonce.Bytes(), 32),
		common.LeftPadBytes(order.Salt.Bytes(), 32),
	)
}

func HashTypedData(domainSep common.Hash, structHash common.Hash) common.Hash {
	return crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSep.Bytes(),
		structHash.Bytes(),
	)
}
