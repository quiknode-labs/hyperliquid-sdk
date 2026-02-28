#!/usr/bin/env npx ts-node
/**
 * gRPC Streaming Example — High-Performance Real-Time Data
 *
 * Stream trades, orders, L2 book, L4 book, and blocks via gRPC.
 * gRPC provides lower latency than WebSocket for high-frequency trading.
 *
 * gRPC is included with all QuickNode Hyperliquid endpoints — no add-on needed.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node stream_grpc.ts
 *
 * The SDK:
 * - Connects to port 10000 automatically
 * - Passes token via x-token header
 * - Handles reconnection with exponential backoff
 * - Manages keepalive pings
 */

import { HyperliquidSDK, GRPCStream, GRPCConnectionState } from 'hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.log("gRPC Streaming Example");
  console.log("=".repeat(60));
  console.log();
  console.log("Usage:");
  console.log("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'");
  console.log("  npx ts-node stream_grpc.ts");
  console.log();
  console.log("gRPC is included with all QuickNode Hyperliquid endpoints.");
  process.exit(1);
}

function timestamp(): string {
  const now = new Date();
  return now.toISOString().slice(11, 23);
}

async function main() {
  console.log("=".repeat(60));
  console.log("gRPC Streaming Examples");
  console.log("=".repeat(60));

  // Single SDK instance — can use sdk.grpc for streaming
  const sdk = new HyperliquidSDK(ENDPOINT);

  // Example 1: Stream Trades
  console.log("\nExample 1: Streaming Trades");
  console.log("-".repeat(60));

  let tradeCount = 0;

  const stream = new GRPCStream(ENDPOINT, {
    reconnect: false,
    onConnect: () => console.log("[CONNECTED]"),
    onError: (err) => console.log(`[ERROR] ${err.message}`),
  });

  stream.trades(["BTC", "ETH"], (data: any) => {
    tradeCount++;
    const coin = data.coin || "?";
    const px = parseFloat(data.px || "0");
    const sz = data.sz || "?";
    const side = data.side === "B" ? "BUY " : "SELL";
    console.log(`[${timestamp()}] ${side} ${sz} ${coin} @ $${px.toLocaleString()}`);

    if (tradeCount >= 5) {
      console.log(`\nReceived ${tradeCount} trades.`);
    }
  });

  console.log("Subscribing to BTC and ETH trades...");

  await stream.start();

  // Wait for trades or timeout
  const start = Date.now();
  while (tradeCount < 5 && Date.now() - start < 15000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  stream.stop();

  // Example 2: Stream L2 Book
  console.log("\nExample 2: Streaming L2 Order Book");
  console.log("-".repeat(60));

  let l2Count = 0;

  const l2Stream = new GRPCStream(ENDPOINT, { reconnect: false });

  l2Stream.l2Book("ETH", (data: any) => {
    l2Count++;
    const bids = data.bids || [];
    const asks = data.asks || [];
    if (bids.length > 0 && asks.length > 0) {
      const bestBid = parseFloat(bids[0][0] || "0");
      const bestAsk = parseFloat(asks[0][0] || "0");
      const spread = bestAsk - bestBid;
      console.log(`[${timestamp()}] ETH: bid=$${bestBid} ask=$${bestAsk} spread=$${spread.toFixed(2)}`);
    }

    if (l2Count >= 3) {
      console.log(`\nReceived ${l2Count} L2 updates.`);
    }
  }, { nLevels: 10 });

  await l2Stream.start();

  const l2Start = Date.now();
  while (l2Count < 3 && Date.now() - l2Start < 10000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  l2Stream.stop();

  // Example 3: Stream Blocks
  console.log("\nExample 3: Streaming Blocks");
  console.log("-".repeat(60));

  let blockCount = 0;

  const blockStream = new GRPCStream(ENDPOINT, { reconnect: false });

  blockStream.blocks((data: any) => {
    blockCount++;
    const bn = data.block_number || "?";
    console.log(`[${timestamp()}] Block #${bn}`);

    if (blockCount >= 3) {
      console.log(`\nReceived ${blockCount} blocks.`);
    }
  });

  await blockStream.start();

  const blockStart = Date.now();
  while (blockCount < 3 && Date.now() - blockStart < 15000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  blockStream.stop();

  console.log("\n" + "=".repeat(60));
  console.log("Done!");
}

main().catch(console.error);
