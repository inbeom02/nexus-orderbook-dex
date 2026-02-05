export type Side = "buy" | "sell";

export type OrderStatus = "open" | "partially_filled" | "filled" | "cancelled";

export interface Order {
  id: string;
  maker: string;
  tokenSell: string;
  tokenBuy: string;
  amountSell: string;
  amountBuy: string;
  expiry: number;
  nonce: number;
  salt: string;
  signature: string;
  side: Side;
  status: OrderStatus;
  filledBase: string;
  pair: string;
  createdAt: string;
  updatedAt: string;
}

export interface OrderSubmission {
  maker: string;
  tokenSell: string;
  tokenBuy: string;
  amountSell: string;
  amountBuy: string;
  expiry: number;
  nonce: number;
  salt: string;
  signature: string;
  side: Side;
  pair: string;
}

export interface Trade {
  id: string;
  buyOrderId: string;
  sellOrderId: string;
  buyer: string;
  seller: string;
  pair: string;
  baseAmount: string;
  quoteAmount: string;
  price: number;
  txHash: string;
  settledOnChain: boolean;
  createdAt: string;
}

export interface PriceLevel {
  price: number;
  amount: string;
  count: number;
}

export interface OrderbookSnapshot {
  bids: PriceLevel[];
  asks: PriceLevel[];
}
