#!/usr/bin/env python3
"""
Staking Example

Stake and unstake HYPE tokens.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python staking.py
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
    print("Hyperliquid Staking Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Stake HYPE tokens
    # result = sdk.stake(100)
    # print(f"Stake: {result}")

    # Unstake HYPE tokens
    # result = sdk.unstake(50)
    # print(f"Unstake: {result}")

    # Delegate to a validator
    # result = sdk.delegate("0x...", 100)
    # print(f"Delegate: {result}")

    # Undelegate from a validator
    # result = sdk.undelegate("0x...", 50)
    # print(f"Undelegate: {result}")

    print()
    print("Staking methods available:")
    print("  sdk.stake(amount)")
    print("  sdk.unstake(amount)")
    print("  sdk.delegate(validator, amount)")
    print("  sdk.undelegate(validator, amount)")


if __name__ == "__main__":
    main()
