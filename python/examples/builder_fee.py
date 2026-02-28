#!/usr/bin/env python3
"""
Builder Fee Example

Approve and revoke builder fee permissions.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python builder_fee.py
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
    print("Hyperliquid Builder Fee Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Check approval status (doesn't require deposit)
    status = sdk.approval_status()
    print(f"Approval status: {status}")

    # Approve builder fee (required before trading via QuickNode)
    # Note: Requires account to have deposited first
    # result = sdk.approve_builder_fee("1%")
    # print(f"Approve builder fee: {result}")

    # Revoke builder fee permission
    # result = sdk.revoke_builder_fee()
    # print(f"Revoke builder fee: {result}")

    print()
    print("Builder fee methods available:")
    print("  sdk.approve_builder_fee(max_fee, builder=None)")
    print("  sdk.revoke_builder_fee(builder=None)")
    print("  sdk.approval_status(user=None)")


if __name__ == "__main__":
    main()
