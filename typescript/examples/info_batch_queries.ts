#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Multi-User Queries Example
 *
 * Shows how to query multiple users' states efficiently.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node info_batch_queries.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.error("Set QUICKNODE_ENDPOINT environment variable");
  process.exit(1);
}

async function main() {
  // Single SDK instance — access everything through sdk.info, sdk.core, sdk.evm, etc.
  const sdk = new HyperliquidSDK(ENDPOINT);
  const info = sdk.info;

  console.log("=".repeat(50));
  console.log("Multi-User Queries");
  console.log("=".repeat(50));

  // Example addresses (use real addresses with activity for better demo)
  const addresses = [
    "0x2ba553d9f990a3b66b03b2dc0d030dfc1c061036", // Active trader
    "0x0000000000000000000000000000000000000001",
    "0x0000000000000000000000000000000000000002",
  ];

  console.log(`\nQuerying ${addresses.length} user accounts...`);

  // Query each user's clearinghouse state
  console.log("\n1. User Account States:");
  for (const addr of addresses) {
    try {
      const state = await info.clearinghouseState(addr);
      const margin = state.marginSummary || {};
      const value = parseFloat(margin.accountValue || "0");
      const positions = (state.assetPositions || []).length;
      console.log(`   ${addr.slice(0, 12)}...: $${value.toLocaleString()} (${positions} positions)`);
    } catch (e) {
      console.log(`   ${addr.slice(0, 12)}...: Error - ${e}`);
    }
  }

  // Query open orders for first user
  console.log("\n2. Open Orders (first user):");
  try {
    const orders = await info.openOrders(addresses[0]);
    console.log(`   ${orders.length} open orders`);
    for (const o of orders.slice(0, 3)) {
      const side = o.side === "B" ? "BUY" : "SELL";
      console.log(`   - ${o.coin}: ${side} ${o.sz} @ ${o.limitPx}`);
    }
  } catch (e) {
    console.log(`   Error: ${e}`);
  }

  // Query user fees
  console.log("\n3. Fee Structure (first user):");
  try {
    const fees = await info.userFees(addresses[0]);
    console.log(`   Maker: ${fees.makerRate || 'N/A'}`);
    console.log(`   Taker: ${fees.takerRate || 'N/A'}`);
  } catch (e) {
    console.log(`   Error: ${e}`);
  }

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
