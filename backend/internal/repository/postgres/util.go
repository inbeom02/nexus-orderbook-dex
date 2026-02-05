package postgres

import "math/big"

func parseBigInt(s string) (*big.Int, bool) {
	n := new(big.Int)
	_, ok := n.SetString(s, 10)
	return n, ok
}
