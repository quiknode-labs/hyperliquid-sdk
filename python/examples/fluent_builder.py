#!/usr/bin/env python3
"""
Fluent Order Builder Example

For power users who want maximum control with IDE autocomplete.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export PRIVATE_KEY="0x..."
    python fluent_builder.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK, Order

PRIVATE_KEY = os.environ.get("PRIVATE_KEY")

if not PRIVATE_KEY:
    print("Error: Set PRIVATE_KEY environment variable")
    print("  export PRIVATE_KEY='0xYourPrivateKey'")
    sys.exit(1)


def main():
    print("Hyperliquid Fluent Order Builder Example")
    print("=" * 50)

    sdk = HyperliquidSDK(private_key=PRIVATE_KEY)
    mid = sdk.get_mid("BTC")
    print(f"BTC mid price: ${mid:,.2f}")

    # Simple limit order with GTC (Good Till Cancelled) - minimum $10 value
    # Use size directly to ensure proper decimal precision (BTC allows 5 decimals)
    order = sdk.order(
        Order.buy("BTC")
        .size(0.00017)  # ~$11 worth at ~$65k (minimum is $10)
        .price(int(mid * 0.97))
        .gtc()
    )
    print(f"Limit GTC: {order}")
    order.cancel()

    # Market order by notional value
    # order = sdk.order(
    #     Order.sell("ETH")
    #     .notional(10)
    #     .market()
    # )
    # print(f"Market: {order}")

    # Reduce-only order (only closes existing position)
    # order = sdk.order(
    #     Order.sell("BTC")
    #     .size(0.001)
    #     .price(int(mid * 1.03))
    #     .gtc()
    #     .reduce_only()
    # )
    # print(f"Reduce-only: {order}")

    # ALO order (Add Liquidity Only / Post-Only)
    # order = sdk.order(
    #     Order.buy("BTC")
    #     .size(0.001)
    #     .price(int(mid * 0.95))
    #     .alo()
    # )
    # print(f"Post-only: {order}")

    print()
    print("Fluent builder methods:")
    print("  .size(0.001)       - Set size in asset units")
    print("  .notional(100)     - Set size in USD")
    print("  .price(65000)      - Set limit price")
    print("  .gtc()             - Good Till Cancelled")
    print("  .ioc()             - Immediate Or Cancel")
    print("  .alo()             - Add Liquidity Only (post-only)")
    print("  .market()          - Market order")
    print("  .reduce_only()     - Only close position")


if __name__ == "__main__":
    main()
