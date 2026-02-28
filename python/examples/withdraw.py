#!/usr/bin/env python3
"""
Withdraw Example

Withdraw USDC to L1 (Arbitrum).

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python withdraw.py
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
    print("Hyperliquid Withdraw Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Withdraw USDC to L1 (Arbitrum)
    # WARNING: This is a real withdrawal - be careful with amounts
    # result = sdk.withdraw("0x1234567890123456789012345678901234567890", 100.0)
    # print(f"Withdraw: {result}")

    print()
    print("Withdraw methods available:")
    print("  sdk.withdraw(destination, amount)")
    print("  Note: Withdraws USDC to your L1 Arbitrum address")


if __name__ == "__main__":
    main()
