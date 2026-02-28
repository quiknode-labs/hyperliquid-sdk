#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Market Data Example
 *
 * Shows how to query market metadata, prices, order book, and recent trades.
 *
 * The SDK handles all Info API methods automatically.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node info_market_data.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.error("Set QUICKNODE_ENDPOINT environment variable");
  console.error("Example: export QUICKNODE_ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
  process.exit(1);
}

async function main() {
  // Single SDK instance — access everything through sdk.info, sdk.core, sdk.evm, etc.
  const sdk = new HyperliquidSDK(ENDPOINT);
  const info = sdk.info;

  console.log("=".repeat(50));
  console.log("Market Data (Info API)");
  console.log("=".repeat(50));

  // Exchange metadata
  console.log("\n1. Exchange Metadata:");
  try {
    const meta = await info.meta();
    console.log(`   Perp Markets: ${(meta.universe || []).length}`);
    for (const asset of (meta.universe || []).slice(0, 5)) {
      console.log(`   - ${asset.name}: max leverage ${asset.maxLeverage}x`);
    }
  } catch (e: any) {
    console.log(`   (meta not available: ${e.code || e.message})`);
  }

  // Spot metadata
  console.log("\n2. Spot Metadata:");
  try {
    const spot = await info.spotMeta();
    console.log(`   Spot Tokens: ${(spot.tokens || []).length}`);
  } catch (e: any) {
    console.log(`   (spotMeta not available: ${e.code || e.message})`);
  }

  // Exchange status
  console.log("\n3. Exchange Status:");
  try {
    const status = await info.exchangeStatus();
    console.log(`   ${JSON.stringify(status)}`);
  } catch (e: any) {
    console.log(`   (exchangeStatus not available: ${e.code || e.message})`);
  }

  // All mid prices
  console.log("\n4. Mid Prices:");
  try {
    const mids = await info.allMids();
    console.log(`   BTC: $${parseFloat(mids.BTC || "0").toLocaleString()}`);
    console.log(`   ETH: $${parseFloat(mids.ETH || "0").toLocaleString()}`);
  } catch (e: any) {
    console.log(`   (allMids not available: ${e.code || e.message})`);
  }

  // Order book
  console.log("\n5. Order Book (BTC):");
  try {
    const book = await info.l2Book("BTC");
    const levels = book.levels || [[], []];
    if (levels[0] && levels[0].length > 0 && levels[1] && levels[1].length > 0) {
      const bestBid = parseFloat(levels[0][0].px || "0");
      const bestAsk = parseFloat(levels[1][0].px || "0");
      const spread = bestAsk - bestBid;
      console.log(`   Best Bid: $${bestBid.toLocaleString()}`);
      console.log(`   Best Ask: $${bestAsk.toLocaleString()}`);
      console.log(`   Spread: $${spread.toFixed(2)}`);
    }
  } catch (e: any) {
    console.log(`   (l2Book not available: ${e.code || e.message})`);
  }

  // Recent trades
  console.log("\n6. Recent Trades (BTC):");
  try {
    const trades = await info.recentTrades("BTC");
    for (const t of trades.slice(0, 3)) {
      const side = t.side === "B" ? "BUY" : "SELL";
      console.log(`   ${side} ${t.sz} @ $${parseFloat(t.px || "0").toLocaleString()}`);
    }
  } catch (e: any) {
    console.log(`   (recentTrades not available: ${e.code || e.message})`);
  }

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
