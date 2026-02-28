#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Vaults & Delegation Example
 *
 * Shows how to query vault information and user delegations.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     export USER_ADDRESS="0x..."  # Optional
 *     npx ts-node info_vaults.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;
const USER = process.env.USER_ADDRESS;

if (!ENDPOINT) {
  console.error("Set QUICKNODE_ENDPOINT environment variable");
  process.exit(1);
}

async function main() {
  // Single SDK instance — access everything through sdk.info, sdk.core, sdk.evm, etc.
  const sdk = new HyperliquidSDK(ENDPOINT);
  const info = sdk.info;

  console.log("=".repeat(50));
  console.log("Vaults & Delegation");
  console.log("=".repeat(50));

  // Vault summaries
  console.log("\n1. Vault Summaries:");
  const vaults = await info.vaultSummaries();
  console.log(`   Total: ${vaults.length}`);
  for (const v of vaults.slice(0, 3)) {
    console.log(`   - ${v.name || 'N/A'}: TVL $${v.tvl || '?'}`);
  }

  // User delegations
  if (USER) {
    console.log(`\n2. Delegations (${USER.slice(0, 10)}...):`);
    const delegations = await info.delegations(USER);
    if (delegations.length > 0) {
      console.log(`   ${delegations.length} active`);
    } else {
      console.log("   None");
    }
  } else {
    console.log("\n(Set USER_ADDRESS for delegation info)");
  }

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
