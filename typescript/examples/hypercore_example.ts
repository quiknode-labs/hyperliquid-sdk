#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * HyperCore API Example - Low-level block and trade data via JSON-RPC.
 *
 * This example shows how to query blocks, trades, and orders using the HyperCore API.
 *
 * Requirements:
 *     npm install hyperliquid-sdk
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node hypercore_example.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

// Get endpoint from environment
const ENDPOINT = process.env.ENDPOINT;

if (!ENDPOINT) {
  console.error("Error: Set ENDPOINT environment variable");
  console.error("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  process.exit(1);
}

async function main() {
  console.log("Hyperliquid HyperCore API Example");
  console.log("=".repeat(50));
  console.log(`Endpoint: ${ENDPOINT.slice(0, 50)}...`);
  console.log();

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT);

  // ==========================================================================
  // Block Data
  // ==========================================================================
  console.log("Block Data");
  console.log("-".repeat(30));

  // Get latest block number
  const blockNum = await sdk.core.latestBlockNumber();
  console.log(`Latest block: ${blockNum}`);

  // Get block by number
  const block = await sdk.core.getBlock(blockNum);
  if (block) {
    const txs = (block as Record<string, unknown>).transactions as unknown[] || [];
    console.log(`Block ${blockNum}:`);
    console.log(`  Transactions: ${txs.length}`);
    console.log(`  Timestamp: ${(block as Record<string, unknown>).timestamp || '?'}`);
  }
  console.log();

  // ==========================================================================
  // Recent Trades
  // ==========================================================================
  console.log("Recent Trades");
  console.log("-".repeat(30));

  // Get latest trades (all coins)
  const trades = await sdk.core.latestTrades({ count: 5 });
  console.log(`Last ${trades.length} trades:`);
  for (const trade of trades.slice(0, 5)) {
    const t = trade as Record<string, unknown>;
    const coin = t.coin || "?";
    const px = parseFloat(String(t.px || 0));
    const sz = t.sz || "?";
    const side = t.side || "?";
    console.log(`  ${coin}: ${sz} @ $${px.toLocaleString()} (${side})`);
  }
  console.log();

  // Get trades for specific coin
  const btcTrades = await sdk.core.latestTrades({ coin: "BTC", count: 3 });
  console.log(`Last ${btcTrades.length} BTC trades:`);
  for (const trade of btcTrades) {
    const t = trade as Record<string, unknown>;
    const px = parseFloat(String(t.px || 0));
    const sz = t.sz || "?";
    const side = t.side || "?";
    console.log(`  ${sz} @ $${px.toLocaleString()} (${side})`);
  }
  console.log();

  // ==========================================================================
  // Recent Orders
  // ==========================================================================
  console.log("Recent Orders");
  console.log("-".repeat(30));

  try {
    // Get latest orders (all coins)
    const orders = await sdk.core.latestOrders({ count: 5 });
    console.log(`Last ${orders.length} orders:`);
    for (const order of orders.slice(0, 5)) {
      const o = order as Record<string, unknown>;
      const coin = o.coin || "?";
      const side = o.side || "?";
      const px = parseFloat(String(o.limitPx || 0));
      const sz = o.sz || "?";
      const status = o.status || "?";
      console.log(`  ${coin}: ${side} ${sz} @ $${px.toLocaleString()} - ${status}`);
    }
  } catch (e) {
    console.log(`  Could not fetch orders: ${e}`);
  }
  console.log();

  // ==========================================================================
  // Block Range Query
  // ==========================================================================
  console.log("Block Range Query");
  console.log("-".repeat(30));

  try {
    // Get batch blocks
    const startBlock = Math.max(0, blockNum - 5);
    const blocks = await sdk.core.getBatchBlocks(startBlock, blockNum);
    console.log(`Blocks ${startBlock} to ${blockNum}: ${blocks.length} blocks`);

    // Safely count transactions
    let totalTxs = 0;
    for (const b of blocks) {
      if (typeof b === 'object' && b !== null) {
        const txs = (b as Record<string, unknown>).transactions as unknown[] || [];
        totalTxs += txs.length;
      }
    }
    console.log(`Total transactions: ${totalTxs}`);
  } catch (e) {
    console.log(`  Could not fetch blocks: ${e}`);
  }

  console.log();
  console.log("=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
