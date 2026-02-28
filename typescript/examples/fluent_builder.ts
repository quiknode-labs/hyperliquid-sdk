#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Fluent Order Builder Example
 *
 * For power users who want maximum control with IDE autocomplete.
 */

import { HyperliquidSDK, Order } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });
  const mid = await sdk.getMid("BTC");

  // Simple limit order with GTC (Good Till Cancelled) - minimum $10 value
  // Use size directly to ensure proper decimal precision (BTC allows 5 decimals)
  const order = await sdk.order(
    Order.buy("BTC")
      .size(0.00017) // ~$11 worth at ~$65k (minimum is $10)
      .price(Math.floor(mid * 0.97))
      .gtc()
  );
  console.log(`Limit GTC: ${order}`);
  await order.cancel();

  // Market order by notional value
  // const order = await sdk.order(
  //   Order.sell("ETH")
  //     .notional(10)
  //     .market()
  // );
  // console.log(`Market: ${order}`);

  // Reduce-only order (only closes existing position)
  // const order = await sdk.order(
  //   Order.sell("BTC")
  //     .size(0.001)
  //     .price(Math.floor(mid * 1.03))
  //     .gtc()
  //     .reduceOnly()
  // );
  // console.log(`Reduce-only: ${order}`);

  // ALO order (Add Liquidity Only / Post-Only)
  // const order = await sdk.order(
  //   Order.buy("BTC")
  //     .size(0.001)
  //     .price(Math.floor(mid * 0.95))
  //     .alo()
  // );
  // console.log(`Post-only: ${order}`);

  console.log("\nFluent builder methods:");
  console.log("  .size(0.001)       - Set size in asset units");
  console.log("  .notional(100)     - Set size in USD");
  console.log("  .price(65000)      - Set limit price");
  console.log("  .gtc()             - Good Till Cancelled");
  console.log("  .ioc()             - Immediate Or Cancel");
  console.log("  .alo()             - Add Liquidity Only (post-only)");
  console.log("  .market()          - Market order");
  console.log("  .reduceOnly()      - Only close position");
}

main().catch(console.error);
