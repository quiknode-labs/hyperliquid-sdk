#!/usr/bin/env python3
"""
Cancel Order Example

Place an order and then cancel it by OID.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python cancel_order.py
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
    print("Hyperliquid Cancel Order Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Get current price
    mid = sdk.get_mid("BTC")
    print(f"BTC mid price: ${mid:,.2f}")

    # Place a resting order 3% below mid
    limit_price = int(mid * 0.97)
    order = sdk.buy("BTC", notional=11, price=limit_price, tif="gtc")
    print(f"Placed order OID: {order.oid}")

    # Cancel using the order object
    order.cancel()
    print("Cancelled via order.cancel()")

    # Alternative: cancel by OID directly
    # sdk.cancel(12345, "BTC")


if __name__ == "__main__":
    main()
