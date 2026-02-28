#!/usr/bin/env npx ts-node
/**
 * WebSocket Streaming Example — Real-Time HyperCore Data
 *
 * Stream trades, orders, book updates, events, and TWAP via WebSocket.
 * These are the data streams available on QuickNode endpoints.
 *
 * Available QuickNode WebSocket streams:
 * - trades: Executed trades with price, size, direction
 * - orders: Order lifecycle events (open, filled, cancelled)
 * - book_updates: Order book changes (incremental deltas)
 * - events: Balance changes, transfers, deposits, withdrawals
 * - twap: TWAP execution data
 * - writer_actions: HyperCore <-> HyperEVM asset transfers
 *
 * Note: L2/L4 order book snapshots are available via gRPC (see stream_orderbook.ts).
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node stream_trades.ts
 */

import { Stream } from 'hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.log("WebSocket Streaming Example");
  console.log("=".repeat(60));
  console.log();
  console.log("Usage:");
  console.log("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'");
  console.log("  npx ts-node stream_trades.ts");
  process.exit(1);
}

function timestamp(): string {
  return new Date().toISOString().slice(11, 23);
}

async function main() {
  console.log("=".repeat(60));
  console.log("WebSocket Trade Streaming");
  console.log("=".repeat(60));

  let tradeCount = 0;

  const stream = new Stream(ENDPOINT!, {
    reconnect: false,
    onConnect: () => console.log("[CONNECTED]"),
    onError: (err) => console.log(`[ERROR] ${err.message}`),
  });

  stream.trades(["BTC", "ETH"], (data: any) => {
    // QuickNode format: {"type": "data", "stream": "hl.trades", "block": {"events": [...]}}
    // Events are [[user, trade_data], ...]
    const block = data.block || {};
    for (const event of block.events || []) {
      if (Array.isArray(event) && event.length >= 2) {
        const t = event[1]; // trade_data is second element
        tradeCount++;
        const coin = t.coin || "?";
        const px = parseFloat(t.px || "0");
        const sz = t.sz || "?";
        const side = t.side === "B" ? "BUY " : "SELL";
        console.log(`[${timestamp()}] ${side} ${sz} ${coin} @ $${px.toLocaleString()}`);

        if (tradeCount >= 10) {
          console.log(`\nReceived ${tradeCount} trades.`);
          return;
        }
      }
    }
  });

  console.log("\nSubscribing to BTC and ETH trades...");
  console.log("-".repeat(60));

  await stream.start();

  const start = Date.now();
  while (tradeCount < 10 && Date.now() - start < 30000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  stream.stop();

  console.log("\n" + "=".repeat(60));
  console.log("Done!");
}

main().catch(console.error);
