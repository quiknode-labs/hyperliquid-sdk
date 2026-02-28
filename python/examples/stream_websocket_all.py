#!/usr/bin/env python3
"""
WebSocket Streaming - Complete Reference

This example demonstrates ALL WebSocket subscription types:
- Market Data: trades, l2_book, book_updates, all_mids, candle, bbo
- User Data: open_orders, user_fills, user_fundings, clearinghouse_state
- TWAP: twap, twap_states, user_twap_slice_fills
- System: events, notification

Requirements:
    pip install hyperliquid-sdk[websocket]

Usage:
    export ENDPOINT="https://your-endpoint.example.com/TOKEN"
    export USER_ADDRESS="0x..."  # Optional, for user data streams
    python stream_websocket_all.py
"""

import os
import signal
import sys
import time
from datetime import datetime

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")
USER = os.environ.get("USER_ADDRESS") or "0x0000000000000000000000000000000000000000"

if not ENDPOINT:
    print("WebSocket Complete Reference")
    print("=" * 60)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'")
    print("  export USER_ADDRESS='0x...'  # Optional, for user data streams")
    print("  python stream_websocket_all.py")
    sys.exit(1)


def timestamp():
    return datetime.now().isoformat()[11:23]


# Global counters
counts = {}


def make_callback(name, max_prints=3):
    counts[name] = 0

    def callback(data):
        counts[name] += 1
        if counts[name] <= max_prints:
            channel = data.get("channel", "unknown")
            print(f"[{timestamp()}] {name.upper()}: {channel} (#{counts[name]})")
            # Print first few fields of data
            inner_data = data.get("data", data)
            if isinstance(inner_data, dict):
                keys = list(inner_data.keys())[:3]
                print(f"             Fields: {', '.join(keys)}")
            elif isinstance(inner_data, list):
                print(f"             Items: {len(inner_data)}")

    return callback


def demo_market_data():
    print("\n" + "=" * 60)
    print("MARKET DATA STREAMS")
    print("=" * 60)
    print()
    print("Available streams:")
    print("  - trades(coins, callback)")
    print("  - book_updates(coins, callback)")
    print("  - l2_book(coin, callback)")
    print("  - all_mids(callback)")
    print("  - candle(coin, interval, callback)")
    print("  - bbo(coin, callback)")
    print()

    # Create SDK client
    sdk = HyperliquidSDK(ENDPOINT)
    stream = sdk.stream()

    # trades: Real-time executed trades
    stream.trades(["BTC", "ETH"], make_callback("trades"))

    # book_updates: Incremental order book changes
    stream.book_updates(["BTC"], make_callback("book_updates"))

    # l2_book: Full L2 order book snapshots
    stream.l2_book("BTC", make_callback("l2_book"))

    print("Starting market data streams for 10 seconds...")
    print("-" * 60)

    stream.start()
    time.sleep(10)
    stream.stop()

    print("\nMarket data summary:")
    for name, count in counts.items():
        print(f"  {name}: {count} messages")


def main():
    print("WebSocket Streaming - Complete Reference")
    print("=" * 60)

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    demo_market_data()

    print("\n" + "=" * 60)
    print("Done!")
    print()
    print("Other available streams (not shown):")
    print("  Market: all_mids, candle, bbo, active_asset_ctx")
    print("  User: open_orders, user_fills, user_fundings, clearinghouse_state")
    print("  TWAP: twap, twap_states, user_twap_slice_fills")
    print("  System: events, notification, writer_actions")


if __name__ == "__main__":
    main()
