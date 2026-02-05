package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	apiURL = "http://localhost:8080"
	// Anvil default private keys (accounts 1 and 2)
	buyerPK  = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
	sellerPK = "5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"

	tokenA   = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"
	tokenB   = "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"
	contract = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
)

var orderTypeHash = crypto.Keccak256Hash([]byte(
	"Order(address maker,address tokenSell,address tokenBuy,uint256 amountSell,uint256 amountBuy,uint256 expiry,uint256 nonce,uint256 salt)",
))

func main() {
	fmt.Println("=== Full Flow with On-Chain Settlement ===")
	fmt.Println()

	buyerKey, _ := crypto.HexToECDSA(buyerPK)
	sellerKey, _ := crypto.HexToECDSA(sellerPK)
	buyer := crypto.PubkeyToAddress(buyerKey.PublicKey)
	seller := crypto.PubkeyToAddress(sellerKey.PublicKey)

	domainSep := computeDomainSeparator()

	// Use unique salts for new orders
	salt := time.Now().UnixNano()

	// 1. Submit sell order
	fmt.Println("1. Seller submits: Sell 100 TKA @ 2 TKB/TKA")
	sellOrder := createOrder(seller, common.HexToAddress(tokenA), common.HexToAddress(tokenB), parseEther("100"), parseEther("200"), salt)
	sellSig := signOrder(sellOrder, sellerKey, domainSep)
	resp := submitOrder(sellOrder, sellSig, "sell")
	fmt.Println("   Response:", truncate(resp, 100))
	fmt.Println()

	// 2. Submit buy order that matches
	fmt.Println("2. Buyer submits: Buy 100 TKA @ 2 TKB/TKA")
	buyOrder := createOrder(buyer, common.HexToAddress(tokenB), common.HexToAddress(tokenA), parseEther("200"), parseEther("100"), salt+1)
	buySig := signOrder(buyOrder, buyerKey, domainSep)
	resp = submitOrder(buyOrder, buySig, "buy")
	fmt.Println("   Response:", truncate(resp, 100))
	fmt.Println()

	// Wait for settlement
	fmt.Println("3. Waiting for on-chain settlement...")
	time.Sleep(5 * time.Second)

	// Check trades
	fmt.Println("4. Checking trades...")
	trades := getTrades()
	if len(trades) > 0 {
		latest := trades[0].(map[string]interface{})
		fmt.Printf("   Latest Trade:\n")
		fmt.Printf("     Base Amount: %v TKA\n", formatWei(latest["baseAmount"]))
		fmt.Printf("     Quote Amount: %v TKB\n", formatWei(latest["quoteAmount"]))
		fmt.Printf("     Price: %v\n", latest["price"])
		fmt.Printf("     Settled On-Chain: %v\n", latest["settledOnChain"])
		fmt.Printf("     Tx Hash: %v\n", latest["txHash"])
	}
	fmt.Println()

	fmt.Println("=== Test Complete ===")
}

func formatWei(v interface{}) string {
	switch val := v.(type) {
	case float64:
		return fmt.Sprintf("%.4f", val/1e18)
	case string:
		n := new(big.Int)
		n.SetString(val, 10)
		f := new(big.Float).SetInt(n)
		f.Quo(f, big.NewFloat(1e18))
		r, _ := f.Float64()
		return fmt.Sprintf("%.4f", r)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

type orderData struct {
	maker      common.Address
	tokenSell  common.Address
	tokenBuy   common.Address
	amountSell *big.Int
	amountBuy  *big.Int
	expiry     *big.Int
	nonce      *big.Int
	salt       *big.Int
}

func createOrder(maker common.Address, tokenSell, tokenBuy common.Address, amountSell, amountBuy *big.Int, salt int64) orderData {
	return orderData{
		maker:      maker,
		tokenSell:  tokenSell,
		tokenBuy:   tokenBuy,
		amountSell: amountSell,
		amountBuy:  amountBuy,
		expiry:     big.NewInt(time.Now().Unix() + 3600),
		nonce:      big.NewInt(0),
		salt:       big.NewInt(salt),
	}
}

func signOrder(order orderData, key *ecdsa.PrivateKey, domainSep common.Hash) string {
	structHash := hashOrder(order)
	digest := crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSep.Bytes(),
		structHash.Bytes(),
	)
	sig, _ := crypto.Sign(digest.Bytes(), key)
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig)
}

func hashOrder(o orderData) common.Hash {
	return crypto.Keccak256Hash(
		orderTypeHash.Bytes(),
		common.LeftPadBytes(o.maker.Bytes(), 32),
		common.LeftPadBytes(o.tokenSell.Bytes(), 32),
		common.LeftPadBytes(o.tokenBuy.Bytes(), 32),
		common.LeftPadBytes(o.amountSell.Bytes(), 32),
		common.LeftPadBytes(o.amountBuy.Bytes(), 32),
		common.LeftPadBytes(o.expiry.Bytes(), 32),
		common.LeftPadBytes(o.nonce.Bytes(), 32),
		common.LeftPadBytes(o.salt.Bytes(), 32),
	)
}

func computeDomainSeparator() common.Hash {
	typeHash := crypto.Keccak256Hash([]byte(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)",
	))
	return crypto.Keccak256Hash(
		typeHash.Bytes(),
		crypto.Keccak256Hash([]byte("NexusOrderBook")).Bytes(),
		crypto.Keccak256Hash([]byte("1")).Bytes(),
		common.LeftPadBytes(big.NewInt(31337).Bytes(), 32),
		common.LeftPadBytes(common.HexToAddress(contract).Bytes(), 32),
	)
}

func submitOrder(o orderData, sig, side string) string {
	body := map[string]interface{}{
		"maker":      o.maker.Hex(),
		"tokenSell":  o.tokenSell.Hex(),
		"tokenBuy":   o.tokenBuy.Hex(),
		"amountSell": o.amountSell.String(),
		"amountBuy":  o.amountBuy.String(),
		"expiry":     o.expiry.Int64(),
		"nonce":      o.nonce.Int64(),
		"salt":       o.salt.String(),
		"signature":  sig,
		"side":       side,
		"pair":       "TKA-TKB",
	}
	jsonBody, _ := json.Marshal(body)
	resp, err := http.Post(apiURL+"/api/orders", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "error: " + err.Error()
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data)
}

func getTrades() []interface{} {
	resp, _ := http.Get(apiURL + "/api/trades?pair=TKA-TKB")
	defer resp.Body.Close()
	var result []interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func parseEther(s string) *big.Int {
	n := new(big.Int)
	n.SetString(s, 10)
	return n.Mul(n, big.NewInt(1e18))
}
