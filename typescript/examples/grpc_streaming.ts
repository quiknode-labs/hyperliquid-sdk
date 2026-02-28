#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * gRPC Streaming Example - High-performance real-time market data via gRPC.
 *
 * This example demonstrates:
 * - Connecting to Hyperliquid's gRPC streaming API
 * - Subscribing to trades, orders, blocks, and L2/L4 order books
 * - Automatic reconnection handling
 * - Graceful shutdown
 *
 * gRPC offers lower latency than WebSocket for high-frequency data.
 *
 * Requirements:
 *     npm install hyperliquid-sdk @grpc/grpc-js @grpc/proto-loader
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node grpc_streaming.ts
 *
 * The SDK automatically:
 * - Extracts the token from any endpoint path
 * - Connects to port 10000 for gRPC
 * - Passes the token via x-token header
 * - Handles keepalive and reconnection
 */

import { HyperliquidSDK, GRPCConnectionState } from '@quicknode/hyperliquid-sdk';

// Get endpoint from args or environment
const ENDPOINT = process.argv[2] || process.env.ENDPOINT;

if (!ENDPOINT) {
  console.log("Hyperliquid gRPC Streaming Example");
  console.log("=".repeat(50));
  console.log();
  console.log("Usage:");
  console.log("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  console.log("  npx ts-node grpc_streaming.ts");
  console.log();
  console.log("Or:");
  console.log("  npx ts-node grpc_streaming.ts 'https://your-endpoint.example.com/TOKEN'");
  process.exit(1);
}

function onTrade(data: Record<string, unknown>) {
  const coin = data.coin || "?";
  const px = parseFloat(String(data.px || 0));
  const sz = data.sz || "?";
  const side = data.side === "B" ? "BUY" : "SELL";
  console.log(`[TRADE] ${coin}: ${side} ${sz} @ $${px.toLocaleString()}`);
}

function onBookUpdate(data: Record<string, unknown>) {
  const coin = data.coin || "?";
  const bids = (data.bids as Record<string, unknown>[]) || [];
  const asks = (data.asks as Record<string, unknown>[]) || [];

  if (bids.length && asks.length) {
    const bestBid = bids[0];
    const bestAsk = asks[0];
    console.log(`[BOOK] ${coin}: Bid $${parseFloat(String(bestBid.price || 0)).toLocaleString()} | Ask $${parseFloat(String(bestAsk.price || 0)).toLocaleString()}`);
  }
}

function onL2Book(data: Record<string, unknown>) {
  const coin = data.coin || "?";
  const bids = (data.bids as unknown[]) || [];
  const asks = (data.asks as unknown[]) || [];
  console.log(`[L2] ${coin}: ${bids.slice(0, 3).length} bid levels, ${asks.slice(0, 3).length} ask levels`);
}

function onBlock(data: Record<string, unknown>) {
  const blockNum = data.block_number || "?";
  console.log(`[BLOCK] #${blockNum}`);
}

function onStateChange(state: GRPCConnectionState) {
  console.log(`[STATE] ${state}`);
}

function onReconnect(attempt: number) {
  console.log(`[RECONNECT] Attempt ${attempt}`);
}

function onError(error: Error) {
  console.log(`[ERROR] ${error.message}`);
}

function onClose() {
  console.log("[CLOSED] gRPC stream stopped");
}

function onConnect() {
  console.log("[CONNECTED] gRPC stream ready");
}

async function main() {
  if (!ENDPOINT) {
    throw new Error("ENDPOINT not set");
  }

  console.log("Hyperliquid gRPC Streaming Example");
  console.log("=".repeat(50));
  console.log(`Endpoint: ${ENDPOINT.slice(0, 60)}${ENDPOINT.length > 60 ? '...' : ''}`);
  console.log();

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT);

  // Subscribe to BTC and ETH trades
  sdk.grpc.trades(["BTC", "ETH"], onTrade);
  console.log("Subscribed to: BTC, ETH trades");

  // Subscribe to book updates
  sdk.grpc.bookUpdates(["BTC"], onBookUpdate);
  console.log("Subscribed to: BTC book updates");

  // Subscribe to L2 order book
  sdk.grpc.l2Book("ETH", onL2Book);
  console.log("Subscribed to: ETH L2 order book");

  // Subscribe to blocks
  sdk.grpc.blocks(onBlock);
  console.log("Subscribed to: blocks");

  // Handle Ctrl+C gracefully
  process.on('SIGINT', () => {
    console.log("\nShutting down gracefully...");
    sdk.grpc.stop();
    process.exit(0);
  });

  console.log();
  console.log("Streaming via gRPC (port 10000)... Press Ctrl+C to stop");
  console.log("-".repeat(50));

  // Start the stream
  await sdk.grpc.start();
}

main().catch(console.error);
