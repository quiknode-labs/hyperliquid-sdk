#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * WebSocket Streaming - Complete Reference
 *
 * This example demonstrates ALL WebSocket subscription types:
 * - Market Data: trades, l2_book, book_updates, all_mids, candle, bbo
 * - User Data: open_orders, user_fills, user_fundings, clearinghouse_state
 * - TWAP: twap, twap_states, user_twap_slice_fills
 * - System: events, notification
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node stream_websocket_all.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.ENDPOINT;
const USER = process.env.USER_ADDRESS || "0x0000000000000000000000000000000000000000";

if (!ENDPOINT) {
  console.log("WebSocket Complete Reference");
  console.log("=".repeat(60));
  console.log();
  console.log("Usage:");
  console.log("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  console.log("  export USER_ADDRESS='0x...'  # Optional, for user data streams");
  console.log("  npx ts-node stream_websocket_all.ts");
  process.exit(1);
}

function timestamp(): string {
  return new Date().toISOString().slice(11, 23);
}

// Global counters
const counts: Record<string, number> = {};

function makeCallback(name: string, maxPrints = 3) {
  counts[name] = 0;

  return (data: any) => {
    counts[name]++;
    if (counts[name] <= maxPrints) {
      const channel = data.channel || "unknown";
      console.log(`[${timestamp()}] ${name.toUpperCase()}: ${channel} (#${counts[name]})`);
      // Print first few fields of data
      const innerData = data.data || data;
      if (typeof innerData === "object" && !Array.isArray(innerData)) {
        const keys = Object.keys(innerData).slice(0, 3);
        console.log(`             Fields: ${keys.join(", ")}`);
      } else if (Array.isArray(innerData)) {
        console.log(`             Items: ${innerData.length}`);
      }
    }
  };
}

async function demoMarketData() {
  console.log("\n" + "=".repeat(60));
  console.log("MARKET DATA STREAMS");
  console.log("=".repeat(60));
  console.log();
  console.log("Available streams:");
  console.log("  - trades(coins, callback)");
  console.log("  - bookUpdates(coins, callback)");
  console.log("  - l2Book(coin, callback)");
  console.log("  - allMids(callback)");
  console.log("  - candle(coin, interval, callback)");
  console.log("  - bbo(coin, callback)");
  console.log();

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT!);

  // trades: Real-time executed trades
  sdk.stream.trades(["BTC", "ETH"], makeCallback("trades"));

  // book_updates: Incremental order book changes
  sdk.stream.bookUpdates(["BTC"], makeCallback("book_updates"));

  // l2_book: Full L2 order book snapshots
  sdk.stream.l2Book("BTC", makeCallback("l2_book"));

  console.log("Starting market data streams for 10 seconds...");
  console.log("-".repeat(60));

  await sdk.stream.start();

  await new Promise(resolve => setTimeout(resolve, 10000));

  sdk.stream.stop();

  console.log("\nMarket data summary:");
  for (const [name, count] of Object.entries(counts)) {
    console.log(`  ${name}: ${count} messages`);
  }
}

async function main() {
  console.log("WebSocket Streaming - Complete Reference");
  console.log("=".repeat(60));

  await demoMarketData();

  console.log("\n" + "=".repeat(60));
  console.log("Done!");
  console.log();
  console.log("Other available streams (not shown):");
  console.log("  Market: allMids, candle, bbo, activeAssetCtx");
  console.log("  User: openOrders, userFills, userFundings, clearinghouseState");
  console.log("  TWAP: twap, twapStates, userTwapSliceFills");
  console.log("  System: events, notification, writerActions");
}

main().catch(console.error);
