"use client";

import { useCallback } from "react";
import { useSignTypedData, useAccount } from "wagmi";
import { parseEther } from "viem";
import { EIP712_DOMAIN, ORDER_TYPES } from "@/lib/eip712";
import { ADDRESSES } from "@/lib/contracts";
import { submitOrder } from "@/lib/api";
import type { Side, OrderSubmission } from "@/types";

interface SignOrderParams {
  side: Side;
  price: string;
  amount: string;
  pair?: string;
}

export function useSignOrder() {
  const { address } = useAccount();
  const { signTypedDataAsync } = useSignTypedData();

  const signAndSubmit = useCallback(
    async ({ side, price, amount, pair = "TKA-TKB" }: SignOrderParams) => {
      if (!address) throw new Error("Wallet not connected");

      const baseAmount = parseEther(amount);
      const priceNum = parseFloat(price);
      const quoteAmount = parseEther((parseFloat(amount) * priceNum).toFixed(18));
      const expiry = BigInt(Math.floor(Date.now() / 1000) + 3600); // 1 hour
      const nonce = 0n;
      const salt = BigInt(Math.floor(Math.random() * 1e18));

      // For buy: tokenSell = quote (TKB), tokenBuy = base (TKA)
      // For sell: tokenSell = base (TKA), tokenBuy = quote (TKB)
      const tokenSell =
        side === "buy" ? ADDRESSES.tokenB : ADDRESSES.tokenA;
      const tokenBuy =
        side === "buy" ? ADDRESSES.tokenA : ADDRESSES.tokenB;
      const amountSell = side === "buy" ? quoteAmount : baseAmount;
      const amountBuy = side === "buy" ? baseAmount : quoteAmount;

      const message = {
        maker: address,
        tokenSell,
        tokenBuy,
        amountSell,
        amountBuy,
        expiry,
        nonce,
        salt,
      };

      const signature = await signTypedDataAsync({
        domain: EIP712_DOMAIN,
        types: ORDER_TYPES,
        primaryType: "Order",
        message,
      });

      const submission: OrderSubmission = {
        maker: address,
        tokenSell,
        tokenBuy,
        amountSell: amountSell.toString(),
        amountBuy: amountBuy.toString(),
        expiry: Number(expiry),
        nonce: Number(nonce),
        salt: salt.toString(),
        signature,
        side,
        pair,
      };

      return submitOrder(submission);
    },
    [address, signTypedDataAsync]
  );

  return { signAndSubmit };
}
