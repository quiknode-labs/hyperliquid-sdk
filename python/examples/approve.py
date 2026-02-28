#!/usr/bin/env python3
"""
Builder Fee Approval Example

Approve the builder fee to enable trading through the API.
Required before placing orders.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python approve.py
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
    print("Hyperliquid Builder Fee Approval Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Check current approval status
    status = sdk.approval_status()
    print(f"Currently approved: {status.get('approved', False)}")
    if status.get("approved"):
        print(f"Max fee rate: {status.get('maxFeeRate')}")

    # Approve builder fee (1% max)
    # result = sdk.approve_builder_fee("1%")
    # print(f"Approved: {result}")

    # Or use auto_approve when creating SDK:
    # sdk = HyperliquidSDK(private_key=PRIVATE_KEY, auto_approve=True)

    # Revoke approval:
    # sdk.revoke_builder_fee()

    print()
    print("Builder fee methods available:")
    print("  sdk.approve_builder_fee(max_fee, builder=None)")
    print("  sdk.revoke_builder_fee(builder=None)")
    print("  sdk.approval_status(user=None)")


if __name__ == "__main__":
    main()
