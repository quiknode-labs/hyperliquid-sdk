#!/usr/bin/env python3
"""
Historical Candles Example

Shows how to fetch historical candlestick (OHLCV) data.

Note: candle_snapshot may not be available on all QuickNode endpoints.
Check the QuickNode docs for method availability.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python info_candles.py
"""

import os
import sys
import time

from hyperliquid_sdk import HyperliquidSDK

ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Error: Set ENDPOINT environment variable")
    print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("=" * 50)
    print("Historical Candles")
    print("=" * 50)

    # Single SDK instance - access everything through sdk.info(), sdk.core(), sdk.evm()
    sdk = HyperliquidSDK(ENDPOINT)
    info = sdk.info()

    # Last 24 hours
    now = int(time.time() * 1000)
    day_ago = now - (24 * 60 * 60 * 1000)

    # Fetch BTC 1-hour candles
    print("\n1. BTC 1-Hour Candles (last 24h):")
    try:
        candles = info.candles("BTC", "1h", day_ago, now)
        print(f"   Retrieved {len(candles)} candles")
        if len(candles) > 0:
            for c in candles[-3:]:
                print(f"   O:{c.get('o')} H:{c.get('h')} L:{c.get('l')} C:{c.get('c')}")
    except Exception as e:
        print(f"   Error: {e}")
        print("   Note: candle_snapshot may not be available on this endpoint")

    # Predicted funding rates (supported on QuickNode)
    print("\n2. Predicted Funding Rates:")
    try:
        fundings = info.predicted_fundings()
        print(f"   {len(fundings)} assets with funding rates:")
        # Structure: [[coin, [[source, {fundingRate, ...}], ...]], ...]
        count = 0
        for item in fundings:
            if count >= 5:
                break
            if isinstance(item, list) and len(item) >= 2:
                coin = item[0]
                sources = item[1]
                if sources and isinstance(sources, list) and len(sources) > 0:
                    # Get HlPerp funding rate if available
                    for src in sources:
                        if isinstance(src, list) and len(src) >= 2 and src[0] == "HlPerp":
                            rate = float(src[1].get("fundingRate", "0")) * 100
                            print(f"   {coin}: {rate:.4f}%")
                            count += 1
                            break
    except Exception as e:
        print(f"   Error: {e}")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
