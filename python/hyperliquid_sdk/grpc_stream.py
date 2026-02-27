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
        L4BookRequest,
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
    - l2_book: Level 2 order book (aggregated price levels)
    - l4_book: Level 4 order book (individual orders)

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
            sub["n_levels"] = n_levels

            self._subscriptions.append(sub)

    def trades(self, coins: List[str], callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to trade stream.

        Fields: coin, px, sz, side (B/A), time, dir, closedPnl, hash, oid, tid

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each trade
        """
        self._add_subscription(GRPCStreamType.TRADES.value, callback, coins=coins)
        return self

    def orders(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        users: Optional[List[str]] = None,
    ) -> "GRPCStream":
        """
        Subscribe to order stream.

        Status: open, filled, triggered, canceled, etc.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each order update
            users: Optional list of user addresses to filter
        """
        self._add_subscription(GRPCStreamType.ORDERS.value, callback, coins=coins, users=users)
        return self

    def book_updates(self, coins: List[str], callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to order book updates.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each book update
        """
        self._add_subscription(GRPCStreamType.BOOK_UPDATES.value, callback, coins=coins)
        return self

    def twap(self, coins: List[str], callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to TWAP execution stream.

        Args:
            coins: List of coin symbols ["BTC", "ETH"]
            callback: Function called for each TWAP update
        """
        self._add_subscription(GRPCStreamType.TWAP.value, callback, coins=coins)
        return self

    def events(self, callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to system events (funding, liquidations, governance).

        Args:
            callback: Function called for each event
        """
        self._add_subscription(GRPCStreamType.EVENTS.value, callback)
        return self

    def blocks(self, callback: Callable[[Dict[str, Any]], None]) -> "GRPCStream":
        """
        Subscribe to block data.

        Args:
            callback: Function called for each block
        """
        self._add_subscription(GRPCStreamType.BLOCKS.value, callback)
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

        Args:
            coin: Coin symbol ("BTC")
            callback: Function called for each book update
        """
        self._add_subscription("L4_BOOK", callback, coin=coin)
        return self

    def _stream_data(self, sub: Dict[str, Any]) -> None:
        """Stream data using bidirectional StreamData RPC."""
        stream_type = sub.get("stream_type")
        callback = sub["callback"]
        coins = sub.get("coins")
        users = sub.get("users")

        while self._running and not self._stop_event.is_set():
            try:
                if not self._streaming_stub:
                    time.sleep(1)
                    continue

                metadata = self._get_metadata()

                # Build request generator
                def request_generator() -> Iterator[SubscribeRequest]:
                    # Send initial subscription request
                    request = SubscribeRequest()
                    request.subscribe.stream_type = _STREAM_TYPE_MAP.get(stream_type, 0)

                    # Add filters
                    if coins:
                        filter_values = FilterValues()
                        filter_values.values.extend(coins)
                        request.subscribe.filters["coin"].CopyFrom(filter_values)

                    if users:
                        filter_values = FilterValues()
                        filter_values.values.extend(users)
                        request.subscribe.filters["user"].CopyFrom(filter_values)

                    yield request

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
                        try:
                            data = json.loads(response.data.data)
                            block_number = response.data.block_number
                            timestamp = response.data.timestamp

                            # Data structure: {"block_number":..., "events":[[user, {...}], ...]}
                            # Extract events and call callback for each
                            events = data.get("events", [])
                            if events:
                                for event in events:
                                    if isinstance(event, list) and len(event) >= 2:
                                        user, event_data = event[0], event[1]
                                        if isinstance(event_data, dict):
                                            # Add metadata
                                            event_data['_block_number'] = block_number
                                            event_data['_timestamp'] = timestamp
                                            event_data['_user'] = user
                                            callback(event_data)
                            else:
                                # Fallback: return raw data if no events structure
                                data['_block_number'] = block_number
                                data['_timestamp'] = timestamp
                                callback(data)
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
                        callback(data)
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
                    callback(data)

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

                    callback(data)

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
