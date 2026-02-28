#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Withdraw Example
 *
 * Withdraw USDC to L1 (Arbitrum).
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

  // Withdraw USDC to L1 (Arbitrum)
  // WARNING: This is a real withdrawal - be careful with amounts
  // const result = await sdk.withdraw("0x1234567890123456789012345678901234567890", 100.0);
  // console.log(`Withdraw: ${JSON.stringify(result)}`);

  console.log("Withdraw methods available:");
  console.log("  sdk.withdraw(destination, amount)");
  console.log("  Note: Withdraws USDC to your L1 Arbitrum address");
}

main().catch(console.error);
