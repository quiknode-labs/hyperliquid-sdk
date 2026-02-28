#!/usr/bin/env python3
"""
Markets Example

List all available markets and HIP-3 DEXes.

No endpoint or private key needed - uses public API.

Requirements:
    pip install hyperliquid-sdk

Usage:
    python markets.py
"""

from hyperliquid_sdk import HyperliquidSDK


def main():
    print("Hyperliquid Markets Example")
    print("=" * 50)

    # No endpoint or private key needed for read-only public queries
    sdk = HyperliquidSDK()

    # Get all markets
    markets = sdk.markets()
    perps = markets.get("perps", []) if isinstance(markets, dict) else []
    spot = markets.get("spot", []) if isinstance(markets, dict) else []
    print(f"Perp markets: {len(perps)}")
    print(f"Spot markets: {len(spot)}")

    # Show first 5 perp markets
    print("\nFirst 5 perp markets:")
    for m in perps[:5]:
        name = m.get("name", "?")
        sz_decimals = m.get("szDecimals", "?")
        print(f"  {name}: szDecimals={sz_decimals}")

    # Get HIP-3 DEXes
    dexes = sdk.dexes()
    print(f"\nHIP-3 DEXes: {len(dexes)}")
    for dex in dexes[:5]:
        name = dex.get("name", "N/A") if isinstance(dex, dict) else str(dex)
        print(f"  {name}")


if __name__ == "__main__":
    main()
