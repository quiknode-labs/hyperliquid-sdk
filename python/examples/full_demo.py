#!/usr/bin/env python3
"""
Full Demo — Comprehensive example of all SDK capabilities.

This example demonstrates all major SDK features:
- Info API (market data, user info)
- HyperCore API (blocks, trades, orders)
- EVM API (chain data, balances)
- WebSocket streaming
- gRPC streaming
- Trading (orders, positions)

Requirements:
    pip install hyperliquid-sdk[all]

Usage:
    export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
    export PRIVATE_KEY="0x..."  # Optional, for trading
    python full_demo.py
"""

import os
import sys
import time

# Optional: Set endpoint via command line
if len(sys.argv) > 1:
    os.environ["ENDPOINT"] = sys.argv[1]

from hyperliquid_sdk import HyperliquidSDK, HyperliquidError


def separator(title: str):
    """Print a section separator."""
    print()
    print("=" * 60)
    print(f"  {title}")
    print("=" * 60)


def subsection(title: str):
    """Print a subsection."""
    print()
    print(f"--- {title} ---")


def demo_info_api(sdk: HyperliquidSDK):
    """Demonstrate Info API capabilities."""
    separator("INFO API")

    subsection("Market Prices")
    mids = sdk.info().all_mids()
    print(f"Total markets: {len(mids)}")
    for coin in ["BTC", "ETH", "SOL", "DOGE"]:
        if coin in mids:
            print(f"  {coin}: ${float(mids[coin]):,.2f}")

    subsection("Order Book")
    book = sdk.info().l2_book("BTC")
    levels = book.get("levels", [[], []])
    bids = levels[0] if len(levels) > 0 else []
    asks = levels[1] if len(levels) > 1 else []
    if bids and asks:
        print(f"  Best Bid: {bids[0]['sz']} @ ${float(bids[0]['px']):,.2f}")
        print(f"  Best Ask: {asks[0]['sz']} @ ${float(asks[0]['px']):,.2f}")
        print(f"  Spread: ${float(asks[0]['px']) - float(bids[0]['px']):,.2f}")

    subsection("Recent Trades")
    trades = sdk.info().recent_trades("ETH")
    print(f"Last 3 ETH trades:")
    for t in trades[:3]:
        print(f"  {t['sz']} @ ${float(t['px']):,.2f} ({t['side']})")

    subsection("Exchange Metadata")
    meta = sdk.info().meta()
    universe = meta.get("universe", [])
    print(f"Total perp markets: {len(universe)}")

    subsection("Predicted Funding")
    fundings = sdk.info().predicted_fundings()
    # Extract funding rates - API returns [[coin, [[venue, fundingInfo], ...]], ...]
    entries = []
    for f in fundings:
        if isinstance(f, list) and len(f) >= 2:
            coin = f[0]
            venues = f[1]
            if isinstance(venues, list) and len(venues) > 0:
                # Use first venue's funding rate
                for v in venues:
                    if isinstance(v, list) and len(v) >= 2:
                        funding_info = v[1]
                        if isinstance(funding_info, dict):
                            rate = float(funding_info.get("fundingRate", 0))
                            entries.append({"coin": coin, "rate": rate})
                            break

    sorted_entries = sorted(entries, key=lambda x: abs(x["rate"]), reverse=True)
    print(f"Top 3 funding rates:")
    for e in sorted_entries[:3]:
        rate = e["rate"] * 100
        print(f"  {e['coin']}: {rate:+.4f}% (8h)")


def demo_hypercore_api(sdk: HyperliquidSDK):
    """Demonstrate HyperCore API capabilities."""
    separator("HYPERCORE API")

    subsection("Latest Block")
    block_num = sdk.core().latest_block_number()
    print(f"Latest block: {block_num:,}")

    block = sdk.core().get_block(block_num)
    if block:
        events = block.get("events", [])
        print(f"Block {block_num}: {len(events)} events")

    subsection("Recent Trades")
    trades = sdk.core().latest_trades(count=5)
    print(f"Last 5 trades across all markets:")
    for t in trades:
        coin = t.get("coin", "?")
        print(f"  {coin}: {t.get('sz', '?')} @ ${float(t.get('px', 0)):,.2f}")

    subsection("Recent Orders")
    orders = sdk.core().latest_orders(count=5)
    print(f"Last 5 orders:")
    for o in orders:
        coin = o.get("coin", "?")
        side = o.get("side", "?")
        status = o.get("status", "?")
        print(f"  {coin}: {side} @ ${float(o.get('limitPx', 0)):,.2f} - {status}")


def demo_evm_api(sdk: HyperliquidSDK):
    """Demonstrate EVM API capabilities."""
    separator("EVM API")

    subsection("Chain Info")
    chain_id = sdk.evm().chain_id()
    block_num = sdk.evm().block_number()
    gas_price = sdk.evm().gas_price()

    print(f"Chain ID: {chain_id} ({'Mainnet' if chain_id == 999 else 'Testnet'})")
    print(f"Block: {block_num:,}")
    print(f"Gas: {gas_price / 1e9:.2f} Gwei")

    subsection("Latest Block")
    block = sdk.evm().get_block_by_number(block_num)
    if block:
        print(f"Block {block_num}:")
        print(f"  Hash: {block.get('hash', '?')[:30]}...")
        print(f"  Gas Used: {int(block.get('gasUsed', '0x0'), 16):,}")


def demo_websocket(sdk: HyperliquidSDK, duration: int = 5):
    """Demonstrate WebSocket streaming."""
    separator("WEBSOCKET STREAMING")

    trade_count = [0]
    book_count = [0]

    def on_trade(data):
        trade_count[0] += 1
        if trade_count[0] <= 3:
            d = data.get("data", {})
            print(f"  [TRADE] {d.get('coin', '?')}: {d.get('sz', '?')} @ {d.get('px', '?')}")

    def on_book(data):
        book_count[0] += 1
        if book_count[0] <= 2:
            d = data.get("data", {})
            print(f"  [BOOK] {d.get('coin', '?')} update")

    def on_error(e):
        print(f"  [ERROR] {e}")

    print(f"Streaming for {duration} seconds...")

    stream = sdk.stream()
    stream.on_error = on_error
    stream.trades(["BTC", "ETH"], on_trade)
    stream.book_updates(["BTC"], on_book)

    # Run in background
    stream.start()
    time.sleep(duration)
    stream.stop()

    print()
    print(f"Received: {trade_count[0]} trades, {book_count[0]} book updates")


def demo_grpc(sdk: HyperliquidSDK, duration: int = 5):
    """Demonstrate gRPC streaming."""
    separator("GRPC STREAMING")

    trade_count = [0]

    def on_trade(data):
        trade_count[0] += 1
        if trade_count[0] <= 3:
            print(f"  [TRADE] {data}")

    def on_error(e):
        print(f"  [ERROR] {e}")

    print(f"Streaming for {duration} seconds...")

    try:
        stream = sdk.grpc()
        stream.on_error = on_error
        stream.trades(["BTC", "ETH"], on_trade)

        # Run in background
        stream.start()
        time.sleep(duration)
        stream.stop()

        print()
        print(f"Received: {trade_count[0]} trades")
    except Exception as e:
        print(f"gRPC not available: {e}")


def demo_trading(sdk: HyperliquidSDK):
    """Demonstrate trading capabilities (dry-run style)."""
    separator("TRADING")

    print(f"Address: {sdk.address}")

    subsection("Account Check")
    try:
        # This would show positions if the account has any
        print("  Trading SDK initialized successfully")
        print("  Ready to place orders (not executing in demo)")
    except Exception as e:
        print(f"  Note: {e}")

    subsection("Order Building (Example)")
    print("  Market buy: sdk.market_buy('BTC', notional=100)")
    print("  Limit sell: sdk.sell('ETH', size=1.0, price=4000)")
    print("  Close pos:  sdk.close_position('BTC')")


def main():
    print()
    print("*" * 60)
    print("  HYPERLIQUID SDK - FULL DEMO")
    print("*" * 60)

    endpoint = os.environ.get("ENDPOINT") or os.environ.get("QUICKNODE_ENDPOINT")
    private_key = os.environ.get("PRIVATE_KEY")

    if not endpoint:
        print()
        print("Error: ENDPOINT not set")
        print()
        print("Usage:")
        print("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
        print("  python full_demo.py")
        print()
        print("Or:")
        print("  python full_demo.py 'https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
        sys.exit(1)

    print()
    print(f"Endpoint: {endpoint[:50]}...")

    # Create SDK once - use it everywhere
    sdk = HyperliquidSDK(endpoint, private_key=private_key)

    # Run all demos
    try:
        demo_info_api(sdk)
        demo_hypercore_api(sdk)
        demo_evm_api(sdk)
        demo_websocket(sdk, duration=5)
        demo_grpc(sdk, duration=5)

        if private_key:
            demo_trading(sdk)
        else:
            print()
            print("--- TRADING (skipped - no PRIVATE_KEY) ---")

    except HyperliquidError as e:
        print(f"\nError: {e}")
        print(f"Code: {e.code}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nInterrupted")
        sys.exit(0)

    separator("DONE")
    print("All demos completed successfully!")
    print()


if __name__ == "__main__":
    main()
