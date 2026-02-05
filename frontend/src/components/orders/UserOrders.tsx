"use client";

import { useAccount } from "wagmi";
import { formatEther } from "viem";
import { useUserOrdersQuery } from "@/hooks/useOrderbook";
import { cancelOrder } from "@/lib/api";
import { useState } from "react";

export function UserOrders() {
  const { address } = useAccount();
  const { data: orders, refetch } = useUserOrdersQuery(address);
  const [cancelling, setCancelling] = useState<string | null>(null);

  if (!address) return null;

  const handleCancel = async (id: string) => {
    setCancelling(id);
    try {
      await cancelOrder(id);
      refetch();
    } catch (err) {
      console.error(err);
    } finally {
      setCancelling(null);
    }
  };

  const openOrders = orders?.filter(
    (o) => o.status === "open" || o.status === "partially_filled"
  );

  return (
    <div className="bg-gray-800 rounded-xl p-4">
      <h2 className="text-lg font-semibold mb-3">Your Orders</h2>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-xs text-gray-500 border-b border-gray-700">
              <th className="text-left py-2">Side</th>
              <th className="text-right py-2">Price</th>
              <th className="text-right py-2">Amount</th>
              <th className="text-right py-2">Filled</th>
              <th className="text-right py-2">Status</th>
              <th className="text-right py-2"></th>
            </tr>
          </thead>
          <tbody>
            {openOrders?.map((order) => {
              const isBuy = order.side === "buy";
              const baseAmount = isBuy ? order.amountBuy : order.amountSell;
              const quoteAmount = isBuy ? order.amountSell : order.amountBuy;
              const price =
                parseFloat(formatEther(BigInt(quoteAmount))) /
                parseFloat(formatEther(BigInt(baseAmount)));

              return (
                <tr key={order.id} className="border-b border-gray-700/50">
                  <td
                    className={`py-1.5 ${isBuy ? "text-green-400" : "text-red-400"}`}
                  >
                    {order.side.toUpperCase()}
                  </td>
                  <td className="text-right font-mono">{price.toFixed(4)}</td>
                  <td className="text-right font-mono">
                    {parseFloat(formatEther(BigInt(baseAmount))).toFixed(4)}
                  </td>
                  <td className="text-right font-mono text-gray-400">
                    {parseFloat(formatEther(BigInt(order.filledBase))).toFixed(4)}
                  </td>
                  <td className="text-right text-xs text-gray-400">
                    {order.status}
                  </td>
                  <td className="text-right">
                    <button
                      onClick={() => handleCancel(order.id)}
                      disabled={cancelling === order.id}
                      className="text-xs text-red-400 hover:text-red-300 disabled:text-gray-600"
                    >
                      {cancelling === order.id ? "..." : "Cancel"}
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {(!openOrders || openOrders.length === 0) && (
          <div className="text-center py-4 text-gray-500 text-sm">
            No open orders
          </div>
        )}
      </div>
    </div>
  );
}
