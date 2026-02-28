#!/usr/bin/env python3
"""
Preflight Validation Example

Validate an order BEFORE signing to catch tick size and lot size errors.
Saves failed transactions by checking validity upfront.

No endpoint or private key needed - uses public API.

Requirements:
    pip install hyperliquid-sdk

Usage:
    python preflight.py
"""

from hyperliquid_sdk import HyperliquidSDK


def main():
    print("Hyperliquid Preflight Validation Example")
    print("=" * 50)

    # No endpoint or private key needed for read-only public queries
    sdk = HyperliquidSDK()

    # Get current price
    mid = sdk.get_mid("BTC")
    print(f"BTC mid: ${mid:,.2f}")

    # Validate a good order
    result = sdk.preflight("BTC", "buy", int(mid * 0.97), 0.001)
    print(f"Valid order: {result}")

    # Validate an order with too many decimals (will fail)
    result2 = sdk.preflight("BTC", "buy", 67000.123456789, 0.001)
    print(f"Invalid price: {result2}")
    if not result2.get("valid") and result2.get("errors"):
        errors = result2.get("errors", [])
        if errors:
            print(f"  Field: {errors[0].get('field')}")
            print(f"  Error: {errors[0].get('error')}")


if __name__ == "__main__":
    main()
