"use client";

import { useState } from "react";
import { useAccount } from "wagmi";
import { useSignOrder } from "@/hooks/useSignOrder";
import type { Side } from "@/types";

export function OrderForm() {
  const { isConnected } = useAccount();
  const { signAndSubmit } = useSignOrder();
  const [side, setSide] = useState<Side>("buy");
  const [price, setPrice] = useState("");
  const [amount, setAmount] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess("");

    if (!price || !amount) {
      setError("Price and amount required");
      return;
    }

    setLoading(true);
    try {
      const result = await signAndSubmit({ side, price, amount });
      setSuccess(
        `Order placed! ${result.matches > 0 ? `${result.matches} match(es)` : "Resting on book"}`
      );
      setPrice("");
      setAmount("");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to place order");
    } finally {
      setLoading(false);
    }
  };

  if (!isConnected) {
    return (
      <div className="bg-gray-800 rounded-xl p-4">
        <h2 className="text-lg font-semibold mb-3">Place Order</h2>
        <p className="text-gray-500 text-sm text-center py-4">
          Connect wallet to trade
        </p>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-xl p-4">
      <h2 className="text-lg font-semibold mb-3">Place Order</h2>

      <div className="flex gap-1 mb-4">
        <button
          onClick={() => setSide("buy")}
          className={`flex-1 py-2 rounded-lg font-medium text-sm ${
            side === "buy"
              ? "bg-green-600 text-white"
              : "bg-gray-700 text-gray-400 hover:bg-gray-600"
          }`}
        >
          Buy
        </button>
        <button
          onClick={() => setSide("sell")}
          className={`flex-1 py-2 rounded-lg font-medium text-sm ${
            side === "sell"
              ? "bg-red-600 text-white"
              : "bg-gray-700 text-gray-400 hover:bg-gray-600"
          }`}
        >
          Sell
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className="text-xs text-gray-500 block mb-1">
            Price (TKB per TKA)
          </label>
          <input
            type="number"
            step="any"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            className="w-full bg-gray-700 rounded-lg px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="0.00"
          />
        </div>
        <div>
          <label className="text-xs text-gray-500 block mb-1">
            Amount (TKA)
          </label>
          <input
            type="number"
            step="any"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="w-full bg-gray-700 rounded-lg px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="0.00"
          />
        </div>

        {price && amount && (
          <div className="text-xs text-gray-500">
            Total:{" "}
            <span className="text-gray-300">
              {(parseFloat(price) * parseFloat(amount)).toFixed(4)} TKB
            </span>
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          className={`w-full py-2.5 rounded-lg font-medium text-sm transition-colors ${
            side === "buy"
              ? "bg-green-600 hover:bg-green-500 disabled:bg-green-900"
              : "bg-red-600 hover:bg-red-500 disabled:bg-red-900"
          }`}
        >
          {loading ? "Signing..." : `${side === "buy" ? "Buy" : "Sell"} TKA`}
        </button>
      </form>

      {error && (
        <div className="mt-2 text-xs text-red-400 break-all">{error}</div>
      )}
      {success && (
        <div className="mt-2 text-xs text-green-400">{success}</div>
      )}
    </div>
  );
}
