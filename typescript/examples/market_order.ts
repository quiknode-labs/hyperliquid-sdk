#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Market Order Example
 *
 * Place a market order that executes immediately at best available price.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // Market buy by notional ($11 worth of BTC - minimum is $10)
  const order = await sdk.marketBuy("BTC", { notional: 11 });
  console.log(`Market buy: ${order}`);
  console.log(`  Status: ${order.status}`);
  console.log(`  OID: ${order.oid}`);

  // Market buy by notional ($10 worth of ETH)
  // const order = await sdk.marketBuy("ETH", { notional: 10 });

  // Market sell
  // const order = await sdk.marketSell("BTC", { size: 0.0001 });
}

main().catch(console.error);
