"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import type { OrderbookSnapshot } from "@/types";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

export function useWebSocket(pair: string = "TKA-TKB") {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [orderbook, setOrderbook] = useState<OrderbookSnapshot>({
    bids: [],
    asks: [],
  });
  const [connected, setConnected] = useState(false);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    const ws = new WebSocket(`${WS_URL}/ws?pair=${pair}`);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "snapshot" || data.type === "orderbook") {
          setOrderbook({
            bids: data.bids || [],
            asks: data.asks || [],
          });
        }
      } catch {
        // ignore parse errors
      }
    };

    ws.onclose = () => {
      setConnected(false);
      // Exponential backoff reconnect
      const delay = Math.min(
        1000 * Math.pow(2, Math.random() * 3),
        10000
      );
      reconnectTimeout.current = setTimeout(connect, delay);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [pair]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimeout.current) clearTimeout(reconnectTimeout.current);
      wsRef.current?.close();
    };
  }, [connect]);

  return { orderbook, connected };
}
