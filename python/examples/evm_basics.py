#!/usr/bin/env python3
"""
HyperEVM Example

Shows how to use standard Ethereum JSON-RPC calls on Hyperliquid's EVM chain.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python evm_basics.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Error: Set ENDPOINT environment variable")
    print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("=" * 50)
    print("HyperEVM (Ethereum JSON-RPC)")
    print("=" * 50)

    # Single SDK instance - access everything through sdk.info(), sdk.core(), sdk.evm()
    sdk = HyperliquidSDK(ENDPOINT)
    evm = sdk.evm()

    # Chain info
    print("\n1. Chain Info:")
    chain_id = evm.chain_id()
    block_num = evm.block_number()
    gas_price = evm.gas_price()
    print(f"   Chain ID: {chain_id}")
    print(f"   Block: {block_num}")
    print(f"   Gas Price: {gas_price / 1e9:.2f} gwei")

    # Latest block
    print("\n2. Latest Block:")
    block = evm.get_block_by_number(block_num)
    if block:
        block_hash = block.get("hash", "?")
        txs = block.get("transactions", [])
        print(f"   Hash: {block_hash[:20]}...")
        print(f"   Txs: {len(txs)}")

    # Check balance
    print("\n3. Balance Check:")
    addr = "0x0000000000000000000000000000000000000000"
    balance = evm.get_balance(addr)
    print(f"   {addr[:12]}...: {balance / 1e18:.6f} ETH")

    print()
    print("=" * 50)
    print("Done!")
    print("\nFor debug/trace APIs, use: sdk.evm(debug=True)")


if __name__ == "__main__":
    main()
