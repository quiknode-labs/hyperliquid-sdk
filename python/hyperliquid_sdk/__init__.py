"""
Hyperliquid SDK — The simplest way to trade on Hyperliquid.

TRADING (no endpoint needed - uses public API):
    >>> from hyperliquid_sdk import HyperliquidSDK
    >>> sdk = HyperliquidSDK()
    >>> sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC

INFO QUERIES (requires QuickNode endpoint):
    >>> from hyperliquid_sdk import Info
    >>> info = Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> info.meta()                  # Exchange metadata
    >>> info.clearinghouse_state("0x...")  # User positions

HYPERCORE (requires QuickNode endpoint - blocks, trades, orders):
    >>> from hyperliquid_sdk import HyperCore
    >>> hc = HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> hc.latest_block_number()     # Latest block
    >>> hc.latest_trades(count=10)   # Recent trades (all coins)

EVM (requires QuickNode endpoint):
    >>> from hyperliquid_sdk import EVM
    >>> evm = EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> evm.block_number()
    >>> evm.get_balance("0x...")

WEBSOCKET STREAMING (requires QuickNode endpoint + WebSocket add-on):
    >>> from hyperliquid_sdk import Stream
    >>> stream = Stream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> stream.trades(["BTC"], lambda t: print(t))
    >>> stream.run()

GRPC STREAMING (requires QuickNode endpoint + gRPC add-on):
    >>> from hyperliquid_sdk import GRPCStream
    >>> stream = GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> stream.trades(["BTC"], lambda t: print(t))
    >>> stream.run()
"""

from .client import HyperliquidSDK
from .order import Order, PlacedOrder, Side, TIF
from .info import Info
from .hypercore import HyperCore
from .evm import EVM
from .websocket import Stream, StreamType, ConnectionState
from .grpc_stream import GRPCStream, GRPCStreamType
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
    # Trading (no endpoint needed)
    "HyperliquidSDK",
    "Order",
    "PlacedOrder",
    "Side",
    "TIF",
    # Info queries (requires endpoint)
    "Info",
    # HyperCore (requires endpoint)
    "HyperCore",
    # EVM (requires endpoint)
    "EVM",
    # WebSocket Streaming (requires endpoint + WebSocket add-on)
    "Stream",
    "StreamType",
    "ConnectionState",
    # gRPC Streaming (requires endpoint + gRPC add-on)
    "GRPCStream",
    "GRPCStreamType",
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

__version__ = "0.5.3"
