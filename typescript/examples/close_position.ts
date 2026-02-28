#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Close Position Example
 *
 * Close an open position completely. The SDK figures out the size and direction.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // Close BTC position (if any)
  // The SDK queries your position and builds the counter-order automatically
  try {
    const result = await sdk.closePosition("BTC");
    console.log(`Closed position: ${result}`);
  } catch (e) {
    console.log(`No position to close or error: ${e}`);
  }
}

main().catch(console.error);
