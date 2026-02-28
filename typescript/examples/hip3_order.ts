#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * HIP-3 Market Order Example
 *
 * Trade on HIP-3 markets (community perps like Hypersea).
 * Same API as regular markets, just use "dex:symbol" format.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // List HIP-3 DEXes
  const dexes = await sdk.dexes();
  console.log("Available HIP-3 DEXes:");
  for (const dex of dexes.slice(0, 5)) {
    console.log(`  ${dex.name || dex}`);
  }

  // Trade on a HIP-3 market
  // Format: "dex:SYMBOL"
  // const order = await sdk.buy("xyz:SILVER", { notional: 11, tif: "ioc" });
  // console.log(`HIP-3 order: ${order}`);

  console.log("\nHIP-3 markets use 'dex:SYMBOL' format");
  console.log("Example: sdk.buy('xyz:SILVER', { notional: 11 })");
}

main().catch(console.error);
