#!/usr/bin/env python3
"""
L2 Order Book Streaming - Aggregated Price Levels

L2 order book shows total size at each price level (aggregated).
Available via both WebSocket and gRPC.

Use L2 for:
- Price monitoring
- Basic trading strategies
- Lower bandwidth requirements

Use L4 (gRPC only) when you need:
- Individual order IDs
- Queue position tracking
- Order flow analysis

Requirements:
    pip install hyperliquid-sdk[grpc]

Usage:
    export ENDPOINT="https://your-endpoint.example.com/TOKEN"
    python stream_l2_book.py
"""

import os
import signal
import sys
import time
from datetime import datetime

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("L2 Order Book Streaming Example")
    print("=" * 60)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'")
    print("  python stream_l2_book.py")
    sys.exit(1)


def timestamp():
    return datetime.now().isoformat()[11:23]


class L2BookTracker:
    def __init__(self, coin):
        self.coin = coin
        self.bids = []
        self.asks = []
        self.update_count = 0

    def update(self, data):
        self.update_count += 1
        self.bids = data.get("bids", [])
        self.asks = data.get("asks", [])

    def best_bid(self):
        if not self.bids:
            return (0, 0)
        bid = self.bids[0]
        if isinstance(bid, list):
            return (float(bid[0]), float(bid[1]))
        return (0, 0)

    def best_ask(self):
        if not self.asks:
            return (0, 0)
        ask = self.asks[0]
        if isinstance(ask, list):
            return (float(ask[0]), float(ask[1]))
        return (0, 0)

    def spread(self):
        bid_px, _ = self.best_bid()
        ask_px, _ = self.best_ask()
        return ask_px - bid_px if bid_px and ask_px else 0

    def spread_bps(self):
        bid_px, _ = self.best_bid()
        ask_px, _ = self.best_ask()
        if not bid_px or not ask_px:
            return 0
        mid = (bid_px + ask_px) / 2
        return ((ask_px - bid_px) / mid) * 10000

    def display(self):
        bid_px, bid_sz = self.best_bid()
        ask_px, ask_sz = self.best_ask()
        print(f"[{timestamp()}] {self.coin}")
        print(f"  Bid: {bid_sz:.4f} @ ${bid_px:,.2f}")
        print(f"  Ask: {ask_sz:.4f} @ ${ask_px:,.2f}")
        print(f"  Spread: ${self.spread():.2f} ({self.spread_bps():.2f} bps)")
        print(f"  Levels: {len(self.bids)} bids, {len(self.asks)} asks")


def main():
    print("=" * 60)
    print("L2 Order Book Streaming")
    print("=" * 60)

    tracker = L2BookTracker("ETH")

    # Create SDK client
    sdk = HyperliquidSDK(ENDPOINT)
    grpc = sdk.grpc()

    def on_connect():
        print("[CONNECTED]")

    def on_error(err):
        print(f"[ERROR] {err}")

    def on_l2_book(data):
        tracker.update(data)
        tracker.display()

    grpc.on_connect = on_connect
    grpc.on_error = on_error

    grpc.l2_book("ETH", on_l2_book, n_levels=20)

    print("\nSubscribing to ETH L2 order book...")
    print("-" * 60)

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        grpc.stop()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    grpc.start()

    start = time.time()
    while tracker.update_count < 10 and time.time() - start < 30:
        time.sleep(0.1)

    print(f"\nReceived {tracker.update_count} L2 updates.")
    grpc.stop()

    print()
    print("=" * 60)
    print("Done!")


if __name__ == "__main__":
    main()
