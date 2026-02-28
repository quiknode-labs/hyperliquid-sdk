#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Builder Fee Approval Example
 *
 * Approve the builder fee to enable trading through the API.
 * Required before placing orders.
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;
const PRIVATE_KEY = process.env.PRIVATE_KEY;

if (!PRIVATE_KEY) {
  console.error("Set PRIVATE_KEY environment variable");
  console.error("Example: export PRIVATE_KEY='0x...'");
  process.exit(1);
}

async function main() {
  const sdk = new HyperliquidSDK(ENDPOINT, { privateKey: PRIVATE_KEY });

  // Check current approval status
  const status = await sdk.approvalStatus();
  console.log(`Currently approved: ${status.approved || false}`);
  if (status.approved) {
    console.log(`Max fee rate: ${status.maxFeeRate}`);
  }

  // Approve builder fee (1% max)
  // await sdk.approveBuilderFee("1%");
  // console.log("Approved!");

  // Or use auto_approve when creating SDK:
  // const sdk = new HyperliquidSDK(ENDPOINT, { privateKey: PRIVATE_KEY, autoApprove: true });

  // Revoke approval:
  // await sdk.revokeBuilderFee();
}

main().catch(console.error);
