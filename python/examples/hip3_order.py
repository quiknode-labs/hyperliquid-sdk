#!/usr/bin/env python3
"""
HIP-3 Market Order Example

Trade on HIP-3 markets (community perps like Hypersea).
Same API as regular markets, just use "dex:symbol" format.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python hip3_order.py
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
    print("Hyperliquid HIP-3 Order Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # List HIP-3 DEXes
    dexes = sdk.dexes()
    print("Available HIP-3 DEXes:")
    for dex in dexes[:5]:
        name = dex.get("name", dex) if isinstance(dex, dict) else str(dex)
        print(f"  {name}")

    # Trade on a HIP-3 market
    # Format: "dex:SYMBOL"
    # order = sdk.buy("xyz:SILVER", notional=11, tif="ioc")
    # print(f"HIP-3 order: {order}")

    print()
    print("HIP-3 markets use 'dex:SYMBOL' format")
    print("Example: sdk.buy('xyz:SILVER', notional=11)")


if __name__ == "__main__":
    main()
