#!/usr/bin/env python3
"""
Isolated Margin Example

Add or remove margin from an isolated position.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python isolated_margin.py
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
    print("Hyperliquid Isolated Margin Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Add $100 margin to BTC long position (is_buy=True for long)
    # result = sdk.update_isolated_margin("BTC", 100, is_buy=True)
    # print(f"Add margin result: {result}")

    # Remove $50 margin from ETH short position (is_buy=False for short)
    # result = sdk.update_isolated_margin("ETH", -50, is_buy=False)
    # print(f"Remove margin result: {result}")

    # Top up isolated-only margin (special maintenance mode)
    # result = sdk.top_up_isolated_only_margin("BTC", 100)
    # print(f"Top up isolated-only margin result: {result}")

    print()
    print("Isolated margin methods available:")
    print("  sdk.update_isolated_margin(asset, amount, is_buy)")
    print("  sdk.top_up_isolated_only_margin(asset, amount)")


if __name__ == "__main__":
    main()
