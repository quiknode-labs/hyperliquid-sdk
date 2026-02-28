#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Order Book Streaming Example - L2 and L4 Order Books via gRPC
 *
 * This example demonstrates how to stream order book data via gRPC:
 * - L2 Book: Aggregated by price level (total size and order count per price)
 * - L4 Book: Individual orders with order IDs
 *
 * Use cases:
 * - L2 Book: Market depth, spread monitoring, analytics dashboards
 * - L4 Book: HFT, quant trading, market making, order flow analysis
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node stream_orderbook.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.ENDPOINT;

if (!ENDPOINT) {
  console.log("Order Book Streaming Example");
  console.log("=".repeat(60));
  console.log();
  console.log("Usage:");
  console.log("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  console.log("  npx ts-node stream_orderbook.ts");
  process.exit(1);
}

function timestamp(): string {
  return new Date().toISOString().slice(11, 23);
}

async function streamL2Example() {
  console.log("\n" + "=".repeat(60));
  console.log("L2 ORDER BOOK (Aggregated Price Levels)");
  console.log("=".repeat(60));

  let count = 0;

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT!);

  sdk.grpc.l2Book("BTC", (data: any) => {
    count++;
    const bids = data.bids || [];
    const asks = data.asks || [];

    if (bids.length > 0 && asks.length > 0) {
      const bestBid = parseFloat(bids[0][0] || "0");
      const bestAsk = parseFloat(asks[0][0] || "0");
      const spread = bestAsk - bestBid;
      const mid = (bestBid + bestAsk) / 2;
      const spreadBps = (spread / mid) * 10000;

      console.log(`[${timestamp()}] BTC L2 Update #${count}`);
      console.log(`  Best Bid: $${bestBid.toLocaleString()}`);
      console.log(`  Best Ask: $${bestAsk.toLocaleString()}`);
      console.log(`  Spread: $${spread.toFixed(2)} (${spreadBps.toFixed(2)} bps)`);
      console.log(`  Depth: ${bids.length} bid levels, ${asks.length} ask levels`);
    }
  }, { nLevels: 10 });

  await sdk.grpc.start();

  const start = Date.now();
  while (count < 3 && Date.now() - start < 15000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  sdk.grpc.stop();
  console.log(`\nReceived ${count} L2 updates.`);
}

async function streamL4Example() {
  console.log("\n" + "=".repeat(60));
  console.log("L4 ORDER BOOK (Individual Orders)");
  console.log("=".repeat(60));

  let count = 0;

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT!);

  sdk.grpc.l4Book("ETH", (data: any) => {
    count++;

    if (data.type === "snapshot") {
      const bids = data.bids || [];
      const asks = data.asks || [];
      console.log(`[${timestamp()}] ETH L4 Snapshot`);
      console.log(`  ${bids.length} individual bid orders`);
      console.log(`  ${asks.length} individual ask orders`);
    } else {
      console.log(`[${timestamp()}] ETH L4 Diff (height: ${data.height})`);
    }
  });

  await sdk.grpc.start();

  const start = Date.now();
  while (count < 3 && Date.now() - start < 20000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  sdk.grpc.stop();
  console.log(`\nReceived ${count} L4 updates.`);
}

async function main() {
  console.log("Order Book Streaming Examples");
  console.log("=".repeat(60));

  await streamL2Example();
  await streamL4Example();

  console.log("\n" + "=".repeat(60));
  console.log("Done!");
}

main().catch(console.error);
