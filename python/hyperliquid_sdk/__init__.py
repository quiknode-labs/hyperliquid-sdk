"""
Hyperliquid SDK — The simplest way to work with Hyperliquid.

All requests route through your QuickNode endpoint — never directly to Hyperliquid.

ONE SDK, ALL APIs:
    >>> from hyperliquid_sdk import HyperliquidSDK
    >>> sdk = HyperliquidSDK("https://your-endpoint.quiknode.pro/TOKEN")

    # Everything through one SDK instance:
    >>> sdk.info.meta()                  # Info API (50+ methods)
    >>> sdk.core.latest_block_number()   # HyperCore (blocks, trades)
    >>> sdk.evm.block_number()           # HyperEVM (Ethereum JSON-RPC)
    >>> sdk.stream.trades(["BTC"], cb)   # WebSocket streaming
    >>> sdk.grpc.trades(["BTC"], cb)     # gRPC streaming
    >>> sdk.evm_stream.new_heads(cb)     # EVM WebSocket (eth_subscribe)

TRADING (add private key):
    >>> sdk = HyperliquidSDK("https://...", private_key="0x...")
    >>> sdk.market_buy("BTC", size=0.001)  # Market buy
    >>> sdk.buy("BTC", size=0.001, price=67000)  # Limit buy
    >>> sdk.close_position("BTC")  # Close position
    >>> sdk.cancel_all()  # Cancel all orders

READ-ONLY (no private key):
    >>> sdk = HyperliquidSDK("https://...")
    >>> sdk.markets()     # Get all markets
    >>> sdk.get_mid("BTC")  # Get mid price
"""

from .client import HyperliquidSDK
from .order import Order, PlacedOrder, Side, TIF, TriggerOrder, TpSl, OrderGrouping
from .info import Info
from .hypercore import HyperCore
from .evm import EVM
from .websocket import Stream, StreamType, ConnectionState
from .grpc_stream import GRPCStream, GRPCStreamType
from .evm_stream import EVMStream, EVMSubscriptionType
from .errors import (
    HyperliquidError,
    BuildError,
    SendError,
    ApprovalError,
    ValidationError,
    SignatureError,
    NoPositionError,
    OrderNotFoundError,
    GeoBlockedError,
    InsufficientMarginError,
    LeverageError,
    RateLimitError,
    MaxOrdersError,
    ReduceOnlyError,
    DuplicateOrderError,
    UserNotFoundError,
    MustDepositError,
    InvalidNonceError,
)

__all__ = [
    # Main SDK (unified entry point)
    "HyperliquidSDK",
    # Order building
    "Order",
    "PlacedOrder",
    "Side",
    "TIF",
    # Trigger orders (stop loss / take profit)
    "TriggerOrder",
    "TpSl",
    "OrderGrouping",
    # Sub-clients (can also be used standalone)
    "Info",
    "HyperCore",
    "EVM",
    "Stream",
    "StreamType",
    "ConnectionState",
    "GRPCStream",
    "GRPCStreamType",
    "EVMStream",
    "EVMSubscriptionType",
    # Errors
    "HyperliquidError",
    "BuildError",
    "SendError",
    "ApprovalError",
    "ValidationError",
    "SignatureError",
    "NoPositionError",
    "OrderNotFoundError",
    "GeoBlockedError",
    "InsufficientMarginError",
    "LeverageError",
    "RateLimitError",
    "MaxOrdersError",
    "ReduceOnlyError",
    "DuplicateOrderError",
    "UserNotFoundError",
    "MustDepositError",
    "InvalidNonceError",
]

__version__ = "0.6.10"
