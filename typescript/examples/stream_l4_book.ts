#!/usr/bin/env npx ts-node
/**
 * L4 Order Book Streaming via gRPC — Individual Orders with Order IDs
 *
 * L4 order book is CRITICAL for:
 * - Market making: Know your exact queue position
 * - Order flow analysis: Detect large orders, icebergs
 * - Optimal execution: See exactly what you're crossing
 * - HFT: Lower latency than WebSocket
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node stream_l4_book.ts
 */

import { GRPCStream } from 'hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.log("L4 Order Book Streaming Example");
  console.log("=".repeat(60));
  console.log();
  console.log("L4 book shows EVERY individual order with order IDs.");
  console.log("This is essential for market making and order flow analysis.");
  console.log();
  console.log("Usage:");
  console.log("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'");
  console.log("  npx ts-node stream_l4_book.ts");
  process.exit(1);
}

function timestamp(): string {
  return new Date().toISOString().slice(11, 23);
}

async function main() {
  console.log("=".repeat(60));
  console.log("L4 Order Book Streaming (Individual Orders)");
  console.log("=".repeat(60));

  let updateCount = 0;

  const stream = new GRPCStream(ENDPOINT!, {
    reconnect: false,
    onConnect: () => console.log("[CONNECTED]"),
    onError: (err) => console.log(`[ERROR] ${err.message}`),
  });

  stream.l4Book("BTC", (data: any) => {
    updateCount++;

    if (data.type === "snapshot") {
      const bids = data.bids || [];
      const asks = data.asks || [];
      console.log(`[${timestamp()}] L4 SNAPSHOT`);
      console.log(`  ${bids.length} bid orders, ${asks.length} ask orders`);

      // Show top 3 orders on each side
      console.log("  Top bids:");
      for (const order of bids.slice(0, 3)) {
        console.log(`    OID ${order.oid}: ${order.sz} @ ${order.limit_px}`);
      }
      console.log("  Top asks:");
      for (const order of asks.slice(0, 3)) {
        console.log(`    OID ${order.oid}: ${order.sz} @ ${order.limit_px}`);
      }
    } else if (data.type === "diff") {
      console.log(`[${timestamp()}] L4 DIFF (height: ${data.height})`);
      const diffData = data.data || {};
      console.log(`  Changes: ${JSON.stringify(diffData).slice(0, 100)}...`);
    }

    if (updateCount >= 5) {
      console.log(`\nReceived ${updateCount} L4 updates.`);
    }
  });

  console.log("\nSubscribing to BTC L4 order book...");
  console.log("-".repeat(60));

  await stream.start();

  const start = Date.now();
  while (updateCount < 5 && Date.now() - start < 30000) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  stream.stop();

  console.log("\n" + "=".repeat(60));
  console.log("Done!");
}

main().catch(console.error);
