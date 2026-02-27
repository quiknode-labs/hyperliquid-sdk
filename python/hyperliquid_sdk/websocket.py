"""
WebSocket Client — Real-time data streams with automatic reconnection.

Subscribe to trades, orders, book updates, TWAP, events, and more.
Handles connection management, ping/pong, and automatic reconnection.

Example:
    >>> from hyperliquid_sdk import Stream
    >>> stream = Stream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> stream.trades(["BTC", "ETH"], lambda t: print(t))
    >>> stream.run()
"""

from __future__ import annotations
import json
import threading
import time
import logging
from typing import Optional, List, Callable, Any, Dict
from urllib.parse import urlparse
from enum import Enum

try:
    import websocket
    HAS_WEBSOCKET = True
except ImportError:
    HAS_WEBSOCKET = False

from .errors import HyperliquidError

logger = logging.getLogger(__name__)


class StreamType(str, Enum):
    """Available stream types."""
    TRADES = "trades"
    ORDERS = "orders"
    BOOK_UPDATES = "bookUpdates"
    TWAP = "twap"
    EVENTS = "events"
    WRITER_ACTIONS = "writerActions"
    # Additional subscription types
    L2_BOOK = "l2Book"
    ALL_MIDS = "allMids"
    CANDLE = "candle"
    BBO = "bbo"
    OPEN_ORDERS = "openOrders"
    ORDER_UPDATES = "orderUpdates"
    USER_EVENTS = "userEvents"
    USER_FILLS = "userFills"
    USER_FUNDINGS = "userFundings"
    USER_NON_FUNDING_LEDGER = "userNonFundingLedgerUpdates"
    CLEARINGHOUSE_STATE = "clearinghouseState"
    ACTIVE_ASSET_CTX = "activeAssetCtx"
    ACTIVE_ASSET_DATA = "activeAssetData"
    TWAP_STATES = "twapStates"
    USER_TWAP_SLICE_FILLS = "userTwapSliceFills"
    USER_TWAP_HISTORY = "userTwapHistory"
    NOTIFICATION = "notification"
    WEB_DATA_3 = "webData3"


class ConnectionState(str, Enum):
    """WebSocket connection states."""
    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    RECONNECTING = "reconnecting"


class Stream:
    """
    WebSocket Client — Real-time data streams with automatic reconnection.

    Features:
    - Automatic reconnection with exponential backoff
    - Ping/pong heartbeat to detect stale connections
    - Thread-safe subscription management
    - Graceful shutdown

    Streams:
    - trades: Executed trades with price, size, direction
    - orders: Order lifecycle events (open, filled, cancelled)
    - book_updates: Order book changes
    - twap: Time-weighted average price execution
    - events: System events (funding, liquidations)
    - writer_actions: Spot token transfers

    Examples:
        stream = Stream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

        # Subscribe to trades
        stream.trades(["BTC", "ETH"], lambda t: print(f"{t['coin']}: {t['sz']} @ {t['px']}"))

        # Subscribe to orders
        stream.orders(["BTC"], lambda o: print(f"Order: {o['status']}"))

        # Run (blocking)
        stream.run()

        # Or run in background
        stream.start()
        # ... do other work ...
        stream.stop()
    """

    # Reconnection settings
    INITIAL_RECONNECT_DELAY = 1.0  # seconds
    MAX_RECONNECT_DELAY = 60.0  # seconds
    RECONNECT_BACKOFF_FACTOR = 2.0
    MAX_RECONNECT_ATTEMPTS = None  # Infinite by default

    # Heartbeat settings
    PING_INTERVAL = 30  # seconds
    PING_TIMEOUT = 10  # seconds

    def __init__(
        self,
        endpoint: str,
        *,
        on_error: Optional[Callable[[Exception], None]] = None,
        on_close: Optional[Callable[[], None]] = None,
        on_open: Optional[Callable[[], None]] = None,
        on_reconnect: Optional[Callable[[int], None]] = None,
        on_state_change: Optional[Callable[[ConnectionState], None]] = None,
        reconnect: bool = True,
        max_reconnect_attempts: Optional[int] = None,
        ping_interval: int = 30,
    ):
        """
        Initialize the stream.

        Args:
            endpoint: Hyperliquid endpoint URL
            on_error: Callback for errors (receives Exception)
            on_close: Callback when connection closes permanently
            on_open: Callback when connected (also on reconnect)
            on_reconnect: Callback on reconnection (receives attempt number)
            on_state_change: Callback when connection state changes
            reconnect: Auto-reconnect on disconnect (default: True)
            max_reconnect_attempts: Max reconnection attempts (None = infinite)
            ping_interval: Heartbeat interval in seconds (default: 30)
        """
        if not HAS_WEBSOCKET:
            raise ImportError("websocket-client required. Install: pip install hyperliquid-sdk[websocket]")

        self._ws_url = self._build_ws_url(endpoint)
        self._on_error = on_error
        self._on_close = on_close
        self._on_open = on_open
        self._on_reconnect = on_reconnect
        self._on_state_change = on_state_change
        self._reconnect_enabled = reconnect
        self._max_reconnect_attempts = max_reconnect_attempts or self.MAX_RECONNECT_ATTEMPTS
        self._ping_interval = ping_interval

        self._ws: Optional[websocket.WebSocketApp] = None
        self._thread: Optional[threading.Thread] = None
        self._ping_thread: Optional[threading.Thread] = None
        self._running = False
        self._state = ConnectionState.DISCONNECTED
        self._reconnect_attempt = 0
        self._reconnect_delay = self.INITIAL_RECONNECT_DELAY
        self._last_pong = 0.0
        self._lock = threading.RLock()
        self._subscriptions: Dict[str, Dict[str, Any]] = {}
        self._callbacks: Dict[str, Callable[[Dict[str, Any]], None]] = {}
        # O(1) lookup: channel -> list of callback functions
        self._channel_callbacks: Dict[str, List[Callable[[Dict[str, Any]], None]]] = {}
        self._sub_id = 0

    def _set_state(self, state: ConnectionState) -> None:
        """Update connection state and notify callback."""
        if self._state != state:
            self._state = state
            if self._on_state_change:
                try:
                    self._on_state_change(state)
                except Exception as e:
                    logger.warning(f"State change callback error: {e}")

    def _build_ws_url(self, url: str) -> str:
        """Build WebSocket URL from endpoint.

        Handles:
        - QuickNode: https://x.quiknode.pro/TOKEN -> wss://x.quiknode.pro/TOKEN/hypercore/ws
        - Public API: wss://api.hyperliquid.xyz/ws -> wss://api.hyperliquid.xyz/ws
        - Direct WSS: wss://... -> wss://...
        """
        parsed = urlparse(url)

        # If already a ws/wss URL, use it directly (possibly with /ws path)
        if parsed.scheme in ("ws", "wss"):
            if parsed.path.rstrip("/").endswith("/ws"):
                return url
            return url.rstrip("/") + "/ws" if not parsed.path.endswith("/ws") else url

        # Convert https to wss
        scheme = "wss" if parsed.scheme == "https" else "ws"
        base = f"{scheme}://{parsed.netloc}"

        # Check if this is the public Hyperliquid API
        if "hyperliquid.xyz" in parsed.netloc or "api.hyperliquid" in parsed.netloc:
            return f"{base}/ws"

        # QuickNode endpoint - extract token and build hypercore/ws path
        path_parts = [p for p in parsed.path.strip("/").split("/") if p]
        token = ""
        for part in path_parts:
            if part not in ("info", "hypercore", "evm", "nanoreth", "ws"):
                token = part
                break

        if token:
            return f"{base}/{token}/hypercore/ws"
        return f"{base}/hypercore/ws"

    def _get_sub_id(self) -> str:
        with self._lock:
            self._sub_id += 1
            return f"sub_{self._sub_id}"

    def subscribe(
        self,
        stream_type: str,
        callback: Callable[[Dict[str, Any]], None],
        *,
        coins: Optional[List[str]] = None,
        users: Optional[List[str]] = None,
    ) -> str:
        """Subscribe to a stream."""
        sub_id = self._get_sub_id()
        params: Dict[str, Any] = {"streamType": stream_type}
        if coins:
            params["coin"] = coins
        if users:
            params["user"] = users

        with self._lock:
            self._subscriptions[sub_id] = params
            self._callbacks[sub_id] = callback
            # Update O(1) channel lookup
            if stream_type not in self._channel_callbacks:
                self._channel_callbacks[stream_type] = []
            self._channel_callbacks[stream_type].append(callback)
            if self._ws and self._state == ConnectionState.CONNECTED:
                self._send_subscribe(params)

        return sub_id

    def trades(self, coins: List[str], callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to trade stream. Fields: coin, px, sz, side (B/A), time, dir, closedPnl, hash, oid, tid"""
        return self.subscribe(StreamType.TRADES.value, callback, coins=coins)

    def orders(
        self,
        coins: List[str],
        callback: Callable[[Dict[str, Any]], None],
        *,
        users: Optional[List[str]] = None,
    ) -> str:
        """Subscribe to order stream. Status: open, filled, triggered, canceled, etc."""
        return self.subscribe(StreamType.ORDERS.value, callback, coins=coins, users=users)

    def book_updates(self, coins: List[str], callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to order book updates."""
        return self.subscribe(StreamType.BOOK_UPDATES.value, callback, coins=coins)

    def twap(self, coins: List[str], callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to TWAP execution stream."""
        return self.subscribe(StreamType.TWAP.value, callback, coins=coins)

    def events(self, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to system events (funding, liquidations, governance)."""
        return self.subscribe(StreamType.EVENTS.value, callback)

    def writer_actions(self, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to spot token transfers."""
        return self.subscribe(StreamType.WRITER_ACTIONS.value, callback)

    # ═══════════════════════════════════════════════════════════════════════════
    # ADDITIONAL SUBSCRIPTION METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def l2_book(self, coin: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to L2 order book snapshots."""
        return self.subscribe(StreamType.L2_BOOK.value, callback, coins=[coin])

    def all_mids(self, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to all mid price updates."""
        return self.subscribe(StreamType.ALL_MIDS.value, callback)

    def candle(self, coin: str, interval: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to candlestick data. Interval: 1m, 5m, 15m, 1h, 4h, 1d."""
        sub_id = self._get_sub_id()
        stream_type = StreamType.CANDLE.value
        params = {"streamType": stream_type, "coin": [coin], "interval": interval}
        with self._lock:
            self._subscriptions[sub_id] = params
            self._callbacks[sub_id] = callback
            # Update O(1) channel lookup
            if stream_type not in self._channel_callbacks:
                self._channel_callbacks[stream_type] = []
            self._channel_callbacks[stream_type].append(callback)
            if self._ws and self._state == ConnectionState.CONNECTED:
                self._send_subscribe(params)
        return sub_id

    def bbo(self, coin: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to best bid/offer updates."""
        return self.subscribe(StreamType.BBO.value, callback, coins=[coin])

    def open_orders(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's open orders."""
        return self.subscribe(StreamType.OPEN_ORDERS.value, callback, users=[user])

    def order_updates(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's order status changes."""
        return self.subscribe(StreamType.ORDER_UPDATES.value, callback, users=[user])

    def user_events(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to comprehensive user events (fills, funding, liquidations)."""
        return self.subscribe(StreamType.USER_EVENTS.value, callback, users=[user])

    def user_fills(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's trade fills."""
        return self.subscribe(StreamType.USER_FILLS.value, callback, users=[user])

    def user_fundings(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's funding payment updates."""
        return self.subscribe(StreamType.USER_FUNDINGS.value, callback, users=[user])

    def user_non_funding_ledger(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's ledger changes (deposits, withdrawals, transfers)."""
        return self.subscribe(StreamType.USER_NON_FUNDING_LEDGER.value, callback, users=[user])

    def clearinghouse_state(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's clearinghouse state updates."""
        return self.subscribe(StreamType.CLEARINGHOUSE_STATE.value, callback, users=[user])

    def active_asset_ctx(self, coin: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to asset context data (pricing, volume, supply)."""
        return self.subscribe(StreamType.ACTIVE_ASSET_CTX.value, callback, coins=[coin])

    def active_asset_data(self, user: str, coin: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user's active asset trading parameters."""
        sub_id = self._get_sub_id()
        stream_type = StreamType.ACTIVE_ASSET_DATA.value
        params = {"streamType": stream_type, "user": [user], "coin": [coin]}
        with self._lock:
            self._subscriptions[sub_id] = params
            self._callbacks[sub_id] = callback
            # Update O(1) channel lookup
            if stream_type not in self._channel_callbacks:
                self._channel_callbacks[stream_type] = []
            self._channel_callbacks[stream_type].append(callback)
            if self._ws and self._state == ConnectionState.CONNECTED:
                self._send_subscribe(params)
        return sub_id

    def twap_states(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to TWAP algorithm states."""
        return self.subscribe(StreamType.TWAP_STATES.value, callback, users=[user])

    def user_twap_slice_fills(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to individual TWAP order slice fills."""
        return self.subscribe(StreamType.USER_TWAP_SLICE_FILLS.value, callback, users=[user])

    def user_twap_history(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to TWAP execution history and status."""
        return self.subscribe(StreamType.USER_TWAP_HISTORY.value, callback, users=[user])

    def notification(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to user notifications."""
        return self.subscribe(StreamType.NOTIFICATION.value, callback, users=[user])

    def web_data_3(self, user: str, callback: Callable[[Dict[str, Any]], None]) -> str:
        """Subscribe to aggregate user information for frontend use."""
        return self.subscribe(StreamType.WEB_DATA_3.value, callback, users=[user])

    def unsubscribe(self, sub_id: str) -> None:
        """Unsubscribe from a stream."""
        with self._lock:
            if sub_id in self._subscriptions:
                params = self._subscriptions.pop(sub_id)
                callback = self._callbacks.pop(sub_id, None)
                # Remove from O(1) channel lookup
                stream_type = params.get("streamType", "")
                if callback and stream_type in self._channel_callbacks:
                    try:
                        self._channel_callbacks[stream_type].remove(callback)
                        # Clean up empty lists
                        if not self._channel_callbacks[stream_type]:
                            del self._channel_callbacks[stream_type]
                    except ValueError:
                        pass  # Callback not in list
                if self._ws and self._state == ConnectionState.CONNECTED:
                    self._send_unsubscribe(params)

    def _send_subscribe(self, params: Dict[str, Any]) -> None:
        """Send subscription message.

        Uses standard Hyperliquid format: {"method": "subscribe", "subscription": {...}}
        """
        if not self._ws:
            return

        try:
            # Convert SDK format to Hyperliquid API format
            subscription: Dict[str, Any] = {"type": params.get("streamType", "trades")}

            # Add coin filter if specified
            coins = params.get("coin")
            if coins:
                # API expects single coin, so subscribe to each
                if isinstance(coins, list):
                    for coin in coins:
                        sub = {**subscription, "coin": coin}
                        self._ws.send(json.dumps({"method": "subscribe", "subscription": sub}))
                    return
                subscription["coin"] = coins

            # Add user filter if specified
            users = params.get("user")
            if users:
                if isinstance(users, list):
                    for user in users:
                        sub = {**subscription, "user": user}
                        self._ws.send(json.dumps({"method": "subscribe", "subscription": sub}))
                    return
                subscription["user"] = users

            self._ws.send(json.dumps({"method": "subscribe", "subscription": subscription}))
        except Exception as e:
            logger.warning(f"Failed to send subscription: {e}")

    def _send_unsubscribe(self, params: Dict[str, Any]) -> None:
        """Send unsubscribe message."""
        if not self._ws:
            return

        try:
            subscription: Dict[str, Any] = {"type": params.get("streamType", "trades")}
            coins = params.get("coin")
            if coins:
                if isinstance(coins, list):
                    for coin in coins:
                        sub = {**subscription, "coin": coin}
                        self._ws.send(json.dumps({"method": "unsubscribe", "subscription": sub}))
                    return
                subscription["coin"] = coins
            self._ws.send(json.dumps({"method": "unsubscribe", "subscription": subscription}))
        except Exception as e:
            logger.warning(f"Failed to send unsubscribe: {e}")

    def _on_message(self, ws: Any, message: str) -> None:
        """Handle incoming WebSocket message."""
        try:
            data = json.loads(message)
            channel = data.get("channel", "")

            # Handle pong response
            if channel == "pong" or data.get("type") == "pong":
                self._last_pong = time.time()
                return

            # Skip subscription confirmations
            if channel == "subscriptionResponse":
                return

            # O(1) lookup: Get callbacks for this channel
            # Copy callbacks under lock, then invoke outside lock to avoid blocking
            callbacks_to_invoke: List[Callable[[Dict[str, Any]], None]] = []
            with self._lock:
                if channel in self._channel_callbacks:
                    callbacks_to_invoke = self._channel_callbacks[channel].copy()
                elif channel == "allMids" and "allMids" in self._channel_callbacks:
                    callbacks_to_invoke = self._channel_callbacks["allMids"].copy()

            # Invoke callbacks outside the lock
            for callback in callbacks_to_invoke:
                try:
                    callback(data)
                except Exception as e:
                    logger.warning(f"Callback error: {e}")

        except json.JSONDecodeError:
            logger.warning(f"Invalid JSON received: {message[:100]}")
        except Exception as e:
            if self._on_error:
                self._on_error(e)

    def _on_ws_error(self, ws: Any, error: Exception) -> None:
        """Handle WebSocket error."""
        logger.warning(f"WebSocket error: {error}")
        if self._on_error:
            try:
                self._on_error(error)
            except Exception as e:
                logger.warning(f"Error callback failed: {e}")

    def _on_ws_close(self, ws: Any, close_status_code: Any, close_msg: Any) -> None:
        """Handle WebSocket close."""
        logger.info(f"WebSocket closed: {close_status_code} {close_msg}")
        self._set_state(ConnectionState.DISCONNECTED)

        if self._reconnect_enabled and self._running:
            self._schedule_reconnect()
        else:
            if self._on_close:
                try:
                    self._on_close()
                except Exception as e:
                    logger.warning(f"Close callback failed: {e}")

    def _on_ws_open(self, ws: Any) -> None:
        """Handle WebSocket open."""
        logger.info("WebSocket connected")
        self._set_state(ConnectionState.CONNECTED)
        self._reconnect_attempt = 0
        self._reconnect_delay = self.INITIAL_RECONNECT_DELAY
        self._last_pong = time.time()

        # Resubscribe to all streams
        with self._lock:
            for params in self._subscriptions.values():
                self._send_subscribe(params)

        # Start ping thread
        self._start_ping_thread()

        if self._on_open:
            try:
                self._on_open()
            except Exception as e:
                logger.warning(f"Open callback failed: {e}")

    def _schedule_reconnect(self) -> None:
        """Schedule a reconnection attempt with exponential backoff."""
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
        time.sleep(self._reconnect_delay)
        self._reconnect_delay = min(
            self._reconnect_delay * self.RECONNECT_BACKOFF_FACTOR,
            self.MAX_RECONNECT_DELAY,
        )

        if self._running:
            self._connect()
            if self._ws:
                try:
                    self._ws.run_forever(ping_interval=0)  # We handle our own pings
                except Exception as e:
                    logger.warning(f"Reconnection failed: {e}")
                    self._schedule_reconnect()

    def _start_ping_thread(self) -> None:
        """Start the ping/pong heartbeat thread."""
        def ping_loop():
            while self._running and self._state == ConnectionState.CONNECTED:
                try:
                    time.sleep(self._ping_interval)
                    if not self._running or self._state != ConnectionState.CONNECTED:
                        break

                    # Check for stale connection
                    if self._last_pong > 0 and time.time() - self._last_pong > self._ping_interval + self.PING_TIMEOUT:
                        logger.warning("Connection stale (no pong), closing")
                        if self._ws:
                            self._ws.close()
                        break

                    # Send ping
                    if self._ws:
                        try:
                            self._ws.send(json.dumps({"method": "ping"}))
                        except Exception:
                            break
                except Exception:
                    break

        self._ping_thread = threading.Thread(target=ping_loop, daemon=True)
        self._ping_thread.start()

    def _connect(self) -> None:
        """Create WebSocket connection."""
        self._set_state(ConnectionState.CONNECTING)
        self._ws = websocket.WebSocketApp(
            self._ws_url,
            on_message=self._on_message,
            on_error=self._on_ws_error,
            on_close=self._on_ws_close,
            on_open=self._on_ws_open,
        )

    def run(self) -> None:
        """Run the stream (blocking)."""
        self._running = True
        self._connect()
        if self._ws:
            try:
                self._ws.run_forever(ping_interval=0)  # We handle our own pings
            except KeyboardInterrupt:
                self.stop()

    def start(self) -> None:
        """Start the stream in background."""
        self._running = True
        self._connect()

        def run_forever():
            while self._running:
                try:
                    if self._ws:
                        self._ws.run_forever(ping_interval=0)
                except Exception as e:
                    logger.warning(f"WebSocket error: {e}")
                if self._running and self._reconnect_enabled:
                    self._schedule_reconnect()
                else:
                    break

        self._thread = threading.Thread(target=run_forever, daemon=True)
        self._thread.start()

    def run_in_background(self) -> None:
        """Alias for start() - run the stream in background."""
        self.start()

    def stop(self) -> None:
        """Stop the stream gracefully."""
        self._running = False
        if self._ws:
            try:
                self._ws.close()
            except Exception:
                pass
        if self._thread:
            self._thread.join(timeout=5)

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

    def __enter__(self) -> "Stream":
        return self

    def __exit__(self, *args) -> None:
        self.stop()

    def __repr__(self) -> str:
        return f"<Stream {self._state.value} {len(self._subscriptions)} subs>"
