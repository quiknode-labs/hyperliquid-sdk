#!/usr/bin/env python3
"""
Vaults Example

Deposit and withdraw from Hyperliquid vaults.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python vaults.py
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
    print("Hyperliquid Vaults Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Wallet: {sdk.address}")

    # Example vault address (HLP vault)
    HLP_VAULT = "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303"

    # Deposit to vault
    # result = sdk.vault_deposit(HLP_VAULT, 100.0)
    # print(f"Vault deposit: {result}")

    # Withdraw from vault
    # result = sdk.vault_withdraw(HLP_VAULT, 50.0)
    # print(f"Vault withdraw: {result}")

    print()
    print("Vault methods available:")
    print("  sdk.vault_deposit(vault_address, amount)")
    print("  sdk.vault_withdraw(vault_address, amount)")


if __name__ == "__main__":
    main()
