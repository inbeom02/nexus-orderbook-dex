import type { Order, OrderSubmission, OrderbookSnapshot, Trade } from "@/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

export async function submitOrder(
  order: OrderSubmission
): Promise<{ order: Order; matches: number }> {
  return fetchJSON("/api/orders", {
    method: "POST",
    body: JSON.stringify(order),
  });
}

export async function getUserOrders(address: string): Promise<Order[]> {
  return fetchJSON(`/api/orders/${address}`);
}

export async function cancelOrder(id: string): Promise<{ status: string }> {
  return fetchJSON(`/api/orders/${id}`, { method: "DELETE" });
}

export async function getOrderbook(
  pair: string = "TKA-TKB"
): Promise<OrderbookSnapshot> {
  return fetchJSON(`/api/orderbook?pair=${pair}`);
}

export async function getTrades(
  pair: string = "TKA-TKB",
  limit: number = 50
): Promise<Trade[]> {
  return fetchJSON(`/api/trades?pair=${pair}&limit=${limit}`);
}
