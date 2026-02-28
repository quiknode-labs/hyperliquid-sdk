#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Cancel by Client Order ID (CLOID) Example
 *
 * Cancel an order using a client-provided order ID instead of the exchange OID.
 * Useful when you track orders by your own IDs.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(undefined, { privateKey: PRIVATE_KEY });

  // Note: CLOIDs are hex strings you provide when placing orders
  // This example shows the cancelByCloid API

  // Cancel by client order ID
  // await sdk.cancelByCloid("0x1234567890abcdef...", "BTC");

  console.log("cancelByCloid() cancels orders by your custom client order ID");
  console.log("Usage: sdk.cancelByCloid(cloid, asset)");
}

main().catch(console.error);
