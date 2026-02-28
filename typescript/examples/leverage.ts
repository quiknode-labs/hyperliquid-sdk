#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Leverage Example
 *
 * Update leverage for a position.
 *
 * Requires: PRIVATE_KEY environment variable
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  console.error("Example: export PRIVATE_KEY='0x...'");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });
  console.log(`Wallet: ${sdk.address}`);

  // Update leverage for BTC to 10x cross margin
  const result = await sdk.updateLeverage("BTC", 10, true);
  console.log(`Update leverage result: ${JSON.stringify(result)}`);

  // Update leverage for ETH to 5x isolated margin
  // const result = await sdk.updateLeverage("ETH", 5, false);
  // console.log(`Update leverage result: ${JSON.stringify(result)}`);

  console.log("\nLeverage methods available:");
  console.log("  sdk.updateLeverage(asset, leverage, isCross)");
}

main().catch(console.error);
