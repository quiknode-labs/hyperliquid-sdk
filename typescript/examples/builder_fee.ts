#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Builder Fee Example
 *
 * Approve and revoke builder fee permissions.
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

  // Check approval status (doesn't require deposit)
  const status = await sdk.approvalStatus();
  console.log(`Approval status: ${JSON.stringify(status)}`);

  // Approve builder fee (required before trading via QuickNode)
  // Note: Requires account to have deposited first
  // const result = await sdk.approveBuilderFee("1%");
  // console.log(`Approve builder fee: ${JSON.stringify(result)}`);

  // Revoke builder fee permission
  // const result = await sdk.revokeBuilderFee();
  // console.log(`Revoke builder fee: ${JSON.stringify(result)}`);

  console.log("\nBuilder fee methods available:");
  console.log("  sdk.approveBuilderFee(maxFee, builder?)");
  console.log("  sdk.revokeBuilderFee(builder?)");
  console.log("  sdk.approvalStatus(user?)");
}

main().catch(console.error);
