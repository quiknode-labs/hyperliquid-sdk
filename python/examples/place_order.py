#!/usr/bin/env python3
"""
Limit Order Example

Place a limit order that rests on the book until filled or cancelled.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python place_order.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK

PRIVATE_KEY = os.environ.get("PRIVATE_KEY")

if not PRIVATE_KEY:
    print("Error: Set PRIVATE_KEY environment variable")
    print("  export PRIVATE_KEY='0xYourPrivateKey'")
    sys.exit(1)


def main():
    print("Hyperliquid Place Order Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Get current price
    mid = sdk.get_mid("BTC")
    print(f"BTC mid price: ${mid:,.2f}")

    # Place limit buy 3% below mid (GTC = Good Till Cancelled)
    limit_price = int(mid * 0.97)
    order = sdk.buy("BTC", notional=11, price=limit_price, tif="gtc")

    print("Placed limit order:")
    print(f"  OID: {order.oid}")
    print(f"  Price: ${limit_price:,}")
    print(f"  Status: {order.status}")

    # Clean up - cancel the order
    order.cancel()
    print("Order cancelled.")


if __name__ == "__main__":
    main()
