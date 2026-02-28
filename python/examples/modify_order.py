#!/usr/bin/env python3
"""
Modify Order Example

Place a resting order and then modify its price.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python modify_order.py
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
    print("Hyperliquid Modify Order Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Get current price
    mid = sdk.get_mid("BTC")
    print(f"BTC mid price: ${mid:,.2f}")

    # Place a resting order
    limit_price = int(mid * 0.97)
    order = sdk.buy("BTC", notional=11, price=limit_price, tif="gtc")
    print(f"Placed order at ${limit_price:,}")
    print(f"  OID: {order.oid}")

    # Modify to a new price (4% below mid)
    new_price = int(mid * 0.96)
    new_order = order.modify(price=new_price)
    print(f"Modified to ${new_price:,}")
    print(f"  New OID: {new_order.oid}")

    # Clean up
    new_order.cancel()
    print("Order cancelled.")


if __name__ == "__main__":
    main()
