#!/usr/bin/env python3
"""
HyperEVM API Example — Interact with Hyperliquid's EVM chain.

This example shows how to query the Hyperliquid EVM chain (chain ID 999 mainnet, 998 testnet).

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python evm_example.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK

# Get endpoint from environment
ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Error: Set ENDPOINT environment variable")
    print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("Hyperliquid EVM API Example")
    print("=" * 50)
    print(f"Endpoint: {ENDPOINT[:50]}...")
    print()

    # Create SDK
    sdk = HyperliquidSDK(ENDPOINT)

    # ==========================================================================
    # Chain Info
    # ==========================================================================
    print("Chain Info")
    print("-" * 30)

    # Get chain ID
    chain_id = sdk.evm().chain_id()
    print(f"Chain ID: {chain_id}")
    print(f"Network: {'Mainnet' if chain_id == 999 else 'Testnet' if chain_id == 998 else 'Unknown'}")

    # Get latest block number
    block_num = sdk.evm().block_number()
    print(f"Latest block: {block_num}")

    # Get gas price
    gas_price = sdk.evm().gas_price()
    gas_gwei = gas_price / 1e9
    print(f"Gas price: {gas_gwei:.2f} Gwei")
    print()

    # ==========================================================================
    # Account Balance
    # ==========================================================================
    print("Account Balance")
    print("-" * 30)

    # Example address - replace with your address
    address = "0x0000000000000000000000000000000000000000"

    balance_wei = sdk.evm().get_balance(address)
    balance_eth = balance_wei / 1e18
    print(f"Address: {address}")
    print(f"Balance: {balance_eth:.6f} HYPE")
    print()

    # ==========================================================================
    # Block Data
    # ==========================================================================
    print("Block Data")
    print("-" * 30)

    # Get latest block
    block = sdk.evm().get_block_by_number(block_num)
    if block:
        print(f"Block {block_num}:")
        print(f"  Hash: {block.get('hash', '?')[:20]}...")
        print(f"  Parent: {block.get('parentHash', '?')[:20]}...")
        print(f"  Timestamp: {block.get('timestamp', '?')}")
        print(f"  Gas Used: {int(block.get('gasUsed', '0x0'), 16):,}")
        txs = block.get("transactions", [])
        print(f"  Transactions: {len(txs)}")
    print()

    # ==========================================================================
    # Transaction Count
    # ==========================================================================
    print("Transaction Count")
    print("-" * 30)

    tx_count = sdk.evm().get_transaction_count(address)
    print(f"Nonce for {address[:10]}...: {tx_count}")
    print()

    # ==========================================================================
    # Smart Contract Call (Example: ERC20 balanceOf)
    # ==========================================================================
    print("Smart Contract Call")
    print("-" * 30)

    # Example: Read a contract (this is just a demonstration)
    # In real usage, you'd use actual contract addresses and proper ABI encoding
    print("  (Contract call example would go here)")
    print("  Use sdk.evm().call() with proper contract address and data")
    print()

    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
