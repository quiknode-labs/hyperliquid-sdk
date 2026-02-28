#!/usr/bin/env python3
"""
HyperCore API Example — Block and trade data via JSON-RPC.

This example shows how to query blocks, trades, and orders using the HyperCore API.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python hypercore_example.py
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
    print("Hyperliquid HyperCore API Example")
    print("=" * 50)
    print(f"Endpoint: {ENDPOINT[:50]}...")
    print()

    # Create SDK
    sdk = HyperliquidSDK(ENDPOINT)

    # ==========================================================================
    # Block Data
    # ==========================================================================
    print("Block Data")
    print("-" * 30)

    # Get latest block number
    block_num = sdk.core().latest_block_number()
    print(f"Latest block: {block_num}")

    # Get block by number
    block = sdk.core().get_block(block_num)
    if block:
        txs = block.get("transactions", [])
        print(f"Block {block_num}:")
        print(f"  Transactions: {len(txs)}")
        print(f"  Timestamp: {block.get('timestamp', '?')}")
    print()

    # ==========================================================================
    # Recent Trades
    # ==========================================================================
    print("Recent Trades")
    print("-" * 30)

    # Get latest trades (all coins)
    trades = sdk.core().latest_trades(count=5)
    print(f"Last {len(trades)} trades:")
    for trade in trades[:5]:
        coin = trade.get("coin", "?")
        px = trade.get("px", "?")
        sz = trade.get("sz", "?")
        side = trade.get("side", "?")
        print(f"  {coin}: {sz} @ ${float(px):,.2f} ({side})")
    print()

    # Get trades for specific coin
    btc_trades = sdk.core().latest_trades(coin="BTC", count=3)
    print(f"Last {len(btc_trades)} BTC trades:")
    for trade in btc_trades:
        px = trade.get("px", "?")
        sz = trade.get("sz", "?")
        side = trade.get("side", "?")
        print(f"  {sz} @ ${float(px):,.2f} ({side})")
    print()

    # ==========================================================================
    # Recent Orders
    # ==========================================================================
    print("Recent Orders")
    print("-" * 30)

    try:
        # Get latest orders (all coins)
        orders = sdk.core().latest_orders(count=5)
        print(f"Last {len(orders)} orders:")
        for order in orders[:5]:
            coin = order.get("coin", "?")
            side = order.get("side", "?")
            px = order.get("limitPx", "?")
            sz = order.get("sz", "?")
            status = order.get("status", "?")
            print(f"  {coin}: {side} {sz} @ ${float(px):,.2f} - {status}")
    except Exception as e:
        print(f"  Could not fetch orders: {e}")
    print()

    # ==========================================================================
    # Block Range Query
    # ==========================================================================
    print("Block Range Query")
    print("-" * 30)

    try:
        # Get batch blocks
        start_block = max(0, block_num - 5)
        blocks = sdk.core().get_batch_blocks(start_block, block_num)
        print(f"Blocks {start_block} to {block_num}: {len(blocks)} blocks")

        # Safely count transactions
        total_txs = 0
        for b in blocks:
            if isinstance(b, dict):
                total_txs += len(b.get("transactions", []))
        print(f"Total transactions: {total_txs}")
    except Exception as e:
        print(f"  Could not fetch blocks: {e}")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
