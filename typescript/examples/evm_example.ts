#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * HyperEVM API Example - Interact with Hyperliquid's EVM chain.
 *
 * This example shows how to query the Hyperliquid EVM chain (chain ID 999 mainnet, 998 testnet).
 *
 * Requirements:
 *     npm install hyperliquid-sdk
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node evm_example.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

// Get endpoint from environment
const ENDPOINT = process.env.ENDPOINT;

if (!ENDPOINT) {
  console.error("Error: Set ENDPOINT environment variable");
  console.error("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  process.exit(1);
}

async function main() {
  console.log("Hyperliquid EVM API Example");
  console.log("=".repeat(50));
  console.log(`Endpoint: ${ENDPOINT.slice(0, 50)}...`);
  console.log();

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT);

  // ==========================================================================
  // Chain Info
  // ==========================================================================
  console.log("Chain Info");
  console.log("-".repeat(30));

  // Get chain ID
  const chainId = await sdk.evm.chainId();
  console.log(`Chain ID: ${chainId}`);
  console.log(`Network: ${chainId === 999 ? 'Mainnet' : chainId === 998 ? 'Testnet' : 'Unknown'}`);

  // Get latest block number
  const blockNum = await sdk.evm.blockNumber();
  console.log(`Latest block: ${blockNum}`);

  // Get gas price
  const gasPrice = await sdk.evm.gasPrice();
  const gasGwei = Number(gasPrice) / 1e9;
  console.log(`Gas price: ${gasGwei.toFixed(2)} Gwei`);
  console.log();

  // ==========================================================================
  // Account Balance
  // ==========================================================================
  console.log("Account Balance");
  console.log("-".repeat(30));

  // Example address - replace with your address
  const address = "0x0000000000000000000000000000000000000000";

  const balanceWei = await sdk.evm.getBalance(address);
  const balanceEth = Number(balanceWei) / 1e18;
  console.log(`Address: ${address}`);
  console.log(`Balance: ${balanceEth.toFixed(6)} HYPE`);
  console.log();

  // ==========================================================================
  // Block Data
  // ==========================================================================
  console.log("Block Data");
  console.log("-".repeat(30));

  // Get latest block
  const block = await sdk.evm.getBlockByNumber(blockNum);
  if (block) {
    const b = block as Record<string, unknown>;
    console.log(`Block ${blockNum}:`);
    console.log(`  Hash: ${String(b.hash || '?').slice(0, 20)}...`);
    console.log(`  Parent: ${String(b.parentHash || '?').slice(0, 20)}...`);
    console.log(`  Timestamp: ${b.timestamp || '?'}`);
    console.log(`  Gas Used: ${parseInt(String(b.gasUsed || '0x0'), 16).toLocaleString()}`);
    const txs = b.transactions as unknown[] || [];
    console.log(`  Transactions: ${txs.length}`);
  }
  console.log();

  // ==========================================================================
  // Transaction Count
  // ==========================================================================
  console.log("Transaction Count");
  console.log("-".repeat(30));

  const txCount = await sdk.evm.getTransactionCount(address);
  console.log(`Nonce for ${address.slice(0, 10)}...: ${txCount}`);
  console.log();

  // ==========================================================================
  // Smart Contract Call (Example: ERC20 balanceOf)
  // ==========================================================================
  console.log("Smart Contract Call");
  console.log("-".repeat(30));

  // Example: Read a contract (this is just a demonstration)
  // In real usage, you'd use actual contract addresses and proper ABI encoding
  console.log("  (Contract call example would go here)");
  console.log("  Use sdk.evm.call() with proper contract address and data");
  console.log();

  console.log("=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
