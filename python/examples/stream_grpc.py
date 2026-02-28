#!/usr/bin/env python3
"""
gRPC Streaming Example - High-Performance Real-Time Data

Stream trades, orders, L2 book, L4 book, and blocks via gRPC.
gRPC provides lower latency than WebSocket for high-frequency trading.

The SDK:
- Connects to port 10000 automatically
- Passes token via x-token header
- Handles reconnection with exponential backoff
- Manages keepalive pings

Requirements:
    pip install hyperliquid-sdk[grpc]

Usage:
    export ENDPOINT="https://your-endpoint.example.com/TOKEN"
    python stream_grpc.py
"""

import os
import signal
import sys
import time
from datetime import datetime

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("gRPC Streaming Example")
    print("=" * 60)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'")
    print("  python stream_grpc.py")
    sys.exit(1)


def timestamp():
    return datetime.now().isoformat()[11:23]


def main():
    print("=" * 60)
    print("gRPC Streaming Examples")
    print("=" * 60)

    # Example 1: Stream Trades
    print("\nExample 1: Streaming Trades")
    print("-" * 60)

    trade_count = [0]

    # Create SDK client
    sdk = HyperliquidSDK(ENDPOINT)
    grpc = sdk.grpc()

    def on_connect():
        print("[CONNECTED]")

    def on_error(err):
        print(f"[ERROR] {err}")

    def on_trade(data):
        trade_count[0] += 1
        coin = data.get("coin", "?")
        px = float(data.get("px", "0"))
        sz = data.get("sz", "?")
        side = "BUY " if data.get("side") == "B" else "SELL"
        print(f"[{timestamp()}] {side} {sz} {coin} @ ${px:,.2f}")

    grpc.on_connect = on_connect
    grpc.on_error = on_error

    grpc.trades(["BTC", "ETH"], on_trade)
    print("Subscribing to BTC and ETH trades...")

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        grpc.stop()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    grpc.start()

    # Wait for trades or timeout
    start = time.time()
    while trade_count[0] < 10 and time.time() - start < 30:
        time.sleep(0.1)

    print(f"\nReceived {trade_count[0]} trades.")
    grpc.stop()

    # Example 2: Stream L2 Book
    print("\nExample 2: Streaming L2 Order Book")
    print("-" * 60)

    l2_count = [0]

    sdk2 = HyperliquidSDK(ENDPOINT)
    grpc2 = sdk2.grpc()

    def on_l2_book(data):
        l2_count[0] += 1
        bids = data.get("bids", [])
        asks = data.get("asks", [])
        if bids and asks:
            best_bid = float(bids[0][0]) if isinstance(bids[0], list) else float(bids[0].get("price", 0))
            best_ask = float(asks[0][0]) if isinstance(asks[0], list) else float(asks[0].get("price", 0))
            spread = best_ask - best_bid
            print(f"[{timestamp()}] ETH: bid=${best_bid:,.2f} ask=${best_ask:,.2f} spread=${spread:.2f}")

    grpc2.l2_book("ETH", on_l2_book, n_levels=10)
    grpc2.start()

    start = time.time()
    while l2_count[0] < 5 and time.time() - start < 15:
        time.sleep(0.1)

    print(f"\nReceived {l2_count[0]} L2 updates.")
    grpc2.stop()

    # Example 3: Stream Blocks
    print("\nExample 3: Streaming Blocks")
    print("-" * 60)

    block_count = [0]

    sdk3 = HyperliquidSDK(ENDPOINT)
    grpc3 = sdk3.grpc()

    def on_block(data):
        block_count[0] += 1
        bn = data.get("block_number", "?")
        print(f"[{timestamp()}] Block #{bn}")

    grpc3.blocks(on_block)
    grpc3.start()

    start = time.time()
    while block_count[0] < 5 and time.time() - start < 20:
        time.sleep(0.1)

    print(f"\nReceived {block_count[0]} blocks.")
    grpc3.stop()

    print()
    print("=" * 60)
    print("Done!")


if __name__ == "__main__":
    main()
