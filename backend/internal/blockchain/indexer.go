package blockchain

import (
	"context"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const eventsABI = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"user","type":"address"},{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"Deposit","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"user","type":"address"},{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"Withdraw","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"buyOrderHash","type":"bytes32"},{"indexed":true,"internalType":"bytes32","name":"sellOrderHash","type":"bytes32"},{"indexed":false,"internalType":"address","name":"buyer","type":"address"},{"indexed":false,"internalType":"address","name":"seller","type":"address"},{"indexed":false,"internalType":"uint256","name":"baseAmount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"quoteAmount","type":"uint256"}],"name":"TradeSettled","type":"event"}]`

type EventHandler interface {
	OnDeposit(user, token common.Address, amount *big.Int)
	OnWithdraw(user, token common.Address, amount *big.Int)
	OnTradeSettled(buyHash, sellHash common.Hash, buyer, seller common.Address, baseAmount, quoteAmount *big.Int)
}

type Indexer struct {
	client  *Client
	handler EventHandler
}

func NewIndexer(client *Client, handler EventHandler) *Indexer {
	return &Indexer{client: client, handler: handler}
}

func (idx *Indexer) Start(ctx context.Context) {
	parsed, err := abi.JSON(strings.NewReader(eventsABI))
	if err != nil {
		log.Fatalf("Failed to parse events ABI: %v", err)
	}

	depositSig := parsed.Events["Deposit"].ID
	withdrawSig := parsed.Events["Withdraw"].ID
	tradeSettledSig := parsed.Events["TradeSettled"].ID

	query := ethereum.FilterQuery{
		Addresses: []common.Address{idx.client.Contract},
	}

	// Poll-based indexing with exponential backoff
	var lastBlock uint64
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		currentBlock, err := idx.client.EthClient.BlockNumber(ctx)
		if err != nil {
			log.Printf("Indexer: failed to get block number: %v", err)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		if currentBlock <= lastBlock {
			time.Sleep(2 * time.Second)
			continue
		}

		backoff = time.Second
		query.FromBlock = new(big.Int).SetUint64(lastBlock + 1)
		query.ToBlock = new(big.Int).SetUint64(currentBlock)

		logs, err := idx.client.EthClient.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Indexer: failed to filter logs: %v", err)
			time.Sleep(backoff)
			continue
		}

		for _, vLog := range logs {
			idx.processLog(vLog, depositSig, withdrawSig, tradeSettledSig, &parsed)
		}

		lastBlock = currentBlock
	}
}

func (idx *Indexer) processLog(vLog types.Log, depositSig, withdrawSig, tradeSettledSig common.Hash, parsed *abi.ABI) {
	if len(vLog.Topics) == 0 {
		return
	}

	switch vLog.Topics[0] {
	case depositSig:
		user := common.BytesToAddress(vLog.Topics[1].Bytes())
		token := common.BytesToAddress(vLog.Topics[2].Bytes())
		amount := new(big.Int).SetBytes(vLog.Data)
		idx.handler.OnDeposit(user, token, amount)

	case withdrawSig:
		user := common.BytesToAddress(vLog.Topics[1].Bytes())
		token := common.BytesToAddress(vLog.Topics[2].Bytes())
		amount := new(big.Int).SetBytes(vLog.Data)
		idx.handler.OnWithdraw(user, token, amount)

	case tradeSettledSig:
		buyHash := vLog.Topics[1]
		sellHash := vLog.Topics[2]
		vals, err := parsed.Events["TradeSettled"].Inputs.NonIndexed().Unpack(vLog.Data)
		if err != nil {
			log.Printf("Indexer: failed to unpack TradeSettled: %v", err)
			return
		}
		buyer := vals[0].(common.Address)
		seller := vals[1].(common.Address)
		baseAmount := vals[2].(*big.Int)
		quoteAmount := vals[3].(*big.Int)
		idx.handler.OnTradeSettled(buyHash, sellHash, buyer, seller, baseAmount, quoteAmount)
	}
}
