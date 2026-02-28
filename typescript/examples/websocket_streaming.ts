#!/usr/bin/env npx ts-node
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
 *     export ENDPOINT="https://YOUR-ENDPOINT.hype-mainnet.quiknode.pro/YOUR-TOKEN"
 *     npx ts-node websocket_streaming.ts
 *
 * The SDK automatically handles URL parsing - you can pass:
 * - https://x.quiknode.pro/TOKEN
 * - https://x.quiknode.pro/TOKEN/info
 * - https://x.quiknode.pro/TOKEN/hypercore
 *
 * All will work correctly - the SDK extracts the token and builds the right WebSocket URL.
 */

import { Stream, StreamConnectionState } from 'hyperliquid-sdk';

// Get endpoint from args or environment
const ENDPOINT = process.argv[2] || process.env.ENDPOINT || process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.log("Hyperliquid WebSocket Streaming Example");
  console.log("=".repeat(50));
  console.log();
  console.log("Usage:");
  console.log("  export ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'");
  console.log("  npx ts-node websocket_streaming.ts");
  console.log();
  console.log("Or:");
  console.log("  npx ts-node websocket_streaming.ts 'https://YOUR-ENDPOINT.quiknode.pro/TOKEN'");
  process.exit(1);
}

function onTrade(data: Record<string, unknown>) {
  const trades = data.data as Record<string, unknown>[] | Record<string, unknown>;

  if (Array.isArray(trades)) {
    for (const trade of trades) {
      const coin = trade.coin || "?";
      const px = parseFloat(String(trade.px || 0));
      const sz = trade.sz || "?";
      const side = trade.side === "B" ? "BUY" : "SELL";
      console.log(`[TRADE] ${coin}: ${side} ${sz} @ $${px.toLocaleString()}`);
    }
  } else if (trades) {
    const coin = trades.coin || "?";
    const px = parseFloat(String(trades.px || 0));
    const sz = trades.sz || "?";
    const side = trades.side === "B" ? "BUY" : "SELL";
    console.log(`[TRADE] ${coin}: ${side} ${sz} @ $${px.toLocaleString()}`);
  }
}

function onBookUpdate(data: Record<string, unknown>) {
  const bookData = data.data as Record<string, unknown> || {};
  const coin = bookData.coin || "?";
  const levels = bookData.levels as [unknown[], unknown[]] || [[], []];
  const bids = (levels[0] || []) as Record<string, unknown>[];
  const asks = (levels[1] || []) as Record<string, unknown>[];

  if (bids.length && asks.length) {
    const bestBid = bids[0];
    const bestAsk = asks[0];
    const spread = parseFloat(String(bestAsk.px || 0)) - parseFloat(String(bestBid.px || 0));
    console.log(`[BOOK] ${coin}: Bid $${parseFloat(String(bestBid.px || 0)).toLocaleString()} | Ask $${parseFloat(String(bestAsk.px || 0)).toLocaleString()} | Spread $${spread.toLocaleString()}`);
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

  // Create stream with all callbacks
  const stream = new Stream(ENDPOINT, {
    onError,
    onClose,
    onStateChange,
    onReconnect,
    reconnect: true,
    pingInterval: 30000,
  });

  // Subscribe to BTC and ETH trades
  stream.trades(["BTC", "ETH"], onTrade);
  console.log("Subscribed to: BTC, ETH trades");

  // Subscribe to BTC book updates
  stream.bookUpdates(["BTC"], onBookUpdate);
  console.log("Subscribed to: BTC book updates");

  // Handle Ctrl+C gracefully
  process.on('SIGINT', () => {
    console.log("\nShutting down gracefully...");
    stream.stop();
    process.exit(0);
  });

  console.log();
  console.log("Streaming... Press Ctrl+C to stop");
  console.log("-".repeat(50));

  // Start the stream
  await stream.start();
}

main().catch(console.error);
