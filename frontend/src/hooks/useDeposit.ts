"use client";

import { useCallback } from "react";
import {
  useAccount,
  useWriteContract,
  useWaitForTransactionReceipt,
  useReadContract,
} from "wagmi";
import { parseEther, maxUint256 } from "viem";
import { NEXUS_ORDERBOOK_ABI, ERC20_ABI, ADDRESSES } from "@/lib/contracts";

export function useDeposit() {
  const { address } = useAccount();
  const { writeContractAsync, data: txHash } = useWriteContract();
  const { isLoading: isConfirming } = useWaitForTransactionReceipt({
    hash: txHash,
  });

  const approve = useCallback(
    async (token: `0x${string}`, amount: string) => {
      return writeContractAsync({
        address: token,
        abi: ERC20_ABI,
        functionName: "approve",
        args: [ADDRESSES.nexusOrderBook, parseEther(amount)],
      });
    },
    [writeContractAsync]
  );

  const approveMax = useCallback(
    async (token: `0x${string}`) => {
      return writeContractAsync({
        address: token,
        abi: ERC20_ABI,
        functionName: "approve",
        args: [ADDRESSES.nexusOrderBook, maxUint256],
      });
    },
    [writeContractAsync]
  );

  const deposit = useCallback(
    async (token: `0x${string}`, amount: string) => {
      return writeContractAsync({
        address: ADDRESSES.nexusOrderBook,
        abi: NEXUS_ORDERBOOK_ABI,
        functionName: "deposit",
        args: [token, parseEther(amount)],
      });
    },
    [writeContractAsync]
  );

  const withdraw = useCallback(
    async (token: `0x${string}`, amount: string) => {
      return writeContractAsync({
        address: ADDRESSES.nexusOrderBook,
        abi: NEXUS_ORDERBOOK_ABI,
        functionName: "withdraw",
        args: [token, parseEther(amount)],
      });
    },
    [writeContractAsync]
  );

  const mint = useCallback(
    async (token: `0x${string}`, amount: string) => {
      if (!address) throw new Error("Not connected");
      return writeContractAsync({
        address: token,
        abi: ERC20_ABI,
        functionName: "mint",
        args: [address, parseEther(amount)],
      });
    },
    [address, writeContractAsync]
  );

  return { approve, approveMax, deposit, withdraw, mint, isConfirming, txHash };
}

export function useVaultBalance(token: `0x${string}`) {
  const { address } = useAccount();
  return useReadContract({
    address: ADDRESSES.nexusOrderBook,
    abi: NEXUS_ORDERBOOK_ABI,
    functionName: "getBalance",
    args: address ? [address, token] : undefined,
    query: { enabled: !!address },
  });
}
