#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Transfers Example
 *
 * Transfer USD and spot assets between accounts and wallets.
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

  // Transfer USD to another address
  // const result = await sdk.transferUsd("0x1234567890123456789012345678901234567890", 10.0);
  // console.log(`USD transfer: ${JSON.stringify(result)}`);

  // Transfer spot asset to another address
  // const result = await sdk.transferSpot("PURR", "0x1234567890123456789012345678901234567890", 100.0);
  // console.log(`Spot transfer: ${JSON.stringify(result)}`);

  // Transfer from spot wallet to perp wallet (internal)
  // const result = await sdk.transferSpotToPerp(100.0);
  // console.log(`Spot to perp: ${JSON.stringify(result)}`);

  // Transfer from perp wallet to spot wallet (internal)
  // const result = await sdk.transferPerpToSpot(100.0);
  // console.log(`Perp to spot: ${JSON.stringify(result)}`);

  // Send asset (generalized transfer)
  // const result = await sdk.sendAsset("USDC", "100.0", "0x1234567890123456789012345678901234567890");
  // console.log(`Send asset: ${JSON.stringify(result)}`);

  console.log("Transfer methods available:");
  console.log("  sdk.transferUsd(destination, amount)");
  console.log("  sdk.transferSpot(token, destination, amount)");
  console.log("  sdk.transferSpotToPerp(amount)");
  console.log("  sdk.transferPerpToSpot(amount)");
  console.log("  sdk.sendAsset(token, amount, destination)");
}

main().catch(console.error);
