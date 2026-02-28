#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Open Orders Example
 *
 * View all open orders with details.
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

  // Get all open orders
  const result = await sdk.openOrders();
  console.log(`Open orders: ${result.count}`);

  for (const o of result.orders) {
    const side = o.side === "B" ? "BUY" : "SELL";
    console.log(`  ${o.name} ${side} ${o.sz} @ ${o.limitPx} (OID: ${o.oid})`);
  }

  // Get order status for a specific order
  // const status = await sdk.orderStatus(12345);
  // console.log(`Order status: ${JSON.stringify(status)}`);
}

main().catch(console.error);
