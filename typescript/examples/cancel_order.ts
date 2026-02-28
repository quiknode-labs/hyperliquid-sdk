#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Cancel Order Example
 *
 * Place an order and then cancel it by OID.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // Place a resting order 3% below mid
  const mid = await sdk.getMid("BTC");
  const limitPrice = Math.floor(mid * 0.97);
  const order = await sdk.buy("BTC", { notional: 11, price: limitPrice, tif: "gtc" });
  console.log(`Placed order OID: ${order.oid}`);

  // Cancel using the order object
  await order.cancel();
  console.log("Cancelled via order.cancel()");

  // Alternative: cancel by OID directly
  // await sdk.cancel(12345, "BTC");
}

main().catch(console.error);
