#!/usr/bin/env python3
"""
Order Book Streaming Example - L2 and L4 Order Books via gRPC

This example demonstrates how to stream order book data via gRPC:
- L2 Book: Aggregated by price level (total size and order count per price)
- L4 Book: Individual orders with order IDs

Use cases:
- L2 Book: Market depth, spread monitoring, analytics dashboards
- L4 Book: HFT, quant trading, market making, order flow analysis

Requirements:
    pip install hyperliquid-sdk[grpc]

Usage:
    export ENDPOINT="https://your-endpoint.example.com/TOKEN"
    python stream_orderbook.py
"""

import os
import signal
import sys
import time
from datetime import datetime

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Order Book Streaming Example")
    print("=" * 60)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'")
    print("  python stream_orderbook.py")
    sys.exit(1)


def timestamp():
    return datetime.now().isoformat()[11:23]


def stream_l2_example():
    print("\n" + "=" * 60)
    print("L2 ORDER BOOK (Aggregated Price Levels)")
    print("=" * 60)

    count = [0]

    sdk = HyperliquidSDK(ENDPOINT)
    grpc = sdk.grpc()

    def on_l2_book(data):
        count[0] += 1
        bids = data.get("bids", [])
        asks = data.get("asks", [])

        if bids and asks:
            best_bid = float(bids[0][0]) if isinstance(bids[0], list) else 0
            best_ask = float(asks[0][0]) if isinstance(asks[0], list) else 0
            spread = best_ask - best_bid
            mid = (best_bid + best_ask) / 2
            spread_bps = (spread / mid) * 10000 if mid else 0

            print(f"[{timestamp()}] BTC L2 Update #{count[0]}")
            print(f"  Best Bid: ${best_bid:,.2f}")
            print(f"  Best Ask: ${best_ask:,.2f}")
            print(f"  Spread: ${spread:.2f} ({spread_bps:.2f} bps)")
            print(f"  Depth: {len(bids)} bid levels, {len(asks)} ask levels")

    grpc.l2_book("BTC", on_l2_book, n_levels=10)
    grpc.start()

    start = time.time()
    while count[0] < 5 and time.time() - start < 20:
        time.sleep(0.1)

    grpc.stop()
    print(f"\nReceived {count[0]} L2 updates.")


def stream_l4_example():
    print("\n" + "=" * 60)
    print("L4 ORDER BOOK (Individual Orders)")
    print("=" * 60)

    count = [0]

    sdk = HyperliquidSDK(ENDPOINT)
    grpc = sdk.grpc()

    def on_l4_book(data):
        count[0] += 1

        if data.get("type") == "snapshot":
            bids = data.get("bids", [])
            asks = data.get("asks", [])
            print(f"[{timestamp()}] ETH L4 Snapshot")
            print(f"  {len(bids)} individual bid orders")
            print(f"  {len(asks)} individual ask orders")
        else:
            height = data.get("height", "?")
            print(f"[{timestamp()}] ETH L4 Diff (height: {height})")

    grpc.l4_book("ETH", on_l4_book)
    grpc.start()

    start = time.time()
    while count[0] < 5 and time.time() - start < 30:
        time.sleep(0.1)

    grpc.stop()
    print(f"\nReceived {count[0]} L4 updates.")


def main():
    print("Order Book Streaming Examples")
    print("=" * 60)

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    stream_l2_example()
    stream_l4_example()

    print("\n" + "=" * 60)
    print("Done!")


if __name__ == "__main__":
    main()
