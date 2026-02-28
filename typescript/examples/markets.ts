#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Markets Example
 *
 * List all available markets and HIP-3 DEXes.
 *
 * No endpoint or private key needed — uses public API.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

async function main() {
  // No endpoint or private key needed for read-only public queries
  const sdk = new HyperliquidSDK();

  // Get all markets
  const markets = await sdk.markets() as { perps?: any[]; spot?: any[] };
  const perps = markets.perps || [];
  const spot = markets.spot || [];
  console.log(`Perp markets: ${perps.length}`);
  console.log(`Spot markets: ${spot.length}`);

  // Show first 5 perp markets
  console.log("\nFirst 5 perp markets:");
  for (const m of perps.slice(0, 5)) {
    console.log(`  ${m.name}: szDecimals=${m.szDecimals}`);
  }

  // Get HIP-3 DEXes
  const dexes = await sdk.dexes() as unknown as any[];
  console.log(`\nHIP-3 DEXes: ${dexes.length}`);
  for (const dex of dexes.slice(0, 5)) {
    console.log(`  ${dex.name || 'N/A'}`);
  }
}

main().catch(console.error);
