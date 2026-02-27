#!/usr/bin/env python3
"""
HyperCore API Example — Low-level block and trade data via JSON-RPC.

This example shows how to query blocks, trades, and orders using the HyperCore API.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python hypercore_example.py
"""

import os
import sys

from hyperliquid_sdk import HyperCore

# Get endpoint from environment
ENDPOINT = os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Error: Set QUICKNODE_ENDPOINT environment variable")
    print("  export QUICKNODE_ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("Hyperliquid HyperCore API Example")
    print("=" * 50)
    print(f"Endpoint: {ENDPOINT[:50]}...")
    print()

    # Create HyperCore client
    hc = HyperCore(ENDPOINT)

    # ==========================================================================
    # Block Data
    # ==========================================================================
    print("Block Data")
    print("-" * 30)

    # Get latest block number
    block_num = hc.latest_block_number()
    print(f"Latest block: {block_num}")

    # Get block by number
    block = hc.block_by_number(block_num)
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
    trades = hc.latest_trades(count=5)
    print(f"Last {len(trades)} trades:")
    for trade in trades[:5]:
        coin = trade.get("coin", "?")
        px = trade.get("px", "?")
        sz = trade.get("sz", "?")
        side = trade.get("side", "?")
        print(f"  {coin}: {sz} @ ${float(px):,.2f} ({side})")
    print()

    # Get trades for specific coin
    btc_trades = hc.trades_by_coin("BTC", count=3)
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

    # Get latest orders (all coins)
    orders = hc.latest_orders(count=5)
    print(f"Last {len(orders)} orders:")
    for order in orders[:5]:
        coin = order.get("coin", "?")
        side = order.get("side", "?")
        px = order.get("limitPx", "?")
        sz = order.get("sz", "?")
        status = order.get("status", "?")
        print(f"  {coin}: {side} {sz} @ ${float(px):,.2f} - {status}")
    print()

    # ==========================================================================
    # Block Range Query
    # ==========================================================================
    print("Block Range Query")
    print("-" * 30)

    # Get blocks in range
    start_block = max(0, block_num - 5)
    blocks = hc.blocks_in_range(start_block, block_num)
    print(f"Blocks {start_block} to {block_num}: {len(blocks)} blocks")

    total_txs = sum(len(b.get("transactions", [])) for b in blocks)
    print(f"Total transactions: {total_txs}")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
