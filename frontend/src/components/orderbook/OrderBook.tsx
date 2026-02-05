"use client";

import { formatEther } from "viem";
import { useWebSocket } from "@/hooks/useWebSocket";
import type { PriceLevel } from "@/types";

function formatAmount(wei: string): string {
  try {
    return parseFloat(formatEther(BigInt(wei))).toFixed(4);
  } catch {
    return "0.0000";
  }
}

function PriceLevelRow({
  level,
  side,
  maxAmount,
}: {
  level: PriceLevel;
  side: "bid" | "ask";
  maxAmount: number;
}) {
  const amount = parseFloat(formatEther(BigInt(level.amount)));
  const width = maxAmount > 0 ? (amount / maxAmount) * 100 : 0;
  const color = side === "bid" ? "bg-green-900/40" : "bg-red-900/40";
  const textColor = side === "bid" ? "text-green-400" : "text-red-400";

  return (
    <div className="relative flex justify-between text-sm font-mono px-2 py-0.5">
      <div
        className={`absolute inset-y-0 ${
          side === "bid" ? "right-0" : "left-0"
        } ${color}`}
        style={{ width: `${Math.min(width, 100)}%` }}
      />
      <span className={`relative z-10 ${textColor}`}>
        {level.price.toFixed(4)}
      </span>
      <span className="relative z-10 text-gray-300">
        {formatAmount(level.amount)}
      </span>
    </div>
  );
}

export function OrderBook() {
  const { orderbook, connected } = useWebSocket();

  const askLevels = [...(orderbook.asks || [])].reverse().slice(-10);
  const bidLevels = (orderbook.bids || []).slice(0, 10);

  const allAmounts = [...askLevels, ...bidLevels].map((l) => {
    try {
      return parseFloat(formatEther(BigInt(l.amount)));
    } catch {
      return 0;
    }
  });
  const maxAmount = Math.max(...allAmounts, 1);

  const spread =
    askLevels.length > 0 && bidLevels.length > 0
      ? askLevels[askLevels.length - 1].price - bidLevels[0].price
      : 0;

  return (
    <div className="bg-gray-800 rounded-xl p-4">
      <div className="flex justify-between items-center mb-3">
        <h2 className="text-lg font-semibold">Order Book</h2>
        <span
          className={`text-xs ${connected ? "text-green-400" : "text-red-400"}`}
        >
          {connected ? "LIVE" : "OFFLINE"}
        </span>
      </div>

      <div className="flex justify-between text-xs text-gray-500 px-2 mb-1">
        <span>Price (TKB)</span>
        <span>Amount (TKA)</span>
      </div>

      <div className="space-y-px">
        {askLevels.map((level, i) => (
          <PriceLevelRow
            key={`ask-${i}`}
            level={level}
            side="ask"
            maxAmount={maxAmount}
          />
        ))}
      </div>

      <div className="text-center py-2 text-sm text-gray-500 border-y border-gray-700 my-1">
        Spread: {spread.toFixed(4)}
      </div>

      <div className="space-y-px">
        {bidLevels.map((level, i) => (
          <PriceLevelRow
            key={`bid-${i}`}
            level={level}
            side="bid"
            maxAmount={maxAmount}
          />
        ))}
      </div>

      {askLevels.length === 0 && bidLevels.length === 0 && (
        <div className="text-center py-8 text-gray-500 text-sm">
          No orders yet
        </div>
      )}
    </div>
  );
}
