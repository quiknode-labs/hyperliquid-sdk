#!/usr/bin/env python3
"""
Info API Example — Query market data and user info.

This example shows how to query exchange metadata, prices, user positions, and more.

Requirements:
    pip install hyperliquid-sdk

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    python info_example.py
"""

import os
import sys

from hyperliquid_sdk import HyperliquidSDK

# Get endpoint from environment
ENDPOINT = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")

if not ENDPOINT:
    print("Error: Set ENDPOINT environment variable")
    print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
    sys.exit(1)


def main():
    print("Hyperliquid Info API Example")
    print("=" * 50)
    print(f"Endpoint: {ENDPOINT[:50]}...")
    print()

    # Create SDK
    sdk = HyperliquidSDK(ENDPOINT)

    # ==========================================================================
    # Market Data
    # ==========================================================================
    print("Market Data")
    print("-" * 30)

    # Get all mid prices
    mids = sdk.info().all_mids()
    print(f"BTC mid: ${float(mids.get('BTC', 0)):,.2f}")
    print(f"ETH mid: ${float(mids.get('ETH', 0)):,.2f}")
    print(f"Total assets: {len(mids)}")
    print()

    # Get L2 order book
    book = sdk.info().l2_book("BTC")
    levels = book.get("levels", [[], []])
    bids = levels[0] if len(levels) > 0 else []
    asks = levels[1] if len(levels) > 1 else []

    if bids and asks:
        best_bid = bids[0]
        best_ask = asks[0]
        spread = float(best_ask["px"]) - float(best_bid["px"])
        print(f"BTC Book:")
        print(f"  Best Bid: {best_bid['sz']} @ ${float(best_bid['px']):,.2f}")
        print(f"  Best Ask: {best_ask['sz']} @ ${float(best_ask['px']):,.2f}")
        print(f"  Spread: ${spread:,.2f}")
    print()

    # Get recent trades
    trades = sdk.info().recent_trades("ETH")
    print(f"Recent ETH trades: {len(trades)}")
    if trades:
        last_trade = trades[0]
        print(f"  Last: {last_trade.get('sz')} @ ${float(last_trade.get('px', 0)):,.2f}")
    print()

    # ==========================================================================
    # Exchange Metadata
    # ==========================================================================
    print("Exchange Metadata")
    print("-" * 30)

    meta = sdk.info().meta()
    universe = meta.get("universe", [])
    print(f"Total perp markets: {len(universe)}")

    # Show a few markets
    for asset in universe[:5]:
        name = asset.get("name", "?")
        sz_decimals = asset.get("szDecimals", "?")
        print(f"  {name}: {sz_decimals} size decimals")
    print()

    # ==========================================================================
    # User Account (requires a valid address)
    # ==========================================================================
    print("User Account")
    print("-" * 30)

    # Example address - replace with your address
    user_address = "0x0000000000000000000000000000000000000000"

    try:
        # Get clearinghouse state (positions, margin)
        state = sdk.info().clearinghouse_state(user_address)
        equity = float(state.get("marginSummary", {}).get("accountValue", 0))
        print(f"Account equity: ${equity:,.2f}")

        positions = state.get("assetPositions", [])
        if positions:
            print(f"Open positions: {len(positions)}")
            for pos in positions[:3]:
                coin = pos.get("position", {}).get("coin", "?")
                size = pos.get("position", {}).get("szi", "0")
                entry = pos.get("position", {}).get("entryPx", "?")
                pnl = pos.get("position", {}).get("unrealizedPnl", "0")
                print(f"  {coin}: {size} @ {entry} (PnL: ${float(pnl):,.2f})")
        else:
            print("  No open positions")
    except Exception as e:
        print(f"  Could not fetch user data: {e}")

    print()

    # ==========================================================================
    # Funding Rates
    # ==========================================================================
    print("Funding Rates")
    print("-" * 30)

    try:
        fundings = sdk.info().predicted_fundings()
        # API returns [[coin, [[venue, fundingInfo], ...]], ...]
        print(f"Predicted funding rates for {len(fundings)} assets:")
        count = 0
        for f in fundings:
            if count >= 5:
                break
            if not isinstance(f, list) or len(f) < 2:
                continue
            coin = f[0]
            venues = f[1]
            if not venues:
                continue
            # Get first venue's funding rate
            for v in venues:
                if not isinstance(v, list) or len(v) < 2:
                    continue
                funding_info = v[1]
                if not isinstance(funding_info, dict):
                    continue
                rate = float(funding_info.get("fundingRate", 0)) * 100
                print(f"  {coin}: {rate:.4f}%")
                count += 1
                break
    except Exception as e:
        print(f"  Could not fetch funding: {e}")

    print()
    print("=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
