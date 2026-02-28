#!/usr/bin/env npx ts-node
/**
 * WebSocket Streaming — Complete Reference
 *
 * This example demonstrates ALL WebSocket subscription types:
 * - Market Data: trades, l2_book, book_updates, all_mids, candle, bbo
 * - User Data: open_orders, user_fills, user_fundings, clearinghouse_state
 * - TWAP: twap, twap_states, user_twap_slice_fills
 * - System: events, notification
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node stream_websocket_all.ts
 */

import { Stream } from 'hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;
const USER = process.env.USER_ADDRESS || "0x0000000000000000000000000000000000000000";

if (!ENDPOINT) {
  console.log("WebSocket Complete Reference");
  console.log("=".repeat(60));
  console.log();
  console.log("Usage:");
  console.log("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'");
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

  const stream = new Stream(ENDPOINT!, { reconnect: false });

  // trades: Real-time executed trades
  stream.trades(["BTC", "ETH"], makeCallback("trades"));

  // book_updates: Incremental order book changes
  stream.bookUpdates(["BTC"], makeCallback("book_updates"));

  // l2_book: Full L2 order book snapshots
  stream.l2Book("BTC", makeCallback("l2_book"));

  console.log("Starting market data streams for 10 seconds...");
  console.log("-".repeat(60));

  await stream.start();

  await new Promise(resolve => setTimeout(resolve, 10000));

  stream.stop();

  console.log("\nMarket data summary:");
  for (const [name, count] of Object.entries(counts)) {
    console.log(`  ${name}: ${count} messages`);
  }
}

async function main() {
  console.log("WebSocket Streaming — Complete Reference");
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
