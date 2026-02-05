package orderbook

import (
	"container/heap"

	"github.com/nexus-orderbook-dex/backend/internal/domain"
)

// OrderEntry is a wrapper for an order in the heap.
type OrderEntry struct {
	Order *domain.Order
	Index int
}

// BuyHeap is a max-heap by price (highest price first), then by time (earliest first).
type BuyHeap []*OrderEntry

func (h BuyHeap) Len() int { return len(h) }
func (h BuyHeap) Less(i, j int) bool {
	pi := h[i].Order.Price()
	pj := h[j].Order.Price()
	if pi != pj {
		return pi > pj // max-heap: higher price = better buy
	}
	return h[i].Order.CreatedAt.Before(h[j].Order.CreatedAt) // time priority
}
func (h BuyHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}
func (h *BuyHeap) Push(x interface{}) {
	entry := x.(*OrderEntry)
	entry.Index = len(*h)
	*h = append(*h, entry)
}
func (h *BuyHeap) Pop() interface{} {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.Index = -1
	*h = old[:n-1]
	return entry
}

// SellHeap is a min-heap by price (lowest price first), then by time (earliest first).
type SellHeap []*OrderEntry

func (h SellHeap) Len() int { return len(h) }
func (h SellHeap) Less(i, j int) bool {
	pi := h[i].Order.Price()
	pj := h[j].Order.Price()
	if pi != pj {
		return pi < pj // min-heap: lower price = better sell
	}
	return h[i].Order.CreatedAt.Before(h[j].Order.CreatedAt)
}
func (h SellHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}
func (h *SellHeap) Push(x interface{}) {
	entry := x.(*OrderEntry)
	entry.Index = len(*h)
	*h = append(*h, entry)
}
func (h *SellHeap) Pop() interface{} {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.Index = -1
	*h = old[:n-1]
	return entry
}

var _ heap.Interface = (*BuyHeap)(nil)
var _ heap.Interface = (*SellHeap)(nil)
