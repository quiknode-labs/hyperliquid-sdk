"""
EVM WebSocket Streaming — eth_subscribe/eth_unsubscribe for HyperEVM.

Stream EVM events via WebSocket on the /nanoreth namespace:
- newHeads: New block headers
- logs: Contract event logs
- newPendingTransactions: Pending transaction hashes

Example:
    >>> from hyperliquid_sdk import EVMStream
    >>> stream = EVMStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> stream.new_heads(lambda h: print(f"New block: {h['number']}"))
    >>> stream.logs({"address": "0x..."}, lambda log: print(log))
    >>> stream.start()
"""

from __future__ import annotations

import json
import threading
import time
from enum import Enum
from typing import Any, Callable, Dict, List, Optional
from urllib.parse import urlparse

from .errors import HyperliquidError

try:
    import websocket
except ImportError:
    websocket = None  # type: ignore


class EVMSubscriptionType(str, Enum):
    """EVM WebSocket subscription types (eth_subscribe)."""

    NEW_HEADS = "newHeads"
    LOGS = "logs"
    NEW_PENDING_TRANSACTIONS = "newPendingTransactions"


class ConnectionState(str, Enum):
    """Connection state enum."""

    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    RECONNECTING = "reconnecting"


class EVMStream:
    """
    EVM WebSocket Streaming — eth_subscribe/eth_unsubscribe.

    Stream EVM events via WebSocket on the /nanoreth namespace.

    Subscription types:
    - newHeads: Fires when a new block header is appended
    - logs: Logs matching filter criteria (address, topics)
    - newPendingTransactions: Pending transaction hashes

    Example:
        stream = EVMStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

        # Subscribe to new block headers
        stream.new_heads(lambda h: print(f"Block {h['number']}"))

        # Subscribe to contract logs
        stream.logs(
            {"address": "0x...", "topics": ["0x..."]},
            lambda log: print(log)
        )

        # Subscribe to pending transactions
        stream.new_pending_transactions(lambda tx: print(f"Pending: {tx}"))

        stream.start()  # Blocking
        # Or: stream.start(blocking=False)
    """

    def __init__(
        self,
        endpoint: str,
        *,
        on_error: Optional[Callable[[Exception], None]] = None,
        on_open: Optional[Callable[[], None]] = None,
        on_close: Optional[Callable[[], None]] = None,
        on_state_change: Optional[Callable[[ConnectionState], None]] = None,
        reconnect: bool = True,
        max_reconnect_attempts: int = 10,
        reconnect_delay: float = 1.0,
        ping_interval: int = 30,
        ping_timeout: int = 10,
    ):
        """
        Initialize EVM WebSocket stream.

        Args:
            endpoint: Hyperliquid endpoint URL
            on_error: Callback for connection errors
            on_open: Callback when connection opens
            on_close: Callback when connection closes
            on_state_change: Callback for state changes
            reconnect: Auto-reconnect on disconnect
            max_reconnect_attempts: Max reconnection attempts
            reconnect_delay: Initial delay between reconnects (exponential backoff)
            ping_interval: WebSocket ping interval in seconds
            ping_timeout: WebSocket ping timeout in seconds
        """
        if websocket is None:
            raise ImportError(
                "websocket-client is required for EVMStream. Install with: pip install websocket-client"
            )

        self._ws_url = self._build_ws_url(endpoint)
        self._on_error = on_error
        self._on_open = on_open
        self._on_close = on_close
        self._on_state_change = on_state_change
        self._reconnect = reconnect
        self._max_reconnect_attempts = max_reconnect_attempts
        self._reconnect_delay = reconnect_delay
        self._ping_interval = ping_interval
        self._ping_timeout = ping_timeout

        self._ws: Optional[websocket.WebSocketApp] = None
        self._thread: Optional[threading.Thread] = None
        self._running = False
        self._state = ConnectionState.DISCONNECTED
        self._reconnect_count = 0

        # Subscription management
        self._request_id = 0
        self._pending_subscriptions: List[Dict[str, Any]] = []
        self._active_subscriptions: Dict[str, Dict[str, Any]] = {}  # sub_id -> subscription info
        self._callbacks: Dict[str, Callable] = {}  # sub_id -> callback

    def _build_ws_url(self, url: str) -> str:
        """Build WebSocket URL for nanoreth namespace."""
        parsed = urlparse(url)
        # Use wss for https, ws for http
        ws_scheme = "wss" if parsed.scheme == "https" else "ws"
        base = f"{ws_scheme}://{parsed.netloc}"

        # Extract token from path
        path_parts = parsed.path.strip("/").split("/")
        token = ""
        for part in path_parts:
            if part not in ("info", "hypercore", "evm", "nanoreth", "ws"):
                token = part
                break

        # WebSocket for EVM is on /nanoreth namespace
        if token:
            return f"{base}/{token}/nanoreth"
        return f"{base}/nanoreth"

    def _set_state(self, state: ConnectionState) -> None:
        """Update connection state."""
        if self._state != state:
            self._state = state
            if self._on_state_change:
                try:
                    self._on_state_change(state)
                except Exception:
                    pass

    def _next_id(self) -> int:
        """Get next request ID."""
        self._request_id += 1
        return self._request_id

    # ═══════════════════════════════════════════════════════════════════════════
    # SUBSCRIPTION METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def new_heads(self, callback: Callable[[Dict[str, Any]], None]) -> "EVMStream":
        """
        Subscribe to new block headers.

        Fires a notification each time a new header is appended to the chain,
        including during chain reorganizations.

        Args:
            callback: Function called with each new block header

        Example:
            stream.new_heads(lambda h: print(f"Block {int(h['number'], 16)}"))
        """
        sub = {
            "type": EVMSubscriptionType.NEW_HEADS,
            "params": None,
            "callback": callback,
        }
        self._pending_subscriptions.append(sub)
        return self

    def logs(
        self,
        filter_params: Optional[Dict[str, Any]],
        callback: Callable[[Dict[str, Any]], None],
    ) -> "EVMStream":
        """
        Subscribe to contract event logs.

        Returns logs that are included in new imported blocks and match
        the given filter criteria.

        Args:
            filter_params: Filter parameters:
                - address: Contract address or list of addresses
                - topics: List of topic filters (up to 4 topics)
            callback: Function called with each matching log

        Example:
            stream.logs(
                {"address": "0x...", "topics": ["0x..."]},
                lambda log: print(log)
            )
        """
        sub = {
            "type": EVMSubscriptionType.LOGS,
            "params": filter_params,
            "callback": callback,
        }
        self._pending_subscriptions.append(sub)
        return self

    def new_pending_transactions(
        self, callback: Callable[[str], None]
    ) -> "EVMStream":
        """
        Subscribe to pending transaction hashes.

        Returns the hash for all transactions that are added to the pending state.

        Args:
            callback: Function called with each pending transaction hash

        Example:
            stream.new_pending_transactions(lambda tx: print(f"Pending: {tx}"))
        """
        sub = {
            "type": EVMSubscriptionType.NEW_PENDING_TRANSACTIONS,
            "params": None,
            "callback": callback,
        }
        self._pending_subscriptions.append(sub)
        return self

    def _send_subscriptions(self) -> None:
        """Send all pending subscriptions to the server."""
        if not self._ws:
            return

        for sub in self._pending_subscriptions:
            req_id = self._next_id()
            params = [sub["type"].value]
            if sub["params"]:
                params.append(sub["params"])

            msg = {
                "jsonrpc": "2.0",
                "method": "eth_subscribe",
                "params": params,
                "id": req_id,
            }

            # Store callback temporarily by request ID
            self._callbacks[f"req_{req_id}"] = sub["callback"]

            try:
                self._ws.send(json.dumps(msg))
            except Exception as e:
                if self._on_error:
                    self._on_error(e)

    def unsubscribe(self, subscription_id: str) -> bool:
        """
        Unsubscribe from a subscription.

        Args:
            subscription_id: The subscription ID to cancel

        Returns:
            True if unsubscription was sent
        """
        if not self._ws or subscription_id not in self._active_subscriptions:
            return False

        req_id = self._next_id()
        msg = {
            "jsonrpc": "2.0",
            "method": "eth_unsubscribe",
            "params": [subscription_id],
            "id": req_id,
        }

        try:
            self._ws.send(json.dumps(msg))
            # Clean up
            self._active_subscriptions.pop(subscription_id, None)
            self._callbacks.pop(subscription_id, None)
            return True
        except Exception as e:
            if self._on_error:
                self._on_error(e)
            return False

    # ═══════════════════════════════════════════════════════════════════════════
    # WEBSOCKET HANDLERS
    # ═══════════════════════════════════════════════════════════════════════════

    def _on_ws_open(self, ws: Any) -> None:
        """Handle WebSocket open."""
        self._set_state(ConnectionState.CONNECTED)
        self._reconnect_count = 0
        self._send_subscriptions()
        if self._on_open:
            try:
                self._on_open()
            except Exception:
                pass

    def _on_ws_close(self, ws: Any, close_status_code: Any, close_msg: Any) -> None:
        """Handle WebSocket close."""
        self._set_state(ConnectionState.DISCONNECTED)
        if self._on_close:
            try:
                self._on_close()
            except Exception:
                pass

        # Attempt reconnection
        if self._running and self._reconnect:
            self._try_reconnect()

    def _on_ws_error(self, ws: Any, error: Any) -> None:
        """Handle WebSocket error."""
        if self._on_error:
            try:
                self._on_error(Exception(str(error)))
            except Exception:
                pass

    def _on_ws_message(self, ws: Any, message: str) -> None:
        """Handle WebSocket message."""
        try:
            data = json.loads(message)
        except json.JSONDecodeError:
            return

        # Check for subscription confirmation
        if "id" in data and "result" in data:
            req_key = f"req_{data['id']}"
            if req_key in self._callbacks:
                sub_id = data["result"]
                # Move callback from request ID to subscription ID
                self._callbacks[sub_id] = self._callbacks.pop(req_key)
                self._active_subscriptions[sub_id] = {"id": sub_id}
            return

        # Check for subscription data
        if data.get("method") == "eth_subscription":
            params = data.get("params", {})
            sub_id = params.get("subscription")
            result = params.get("result")

            if sub_id and sub_id in self._callbacks:
                try:
                    self._callbacks[sub_id](result)
                except Exception as e:
                    if self._on_error:
                        self._on_error(e)

    def _try_reconnect(self) -> None:
        """Attempt to reconnect with exponential backoff."""
        if self._reconnect_count >= self._max_reconnect_attempts:
            self._running = False
            return

        self._set_state(ConnectionState.RECONNECTING)
        delay = self._reconnect_delay * (2**self._reconnect_count)
        self._reconnect_count += 1
        time.sleep(min(delay, 30))  # Cap at 30 seconds

        if self._running:
            self._connect()

    def _connect(self) -> None:
        """Create and connect WebSocket."""
        self._set_state(ConnectionState.CONNECTING)
        self._ws = websocket.WebSocketApp(
            self._ws_url,
            on_open=self._on_ws_open,
            on_message=self._on_ws_message,
            on_error=self._on_ws_error,
            on_close=self._on_ws_close,
        )
        self._ws.run_forever(
            ping_interval=self._ping_interval,
            ping_timeout=self._ping_timeout,
        )

    # ═══════════════════════════════════════════════════════════════════════════
    # PUBLIC CONTROL METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def start(self, blocking: bool = True) -> None:
        """
        Start the WebSocket connection.

        Args:
            blocking: If True, block until stop() is called.
                     If False, run in background thread.
        """
        if self._running:
            return

        self._running = True

        if blocking:
            self._connect()
        else:
            self._thread = threading.Thread(target=self._connect, daemon=True)
            self._thread.start()

    def stop(self) -> None:
        """Stop the WebSocket connection."""
        self._running = False
        if self._ws:
            self._ws.close()
        if self._thread:
            self._thread.join(timeout=5)

    @property
    def state(self) -> ConnectionState:
        """Get current connection state."""
        return self._state

    @property
    def connected(self) -> bool:
        """Check if connected."""
        return self._state == ConnectionState.CONNECTED

    @property
    def subscriptions(self) -> List[str]:
        """Get list of active subscription IDs."""
        return list(self._active_subscriptions.keys())

    def __enter__(self) -> "EVMStream":
        return self

    def __exit__(self, *args: Any) -> None:
        self.stop()

    def __repr__(self) -> str:
        return f"<EVMStream {self._ws_url[:40]}... state={self._state.value}>"
