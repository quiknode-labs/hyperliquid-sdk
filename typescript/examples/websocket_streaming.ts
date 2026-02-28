#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * WebSocket Streaming Example - Real-time market data via WebSocket.
 *
 * This example demonstrates:
 * - Connecting to Hyperliquid's WebSocket API
 * - Subscribing to trades, orders, and book updates
 * - Automatic reconnection handling
 * - Graceful shutdown
 *
 * Requirements:
 *     npm install hyperliquid-sdk ws
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node websocket_streaming.ts
 *
 * The SDK automatically handles URL parsing - you can pass any valid endpoint URL.
 */

import { HyperliquidSDK, StreamConnectionState } from '@quicknode/hyperliquid-sdk';

// Get endpoint from args or environment
const ENDPOINT = process.argv[2] || process.env.ENDPOINT;

if (!ENDPOINT) {
  console.log("Hyperliquid WebSocket Streaming Example");
  console.log("=".repeat(50));
  console.log();
  console.log("Usage:");
  console.log("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  console.log("  npx ts-node websocket_streaming.ts");
  console.log();
  console.log("Or:");
  console.log("  npx ts-node websocket_streaming.ts 'https://your-endpoint.example.com/TOKEN'");
  process.exit(1);
}

function onTrade(data: Record<string, unknown>) {
  const block = data.block as Record<string, unknown> | undefined;
  const events = block?.events as [string, Record<string, unknown>][] | undefined;

  if (events && Array.isArray(events)) {
    for (const [, trade] of events) {
      const coin = trade.coin || "?";
      const px = parseFloat(String(trade.px || 0));
      const sz = trade.sz || "?";
      const side = trade.side === "B" ? "BUY" : "SELL";
      console.log(`[TRADE] ${coin}: ${side} ${sz} @ $${px.toLocaleString()}`);
    }
  }
}

function onBookUpdate(data: Record<string, unknown>) {
  const block = data.block as Record<string, unknown> | undefined;
  const events = block?.events as [string, Record<string, unknown>][] | undefined;

  if (events && Array.isArray(events)) {
    for (const [, update] of events) {
      const coin = update.coin || "?";
      const levels = update.levels as { px: string; sz: string; n: number }[][] | undefined;
      if (levels && levels.length >= 2) {
        const bids = levels[0] || [];
        const asks = levels[1] || [];
        if (bids.length && asks.length) {
          const bestBid = bids[0];
          const bestAsk = asks[0];
          const bidPx = parseFloat(bestBid.px || "0");
          const askPx = parseFloat(bestAsk.px || "0");
          const spread = askPx - bidPx;
          console.log(`[BOOK] ${coin}: Bid $${bidPx.toLocaleString()} | Ask $${askPx.toLocaleString()} | Spread $${spread.toLocaleString()}`);
        }
      }
    }
  }
}

function onStateChange(state: StreamConnectionState) {
  console.log(`[STATE] ${state}`);
}

function onReconnect(attempt: number) {
  console.log(`[RECONNECT] Attempt ${attempt}`);
}

function onError(error: Error) {
  console.log(`[ERROR] ${error.message}`);
}

function onClose() {
  console.log("[CLOSED] Stream stopped");
}

async function main() {
  console.log("Hyperliquid WebSocket Streaming Example");
  console.log("=".repeat(50));
  console.log(`Endpoint: ${ENDPOINT.slice(0, 60)}${ENDPOINT.length > 60 ? '...' : ''}`);
  console.log();

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT);

  // Configure stream callbacks
  sdk.stream.onError = onError;
  sdk.stream.onClose = onClose;
  sdk.stream.onStateChange = onStateChange;
  sdk.stream.onReconnect = onReconnect;

  // Subscribe to BTC and ETH trades
  sdk.stream.trades(["BTC", "ETH"], onTrade);
  console.log("Subscribed to: BTC, ETH trades");

  // Subscribe to BTC book updates
  sdk.stream.bookUpdates(["BTC"], onBookUpdate);
  console.log("Subscribed to: BTC book updates");

  // Handle Ctrl+C gracefully
  process.on('SIGINT', () => {
    console.log("\nShutting down gracefully...");
    sdk.stream.stop();
    process.exit(0);
  });

  console.log();
  console.log("Streaming... Press Ctrl+C to stop");
  console.log("-".repeat(50));

  // Start the stream
  await sdk.stream.start();
}

main().catch(console.error);
