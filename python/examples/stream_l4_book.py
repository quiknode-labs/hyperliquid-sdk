#!/usr/bin/env python3
"""
L4 Order Book Streaming via gRPC - Individual Orders with Order IDs

L4 order book is CRITICAL for:
- Market making: Know your exact queue position
- Order flow analysis: Detect large orders, icebergs
- Optimal execution: See exactly what you're crossing
- HFT: Lower latency than WebSocket

Requirements:
    pip install hyperliquid-sdk[grpc]

Usage:
    export ENDPOINT="https://your-endpoint.example.com/TOKEN"
    python stream_l4_book.py
"""

import os
import signal
import sys
import time
from datetime import datetime

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("L4 Order Book Streaming Example")
    print("=" * 60)
    print()
    print("L4 book shows EVERY individual order with order IDs.")
    print("This is essential for market making and order flow analysis.")
    print()
    print("Usage:")
    print("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'")
    print("  python stream_l4_book.py")
    sys.exit(1)


def timestamp():
    return datetime.now().isoformat()[11:23]


def main():
    print("=" * 60)
    print("L4 Order Book Streaming (Individual Orders)")
    print("=" * 60)

    update_count = [0]

    # Create SDK client
    sdk = HyperliquidSDK(ENDPOINT)
    grpc = sdk.grpc()

    def on_connect():
        print("[CONNECTED]")

    def on_error(err):
        print(f"[ERROR] {err}")

    def on_l4_book(data):
        update_count[0] += 1

        if data.get("type") == "snapshot":
            bids = data.get("bids", [])
            asks = data.get("asks", [])
            print(f"[{timestamp()}] L4 SNAPSHOT")
            print(f"  {len(bids)} bid orders, {len(asks)} ask orders")

            # Show top 3 orders on each side
            print("  Top bids:")
            for order in bids[:3]:
                oid = order.get("oid", "?")
                sz = order.get("sz", "?")
                limit_px = order.get("limit_px", "?")
                print(f"    OID {oid}: {sz} @ {limit_px}")
            print("  Top asks:")
            for order in asks[:3]:
                oid = order.get("oid", "?")
                sz = order.get("sz", "?")
                limit_px = order.get("limit_px", "?")
                print(f"    OID {oid}: {sz} @ {limit_px}")
        elif data.get("type") == "diff":
            height = data.get("height", "?")
            print(f"[{timestamp()}] L4 DIFF (height: {height})")
            diff_data = data.get("data", {})
            print(f"  Changes: {str(diff_data)[:100]}...")

    grpc.on_connect = on_connect
    grpc.on_error = on_error

    grpc.l4_book("BTC", on_l4_book)

    print("\nSubscribing to BTC L4 order book...")
    print("-" * 60)

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        grpc.stop()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    grpc.start()

    start = time.time()
    while update_count[0] < 10 and time.time() - start < 45:
        time.sleep(0.1)

    print(f"\nReceived {update_count[0]} L4 updates.")
    grpc.stop()

    print()
    print("=" * 60)
    print("Done!")


if __name__ == "__main__":
    main()
