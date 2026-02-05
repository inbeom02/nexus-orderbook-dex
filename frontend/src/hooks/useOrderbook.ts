"use client";

import { useQuery } from "@tanstack/react-query";
import { getOrderbook, getTrades, getUserOrders } from "@/lib/api";

export function useOrderbookQuery(pair: string = "TKA-TKB") {
  return useQuery({
    queryKey: ["orderbook", pair],
    queryFn: () => getOrderbook(pair),
    refetchInterval: 5000,
  });
}

export function useTradesQuery(pair: string = "TKA-TKB") {
  return useQuery({
    queryKey: ["trades", pair],
    queryFn: () => getTrades(pair),
    refetchInterval: 5000,
  });
}

export function useUserOrdersQuery(address: string | undefined) {
  return useQuery({
    queryKey: ["userOrders", address],
    queryFn: () => getUserOrders(address!),
    enabled: !!address,
    refetchInterval: 5000,
  });
}
