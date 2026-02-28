#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Preflight Validation Example
 *
 * Validate an order BEFORE signing to catch tick size and lot size errors.
 * Saves failed transactions by checking validity upfront.
 *
 * No endpoint or private key needed — uses public API.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

async function main() {
  // No endpoint or private key needed for read-only public queries
  const sdk = new HyperliquidSDK();

  // Get current price
  const mid = await sdk.getMid("BTC");
  console.log(`BTC mid: $${mid.toLocaleString()}`);

  // Validate a good order
  const result = await sdk.preflight("BTC", "buy", Math.floor(mid * 0.97), 0.001) as any;
  console.log(`Valid order: ${JSON.stringify(result)}`);

  // Validate an order with too many decimals (will fail)
  const result2 = await sdk.preflight("BTC", "buy", 67000.123456789, 0.001) as any;
  console.log(`Invalid price: ${JSON.stringify(result2)}`);
  if (!result2.valid && result2.errors && result2.errors.length > 0) {
    console.log(`  Field: ${result2.errors[0].field}`);
    console.log(`  Error: ${result2.errors[0].error}`);
  }
}

main().catch(console.error);
