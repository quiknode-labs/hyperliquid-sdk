#!/usr/bin/env python3
"""
Schedule Cancel Example (Dead Man's Switch)

Schedule automatic cancellation of all orders after a delay.
If you don't send another schedule_cancel before the time expires,
all your orders are cancelled. Useful as a safety mechanism.

NOTE: Requires $1M trading volume on your account to use this feature.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python schedule_cancel.py
"""

import os
import sys
import time

from hyperliquid_sdk import HyperliquidSDK

PRIVATE_KEY = os.environ.get("PRIVATE_KEY")

if not PRIVATE_KEY:
    print("Error: Set PRIVATE_KEY environment variable")
    print("  export PRIVATE_KEY='0xYourPrivateKey'")
    sys.exit(1)


def main():
    print("Hyperliquid Schedule Cancel Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    print(f"Address: {sdk.address}")

    # Schedule cancel all orders in 60 seconds
    # cancel_time = int(time.time() * 1000) + 60000  # 60 seconds from now
    # result = sdk.schedule_cancel(cancel_time)
    # print(f"Scheduled cancel at {cancel_time}: {result}")

    # To cancel the scheduled cancel (keep orders alive):
    # result = sdk.schedule_cancel(None)
    # print(f"Cancelled scheduled cancel: {result}")

    print()
    print("Schedule cancel methods available:")
    print("  sdk.schedule_cancel(time_ms)  # Schedule cancel at timestamp")
    print("  sdk.schedule_cancel(None)     # Cancel the scheduled cancel")
    print()
    print("NOTE: Requires $1M trading volume on your account")


if __name__ == "__main__":
    main()
