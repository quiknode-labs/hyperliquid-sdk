#!/usr/bin/env python3
"""
Trigger Orders Example

Stop loss and take profit orders.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python trigger_orders.py
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
    print("Hyperliquid Trigger Orders Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    mid = sdk.get_mid("BTC")
    print(f"BTC mid: ${mid:,.2f}")

    # Stop loss order (market) - triggers when price falls below stop price
    # No limit_price means market order when triggered
    # result = sdk.stop_loss("BTC", size=0.001, trigger_price=mid * 0.95)
    # print(f"Stop loss (market): {result}")

    # Stop loss order (limit) - triggers and places limit order at limit_price
    # result = sdk.stop_loss("BTC", size=0.001, trigger_price=mid * 0.95, limit_price=mid * 0.94)
    # print(f"Stop loss (limit): {result}")

    # Take profit order (market) - triggers when price rises above trigger
    # result = sdk.take_profit("BTC", size=0.001, trigger_price=mid * 1.05)
    # print(f"Take profit (market): {result}")

    # Take profit order (limit)
    # result = sdk.take_profit("BTC", size=0.001, trigger_price=mid * 1.05, limit_price=mid * 1.06)
    # print(f"Take profit (limit): {result}")

    # For buy-side stop/TP (e.g., closing a short position), use side="buy"
    # result = sdk.stop_loss("BTC", size=0.001, trigger_price=mid * 1.05, side="buy")

    print()
    print("Trigger order methods available:")
    print("  sdk.stop_loss(asset, size, trigger_price, limit_price=None, side=None)")
    print("  sdk.take_profit(asset, size, trigger_price, limit_price=None, side=None)")
    print("  sdk.trigger_order(TriggerOrder(...))")
    print()
    print("Note: Omit limit_price for market orders when triggered")


if __name__ == "__main__":
    main()
