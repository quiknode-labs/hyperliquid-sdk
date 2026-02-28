#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Trading Example - Place orders on Hyperliquid.
 *
 * This example shows how to place market and limit orders using the SDK.
 *
 * Requirements:
 *     npm install hyperliquid-sdk
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
 *     export PRIVATE_KEY="0x..."
 *     npx ts-node trading_example.ts
 */

import { HyperliquidSDK, Order, Side, TriggerOrder } from '@quicknode/hyperliquid-sdk';

// Get endpoint and private key from environment
const ENDPOINT = process.env.QUICKNODE_ENDPOINT;
const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!ENDPOINT) {
  console.error("Error: Set QUICKNODE_ENDPOINT environment variable");
  console.error("  export QUICKNODE_ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
  process.exit(1);
}

if (!PRIVATE_KEY) {
  console.error("Error: Set PRIVATE_KEY environment variable");
  console.error("  export PRIVATE_KEY='0xYourPrivateKey'");
  process.exit(1);
}

async function main() {
  console.log("Hyperliquid Trading Example");
  console.log("=".repeat(50));

  // Initialize SDK with QuickNode endpoint and private key
  // All requests route through QuickNode - never directly to Hyperliquid
  const sdk = new HyperliquidSDK(ENDPOINT, { privateKey: PRIVATE_KEY });

  console.log(`Address: ${sdk.address}`);
  console.log(`Endpoint: ${ENDPOINT.slice(0, 50)}...`);
  console.log();

  // ==========================================================================
  // Example 1: Market Buy $100 of BTC
  // ==========================================================================
  console.log("Example 1: Market Buy");
  console.log("-".repeat(30));

  try {
    const order = await sdk.marketBuy("BTC", { notional: 100 });
    console.log(`Order placed: ${order}`);
    console.log(`  Order ID: ${order.oid}`);
    console.log(`  Status: ${order.status}`);
    console.log(`  Filled: ${order.filledSize} @ avg ${order.avgPrice}`);
  } catch (e) {
    console.log(`  Error: ${e}`);
  }

  console.log();

  // ==========================================================================
  // Example 2: Limit Order using fluent builder
  // ==========================================================================
  console.log("Example 2: Limit Order");
  console.log("-".repeat(30));

  try {
    // Build limit order with fluent API
    const order = Order.buy("ETH")
      .size(0.1)
      .price(2000)
      .gtc();

    // Place order
    const result = await sdk.order(order);
    console.log(`Order placed: ${result}`);
    console.log(`  Order ID: ${result.oid}`);
    console.log(`  Status: ${result.status}`);
  } catch (e) {
    console.log(`  Error: ${e}`);
  }

  console.log();

  // ==========================================================================
  // Example 3: Stop Loss Order using TriggerOrder builder
  // ==========================================================================
  console.log("Example 3: Stop Loss Order");
  console.log("-".repeat(30));

  try {
    const trigger = TriggerOrder.stopLoss("BTC")
      .size(0.01)
      .triggerPrice(60000)
      .limit(59900)
      .reduceOnly();

    const result = await sdk.triggerOrder(trigger);
    console.log(`Stop loss placed: ${result}`);
  } catch (e) {
    console.log(`  Error: ${e}`);
  }

  console.log();

  // ==========================================================================
  // Example 4: Cancel Orders
  // ==========================================================================
  console.log("Example 4: Cancel All Orders");
  console.log("-".repeat(30));

  try {
    // Cancel all BTC orders
    await sdk.cancelAll("BTC");
    console.log("Cancelled all BTC orders");
  } catch (e) {
    console.log(`  Error: ${e}`);
  }

  console.log();

  // ==========================================================================
  // Example 5: Close Position
  // ==========================================================================
  console.log("Example 5: Close Position");
  console.log("-".repeat(30));

  try {
    // Market close BTC position
    const result = await sdk.closePosition("BTC");
    console.log(`Position closed: ${result}`);
  } catch (e) {
    console.log(`  Error: ${e}`);
  }

  console.log();
  console.log("=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
