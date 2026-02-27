"""
Hyperliquid SDK — The simplest way to trade on Hyperliquid.

One line to place orders, zero ceremony:

    >>> from hyperliquid_sdk import HyperliquidSDK
    >>> sdk = HyperliquidSDK()
    >>> order = sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC

That's it. No build-sign-send ceremony. No manual hash signing. Just trading.

Examples:
    # Market orders
    sdk.market_buy("BTC", size=0.001)
    sdk.market_sell("ETH", notional=100)

    # Limit orders
    sdk.buy("BTC", size=0.001, price=65000, tif="gtc")

    # Order management
    order = sdk.buy("BTC", size=0.001, price=60000, tif="gtc")
    order.modify(price=61000)
    order.cancel()

    # Position management
    sdk.close_position("BTC")
    sdk.cancel_all()

    # Fluent builder (power users)
    sdk.order(Order.buy("BTC").size(0.001).price(65000).gtc())
"""

from .client import HyperliquidSDK
from .order import Order, PlacedOrder, Side, TIF
from .errors import (
    HyperliquidError,
    BuildError,
    SendError,
    ApprovalError,
    ValidationError,
    GeoBlockedError,
    NoPositionError,
    OrderNotFoundError,
)

__all__ = [
    # Main client
    "HyperliquidSDK",
    # Order types
    "Order",
    "PlacedOrder",
    "Side",
    "TIF",
    # Errors
    "HyperliquidError",
    "BuildError",
    "SendError",
    "ApprovalError",
    "ValidationError",
    "GeoBlockedError",
    "NoPositionError",
    "OrderNotFoundError",
]

__version__ = "0.1.0"
