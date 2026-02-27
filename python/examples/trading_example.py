#!/usr/bin/env python3
"""
Trading Example — Place orders on Hyperliquid.

This example shows how to place market and limit orders using the SDK.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    export PRIVATE_KEY="0x..."
    python trading_example.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK, Order, Side

# Get endpoint and private key from environment
ENDPOINT = os.environ.get("QUICKNODE_ENDPOINT")
PRIVATE_KEY = os.environ.get("PRIVATE_KEY")

if not ENDPOINT:
    print("Error: Set QUICKNODE_ENDPOINT environment variable")
    print("  export QUICKNODE_ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)

if not PRIVATE_KEY:
    print("Error: Set PRIVATE_KEY environment variable")
    print("  export PRIVATE_KEY='0xYourPrivateKey'")
    sys.exit(1)


def main():
    print("Hyperliquid Trading Example")
    print("=" * 50)

    # Initialize SDK with QuickNode endpoint and private key
    # All requests route through QuickNode - never directly to Hyperliquid
    sdk = HyperliquidSDK(ENDPOINT, private_key=PRIVATE_KEY)

    print(f"Address: {sdk.address}")
    print(f"Endpoint: {ENDPOINT[:50]}...")
    print()

    # ==========================================================================
    # Example 1: Market Buy $100 of BTC
    # ==========================================================================
    print("Example 1: Market Buy")
    print("-" * 30)

    try:
        order = sdk.market_buy("BTC", notional=100)
        print(f"Order placed: {order}")
        print(f"  Order ID: {order.oid}")
        print(f"  Status: {order.status}")
        print(f"  Filled: {order.filled_size} @ avg {order.avg_price}")
    except Exception as e:
        print(f"  Error: {e}")

    print()

    # ==========================================================================
    # Example 2: Limit Order
    # ==========================================================================
    print("Example 2: Limit Order")
    print("-" * 30)

    try:
        # Build limit order
        order = Order(
            coin="ETH",
            side=Side.BUY,
            size=0.1,
            limit_price=2000.0,  # Limit price
            reduce_only=False,
        )

        # Place order
        result = sdk.place_order(order)
        print(f"Order placed: {result}")
        print(f"  Order ID: {result.oid}")
        print(f"  Status: {result.status}")
    except Exception as e:
        print(f"  Error: {e}")

    print()

    # ==========================================================================
    # Example 3: Stop Loss Order
    # ==========================================================================
    print("Example 3: Stop Loss Order")
    print("-" * 30)

    try:
        order = Order(
            coin="BTC",
            side=Side.SELL,
            size=0.01,
            trigger_price=60000.0,  # Stop triggers at this price
            limit_price=59900.0,    # Then executes as limit at this price
            reduce_only=True,       # Only reduce existing position
        )

        result = sdk.place_order(order)
        print(f"Stop loss placed: {result}")
    except Exception as e:
        print(f"  Error: {e}")

    print()

    # ==========================================================================
    # Example 4: Cancel Orders
    # ==========================================================================
    print("Example 4: Cancel All Orders")
    print("-" * 30)

    try:
        # Cancel all BTC orders
        sdk.cancel_all_orders("BTC")
        print("Cancelled all BTC orders")
    except Exception as e:
        print(f"  Error: {e}")

    print()

    # ==========================================================================
    # Example 5: Close Position
    # ==========================================================================
    print("Example 5: Close Position")
    print("-" * 30)

    try:
        # Market close BTC position
        result = sdk.market_close("BTC")
        print(f"Position closed: {result}")
    except Exception as e:
        print(f"  Error: {e}")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
