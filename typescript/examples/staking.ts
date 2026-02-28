#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Staking Example
 *
 * Stake and unstake HYPE tokens.
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

  // Stake HYPE tokens
  // const result = await sdk.stake(100);
  // console.log(`Stake: ${JSON.stringify(result)}`);

  // Unstake HYPE tokens
  // const result = await sdk.unstake(50);
  // console.log(`Unstake: ${JSON.stringify(result)}`);

  // Delegate to a validator
  // const result = await sdk.delegate("0x...", 100);
  // console.log(`Delegate: ${JSON.stringify(result)}`);

  // Undelegate from a validator
  // const result = await sdk.undelegate("0x...", 50);
  // console.log(`Undelegate: ${JSON.stringify(result)}`);

  console.log("Staking methods available:");
  console.log("  sdk.stake(amount)");
  console.log("  sdk.unstake(amount)");
  console.log("  sdk.delegate(validator, amount)");
  console.log("  sdk.undelegate(validator, amount)");
}

main().catch(console.error);
