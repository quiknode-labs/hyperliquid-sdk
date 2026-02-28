#!/usr/bin/env python3
"""
WebSocket Streaming Example - Real-Time Trade Data

Stream trades via WebSocket.

Available WebSocket streams:
- trades: Executed trades with price, size, direction
- orders: Order lifecycle events (open, filled, cancelled)
- book_updates: Order book changes (incremental deltas)
- events: Balance changes, transfers, deposits, withdrawals
- twap: TWAP execution data
- writer_actions: HyperCore <-> HyperEVM asset transfers

Note: L2/L4 order book snapshots are available via gRPC (see stream_orderbook.py).

Requirements:
    pip install hyperliquid-sdk[websocket]

Usage:
    export ENDPOINT="https://your-endpoint.example.com/TOKEN"
    python stream_trades.py
"""

import os
import signal
import sys
import time
from datetime import datetime

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("WebSocket Streaming Example")
    print("=" * 60)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'")
    print("  python stream_trades.py")
    sys.exit(1)


def timestamp():
    return datetime.now().isoformat()[11:23]


def main():
    print("=" * 60)
    print("WebSocket Trade Streaming")
    print("=" * 60)

    trade_count = [0]

    # Create SDK client
    sdk = HyperliquidSDK(ENDPOINT)
    stream = sdk.stream()

    def on_connect():
        print("[CONNECTED]")

    def on_error(err):
        print(f"[ERROR] {err}")

    def on_trade(data):
        # Events are [[user, trade_data], ...]
        block = data.get("block", {})
        for event in block.get("events", []):
            if isinstance(event, list) and len(event) >= 2:
                t = event[1]  # trade_data is second element
                trade_count[0] += 1
                coin = t.get("coin", "?")
                px = float(t.get("px", "0"))
                sz = t.get("sz", "?")
                side = "BUY " if t.get("side") == "B" else "SELL"
                print(f"[{timestamp()}] {side} {sz} {coin} @ ${px:,.2f}")

    stream.on_connect = on_connect
    stream.on_error = on_error

    stream.trades(["BTC", "ETH"], on_trade)

    print("\nSubscribing to BTC and ETH trades...")
    print("-" * 60)

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        stream.stop()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    stream.start()

    start = time.time()
    while trade_count[0] < 20 and time.time() - start < 60:
        time.sleep(0.1)

    print(f"\nReceived {trade_count[0]} trades.")
    stream.stop()

    print()
    print("=" * 60)
    print("Done!")


if __name__ == "__main__":
    main()
