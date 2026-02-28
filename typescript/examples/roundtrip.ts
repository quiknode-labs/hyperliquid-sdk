#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Round Trip Example
 *
 * Complete trade cycle: buy then sell to end up flat.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // Buy $11 worth of BTC
  console.log("Buying BTC...");
  const buy = await sdk.marketBuy("BTC", { notional: 11 });
  console.log(`  Bought: ${buy.filledSize || buy.size} BTC`);
  console.log(`  Status: ${buy.status}`);

  // Sell the same amount
  console.log("Selling BTC...");
  const sell = await sdk.marketSell("BTC", { size: buy.filledSize || buy.size });
  console.log(`  Sold: ${sell.filledSize || sell.size} BTC`);
  console.log(`  Status: ${sell.status}`);

  console.log("Done! Position should be flat.");
}

main().catch(console.error);
