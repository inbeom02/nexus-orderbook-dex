package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/nexus-orderbook-dex/backend/internal/domain"
	"github.com/nexus-orderbook-dex/backend/internal/orderbook"
)

const nexusABI = `[{"inputs":[{"components":[{"internalType":"address","name":"maker","type":"address"},{"internalType":"address","name":"tokenSell","type":"address"},{"internalType":"address","name":"tokenBuy","type":"address"},{"internalType":"uint256","name":"amountSell","type":"uint256"},{"internalType":"uint256","name":"amountBuy","type":"uint256"},{"internalType":"uint256","name":"expiry","type":"uint256"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"uint256","name":"salt","type":"uint256"}],"internalType":"struct OrderTypes.Order","name":"buyOrder","type":"tuple"},{"internalType":"bytes","name":"buySig","type":"bytes"},{"components":[{"internalType":"address","name":"maker","type":"address"},{"internalType":"address","name":"tokenSell","type":"address"},{"internalType":"address","name":"tokenBuy","type":"address"},{"internalType":"uint256","name":"amountSell","type":"uint256"},{"internalType":"uint256","name":"amountBuy","type":"uint256"},{"internalType":"uint256","name":"expiry","type":"uint256"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"uint256","name":"salt","type":"uint256"}],"internalType":"struct OrderTypes.Order","name":"sellOrder","type":"tuple"},{"internalType":"bytes","name":"sellSig","type":"bytes"},{"internalType":"uint256","name":"fillAmount","type":"uint256"}],"name":"settleMatch","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

// SettlementWorker serializes on-chain tx submissions via a channel.
type SettlementWorker struct {
	client    *Client
	parsedABI abi.ABI
	nonceMu   sync.Mutex
}

type SettleJob struct {
	Match   orderbook.MatchResult
	TradeID string
	Result  chan SettleResult
}

type SettleResult struct {
	TxHash string
	Err    error
}

func NewSettlementWorker(client *Client) (*SettlementWorker, error) {
	parsed, err := abi.JSON(strings.NewReader(nexusABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	return &SettlementWorker{
		client:    client,
		parsedABI: parsed,
	}, nil
}

// Run starts the settlement worker reading from the jobs channel.
func (w *SettlementWorker) Run(ctx context.Context, jobs <-chan SettleJob) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			result := w.settle(ctx, job.Match)
			job.Result <- result
		}
	}
}

func (w *SettlementWorker) settle(ctx context.Context, match orderbook.MatchResult) SettleResult {
	w.nonceMu.Lock()
	defer w.nonceMu.Unlock()

	buyOrder := tuplifyOrder(match.BuyOrder)
	sellOrder := tuplifyOrder(match.SellOrder)
	buySig := common.FromHex(match.BuyOrder.Signature)
	sellSig := common.FromHex(match.SellOrder.Signature)

	data, err := w.parsedABI.Pack("settleMatch", buyOrder, buySig, sellOrder, sellSig, match.FillAmount)
	if err != nil {
		return SettleResult{Err: fmt.Errorf("pack failed: %w", err)}
	}

	nonce, err := w.client.EthClient.PendingNonceAt(ctx, w.client.Address)
	if err != nil {
		return SettleResult{Err: fmt.Errorf("nonce failed: %w", err)}
	}

	gasPrice, err := w.client.EthClient.SuggestGasPrice(ctx)
	if err != nil {
		return SettleResult{Err: fmt.Errorf("gas price failed: %w", err)}
	}

	tx := types.NewTransaction(
		nonce,
		w.client.Contract,
		big.NewInt(0),
		uint64(500000),
		gasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(w.client.ChainID), w.client.PrivateKey)
	if err != nil {
		return SettleResult{Err: fmt.Errorf("sign failed: %w", err)}
	}

	err = w.client.EthClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return SettleResult{Err: fmt.Errorf("send failed: %w", err)}
	}

	log.Printf("Settlement tx sent: %s", signedTx.Hash().Hex())

	// Wait for receipt
	receipt, err := bind.WaitMined(ctx, w.client.EthClient, signedTx)
	if err != nil {
		return SettleResult{Err: fmt.Errorf("wait mined failed: %w", err)}
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return SettleResult{Err: fmt.Errorf("tx reverted: %s", signedTx.Hash().Hex())}
	}

	return SettleResult{TxHash: signedTx.Hash().Hex()}
}

type abiOrder struct {
	Maker      common.Address
	TokenSell  common.Address
	TokenBuy   common.Address
	AmountSell *big.Int
	AmountBuy  *big.Int
	Expiry     *big.Int
	Nonce      *big.Int
	Salt       *big.Int
}

func tuplifyOrder(o *domain.Order) abiOrder {
	return abiOrder{
		Maker:      common.HexToAddress(o.Maker),
		TokenSell:  common.HexToAddress(o.TokenSell),
		TokenBuy:   common.HexToAddress(o.TokenBuy),
		AmountSell: o.AmountSell,
		AmountBuy:  o.AmountBuy,
		Expiry:     new(big.Int).SetUint64(o.Expiry),
		Nonce:      new(big.Int).SetUint64(o.Nonce),
		Salt:       o.Salt,
	}
}
