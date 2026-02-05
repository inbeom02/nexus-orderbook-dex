"use client";

import { formatEther } from "viem";
import { useTradesQuery } from "@/hooks/useOrderbook";

export function TradeHistory() {
  const { data: trades } = useTradesQuery();

  return (
    <div className="bg-gray-800 rounded-xl p-4">
      <h2 className="text-lg font-semibold mb-3">Recent Trades</h2>

      <div className="flex justify-between text-xs text-gray-500 px-1 mb-1">
        <span>Price</span>
        <span>Amount</span>
        <span>Time</span>
      </div>

      <div className="space-y-px max-h-64 overflow-y-auto">
        {trades?.map((trade) => (
          <div
            key={trade.id}
            className="flex justify-between text-sm font-mono px-1 py-0.5"
          >
            <span className="text-gray-300">{trade.price.toFixed(4)}</span>
            <span className="text-gray-400">
              {parseFloat(formatEther(BigInt(trade.baseAmount))).toFixed(4)}
            </span>
            <span className="text-gray-600 text-xs">
              {new Date(trade.createdAt).toLocaleTimeString()}
            </span>
          </div>
        ))}

        {(!trades || trades.length === 0) && (
          <div className="text-center py-4 text-gray-500 text-sm">
            No trades yet
          </div>
        )}
      </div>
    </div>
  );
}
