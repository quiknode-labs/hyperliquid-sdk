#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * L2 Order Book Streaming - Aggregated Price Levels
 *
 * L2 order book shows total size at each price level (aggregated).
 * Available via both WebSocket and gRPC.
 *
 * Use L2 for:
 * - Price monitoring
 * - Basic trading strategies
 * - Lower bandwidth requirements
 *
 * Use L4 (gRPC only) when you need:
 * - Individual order IDs
 * - Queue position tracking
 * - Order flow analysis
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node stream_l2_book.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.ENDPOINT;

if (!ENDPOINT) {
  console.log("L2 Order Book Streaming Example");
  console.log("=".repeat(60));
  console.log();
  console.log("Usage:");
  console.log("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  console.log("  npx ts-node stream_l2_book.ts");
  process.exit(1);
}

function timestamp(): string {
  return new Date().toISOString().slice(11, 23);
}

class L2BookTracker {
  coin: string;
  bids: any[] = [];
  asks: any[] = [];
  updateCount = 0;

  constructor(coin: string) {
    this.coin = coin;
  }

  update(data: any): void {
    this.updateCount++;
    this.bids = data.bids || [];
    this.asks = data.asks || [];
  }

  bestBid(): [number, number] {
    if (this.bids.length === 0) return [0, 0];
    const bid = this.bids[0];
    return Array.isArray(bid) ? [parseFloat(bid[0]), parseFloat(bid[1])] : [0, 0];
  }

  bestAsk(): [number, number] {
    if (this.asks.length === 0) return [0, 0];
    const ask = this.asks[0];
    return Array.isArray(ask) ? [parseFloat(ask[0]), parseFloat(ask[1])] : [0, 0];
  }

  spread(): number {
    const [bidPx] = this.bestBid();
    const [askPx] = this.bestAsk();
    return bidPx && askPx ? askPx - bidPx : 0;
  }

  spreadBps(): number {
    const [bidPx] = this.bestBid();
    const [askPx] = this.bestAsk();
    if (!bidPx || !askPx) return 0;
    const mid = (bidPx + askPx) / 2;
    return ((askPx - bidPx) / mid) * 10000;
  }

  display(): void {
    const [bidPx, bidSz] = this.bestBid();
    const [askPx, askSz] = this.bestAsk();
    console.log(`[${timestamp()}] ${this.coin}`);
    console.log(`  Bid: ${bidSz.toFixed(4)} @ $${bidPx.toLocaleString()}`);
    console.log(`  Ask: ${askSz.toFixed(4)} @ $${askPx.toLocaleString()}`);
    console.log(`  Spread: $${this.spread().toFixed(2)} (${this.spreadBps().toFixed(2)} bps)`);
    console.log(`  Levels: ${this.bids.length} bids, ${this.asks.length} asks`);
  }
}

async function main() {
  console.log("=".repeat(60));
  console.log("L2 Order Book Streaming");
  console.log("=".repeat(60));

  const tracker = new L2BookTracker("ETH");

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT!);

  // Configure gRPC stream
  sdk.grpc.onConnect = () => console.log("[CONNECTED]");
  sdk.grpc.onError = (err) => console.log(`[ERROR] ${err.message}`);

  sdk.grpc.l2Book("ETH", (data: any) => {
    tracker.update(data);
    tracker.display();

    if (tracker.updateCount >= 5) {
      console.log(`\nReceived ${tracker.updateCount} L2 updates.`);
    }
  }, { nLevels: 20 });

  console.log("\nSubscribing to ETH L2 order book...");
  console.log("-".repeat(60));

  await sdk.grpc.start();

  const start = Date.now();
  while (tracker.updateCount < 5 && Date.now() - start < 20000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  sdk.grpc.stop();

  console.log("\n" + "=".repeat(60));
  console.log("Done!");
}

main().catch(console.error);
