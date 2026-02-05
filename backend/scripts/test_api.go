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
	fmt.Println("=== API E2E Test ===")
	fmt.Println()

	// Get buyer and seller addresses
	buyerKey, _ := crypto.HexToECDSA(buyerPK)
	sellerKey, _ := crypto.HexToECDSA(sellerPK)
	buyer := crypto.PubkeyToAddress(buyerKey.PublicKey)
	seller := crypto.PubkeyToAddress(sellerKey.PublicKey)

	fmt.Println("Buyer:", buyer.Hex())
	fmt.Println("Seller:", seller.Hex())
	fmt.Println()

	// Domain separator
	domainSep := computeDomainSeparator()
	fmt.Println("Domain Separator:", domainSep.Hex())
	fmt.Println()

	// 1. Submit a sell order (seller sells 50 TKA for 100 TKB, price = 2)
	fmt.Println("1. Submitting sell order (50 TKA @ 2 TKB/TKA)...")
	sellOrder := createOrder(
		seller,
		common.HexToAddress(tokenA), // tokenSell
		common.HexToAddress(tokenB), // tokenBuy
		parseEther("50"),            // amountSell
		parseEther("100"),           // amountBuy
		1,
	)
	sellSig := signOrder(sellOrder, sellerKey, domainSep)
	resp := submitOrder(sellOrder, sellSig, "sell")
	fmt.Println("   Response:", resp)
	fmt.Println()

	// 2. Submit another sell order at different price
	fmt.Println("2. Submitting sell order (30 TKA @ 2.5 TKB/TKA)...")
	sellOrder2 := createOrder(
		seller,
		common.HexToAddress(tokenA),
		common.HexToAddress(tokenB),
		parseEther("30"),
		parseEther("75"),
		2,
	)
	sellSig2 := signOrder(sellOrder2, sellerKey, domainSep)
	resp = submitOrder(sellOrder2, sellSig2, "sell")
	fmt.Println("   Response:", resp)
	fmt.Println()

	// 3. Check orderbook
	fmt.Println("3. Checking orderbook...")
	ob := getOrderbook()
	fmt.Println("   Bids:", ob["bids"])
	fmt.Println("   Asks:", ob["asks"])
	fmt.Println()

	// 4. Submit a buy order that matches (buyer buys 50 TKA @ 2 TKB/TKA)
	fmt.Println("4. Submitting buy order (50 TKA @ 2 TKB/TKA) - should match!")
	buyOrder := createOrder(
		buyer,
		common.HexToAddress(tokenB), // tokenSell
		common.HexToAddress(tokenA), // tokenBuy
		parseEther("100"),           // amountSell
		parseEther("50"),            // amountBuy
		1,
	)
	buySig := signOrder(buyOrder, buyerKey, domainSep)
	resp = submitOrder(buyOrder, buySig, "buy")
	fmt.Println("   Response:", resp)
	fmt.Println()

	// 5. Check orderbook after match
	fmt.Println("5. Orderbook after match...")
	ob = getOrderbook()
	fmt.Println("   Bids:", ob["bids"])
	fmt.Println("   Asks:", ob["asks"])
	fmt.Println()

	// 6. Check trades
	fmt.Println("6. Recent trades...")
	trades := getTrades()
	for _, t := range trades {
		fmt.Printf("   Trade: %v\n", t)
	}
	fmt.Println()

	// 7. Check user orders
	fmt.Println("7. User orders for seller...")
	orders := getUserOrders(seller.Hex())
	for _, o := range orders {
		m := o.(map[string]interface{})
		fmt.Printf("   Order ID=%s, side=%s, status=%s\n", m["id"], m["side"], m["status"])
	}
	fmt.Println()

	fmt.Println("=== Test Complete ===")
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

func getOrderbook() map[string]interface{} {
	resp, _ := http.Get(apiURL + "/api/orderbook?pair=TKA-TKB")
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func getTrades() []interface{} {
	resp, _ := http.Get(apiURL + "/api/trades?pair=TKA-TKB")
	defer resp.Body.Close()
	var result []interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func getUserOrders(addr string) []interface{} {
	resp, _ := http.Get(apiURL + "/api/orders/" + addr)
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

