#!/usr/bin/env python3
"""
Cancel by Client Order ID (CLOID) Example

Cancel an order using a client-provided order ID instead of the exchange OID.
Useful when you track orders by your own IDs.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python cancel_by_cloid.py
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
    print("Hyperliquid Cancel by CLOID Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Note: CLOIDs are hex strings you provide when placing orders
    # This example shows the cancel_by_cloid API

    # Cancel by client order ID
    # sdk.cancel_by_cloid("0x1234567890abcdef...", "BTC")

    print()
    print("cancel_by_cloid() cancels orders by your custom client order ID")
    print("Usage: sdk.cancel_by_cloid(cloid, asset)")


if __name__ == "__main__":
    main()
