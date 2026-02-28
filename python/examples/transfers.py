#!/usr/bin/env python3
"""
Transfers Example

Transfer USD and spot assets between accounts and wallets.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python transfers.py
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
    print("Hyperliquid Transfers Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Transfer USD to another address
    # result = sdk.transfer_usd("0x1234567890123456789012345678901234567890", 10.0)
    # print(f"USD transfer: {result}")

    # Transfer spot asset to another address
    # result = sdk.transfer_spot("PURR", "0x1234567890123456789012345678901234567890", 100.0)
    # print(f"Spot transfer: {result}")

    # Transfer from spot wallet to perp wallet (internal)
    # result = sdk.transfer_spot_to_perp(100.0)
    # print(f"Spot to perp: {result}")

    # Transfer from perp wallet to spot wallet (internal)
    # result = sdk.transfer_perp_to_spot(100.0)
    # print(f"Perp to spot: {result}")

    # Send asset (generalized transfer)
    # result = sdk.send_asset("USDC", "100.0", "0x1234567890123456789012345678901234567890")
    # print(f"Send asset: {result}")

    print()
    print("Transfer methods available:")
    print("  sdk.transfer_usd(destination, amount)")
    print("  sdk.transfer_spot(token, destination, amount)")
    print("  sdk.transfer_spot_to_perp(amount)")
    print("  sdk.transfer_perp_to_spot(amount)")
    print("  sdk.send_asset(token, amount, destination)")


if __name__ == "__main__":
    main()
