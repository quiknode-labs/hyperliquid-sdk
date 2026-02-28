#!/usr/bin/env python3
"""
Cancel All Orders Example

Cancel all open orders, or all orders for a specific asset.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python cancel_all.py
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
    print("Hyperliquid Cancel All Orders Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Check open orders first
    orders = sdk.open_orders()
    print(f"Open orders: {orders.count}")

    # Cancel all orders
    result = sdk.cancel_all()
    print(f"Cancel all: {result}")

    # Or cancel just BTC orders:
    # sdk.cancel_all("BTC")


if __name__ == "__main__":
    main()
