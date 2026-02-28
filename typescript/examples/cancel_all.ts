#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Cancel All Orders Example
 *
 * Cancel all open orders, or all orders for a specific asset.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // Check open orders first
  const orders = await sdk.openOrders();
  console.log(`Open orders: ${orders.count}`);

  // Cancel all orders
  const result = await sdk.cancelAll();
  console.log(`Cancel all: ${JSON.stringify(result)}`);

  // Or cancel just BTC orders:
  // await sdk.cancelAll("BTC");
}

main().catch(console.error);
