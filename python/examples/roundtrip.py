#!/usr/bin/env python3
"""
Round Trip Example

Complete trade cycle: buy then sell to end up flat.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python roundtrip.py
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
    print("Hyperliquid Round Trip Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Buy $11 worth of BTC
    print("Buying BTC...")
    buy = sdk.market_buy("BTC", notional=11)
    filled_size = buy.filled_size or buy.size
    print(f"  Bought: {filled_size} BTC")
    print(f"  Status: {buy.status}")

    # Sell the same amount
    print("Selling BTC...")
    sell = sdk.market_sell("BTC", size=filled_size)
    print(f"  Sold: {sell.filled_size or sell.size} BTC")
    print(f"  Status: {sell.status}")

    print("Done! Position should be flat.")


if __name__ == "__main__":
    main()
