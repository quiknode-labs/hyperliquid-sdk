#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * User Account Data Example
 *
 * Shows how to query user positions, orders, and account state.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     export USER_ADDRESS="0x..."
 *     npx ts-node info_user_data.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;
const USER = process.env.USER_ADDRESS || "0x2ba553d9f990a3b66b03b2dc0d030dfc1c061036";

if (!ENDPOINT) {
  console.error("Set QUICKNODE_ENDPOINT environment variable");
  process.exit(1);
}

async function main() {
  // Single SDK instance — access everything through sdk.info, sdk.core, sdk.evm, etc.
  const sdk = new HyperliquidSDK(ENDPOINT);
  const info = sdk.info;

  console.log("=".repeat(50));
  console.log(`User Data: ${USER.slice(0, 10)}...`);
  console.log("=".repeat(50));

  // Clearinghouse state (positions + margin)
  console.log("\n1. Positions & Margin:");
  try {
    const state = await info.clearinghouseState(USER);
    const margin = state.marginSummary || {};
    console.log(`   Account Value: $${margin.accountValue || "0"}`);
    console.log(`   Margin Used: $${margin.totalMarginUsed || "0"}`);

    const positions = state.assetPositions || [];
    if (positions.length > 0) {
      console.log(`   Positions: ${positions.length}`);
      for (const pos of positions.slice(0, 3)) {
        const p = pos.position || {};
        console.log(`   - ${p.coin}: ${p.szi} @ ${p.entryPx}`);
      }
    } else {
      console.log("   No positions");
    }
  } catch (e: any) {
    console.log(`   (clearinghouseState not available: ${e.code || e.message})`);
  }

  // Open orders
  console.log("\n2. Open Orders:");
  try {
    const orders = await info.openOrders(USER);
    if (orders.length > 0) {
      console.log(`   ${orders.length} orders:`);
      for (const o of orders.slice(0, 3)) {
        const side = o.side === "B" ? "BUY" : "SELL";
        console.log(`   - ${o.coin}: ${side} ${o.sz} @ ${o.limitPx}`);
      }
    } else {
      console.log("   No open orders");
    }
  } catch (e: any) {
    console.log(`   (openOrders not available: ${e.code || e.message})`);
  }

  // User fees
  console.log("\n3. Fee Structure:");
  try {
    const fees = await info.userFees(USER);
    console.log(`   Maker: ${fees.makerRate || 'N/A'}`);
    console.log(`   Taker: ${fees.takerRate || 'N/A'}`);
  } catch (e: any) {
    console.log(`   (userFees not available: ${e.code || e.message})`);
  }

  // Spot balances
  console.log("\n4. Spot Balances:");
  try {
    const spot = await info.spotClearinghouseState(USER);
    const balances = spot.balances || [];
    if (balances.length > 0) {
      for (const b of balances.slice(0, 5)) {
        console.log(`   - ${b.coin}: ${b.total}`);
      }
    } else {
      console.log("   No spot balances");
    }
  } catch (e: any) {
    console.log(`   (spotClearinghouseState not available: ${e.code || e.message})`);
  }

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
