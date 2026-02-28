#!/usr/bin/env python3
"""
Market Data Example

Shows how to query market metadata, prices, order book, and recent trades.

The SDK handles all Info API methods automatically.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python info_market_data.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Error: Set ENDPOINT environment variable")
    print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("=" * 50)
    print("Market Data (Info API)")
    print("=" * 50)

    # Single SDK instance - access everything through sdk.info(), sdk.core(), sdk.evm()
    sdk = HyperliquidSDK(ENDPOINT)
    info = sdk.info()

    # Exchange metadata
    print("\n1. Exchange Metadata:")
    try:
        meta = info.meta()
        universe = meta.get("universe", [])
        print(f"   Perp Markets: {len(universe)}")
        for asset in universe[:5]:
            print(f"   - {asset.get('name')}: max leverage {asset.get('maxLeverage')}x")
    except Exception as e:
        print(f"   (meta not available: {e})")

    # Spot metadata
    print("\n2. Spot Metadata:")
    try:
        spot = info.spot_meta()
        tokens = spot.get("tokens", [])
        print(f"   Spot Tokens: {len(tokens)}")
    except Exception as e:
        print(f"   (spot_meta not available: {e})")

    # Exchange status
    print("\n3. Exchange Status:")
    try:
        status = info.exchange_status()
        print(f"   {status}")
    except Exception as e:
        print(f"   (exchange_status not available: {e})")

    # All mid prices
    print("\n4. Mid Prices:")
    try:
        mids = info.all_mids()
        btc_price = float(mids.get("BTC", "0"))
        eth_price = float(mids.get("ETH", "0"))
        print(f"   BTC: ${btc_price:,.2f}")
        print(f"   ETH: ${eth_price:,.2f}")
    except Exception as e:
        print(f"   (all_mids not available: {e})")

    # Order book
    print("\n5. Order Book (BTC):")
    try:
        book = info.l2_book("BTC")
        levels = book.get("levels", [[], []])
        if levels[0] and levels[1]:
            best_bid = float(levels[0][0].get("px", "0"))
            best_ask = float(levels[1][0].get("px", "0"))
            spread = best_ask - best_bid
            print(f"   Best Bid: ${best_bid:,.2f}")
            print(f"   Best Ask: ${best_ask:,.2f}")
            print(f"   Spread: ${spread:.2f}")
    except Exception as e:
        print(f"   (l2_book not available: {e})")

    # Recent trades
    print("\n6. Recent Trades (BTC):")
    try:
        trades = info.recent_trades("BTC")
        for t in trades[:3]:
            side = "BUY" if t.get("side") == "B" else "SELL"
            px = float(t.get("px", "0"))
            print(f"   {side} {t.get('sz')} @ ${px:,.2f}")
    except Exception as e:
        print(f"   (recent_trades not available: {e})")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
