#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * HyperEVM Example
 *
 * Shows how to use standard Ethereum JSON-RPC calls on Hyperliquid's EVM chain.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node evm_basics.ts
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
  const evm = sdk.evm;

  console.log("=".repeat(50));
  console.log("HyperEVM (Ethereum JSON-RPC)");
  console.log("=".repeat(50));

  // Chain info
  console.log("\n1. Chain Info:");
  const chainId = await evm.chainId();
  const blockNum = await evm.blockNumber();
  const gasPrice = await evm.gasPrice();
  console.log(`   Chain ID: ${chainId}`);
  console.log(`   Block: ${blockNum}`);
  console.log(`   Gas Price: ${(Number(gasPrice) / 1e9).toFixed(2)} gwei`);

  // Latest block
  console.log("\n2. Latest Block:");
  const block = await evm.getBlockByNumber("latest");
  if (block) {
    console.log(`   Hash: ${block.hash?.slice(0, 20)}...`);
    console.log(`   Txs: ${(block.transactions || []).length}`);
  }

  // Check balance
  console.log("\n3. Balance Check:");
  const addr = "0x0000000000000000000000000000000000000000";
  const balance = await evm.getBalance(addr);
  console.log(`   ${addr.slice(0, 12)}...: ${(Number(balance) / 1e18).toFixed(6)} ETH`);

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
  console.log("\nFor debug/trace APIs, use: new EVM(endpoint, { debug: true })");
}

main().catch(console.error);
