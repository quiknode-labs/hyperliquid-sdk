#!/usr/bin/env python3
"""
Close Position Example

Close an open position completely. The SDK figures out the size and direction.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python close_position.py
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
    print("Hyperliquid Close Position Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Close BTC position (if any)
    # The SDK queries your position and builds the counter-order automatically
    try:
        result = sdk.close_position("BTC")
        print(f"Closed position: {result}")
    except Exception as e:
        print(f"No position to close or error: {e}")


if __name__ == "__main__":
    main()
