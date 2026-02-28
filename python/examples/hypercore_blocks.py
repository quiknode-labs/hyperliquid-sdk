#!/usr/bin/env python3
"""
HyperCore Block Data Example

Shows how to get real-time trades, orders, and block data via the HyperCore API.

This is the alternative to Info methods (all_mids, l2_book, recent_trades) that
are not available on QuickNode endpoints.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python hypercore_blocks.py
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
    print("HyperCore Block Data")
    print("=" * 50)

    # Single SDK instance - access everything through sdk.info(), sdk.core(), sdk.evm()
    sdk = HyperliquidSDK(ENDPOINT)
    hc = sdk.core()

    # Latest block number
    print("\n1. Latest Block:")
    block_num = hc.latest_block_number()
    print(f"   Block #{block_num}")

    # Recent trades
    print("\n2. Recent Trades (all coins):")
    trades = hc.latest_trades(count=5)
    for t in trades[:5]:
        side = "BUY" if t.get("side") == "B" else "SELL"
        print(f"   {side} {t.get('sz')} {t.get('coin')} @ ${t.get('px')}")

    # Recent BTC trades only
    print("\n3. BTC Trades:")
    btc_trades = hc.latest_trades(count=10, coin="BTC")
    for t in btc_trades[:3]:
        side = "BUY" if t.get("side") == "B" else "SELL"
        print(f"   {side} {t.get('sz')} @ ${t.get('px')}")
    if len(btc_trades) == 0:
        print("   No BTC trades in recent blocks")

    # Get a specific block
    print("\n4. Get Block Data:")
    block = hc.get_block(block_num - 1)
    if block:
        print(f"   Block #{block_num - 1}")
        print(f"   Time: {block.get('block_time', 'N/A')}")
        print(f"   Events: {len(block.get('events', []))}")

    # Get batch of blocks
    print("\n5. Batch Blocks:")
    blocks = hc.get_batch_blocks(block_num - 5, block_num - 1)
    block_list = blocks.get("blocks", []) if isinstance(blocks, dict) else blocks
    print(f"   Retrieved {len(block_list)} blocks")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
