#!/usr/bin/env python3
"""
WebSocket Streaming Example — Real-time market data via WebSocket.

This example demonstrates:
- Subscribing to trades, orders, and book updates
- Automatic reconnection handling
- Graceful shutdown

Requirements:
    pip install hyperliquid-sdk[websocket]

Usage:
    export ENDPOINT="https://YOUR-ENDPOINT.hype-mainnet.quiknode.pro/YOUR-TOKEN"
    python websocket_streaming.py
"""

import os
import signal
import sys

from hyperliquid_sdk import HyperliquidSDK, ConnectionState

# Get endpoint from args or environment
if len(sys.argv) > 1:
    ENDPOINT = sys.argv[1]
else:
    ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Hyperliquid WebSocket Streaming Example")
    print("=" * 50)
    print()
    print("Usage:")
    print("  export ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
    print("  python websocket_streaming.py")
    sys.exit(1)


def on_trade(data):
    """Handle incoming trade data."""
    trades = data.get("data", [])
    if isinstance(trades, list):
        for trade in trades:
            coin = trade.get("coin", "?")
            px = float(trade.get("px", 0))
            sz = trade.get("sz", "?")
            side = "BUY" if trade.get("side") == "B" else "SELL"
            print(f"[TRADE] {coin}: {side} {sz} @ ${px:,.2f}")
    else:
        coin = trades.get("coin", "?")
        px = float(trades.get("px", 0))
        sz = trades.get("sz", "?")
        side = "BUY" if trades.get("side") == "B" else "SELL"
        print(f"[TRADE] {coin}: {side} {sz} @ ${px:,.2f}")


def on_book_update(data):
    """Handle incoming book update."""
    book_data = data.get("data", {})
    coin = book_data.get("coin", "?")
    levels = book_data.get("levels", [[], []])
    bids = levels[0] if len(levels) > 0 else []
    asks = levels[1] if len(levels) > 1 else []

    if bids and asks:
        best_bid = bids[0]
        best_ask = asks[0]
        spread = float(best_ask.get("px", 0)) - float(best_bid.get("px", 0))
        print(f"[BOOK] {coin}: Bid ${float(best_bid.get('px', 0)):,.2f} | Ask ${float(best_ask.get('px', 0)):,.2f} | Spread ${spread:,.2f}")


def on_state_change(state: ConnectionState):
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
    print("[CLOSED] Stream stopped")


def main():
    print("Hyperliquid WebSocket Streaming Example")
    print("=" * 50)
    print(f"Endpoint: {ENDPOINT[:60]}{'...' if len(ENDPOINT) > 60 else ''}")
    print()

    # Create SDK and get stream
    sdk = HyperliquidSDK(ENDPOINT)
    stream = sdk.stream()

    # Configure callbacks
    stream.on_error = on_error
    stream.on_close = on_close
    stream.on_state_change = on_state_change
    stream.on_reconnect = on_reconnect

    # Subscribe to BTC and ETH trades
    stream.trades(["BTC", "ETH"], on_trade)
    print("Subscribed to: BTC, ETH trades")

    # Subscribe to BTC book updates
    stream.book_updates(["BTC"], on_book_update)
    print("Subscribed to: BTC book updates")

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
