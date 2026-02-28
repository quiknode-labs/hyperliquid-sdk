#!/usr/bin/env python3
"""
gRPC Streaming Example — High-performance real-time market data via gRPC.

This example demonstrates:
- Subscribing to trades, orders, blocks, and L2/L4 order books
- Automatic reconnection handling
- Graceful shutdown

gRPC offers lower latency than WebSocket for high-frequency data.

Requirements:
    pip install hyperliquid-sdk[grpc]

Usage:
    export ENDPOINT="https://YOUR-ENDPOINT.hype-mainnet.quiknode.pro/YOUR-TOKEN"
    python grpc_streaming.py
"""

import os
import signal
import sys

from hyperliquid_sdk import HyperliquidSDK

# Get endpoint from args or environment
if len(sys.argv) > 1:
    ENDPOINT = sys.argv[1]
else:
    ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Hyperliquid gRPC Streaming Example")
    print("=" * 50)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
    print("  python grpc_streaming.py")
    sys.exit(1)


def on_trade(data):
    """Handle incoming trade."""
    coin = data.get("coin", "?")
    px = float(data.get("px", 0))
    sz = data.get("sz", "?")
    side = "BUY" if data.get("side") == "B" else "SELL"
    print(f"[TRADE] {coin}: {side} {sz} @ ${px:,.2f}")


def on_book_update(data):
    """Handle incoming book update."""
    coin = data.get("coin", "?")
    bids = data.get("bids", [])
    asks = data.get("asks", [])

    if bids and asks:
        best_bid = bids[0]
        best_ask = asks[0]
        print(f"[BOOK] {coin}: Bid ${float(best_bid.get('price', 0)):,.2f} | Ask ${float(best_ask.get('price', 0)):,.2f}")


def on_l2_book(data):
    """Handle L2 order book update."""
    coin = data.get("coin", "?")
    bids = data.get("bids", [])[:3]
    asks = data.get("asks", [])[:3]
    print(f"[L2] {coin}: {len(bids)} bid levels, {len(asks)} ask levels")


def on_block(data):
    """Handle incoming block."""
    block_num = data.get("block_number", "?")
    print(f"[BLOCK] #{block_num}")


def on_state_change(state):
    """Handle connection state changes."""
    print(f"[STATE] {state.value}")


def on_reconnect(attempt: int):
    """Handle reconnection attempts."""
    print(f"[RECONNECT] Attempt {attempt}")


def on_error(error):
    """Handle errors."""
    print(f"[ERROR] {error}")


def on_close():
    """Handle final connection close."""
    print("[CLOSED] gRPC stream stopped")


def on_connect():
    """Handle connection established."""
    print("[CONNECTED] gRPC stream ready")


def main():
    print("Hyperliquid gRPC Streaming Example")
    print("=" * 50)
    print(f"Endpoint: {ENDPOINT[:60]}{'...' if len(ENDPOINT) > 60 else ''}")
    print()

    # Create SDK and get gRPC stream
    sdk = HyperliquidSDK(ENDPOINT)
    stream = sdk.grpc()

    # Configure callbacks
    stream.on_error = on_error
    stream.on_close = on_close
    stream.on_connect = on_connect
    stream.on_state_change = on_state_change
    stream.on_reconnect = on_reconnect

    # Subscribe to BTC and ETH trades
    stream.trades(["BTC", "ETH"], on_trade)
    print("Subscribed to: BTC, ETH trades")

    # Subscribe to book updates
    stream.book_updates(["BTC"], on_book_update)
    print("Subscribed to: BTC book updates")

    # Subscribe to L2 order book
    stream.l2_book("ETH", on_l2_book)
    print("Subscribed to: ETH L2 order book")

    # Subscribe to blocks
    stream.blocks(on_block)
    print("Subscribed to: blocks")

    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print("\nShutting down gracefully...")
        stream.stop()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    print()
    print("Streaming... Press Ctrl+C to stop")
    print("-" * 50)

    # Run the stream (blocking)
    # Use stream.start() for non-blocking background mode
    stream.run()


if __name__ == "__main__":
    main()
