#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * HyperCore Block Data Example
 *
 * Shows how to get real-time trades, orders, and block data via the HyperCore API.
 *
 * This is the alternative to Info methods (allMids, l2Book, recentTrades) that
 * are not available on QuickNode endpoints.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node hypercore_blocks.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.error("Set QUICKNODE_ENDPOINT environment variable");
  process.exit(1);
}

async function main() {
  // Single SDK instance — access everything through sdk.info, sdk.core, sdk.evm, etc.
  const sdk = new HyperliquidSDK(ENDPOINT);
  const hc = sdk.core;

  console.log("=".repeat(50));
  console.log("HyperCore Block Data");
  console.log("=".repeat(50));

  // Latest block number
  console.log("\n1. Latest Block:");
  const blockNum = await hc.latestBlockNumber();
  console.log(`   Block #${blockNum}`);

  // Recent trades
  console.log("\n2. Recent Trades (all coins):");
  const trades = await hc.latestTrades(5);
  for (const t of trades.slice(0, 5)) {
    const side = t.side === "B" ? "BUY" : "SELL";
    console.log(`   ${side} ${t.sz} ${t.coin} @ $${t.px}`);
  }

  // Recent BTC trades only
  console.log("\n3. BTC Trades:");
  const btcTrades = await hc.latestTrades(10, "BTC");
  for (const t of btcTrades.slice(0, 3)) {
    const side = t.side === "B" ? "BUY" : "SELL";
    console.log(`   ${side} ${t.sz} @ $${t.px}`);
  }
  if (btcTrades.length === 0) {
    console.log("   No BTC trades in recent blocks");
  }

  // Get a specific block
  console.log("\n4. Get Block Data:");
  const block = await hc.getBlock(blockNum - 1);
  console.log(`   Block #${blockNum - 1}`);
  console.log(`   Time: ${block.block_time || 'N/A'}`);
  console.log(`   Events: ${(block.events || []).length}`);

  // Get batch of blocks
  console.log("\n5. Batch Blocks:");
  const blocks = await hc.getBatchBlocks(blockNum - 5, blockNum - 1);
  console.log(`   Retrieved ${(blocks.blocks || []).length} blocks`);

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
