#!/usr/bin/env python3
"""
Leverage Example

Update leverage for a position.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python leverage.py
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
    print("Hyperliquid Leverage Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Update leverage for BTC to 10x cross margin
    result = sdk.update_leverage("BTC", 10, is_cross=True)
    print(f"Update leverage result: {result}")

    # Update leverage for ETH to 5x isolated margin
    # result = sdk.update_leverage("ETH", 5, is_cross=False)
    # print(f"Update leverage result: {result}")

    print()
    print("Leverage methods available:")
    print("  sdk.update_leverage(asset, leverage, is_cross)")


if __name__ == "__main__":
    main()
