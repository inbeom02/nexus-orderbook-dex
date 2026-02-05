package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type OrderbookCache struct {
	client *redis.Client
}

func NewOrderbookCache(client *redis.Client) *OrderbookCache {
	return &OrderbookCache{client: client}
}

type PriceLevelData struct {
	Price  float64 `json:"price"`
	Amount string  `json:"amount"`
	Count  int     `json:"count"`
}

func (c *OrderbookCache) SetSnapshot(ctx context.Context, pair string, bids, asks []PriceLevelData) error {
	bidsJSON, err := json.Marshal(bids)
	if err != nil {
		return err
	}
	asksJSON, err := json.Marshal(asks)
	if err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	pipe.Set(ctx, fmt.Sprintf("ob:%s:bids", pair), bidsJSON, 0)
	pipe.Set(ctx, fmt.Sprintf("ob:%s:asks", pair), asksJSON, 0)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *OrderbookCache) GetSnapshot(ctx context.Context, pair string) (bids, asks []PriceLevelData, err error) {
	bidsJSON, err := c.client.Get(ctx, fmt.Sprintf("ob:%s:bids", pair)).Bytes()
	if err != nil && err != redis.Nil {
		return nil, nil, err
	}
	asksJSON, err := c.client.Get(ctx, fmt.Sprintf("ob:%s:asks", pair)).Bytes()
	if err != nil && err != redis.Nil {
		return nil, nil, err
	}

	if len(bidsJSON) > 0 {
		json.Unmarshal(bidsJSON, &bids)
	}
	if len(asksJSON) > 0 {
		json.Unmarshal(asksJSON, &asks)
	}
	return bids, asks, nil
}

// PublishUpdate sends a real-time update to subscribers.
func (c *OrderbookCache) PublishUpdate(ctx context.Context, pair string, data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Publish(ctx, fmt.Sprintf("ob:updates:%s", pair), msg).Err()
}

// Subscribe returns a channel for orderbook updates.
func (c *OrderbookCache) Subscribe(ctx context.Context, pair string) *redis.PubSub {
	return c.client.Subscribe(ctx, fmt.Sprintf("ob:updates:%s", pair))
}
