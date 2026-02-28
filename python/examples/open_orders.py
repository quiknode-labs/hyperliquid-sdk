#!/usr/bin/env python3
"""
Open Orders Example

View all open orders with details.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python open_orders.py
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
    print("Hyperliquid Open Orders Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Get all open orders
    result = sdk.open_orders()
    print(f"Open orders: {result.count}")

    for o in result.orders:
        side = "BUY" if o.side == "B" else "SELL"
        print(f"  {o.name} {side} {o.sz} @ {o.limit_px} (OID: {o.oid})")

    # Get order status for a specific order
    # status = sdk.order_status(12345)
    # print(f"Order status: {status}")


if __name__ == "__main__":
    main()
