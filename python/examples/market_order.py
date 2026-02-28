#!/usr/bin/env python3
"""
Market Order Example

Place a market order that executes immediately at best available price.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python market_order.py
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
    print("Hyperliquid Market Order Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Market buy by notional ($11 worth of BTC - minimum is $10)
    order = sdk.market_buy("BTC", notional=11)
    print(f"Market buy: {order}")
    print(f"  Status: {order.status}")
    print(f"  OID: {order.oid}")

    # Market buy by notional ($10 worth of ETH)
    # order = sdk.market_buy("ETH", notional=10)

    # Market sell
    # order = sdk.market_sell("BTC", size=0.0001)


if __name__ == "__main__":
    main()
