"use client";

import { useState } from "react";
import { useAccount } from "wagmi";
import { formatEther } from "viem";
import { useDeposit, useVaultBalance } from "@/hooks/useDeposit";
import { ADDRESSES } from "@/lib/contracts";

const tokens = [
  { symbol: "TKA", address: ADDRESSES.tokenA },
  { symbol: "TKB", address: ADDRESSES.tokenB },
] as const;

export function DepositWithdraw() {
  const { isConnected } = useAccount();
  const [selectedToken, setSelectedToken] = useState<0 | 1>(0);
  const [amount, setAmount] = useState("");
  const [action, setAction] = useState<"deposit" | "withdraw" | "mint">(
    "deposit"
  );
  const { approve, deposit, withdraw, mint, isConfirming } = useDeposit();
  const { data: balanceA } = useVaultBalance(ADDRESSES.tokenA);
  const { data: balanceB } = useVaultBalance(ADDRESSES.tokenB);

  if (!isConnected) return null;

  const token = tokens[selectedToken];
  const vaultBalance = selectedToken === 0 ? balanceA : balanceB;

  const handleAction = async () => {
    if (!amount) return;
    try {
      if (action === "mint") {
        await mint(token.address, amount);
      } else if (action === "deposit") {
        await approve(token.address, amount);
        await deposit(token.address, amount);
      } else {
        await withdraw(token.address, amount);
      }
      setAmount("");
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div className="bg-gray-800 rounded-xl p-4">
      <h2 className="text-lg font-semibold mb-3">Vault</h2>

      <div className="flex gap-2 mb-3">
        {tokens.map((t, i) => (
          <button
            key={t.symbol}
            onClick={() => setSelectedToken(i as 0 | 1)}
            className={`px-3 py-1 rounded-lg text-sm ${
              selectedToken === i
                ? "bg-blue-600"
                : "bg-gray-700 hover:bg-gray-600"
            }`}
          >
            {t.symbol}
          </button>
        ))}
      </div>

      <div className="text-sm text-gray-400 mb-3">
        Vault Balance:{" "}
        <span className="text-white font-mono">
          {vaultBalance ? formatEther(vaultBalance as bigint) : "0"}{" "}
          {token.symbol}
        </span>
      </div>

      <div className="flex gap-1 mb-3">
        {(["deposit", "withdraw", "mint"] as const).map((a) => (
          <button
            key={a}
            onClick={() => setAction(a)}
            className={`px-3 py-1 rounded text-xs capitalize ${
              action === a ? "bg-blue-600" : "bg-gray-700 hover:bg-gray-600"
            }`}
          >
            {a}
          </button>
        ))}
      </div>

      <div className="flex gap-2">
        <input
          type="number"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="Amount"
          className="flex-1 bg-gray-700 rounded-lg px-3 py-2 text-sm outline-none"
        />
        <button
          onClick={handleAction}
          disabled={isConfirming || !amount}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-600 rounded-lg text-sm font-medium transition-colors"
        >
          {isConfirming ? "..." : action}
        </button>
      </div>
    </div>
  );
}
