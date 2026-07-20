"""
gRPC Stream Client — High-performance real-time data streams with automatic reconnection.

Stream trades, orders, book updates, blocks, and more via gRPC.
Handles connection management, keepalive, and automatic reconnection.

The gRPC API uses Protocol Buffers over HTTP/2 on port 10000.
Authentication is via x-token header with your QuickNode API token.

Example:
    >>> from hyperliquid_sdk import GRPCStream
    >>> stream = GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> stream.trades(["BTC", "ETH"], lambda t: print(t))
    >>> stream.run()
"""

from __future__ import annotations
import json
import threading
import time
import logging
from typing import Optional, List, Callable, Any, Dict, Tuple, Iterator
from urllib.parse import urlparse
from enum import Enum

try:
    import grpc
    HAS_GRPC = True
except ImportError:
    HAS_GRPC = False

# Import proto types (generated from streaming.proto and orderbook.proto)
try:
    from .proto import (
        StreamType as ProtoStreamType,
        SubscribeRequest,
        StreamSubscribe,
        FilterValues,
        Ping,
        PingRequest,
        Timestamp,
        L2BookRequest,
        L2BookDiffRequest,
        L4BookRequest,
        L4BookUpdatesRequest,
        L4OrderDiffType,
        BboBookRequest,
        TpslUpdatesRequest,
        TpslDiffType,
        StreamingStub,
        BlockStreamingStub,
        OrderBookStreamingStub,
    )
    HAS_PROTO = True
except ImportError:
    HAS_PROTO = False

from .errors import HyperliquidError

logger = logging.getLogger(__name__)


class GRPCStreamType(str, Enum):
    """Available gRPC stream types."""
    TRADES = "TRADES"
    ORDERS = "ORDERS"
    BOOK_UPDATES = "BOOK_UPDATES"
    TWAP = "TWAP"
    EVENTS = "EVENTS"
    BLOCKS = "BLOCKS"
    WRITER_ACTIONS = "WRITER_ACTIONS"
    MEMPOOL_TXS = "MEMPOOL_TXS"
    ORDER_PRIORITY = "ORDER_PRIORITY"
    GOSSIP_PRIORITY = "GOSSIP_PRIORITY"


class ConnectionState(str, Enum):
    """gRPC connection states."""
    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    RECONNECTING = "reconnecting"


# Map string stream types to proto enum values
_STREAM_TYPE_MAP = {
    "TRADES": 1,  # ProtoStreamType.TRADES
    "ORDERS": 2,  # ProtoStreamType.ORDERS
    "BOOK_UPDATES": 3,  # ProtoStreamType.BOOK_UPDATES
    "TWAP": 4,  # ProtoStreamType.TWAP
    "EVENTS": 5,  # ProtoStreamType.EVENTS
    "BLOCKS": 6,  # ProtoStreamType.BLOCKS
    "WRITER_ACTIONS": 7,  # ProtoStreamType.WRITER_ACTIONS
    "MEMPOOL_TXS": 8,  # ProtoStreamType.MEMPOOL_TXS
    "ORDER_PRIORITY": 9,  # ProtoStreamType.ORDER_PRIORITY
    "GOSSIP_PRIORITY": 10,  # ProtoStreamType.GOSSIP_PRIORITY
}

# Orderbook stream types (internal) -> OrderBookStreaming RPC method names
_ORDERBOOK_RPC_MAP = {
    "BBO_BOOK": "StreamBboBook",
    "L2_BOOK_DIFF": "StreamL2BookDiff",
    "L4_BOOK_UPDATES": "StreamL4BookUpdates",
    "TPSL_UPDATES": "StreamTpslUpdates",
    "L2_BOOK_PACKED": "StreamL2BookPacked",
    "BBO_BOOK_PACKED": "StreamBboBookPacked",
    "L4_BOOK_BYTES": "StreamL4BookBytes",
}


class GRPCStream:
    """
    gRPC Stream Client — High-performance real-time data streams.

    Features:
    - Automatic reconnection with exponential backoff
    - Keepalive pings to maintain connection
    - Thread-safe subscription management
    - Graceful shutdown
    - Native Protocol Buffer support

    Streams:
    - trades: Executed trades with price, size, direction
    - orders: Order lifecycle events (open, filled, cancelled)
    - book_updates: Order book changes
    - twap: Time-weighted average price execution
    - events: System events (funding, liquidations)
    - blocks: Block data
    - mempool_txs: Pre-consensus mempool transactions
    - order_priority: Derived order/write priority actions
    - gossip_priority: Derived gossip/read priority bid actions
    - l2_book: Level 2 order book (aggregated price levels)
    - l4_book: Level 4 order book (individual orders)
    - bbo_book: Top-of-book (best bid/offer) changes
    - l2_book_diff: Incremental L2 price-level changes
    - l4_book_updates: Typed L4 order diffs (new/update/remove)
    - tpsl_updates: Trigger/TP-SL order add/remove updates
    - l2_book_packed / bbo_book_packed: Fast-path fixed-point (1e8) variants
    - l4_book_bytes: Fast-path L4 with JSON-bytes diffs
    - stream_bytes: Low-level bytes fast path for any stream type

    Examples:
        stream = GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

        # Subscribe to trades
        stream.trades(["BTC", "ETH"], lambda t: print(f"{t['coin']}: {t['sz']} @ {t['px']}"))

        # Subscribe to L2 order book
        stream.l2_book("BTC", lambda b: print(f"Bid: {b['bids'][0]}, Ask: {b['asks'][0]}"))

        # Run (blocking)
        stream.run()

        # Or run in background
        stream.start()
        # ... do other work ...
        stream.stop()
    """

    # gRPC port for Hyperliquid
    GRPC_PORT = 10000

    # Reconnection settings
    INITIAL_RECONNECT_DELAY = 1.0  # seconds
    MAX_RECONNECT_DELAY = 60.0  # seconds
    RECONNECT_BACKOFF_FACTOR = 2.0
    MAX_RECONNECT_ATTEMPTS = None  # Infinite by default

    # Keepalive settings
    KEEPALIVE_TIME_MS = 30000  # Send keepalive every 30 seconds
    KEEPALIVE_TIMEOUT_MS = 10000  # Wait 10 seconds for keepalive response

    def __init__(
        self,
        endpoint: str,
        *,
        on_error: Optional[Callable[[Exception], None]] = None,
        on_close: Optional[Callable[[], None]] = None,
        on_connect: Optional[Callable[[], None]] = None,
        on_reconnect: Optional[Callable[[int], None]] = None,
        on_state_change: Optional[Callable[[ConnectionState], None]] = None,
        secure: bool = True,
        reconnect: bool = True,
        max_reconnect_attempts: Optional[int] = None,
    ):
        """
        Initialize the gRPC stream.

        Args:
            endpoint: Hyperliquid endpoint URL (the token is extracted automatically)
            on_error: Callback for errors (receives Exception)
            on_close: Callback when connection closes permanently
            on_connect: Callback when connected (also on reconnect)
            on_reconnect: Callback on reconnection (receives attempt number)
            on_state_change: Callback when connection state changes
            secure: Use secure channel (TLS). Default: True
            reconnect: Auto-reconnect on disconnect (default: True)
            max_reconnect_attempts: Max reconnection attempts (None = infinite)
        """
        if not HAS_GRPC:
            raise ImportError("grpcio required. Install: pip install hyperliquid-sdk[grpc]")
        if not HAS_PROTO:
            raise ImportError("Proto files not found. Reinstall hyperliquid-sdk.")

        self._host, self._token = self._parse_endpoint(endpoint)
        self._on_error = on_error
        self._on_close = on_close
        self._on_connect = on_connect
        self._on_reconnect = on_reconnect
        self._on_state_change = on_state_change
        self._secure = secure
        self._reconnect_enabled = reconnect
        self._max_reconnect_attempts = max_reconnect_attempts or self.MAX_RECONNECT_ATTEMPTS

        self._channel: Optional[grpc.Channel] = None
        self._streaming_stub: Optional[StreamingStub] = None
        self._block_stub: Optional[BlockStreamingStub] = None
        self._orderbook_stub: Optional[OrderBookStreamingStub] = None
        self._threads: List[threading.Thread] = []
        self._running = False
        self._state = ConnectionState.DISCONNECTED
        self._reconnect_attempt = 0
        self._reconnect_delay = self.INITIAL_RECONNECT_DELAY
        self._lock = threading.RLock()
        self._subscriptions: List[Dict[str, Any]] = []
        self._stop_event = threading.Event()

    def _set_state(self, state: ConnectionState) -> None:
        """Update connection state and notify callback."""
        if self._state != state:
            self._state = state
            if self._on_state_change:
                try:
                    self._on_state_change(state)
                except Exception as e:
                    logger.warning(f"State change callback error: {e}")

    def _safe_callback(self, callback: Callable, data: Any) -> None:
        """Safely invoke a user callback, catching and logging any exceptions."""
        try:
            callback(data)
        except Exception as e:
            logger.warning(f"Callback error: {e}")
            if self._on_error:
                try:
                    self._on_error(e)
                except Exception:
                    pass  # Don't let error callback errors propagate

    def _parse_endpoint(self, url: str) -> Tuple[str, str]:
        """Parse endpoint URL to extract host and token."""
        parsed = urlparse(url)
        host = parsed.netloc

        # Remove port if present
        if ":" in host:
            host = host.split(":")[0]

        # Extract token from path
        path_parts = [p for p in parsed.path.strip("/").split("/") if p]
        token = ""
        for part in path_parts:
            if part not in ("info", "hypercore", "evm", "nanoreth", "ws"):
                token = part
                break

        return host, token

    def _get_target(self) -> str:
        """Get the gRPC target address."""
        return f"{self._host}:{self.GRPC_PORT}"

    def _get_metadata(self) -> List[Tuple[str, str]]:
        """Get gRPC metadata (headers) including auth token."""
        return [("x-token", self._token)]

    def _get_channel_options(self) -> List[Tuple[str, Any]]:
        """Get gRPC channel options for keepalive."""
        return [
            ("grpc.keepalive_time_ms", self.KEEPALIVE_TIME_MS),
            ("grpc.keepalive_timeout_ms", self.KEEPALIVE_TIMEOUT_MS),
            ("grpc.keepalive_permit_without_calls", True),
            ("grpc.http2.max_pings_without_data", 0),
            ("grpc.http2.min_time_between_pings_ms", 10000),
            ("grpc.http2.min_ping_interval_without_data_ms", 5000),
            ("grpc.max_receive_message_length", 100 * 1024 * 1024),  # 100MB
            ("grpc.max_send_message_length", 100 * 1024 * 1024),     # 100MB
        ]

    def _create_channel(self) -> grpc.Channel:
        """Create a gRPC channel with keepalive options."""
        target = self._get_target()
        options = self._get_channel_options()

        if self._secure:
            credentials = grpc.ssl_channel_credentials()
            return grpc.secure_channel(target, credentials, options=options)
        return grpc.insecure_channel(target, options=options)

    def _create_stubs(self) -> None:
        """Create gRPC stubs for all services."""
        if self._channel:
            self._streaming_stub = StreamingStub(self._channel)
            self._block_stub = BlockStreamingStub(self._channel)
            self._orderbook_stub = OrderBookStreamingStub(self._channel)

    def _add_subscription(
        self,
        stream_type: str,
        callback: Callable[[Dict[str, Any]], None],
        coins: Optional[List[str]] = None,
        users: Optional[List[str]] = None,
        coin: Optional[str] = None,
        n_sig_figs: Optional[int] = None,
        n_levels: int = 20,
        raw: bool = False,
        start_block: Optional[int] = None,
        mantissa: Optional[int] = None,
        skip_initial_snapshot: bool = False,
        bytes_stream_type: Optional[str] = None,
    ) -> None:
        """Add a subscription to be started when run() is called."""
        with self._lock:
            sub = {
                "stream_type": stream_type,
                "callback": callback,
            }
            if coins:
                sub["coins"] = coins
            if users:
                sub["users"] = users
            if coin:
                sub["coin"] = coin
            if n_sig_figs is not None:
                sub["n_sig_figs"] = n_sig_figs
            if start_block is not None:
                sub["start_block"] = start_block
            if mantissa is not None:
                sub["mantissa"] = mantissa
            if bytes_stream_type is not None:
                sub["bytes_stream_type"] = bytes_stream_type
            sub["skip_initial_snapshot"] = skip_initial_snapshot
            sub["n_levels"] = n_levels
            sub["raw"] = raw

            self._subscriptions.append(sub)

    def trades(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to trade stream.

        Fields: coin, px, sz, side (B/A), time, dir, closedPnl, hash, oid, tid

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each trade
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.TRADES.value, callback, coins=coins, start_block=start_block)
        return self

    def raw_trades(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw trade blocks.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each raw trade block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.TRADES.value, callback, coins=coins, raw=True, start_block=start_block
        )
        return self

    def orders(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        users: Optional[List[str]] = None,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to order stream.

        Status: open, filled, triggered, canceled, etc.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each order update
            users: Optional list of user addresses to filter
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.ORDERS.value, callback, coins=coins, users=users, start_block=start_block
        )
        return self

    def raw_orders(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        users: Optional[List[str]] = None,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw order blocks.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each raw order block
            users: Optional list of user addresses to filter
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.ORDERS.value, callback, coins=coins, users=users, raw=True, start_block=start_block
        )
        return self

    def book_updates(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to order book updates.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each book update
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.BOOK_UPDATES.value, callback, coins=coins, start_block=start_block
        )
        return self

    def raw_book_updates(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw order book update blocks.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each raw book update block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.BOOK_UPDATES.value, callback, coins=coins, raw=True, start_block=start_block
        )
        return self

    def twap(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to TWAP execution stream.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each TWAP update
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.TWAP.value, callback, coins=coins, start_block=start_block)
        return self

    def raw_twap(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw TWAP execution blocks.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each raw TWAP block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.TWAP.value, callback, coins=coins, raw=True, start_block=start_block
        )
        return self

    def events(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to system events (funding, liquidations, governance).

        Args:
            callback: Function called for each event
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.EVENTS.value, callback, start_block=start_block)
        return self

    def raw_events(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw system event blocks.

        Args:
            callback: Function called for each raw event block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.EVENTS.value, callback, raw=True, start_block=start_block)
        return self

    def blocks(self, callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to block data.

        Args:
            callback: Function called for each block
        """
        self._add_subscription(GRPCStreamType.BLOCKS.value, callback)
        return self

    def writer_actions(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to writer actions (HyperCore <-> HyperEVM asset transfers).

        Args:
            callback: Function called for each writer action
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.WRITER_ACTIONS.value, callback, start_block=start_block)
        return self

    def raw_writer_actions(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw writer action blocks.

        Args:
            callback: Function called for each raw writer action block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.WRITER_ACTIONS.value, callback, raw=True, start_block=start_block
        )
        return self

    def mempool_txs(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to pre-consensus mempool transactions.

        Args:
            callback: Function called for each mempool transaction event
            coins: Optional list of coin symbols to filter ["BTC", "ETH"] (None = all)
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.MEMPOOL_TXS.value, callback, coins=coins, start_block=start_block
        )
        return self

    def raw_mempool_txs(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw pre-consensus mempool transaction blocks.

        Args:
            callback: Function called for each raw mempool block
            coins: Optional list of coin symbols to filter ["BTC", "ETH"] (None = all)
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.MEMPOOL_TXS.value, callback, coins=coins, raw=True, start_block=start_block
        )
        return self

    def order_priority(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to derived order/write priority actions (from mempool and confirmed replica data).

        Events carry server-enriched fields: coin, market_type, sz_decimals.

        Args:
            callback: Function called for each order priority event
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.ORDER_PRIORITY.value, callback, start_block=start_block)
        return self

    def raw_order_priority(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw order priority blocks.

        Args:
            callback: Function called for each raw order priority block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.ORDER_PRIORITY.value, callback, raw=True, start_block=start_block
        )
        return self

    def gossip_priority(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to derived gossip/read priority bid actions (does not measure delivery latency).

        Events carry server-enriched fields: coin, market_type, sz_decimals.

        Args:
            callback: Function called for each gossip priority event
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(GRPCStreamType.GOSSIP_PRIORITY.value, callback, start_block=start_block)
        return self

    def raw_gossip_priority(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to raw gossip priority blocks.

        Args:
            callback: Function called for each raw gossip priority block
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            GRPCStreamType.GOSSIP_PRIORITY.value, callback, raw=True, start_block=start_block
        )
        return self

    def stream_bytes(
        self,
        stream_type: str,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
        users: Optional[List[str]] = None,
        start_block: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to the low-level bytes fast path (StreamDataBytes RPC).

        The payload is NOT parsed: the callback receives
        {"block_number": int, "timestamp": int, "data": bytes} for each update.
        Fast-path clients should prefer this over the JSON string streams.

        Args:
            stream_type: Stream type name (e.g. "TRADES", GRPCStreamType.ORDERS, ...)
            callback: Function called for each raw bytes update
            coins: Optional list of coin symbols to filter
            users: Optional list of user addresses to filter
            start_block: Optional block number to start streaming from
        """
        self._add_subscription(
            "STREAM_BYTES",
            callback,
            coins=coins,
            users=users,
            start_block=start_block,
            bytes_stream_type=str(GRPCStreamType(stream_type).value),
        )
        return self

    def l2_book(
        self,
        coin: str,
        callback: Callable[[Dict[str, Any]], None],
        *,
        n_sig_figs: Optional[int] = None,
        n_levels: int = 20,
    ) -> "GRPCStream":
        """
        Subscribe to Level 2 order book updates (aggregated price levels).

        Args:
            coin: Coin symbol ("BTC")
            callback: Function called for each book update
            n_sig_figs: Optional number of significant figures for price aggregation
            n_levels: Number of price levels to return (default: 20)
        """
        self._add_subscription("L2_BOOK", callback, coin=coin, n_sig_figs=n_sig_figs, n_levels=n_levels)
        return self

    def l4_book(self, coin: str, callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to Level 4 order book updates (individual orders).

        Note: the server may send an unsolicited full snapshot at any time after
        subscribe (e.g. after ALO queue-priority anchored insertions). Clients MUST
        discard local book state and replace it with any snapshot received mid-stream.

        Args:
            coin: Coin symbol ("BTC")
            callback: Function called for each book update
        """
        self._add_subscription("L4_BOOK", callback, coin=coin)
        return self

    def bbo_book(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
    ) -> "GRPCStream":
        """
        Subscribe to top-of-book (best bid/offer) changes.

        Emits only when the best bid or ask changes for a coin.
        Fields: coin, time, block_number, bid ([px, sz, n] or None), ask ([px, sz, n] or None)

        Args:
            callback: Function called for each BBO update
            coins: Optional list of coin symbols (None = all coins)
        """
        self._add_subscription("BBO_BOOK", callback, coins=coins)
        return self

    def l2_book_diff(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
        n_levels: int = 20,
        n_sig_figs: Optional[int] = None,
        mantissa: Optional[int] = None,
        skip_initial_snapshot: bool = False,
    ) -> "GRPCStream":
        """
        Subscribe to incremental L2 price-level changes.

        Each update carries only changed levels per coin; a level with sz == "0"
        means the level was removed. Unless skip_initial_snapshot is True, the
        first update per coin contains the current levels (snapshot=True).

        Args:
            callback: Function called for each diff batch
            coins: Optional list of coin symbols (None = all coins)
            n_levels: Max tracked levels per side (default: 20, max 100)
            n_sig_figs: Optional significant figures for price bucketing (2-5)
            mantissa: Optional mantissa for bucketing (1, 2, or 5)
            skip_initial_snapshot: Skip the initial per-coin snapshot (default: False)
        """
        self._add_subscription(
            "L2_BOOK_DIFF",
            callback,
            coins=coins,
            n_levels=n_levels,
            n_sig_figs=n_sig_figs,
            mantissa=mantissa,
            skip_initial_snapshot=skip_initial_snapshot,
        )
        return self

    def l4_book_updates(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
    ) -> "GRPCStream":
        """
        Subscribe to typed L4 order book updates (new/update/remove diffs per block).

        Note: the server may send an unsolicited full snapshot batch at any time
        (snapshot=True); clients MUST discard local book state and rebuild from it.

        Args:
            callback: Function called for each update batch
            coins: Optional list of coin symbols (None = all coins)
        """
        self._add_subscription("L4_BOOK_UPDATES", callback, coins=coins)
        return self

    def tpsl_updates(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
    ) -> "GRPCStream":
        """
        Subscribe to trigger/TP-SL order add/remove updates.

        Args:
            callback: Function called for each update batch
            coins: Optional list of coin symbols (None = all perp coins)
        """
        self._add_subscription("TPSL_UPDATES", callback, coins=coins)
        return self

    def l2_book_packed(
        self,
        coin: str,
        callback: Callable[[Dict[str, Any]], None],
        *,
        n_sig_figs: Optional[int] = None,
        n_levels: int = 20,
        mantissa: Optional[int] = None,
    ) -> "GRPCStream":
        """
        Subscribe to fast-path L2 order book updates with fixed-point integers.

        Prices/sizes are uint64 fixed-point integers scaled by 1e8.

        Args:
            coin: Coin symbol ("BTC")
            callback: Function called for each book update
            n_sig_figs: Optional number of significant figures for price aggregation
            n_levels: Number of price levels to return (default: 20)
            mantissa: Optional mantissa for bucketing (1, 2, or 5)
        """
        self._add_subscription(
            "L2_BOOK_PACKED", callback, coin=coin, n_sig_figs=n_sig_figs, n_levels=n_levels, mantissa=mantissa
        )
        return self

    def bbo_book_packed(
        self,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
    ) -> "GRPCStream":
        """
        Subscribe to fast-path top-of-book changes with fixed-point integers.

        Prices/sizes are uint64 fixed-point integers scaled by 1e8.

        Args:
            callback: Function called for each BBO update
            coins: Optional list of coin symbols (None = all coins)
        """
        self._add_subscription("BBO_BOOK_PACKED", callback, coins=coins)
        return self

    def l4_book_bytes(self, coin: str, callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to fast-path Level 4 order book updates (diff payload as JSON bytes).

        Diff updates carry the raw JSON payload as bytes (not parsed). Snapshots are
        delivered as structured dicts like l4_book. The server may send an unsolicited
        full snapshot at any time; clients MUST discard local book state and replace
        it with any snapshot received mid-stream.

        Args:
            coin: Coin symbol ("BTC")
            callback: Function called for each book update
        """
        self._add_subscription("L4_BOOK_BYTES", callback, coin=coin)
        return self

    def _build_subscribe_request(
        self,
        sub: Dict[str, Any],
        stream_type: Optional[str] = None,
        *,
        reconnect: bool = False,
    ) -> SubscribeRequest:
        """Build the SubscribeRequest for a StreamData/StreamDataBytes subscription.

        On reconnect, a user-set start_block is advanced past the highest block
        already delivered so already-processed blocks are not replayed. An unset
        start_block stays unset (live-tip semantics are preserved).
        """
        request = SubscribeRequest()
        request.subscribe.stream_type = _STREAM_TYPE_MAP.get(
            stream_type or sub.get("stream_type"), 0
        )

        # Resume from a specific block (0 = live tip)
        start_block = sub.get("start_block")
        if start_block:
            if reconnect:
                # Don't replay blocks already delivered on a previous connection.
                start_block = max(start_block, sub.get("_last_seen_block", 0) + 1)
            request.subscribe.start_block = start_block

        # Add filters
        coins = sub.get("coins")
        if coins:
            filter_values = FilterValues()
            filter_values.values.extend(coins)
            request.subscribe.filters["coin"].CopyFrom(filter_values)

        users = sub.get("users")
        if users:
            filter_values = FilterValues()
            filter_values.values.extend(users)
            request.subscribe.filters["user"].CopyFrom(filter_values)

        return request

    def _stream_data(self, sub: Dict[str, Any]) -> None:
        """Stream data using bidirectional StreamData RPC."""
        callback = sub["callback"]
        first_connect = True

        while self._running and not self._stop_event.is_set():
            try:
                if not self._streaming_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()
                initial_request = self._build_subscribe_request(sub, reconnect=not first_connect)
                first_connect = False

                # Build request generator
                def request_generator() -> Iterator[SubscribeRequest]:
                    # Send initial subscription request
                    yield initial_request

                    # Keep sending pings to maintain connection
                    while self._running and not self._stop_event.is_set():
                        time.sleep(30)
                        ping_request = SubscribeRequest()
                        ping_request.ping.timestamp = int(time.time() * 1000)
                        yield ping_request

                # Create bidirectional stream
                stream = self._streaming_stub.StreamData(request_generator(), metadata=metadata)

                # Handle responses
                for response in stream:
                    if not self._running or self._stop_event.is_set():
                        break

                    if response.HasField('data'):
                        block_number = response.data.block_number
                        try:
                            data = json.loads(response.data.data)
                            # Advance the resume cursor only after the payload
                            # parsed, so a corrupt block is re-requested on
                            # reconnect instead of skipped permanently.
                            if block_number > sub.get("_last_seen_block", 0):
                                sub["_last_seen_block"] = block_number
                            timestamp = response.data.timestamp

                            if sub.get("raw"):
                                data['_block_number'] = block_number
                                data['_timestamp'] = timestamp
                                self._safe_callback(callback, data)
                                continue

                            # Data can contain events as objects or legacy [user, event] tuples.
                            events = data.get("events")
                            if isinstance(events, list) and events:
                                emitted_events = False
                                for index, event in enumerate(events):
                                    user = None
                                    event_data = None

                                    if isinstance(event, list) and len(event) >= 2:
                                        user, candidate = event[0], event[1]
                                        if isinstance(candidate, dict):
                                            event_data = candidate
                                    elif isinstance(event, dict):
                                        event_data = event

                                    if event_data is not None:
                                        event_data['_block_number'] = block_number
                                        event_data['_timestamp'] = timestamp
                                        event_data['_event_index'] = index
                                        if user is not None:
                                            event_data['_user'] = user
                                        self._safe_callback(callback, event_data)
                                        emitted_events = True

                                if not emitted_events:
                                    data['_block_number'] = block_number
                                    data['_timestamp'] = timestamp
                                    self._safe_callback(callback, data)
                            else:
                                # Fallback: return raw data if no events structure
                                data['_block_number'] = block_number
                                data['_timestamp'] = timestamp
                                self._safe_callback(callback, data)
                        except json.JSONDecodeError as e:
                            logger.warning(f"Failed to parse data: {e}")
                    elif response.HasField('pong'):
                        logger.debug(f"Pong received: {response.pong.timestamp}")

            except grpc.RpcError as e:
                if not self._running:
                    break

                error = HyperliquidError(
                    f"gRPC error: {e.code()} - {e.details()}",
                    code="GRPC_ERROR",
                    raw={"code": str(e.code()), "details": e.details()},
                )

                if self._on_error:
                    try:
                        self._on_error(error)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

            except Exception as e:
                if not self._running:
                    break

                if self._on_error:
                    try:
                        self._on_error(e)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

    def _stream_blocks(self, sub: Dict[str, Any]) -> None:
        """Stream raw block data using BlockStreaming RPC."""
        callback = sub["callback"]

        while self._running and not self._stop_event.is_set():
            try:
                if not self._block_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()
                request = Timestamp(timestamp=int(time.time() * 1000))

                # Create stream
                stream = self._block_stub.StreamBlocks(request, metadata=metadata)

                for block in stream:
                    if not self._running or self._stop_event.is_set():
                        break

                    try:
                        data = json.loads(block.data_json)
                        self._safe_callback(callback, data)
                    except json.JSONDecodeError as e:
                        logger.warning(f"Failed to parse block: {e}")

            except grpc.RpcError as e:
                if not self._running:
                    break

                error = HyperliquidError(
                    f"gRPC error: {e.code()} - {e.details()}",
                    code="GRPC_ERROR",
                    raw={"code": str(e.code()), "details": e.details()},
                )

                if self._on_error:
                    try:
                        self._on_error(error)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

            except Exception as e:
                if not self._running:
                    break

                if self._on_error:
                    try:
                        self._on_error(e)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

    def _stream_l2_book(self, sub: Dict[str, Any]) -> None:
        """Stream L2 order book using OrderBookStreaming RPC."""
        callback = sub["callback"]
        coin = sub.get("coin")
        n_levels = sub.get("n_levels", 20)
        n_sig_figs = sub.get("n_sig_figs")

        while self._running and not self._stop_event.is_set():
            try:
                if not self._orderbook_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()

                # Build request
                request = L2BookRequest(coin=coin, n_levels=n_levels)
                if n_sig_figs is not None:
                    request.n_sig_figs = n_sig_figs

                # Create stream
                stream = self._orderbook_stub.StreamL2Book(request, metadata=metadata)

                for update in stream:
                    if not self._running or self._stop_event.is_set():
                        break

                    # Convert protobuf to dict
                    data = {
                        "coin": update.coin,
                        "time": update.time,
                        "block_number": update.block_number,
                        "bids": [[level.px, level.sz, level.n] for level in update.bids],
                        "asks": [[level.px, level.sz, level.n] for level in update.asks],
                    }
                    self._safe_callback(callback, data)

            except grpc.RpcError as e:
                if not self._running:
                    break

                error = HyperliquidError(
                    f"gRPC error: {e.code()} - {e.details()}",
                    code="GRPC_ERROR",
                    raw={"code": str(e.code()), "details": e.details()},
                )

                if self._on_error:
                    try:
                        self._on_error(error)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

            except Exception as e:
                if not self._running:
                    break

                if self._on_error:
                    try:
                        self._on_error(e)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

    def _stream_l4_book(self, sub: Dict[str, Any]) -> None:
        """Stream L4 order book using OrderBookStreaming RPC."""
        callback = sub["callback"]
        coin = sub.get("coin")

        while self._running and not self._stop_event.is_set():
            try:
                if not self._orderbook_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()
                request = L4BookRequest(coin=coin)

                # Create stream
                stream = self._orderbook_stub.StreamL4Book(request, metadata=metadata)

                for update in stream:
                    if not self._running or self._stop_event.is_set():
                        break

                    # Convert protobuf to dict based on update type
                    if update.HasField('snapshot'):
                        snapshot = update.snapshot
                        data = {
                            "type": "snapshot",
                            "coin": snapshot.coin,
                            "time": snapshot.time,
                            "height": snapshot.height,
                            "bids": [self._l4_order_to_dict(o) for o in snapshot.bids],
                            "asks": [self._l4_order_to_dict(o) for o in snapshot.asks],
                        }
                    elif update.HasField('diff'):
                        diff = update.diff
                        data = {
                            "type": "diff",
                            "time": diff.time,
                            "height": diff.height,
                            "data": json.loads(diff.data) if diff.data else {},
                        }
                    else:
                        continue

                    self._safe_callback(callback, data)

            except grpc.RpcError as e:
                if not self._running:
                    break

                error = HyperliquidError(
                    f"gRPC error: {e.code()} - {e.details()}",
                    code="GRPC_ERROR",
                    raw={"code": str(e.code()), "details": e.details()},
                )

                if self._on_error:
                    try:
                        self._on_error(error)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

            except Exception as e:
                if not self._running:
                    break

                if self._on_error:
                    try:
                        self._on_error(e)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

    def _l4_order_to_dict(self, order) -> Dict[str, Any]:
        """Convert L4Order protobuf to dict."""
        return {
            "user": order.user,
            "coin": order.coin,
            "side": order.side,
            "limit_px": order.limit_px,
            "sz": order.sz,
            "oid": order.oid,
            "timestamp": order.timestamp,
            "trigger_condition": order.trigger_condition,
            "is_trigger": order.is_trigger,
            "trigger_px": order.trigger_px,
            "is_position_tpsl": order.is_position_tpsl,
            "reduce_only": order.reduce_only,
            "order_type": order.order_type,
            "tif": order.tif if order.HasField('tif') else None,
            "cloid": order.cloid if order.HasField('cloid') else None,
        }

    def _stream_data_bytes(self, sub: Dict[str, Any]) -> None:
        """Stream raw payload bytes using bidirectional StreamDataBytes RPC."""
        callback = sub["callback"]
        first_connect = True

        while self._running and not self._stop_event.is_set():
            try:
                if not self._streaming_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()
                initial_request = self._build_subscribe_request(
                    sub, sub.get("bytes_stream_type"), reconnect=not first_connect
                )
                first_connect = False

                # Build request generator
                def request_generator() -> Iterator[SubscribeRequest]:
                    # Send initial subscription request
                    yield initial_request

                    # Keep sending pings to maintain connection
                    while self._running and not self._stop_event.is_set():
                        time.sleep(30)
                        ping_request = SubscribeRequest()
                        ping_request.ping.timestamp = int(time.time() * 1000)
                        yield ping_request

                # Create bidirectional stream
                stream = self._streaming_stub.StreamDataBytes(request_generator(), metadata=metadata)

                # Handle responses
                for response in stream:
                    if not self._running or self._stop_event.is_set():
                        break

                    if response.HasField('data'):
                        block_number = response.data.block_number
                        # Fast path: hand the payload bytes through unparsed
                        self._safe_callback(callback, {
                            "block_number": block_number,
                            "timestamp": response.data.timestamp,
                            "data": response.data.data,
                        })
                        # Cursor advances only after delivery so reconnects
                        # never skip an undelivered block.
                        if block_number > sub.get("_last_seen_block", 0):
                            sub["_last_seen_block"] = block_number
                    elif response.HasField('pong'):
                        logger.debug(f"Pong received: {response.pong.timestamp}")

            except grpc.RpcError as e:
                if not self._running:
                    break

                error = HyperliquidError(
                    f"gRPC error: {e.code()} - {e.details()}",
                    code="GRPC_ERROR",
                    raw={"code": str(e.code()), "details": e.details()},
                )

                if self._on_error:
                    try:
                        self._on_error(error)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

            except Exception as e:
                if not self._running:
                    break

                if self._on_error:
                    try:
                        self._on_error(e)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

    def _build_orderbook_request(self, sub: Dict[str, Any], *, reconnect: bool = False) -> Any:
        """Build the request message for an OrderBookStreaming subscription.

        On reconnect, skip_initial_snapshot is forced to False so the server
        resends the snapshot and the consumer can resync its local book.
        """
        stream_type = sub.get("stream_type")
        coins = sub.get("coins") or []

        if stream_type in ("BBO_BOOK", "BBO_BOOK_PACKED"):
            return BboBookRequest(coins=coins)

        if stream_type == "L2_BOOK_DIFF":
            # skip_initial_snapshot only applies to the first connection.
            skip_initial_snapshot = False if reconnect else sub.get("skip_initial_snapshot", False)
            request = L2BookDiffRequest(
                coins=coins,
                n_levels=sub.get("n_levels", 20),
                skip_initial_snapshot=skip_initial_snapshot,
            )
            if sub.get("n_sig_figs") is not None:
                request.n_sig_figs = sub["n_sig_figs"]
            if sub.get("mantissa") is not None:
                request.mantissa = sub["mantissa"]
            return request

        if stream_type == "L4_BOOK_UPDATES":
            return L4BookUpdatesRequest(coins=coins)

        if stream_type == "TPSL_UPDATES":
            return TpslUpdatesRequest(coins=coins)

        if stream_type == "L2_BOOK_PACKED":
            request = L2BookRequest(coin=sub.get("coin"), n_levels=sub.get("n_levels", 20))
            if sub.get("n_sig_figs") is not None:
                request.n_sig_figs = sub["n_sig_figs"]
            if sub.get("mantissa") is not None:
                request.mantissa = sub["mantissa"]
            return request

        if stream_type == "L4_BOOK_BYTES":
            return L4BookRequest(coin=sub.get("coin"))

        raise ValueError(f"Unknown orderbook stream type: {stream_type}")

    def _bbo_update_to_dict(self, update) -> Dict[str, Any]:
        """Convert BboBookUpdate/BboBookPackedUpdate protobuf to dict."""
        return {
            "coin": update.coin,
            "time": update.time,
            "block_number": update.block_number,
            "bid": [update.bid.px, update.bid.sz, update.bid.n] if update.HasField('bid') else None,
            "ask": [update.ask.px, update.ask.sz, update.ask.n] if update.HasField('ask') else None,
        }

    def _orderbook_update_to_dict(self, stream_type: str, update) -> Optional[Dict[str, Any]]:
        """Convert an OrderBookStreaming update protobuf to dict."""
        if stream_type in ("BBO_BOOK", "BBO_BOOK_PACKED"):
            return self._bbo_update_to_dict(update)

        if stream_type == "L2_BOOK_DIFF":
            return {
                "time": update.time,
                "height": update.height,
                "snapshot": update.snapshot,
                "diffs": [
                    {
                        "coin": diff.coin,
                        "seq": diff.seq,
                        "prev_seq": diff.prev_seq,
                        "snapshot": diff.snapshot,
                        "bids": [[level.px, level.sz, level.n] for level in diff.bids],
                        "asks": [[level.px, level.sz, level.n] for level in diff.asks],
                    }
                    for diff in update.diffs
                ],
            }

        if stream_type == "L4_BOOK_UPDATES":
            return {
                "time": update.time,
                "height": update.height,
                "snapshot": update.snapshot,
                "diffs": [
                    {
                        "diff_type": L4OrderDiffType.Name(diff.diff_type),
                        "coin": diff.coin,
                        "oid": diff.oid,
                        "user": diff.user,
                        "side": diff.side,
                        "px": diff.px,
                        "sz": diff.sz,
                    }
                    for diff in update.diffs
                ],
            }

        if stream_type == "TPSL_UPDATES":
            return {
                "time": update.time,
                "height": update.height,
                "snapshot": update.snapshot,
                "diffs": [
                    {
                        "diff_type": TpslDiffType.Name(diff.diff_type),
                        "oid": diff.oid,
                        "coin": diff.coin,
                        "user": diff.user,
                        "side": diff.side,
                        "trigger_px": diff.trigger_px,
                        "limit_px": diff.limit_px,
                        "sz": diff.sz,
                        "trigger_condition": diff.trigger_condition,
                        "order_type": diff.order_type,
                        "is_position_tpsl": diff.is_position_tpsl,
                        "reduce_only": diff.reduce_only,
                        "timestamp": diff.timestamp,
                        "reason": diff.reason,
                    }
                    for diff in update.diffs
                ],
            }

        if stream_type == "L2_BOOK_PACKED":
            return {
                "coin": update.coin,
                "time": update.time,
                "block_number": update.block_number,
                "bids": [[level.px, level.sz, level.n] for level in update.bids],
                "asks": [[level.px, level.sz, level.n] for level in update.asks],
            }

        if stream_type == "L4_BOOK_BYTES":
            if update.HasField('snapshot'):
                snapshot = update.snapshot
                return {
                    "type": "snapshot",
                    "coin": snapshot.coin,
                    "time": snapshot.time,
                    "height": snapshot.height,
                    "bids": [self._l4_order_to_dict(o) for o in snapshot.bids],
                    "asks": [self._l4_order_to_dict(o) for o in snapshot.asks],
                }
            if update.HasField('diff'):
                diff = update.diff
                return {
                    "type": "diff",
                    "time": diff.time,
                    "height": diff.height,
                    "data": diff.data,  # Raw JSON bytes (not parsed)
                }
            return None

        return None

    def _stream_orderbook(self, sub: Dict[str, Any]) -> None:
        """Stream order book data using the newer OrderBookStreaming RPCs."""
        callback = sub["callback"]
        stream_type = sub.get("stream_type")
        first_connect = True

        while self._running and not self._stop_event.is_set():
            try:
                if not self._orderbook_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()
                request = self._build_orderbook_request(sub, reconnect=not first_connect)
                first_connect = False
                rpc = getattr(self._orderbook_stub, _ORDERBOOK_RPC_MAP[stream_type])

                # Create stream
                stream = rpc(request, metadata=metadata)

                for update in stream:
                    if not self._running or self._stop_event.is_set():
                        break

                    data = self._orderbook_update_to_dict(stream_type, update)
                    if data is not None:
                        self._safe_callback(callback, data)

            except grpc.RpcError as e:
                if not self._running:
                    break

                error = HyperliquidError(
                    f"gRPC error: {e.code()} - {e.details()}",
                    code="GRPC_ERROR",
                    raw={"code": str(e.code()), "details": e.details()},
                )

                if self._on_error:
                    try:
                        self._on_error(error)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

            except Exception as e:
                if not self._running:
                    break

                if self._on_error:
                    try:
                        self._on_error(e)
                    except Exception:
                        pass

                if self._reconnect_enabled and self._running:
                    self._handle_reconnect()
                else:
                    break

    def _handle_reconnect(self) -> None:
        """Handle reconnection with exponential backoff."""
        if not self._running:
            return

        if self._max_reconnect_attempts and self._reconnect_attempt >= self._max_reconnect_attempts:
            logger.error(f"Max reconnection attempts ({self._max_reconnect_attempts}) reached")
            self._running = False
            if self._on_close:
                try:
                    self._on_close()
                except Exception:
                    pass
            return

        self._reconnect_attempt += 1
        self._set_state(ConnectionState.RECONNECTING)

        logger.info(f"Reconnecting in {self._reconnect_delay:.1f}s (attempt {self._reconnect_attempt})")

        if self._on_reconnect:
            try:
                self._on_reconnect(self._reconnect_attempt)
            except Exception as e:
                logger.warning(f"Reconnect callback failed: {e}")

        # Wait with backoff
        self._stop_event.wait(self._reconnect_delay)
        self._reconnect_delay = min(
            self._reconnect_delay * self.RECONNECT_BACKOFF_FACTOR,
            self.MAX_RECONNECT_DELAY,
        )

        # Recreate channel and stubs
        if self._running:
            try:
                if self._channel:
                    self._channel.close()
            except Exception:
                pass
            self._channel = self._create_channel()
            self._create_stubs()
            self._set_state(ConnectionState.CONNECTED)
            self._reconnect_attempt = 0
            self._reconnect_delay = self.INITIAL_RECONNECT_DELAY

            if self._on_connect:
                try:
                    self._on_connect()
                except Exception as e:
                    logger.warning(f"Connect callback failed: {e}")

    def _start_streams(self) -> None:
        """Start all subscription streams."""
        with self._lock:
            for sub in self._subscriptions:
                stream_type = sub.get("stream_type")

                if stream_type == "L2_BOOK":
                    thread = threading.Thread(
                        target=self._stream_l2_book,
                        args=(sub,),
                        daemon=True,
                    )
                elif stream_type == "L4_BOOK":
                    thread = threading.Thread(
                        target=self._stream_l4_book,
                        args=(sub,),
                        daemon=True,
                    )
                elif stream_type in _ORDERBOOK_RPC_MAP:
                    thread = threading.Thread(
                        target=self._stream_orderbook,
                        args=(sub,),
                        daemon=True,
                    )
                elif stream_type == "STREAM_BYTES":
                    thread = threading.Thread(
                        target=self._stream_data_bytes,
                        args=(sub,),
                        daemon=True,
                    )
                elif stream_type == "BLOCKS":
                    thread = threading.Thread(
                        target=self._stream_blocks,
                        args=(sub,),
                        daemon=True,
                    )
                else:
                    thread = threading.Thread(
                        target=self._stream_data,
                        args=(sub,),
                        daemon=True,
                    )
                thread.start()
                self._threads.append(thread)

    def ping(self) -> bool:
        """
        Test connectivity with a ping request.

        Returns:
            True if ping successful, False otherwise
        """
        if not self._streaming_stub:
            return False

        try:
            request = PingRequest(count=1)
            response = self._streaming_stub.Ping(request, metadata=self._get_metadata())
            return response.count == 1
        except grpc.RpcError:
            return False

    def run(self) -> None:
        """Run the stream (blocking)."""
        self._running = True
        self._stop_event.clear()
        self._set_state(ConnectionState.CONNECTING)
        self._channel = self._create_channel()
        self._create_stubs()
        self._set_state(ConnectionState.CONNECTED)

        if self._on_connect:
            try:
                self._on_connect()
            except Exception as e:
                logger.warning(f"Connect callback failed: {e}")

        self._start_streams()

        # Wait for all threads or stop
        try:
            while self._running and any(t.is_alive() for t in self._threads):
                time.sleep(0.5)
        except KeyboardInterrupt:
            self.stop()

    def start(self) -> None:
        """Start the stream in background."""
        self._running = True
        self._stop_event.clear()
        self._set_state(ConnectionState.CONNECTING)
        self._channel = self._create_channel()
        self._create_stubs()
        self._set_state(ConnectionState.CONNECTED)

        if self._on_connect:
            try:
                self._on_connect()
            except Exception as e:
                logger.warning(f"Connect callback failed: {e}")

        self._start_streams()

    def run_in_background(self) -> None:
        """Alias for start() - run the stream in background."""
        self.start()

    def stop(self) -> None:
        """Stop the stream gracefully."""
        self._running = False
        self._stop_event.set()

        # Close channel
        if self._channel:
            try:
                self._channel.close()
            except Exception:
                pass
            self._channel = None

        # Clear stubs
        self._streaming_stub = None
        self._block_stub = None
        self._orderbook_stub = None

        # Wait for threads to finish
        for thread in self._threads:
            thread.join(timeout=2)

        self._threads.clear()
        self._set_state(ConnectionState.DISCONNECTED)

        if self._on_close:
            try:
                self._on_close()
            except Exception:
                pass

    @property
    def connected(self) -> bool:
        """Check if stream is connected."""
        return self._state == ConnectionState.CONNECTED

    @property
    def state(self) -> ConnectionState:
        """Get current connection state."""
        return self._state

    @property
    def reconnect_attempts(self) -> int:
        """Get number of reconnection attempts since last successful connection."""
        return self._reconnect_attempt

    def __enter__(self) -> "GRPCStream":
        return self

    def __exit__(self, *args) -> None:
        self.stop()

    def __repr__(self) -> str:
        return f"<GRPCStream {self._state.value} {len(self._subscriptions)} subs>"
