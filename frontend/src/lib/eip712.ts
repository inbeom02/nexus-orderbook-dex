import { ADDRESSES } from "./contracts";

export const EIP712_DOMAIN = {
  name: "NexusOrderBook",
  version: "1",
  chainId: Number(process.env.NEXT_PUBLIC_CHAIN_ID || 31337),
  verifyingContract: ADDRESSES.nexusOrderBook,
} as const;

export const ORDER_TYPES = {
  Order: [
    { name: "maker", type: "address" },
    { name: "tokenSell", type: "address" },
    { name: "tokenBuy", type: "address" },
    { name: "amountSell", type: "uint256" },
    { name: "amountBuy", type: "uint256" },
    { name: "expiry", type: "uint256" },
    { name: "nonce", type: "uint256" },
    { name: "salt", type: "uint256" },
  ],
} as const;
