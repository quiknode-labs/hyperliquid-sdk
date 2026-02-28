#!/usr/bin/env python3
"""
Vaults & Delegation Example

Shows how to query vault information and user delegations.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    export USER_ADDRESS="0x..."  # Optional
    python info_vaults.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")
USER = os.environ.get("USER_ADDRESS")

if not ENDPOINT:
    print("Error: Set ENDPOINT environment variable")
    print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("=" * 50)
    print("Vaults & Delegation")
    print("=" * 50)

    # Single SDK instance - access everything through sdk.info(), sdk.core(), sdk.evm()
    sdk = HyperliquidSDK(ENDPOINT)
    info = sdk.info()

    # Vault summaries
    print("\n1. Vault Summaries:")
    vaults = info.vault_summaries()
    print(f"   Total: {len(vaults)}")
    for v in vaults[:3]:
        name = v.get("name", "N/A")
        tvl = v.get("tvl", "?")
        print(f"   - {name}: TVL ${tvl}")

    # User delegations
    if USER:
        print(f"\n2. Delegations ({USER[:10]}...):")
        delegations = info.delegations(USER)
        if delegations:
            print(f"   {len(delegations)} active")
        else:
            print("   None")
    else:
        print("\n(Set USER_ADDRESS for delegation info)")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
