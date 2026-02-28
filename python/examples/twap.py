#!/usr/bin/env python3
"""
TWAP Orders Example

Time-Weighted Average Price orders for large trades.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python twap.py
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
    print("Hyperliquid TWAP Orders Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    mid = sdk.get_mid("BTC")
    print(f"BTC mid: ${mid:,.2f}")

    # TWAP order - executes over time to minimize market impact
    # result = sdk.twap_order(
    #     "BTC",
    #     size=0.01,           # Total size to execute
    #     is_buy=True,
    #     duration_minutes=60,  # Execute over 60 minutes
    #     randomize=True,       # Randomize execution times
    #     reduce_only=False
    # )
    # print(f"TWAP order: {result}")
    # twap_id = result.get("response", {}).get("data", {}).get("running", {}).get("id")

    # Cancel TWAP order
    # result = sdk.twap_cancel("BTC", twap_id)
    # print(f"TWAP cancel: {result}")

    print()
    print("TWAP methods available:")
    print("  sdk.twap_order(asset, size, is_buy, duration_minutes, randomize=None, reduce_only=None)")
    print("  sdk.twap_cancel(asset, twap_id)")


if __name__ == "__main__":
    main()
