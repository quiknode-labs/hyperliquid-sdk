"""
HyperCore JSON-RPC API Client — Blocks, trading, and real-time data.

Access HyperCore-specific methods via QuickNode's /hypercore endpoint.

Example:
    >>> from hyperliquid_sdk import HyperCore
    >>> hc = HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> print(hc.latest_block_number())  # Get latest block
    >>> print(hc.latest_trades(count=10))  # Get recent trades
"""

from __future__ import annotations
from typing import Optional, List, Dict, Any
from urllib.parse import urlparse

import requests

from .errors import HyperliquidError


class HyperCore:
    """
    HyperCore JSON-RPC API — Blocks, trading, and real-time data.

    Access block data, trading operations, and real-time streams via JSON-RPC.

    Examples:
        hc = HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

        # Block data
        hc.latest_block_number()
        hc.get_block(12345)
        hc.get_batch_blocks([100, 101, 102])
        hc.latest_blocks(count=10)

        # Recent data (alternative to Info methods not on QuickNode)
        hc.latest_trades(count=10)
        hc.latest_orders(count=10)
        hc.latest_book_updates(count=10)

        # Discovery
        hc.list_dexes()
        hc.list_markets()
        hc.list_markets(dex="hyperliquidity")
    """

    def __init__(self, endpoint: str, *, timeout: int = 30):
        """
        Initialize the HyperCore client.

        Args:
            endpoint: Hyperliquid endpoint URL (e.g., https://your-endpoint.quiknode.pro/TOKEN)
            timeout: Request timeout in seconds (default: 30)
        """
        self._hypercore_url = self._build_url(endpoint)
        self._timeout = timeout
        self._session = requests.Session()
        self._id = 0

    def _build_url(self, url: str) -> str:
        """Build the /hypercore endpoint URL."""
        parsed = urlparse(url)
        base = f"{parsed.scheme}://{parsed.netloc}"
        path_parts = [p for p in parsed.path.strip("/").split("/") if p]

        # Find the token (not a known path)
        token = None
        for part in path_parts:
            if part not in ("info", "hypercore", "evm", "nanoreth", "ws"):
                token = part
                break

        if token:
            return f"{base}/{token}/hypercore"
        return f"{base}/hypercore"

    def _rpc(self, method: str, params: Any = None) -> Any:
        """Make a JSON-RPC 2.0 request."""
        self._id += 1
        body: Dict[str, Any] = {
            "jsonrpc": "2.0",
            "method": method,
            "id": self._id,
        }
        if params is not None:
            body["params"] = params
        else:
            body["params"] = {}

        try:
            resp = self._session.post(self._hypercore_url, json=body, timeout=self._timeout)
        except requests.exceptions.Timeout:
            raise HyperliquidError(
                f"Request timed out after {self._timeout}s",
                code="TIMEOUT",
                raw={"method": method, "timeout": self._timeout},
            )
        except requests.exceptions.ConnectionError as e:
            raise HyperliquidError(
                f"Connection failed: {e}",
                code="CONNECTION_ERROR",
                raw={"method": method, "error": str(e)},
            ) from e

        if resp.status_code != 200:
            raise HyperliquidError(
                f"Request failed with status {resp.status_code}",
                code="HTTP_ERROR",
                raw={"status": resp.status_code, "body": resp.text},
            )

        try:
            data = resp.json()
        except ValueError:
            raise HyperliquidError(
                "Invalid JSON response",
                code="PARSE_ERROR",
                raw={"body": resp.text[:500]},
            )

        if "error" in data:
            error = data["error"]
            # Handle both dict and string error formats
            if isinstance(error, dict):
                message = error.get("message", str(error))
                code = error.get("code", "RPC_ERROR")
            else:
                message = str(error)
                code = "RPC_ERROR"
            raise HyperliquidError(message, code=str(code), raw=error)

        return data.get("result")

    # ═══════════════════════════════════════════════════════════════════════════
    # BLOCK DATA
    # ═══════════════════════════════════════════════════════════════════════════

    def latest_block_number(self, stream: str = "trades") -> int:
        """
        Get the latest block number for a stream.

        Args:
            stream: Stream type ("trades", "orders", "events", "book", "twap", "writer_actions")
        """
        return self._rpc("hl_getLatestBlockNumber", {"stream": stream})

    def get_block(self, block_number: int, stream: str = "trades") -> Dict[str, Any]:
        """
        Get a specific block by number.

        Args:
            block_number: Block number to fetch
            stream: Stream type ("trades", "orders", "events", "book", "twap", "writer_actions")
        """
        # Uses array format: [stream, block_number]
        return self._rpc("hl_getBlock", [stream, block_number])

    def get_batch_blocks(self, from_block: int, to_block: int, stream: str = "trades") -> List[Dict[str, Any]]:
        """
        Get a range of blocks.

        Args:
            from_block: Starting block number
            to_block: Ending block number (inclusive)
            stream: Stream type ("trades", "orders", "events", "book", "twap", "writer_actions")
        """
        return self._rpc("hl_getBatchBlocks", {"stream": stream, "from": from_block, "to": to_block})

    def latest_blocks(self, stream: str = "trades", *, count: int = 10) -> Dict[str, Any]:
        """
        Get the latest blocks for a stream.

        Args:
            stream: Stream type ("trades", "orders", "book_updates", "twap", "events", "writer_actions")
            count: Number of blocks to return (default: 10)
        """
        return self._rpc("hl_getLatestBlocks", {"stream": stream, "count": count})

    # ═══════════════════════════════════════════════════════════════════════════
    # RECENT DATA (Alternative to unsupported Info methods)
    # ═══════════════════════════════════════════════════════════════════════════

    def latest_trades(self, *, count: int = 10, coin: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Get recent trades from latest blocks.

        This is an alternative to Info.recent_trades() which is not available on QuickNode.

        Args:
            count: Number of blocks to fetch (default: 10)
            coin: Optional coin filter (e.g., "BTC", "ETH")

        Returns:
            List of trade events from recent blocks
        """
        result = self.latest_blocks("trades", count=count)
        trades = []
        for block in result.get("blocks", []):
            for event in block.get("events", []):
                if len(event) >= 2:
                    user, trade = event[0], event[1]
                    if coin is None or trade.get("coin") == coin:
                        trades.append({"user": user, **trade})
        return trades

    def latest_orders(self, *, count: int = 10, coin: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Get recent order events from latest blocks.

        Args:
            count: Number of blocks to fetch (default: 10)
            coin: Optional coin filter

        Returns:
            List of order events from recent blocks
        """
        result = self.latest_blocks("orders", count=count)
        orders = []
        for block in result.get("blocks", []):
            for event in block.get("events", []):
                if len(event) >= 2:
                    user, order = event[0], event[1]
                    if coin is None or order.get("coin") == coin:
                        orders.append({"user": user, **order})
        return orders

    def latest_book_updates(self, *, count: int = 10, coin: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Get recent book updates from latest blocks.

        This is an alternative to Info.l2_book() for real-time updates.

        Args:
            count: Number of blocks to fetch (default: 10)
            coin: Optional coin filter

        Returns:
            List of book update events from recent blocks
        """
        result = self.latest_blocks("book_updates", count=count)
        updates = []
        for block in result.get("blocks", []):
            for event in block.get("events", []):
                if isinstance(event, dict):
                    if coin is None or event.get("coin") == coin:
                        updates.append(event)
        return updates

    # ═══════════════════════════════════════════════════════════════════════════
    # DISCOVERY
    # ═══════════════════════════════════════════════════════════════════════════

    def list_dexes(self) -> List[Dict[str, Any]]:
        """List all available DEXes."""
        return self._rpc("hl_listDexes")

    def list_markets(self, *, dex: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        List available markets.

        Args:
            dex: Optional DEX filter (e.g., "hyperliquidity")
        """
        params = {}
        if dex:
            params["dex"] = dex
        return self._rpc("hl_listMarkets", params if params else None)

    # ═══════════════════════════════════════════════════════════════════════════
    # ORDER QUERIES
    # ═══════════════════════════════════════════════════════════════════════════

    def open_orders(self, user: str) -> List[Dict[str, Any]]:
        """Get open orders for a user."""
        return self._rpc("hl_openOrders", {"user": user})

    def order_status(self, user: str, oid: int) -> Dict[str, Any]:
        """Get status of a specific order."""
        return self._rpc("hl_orderStatus", {"user": user, "oid": oid})

    def preflight(
        self,
        coin: str,
        is_buy: bool,
        limit_px: str,
        sz: str,
        user: str,
        *,
        reduce_only: bool = False,
        order_type: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Validate an order before signing (preflight check).

        Args:
            coin: Trading pair (e.g., "BTC")
            is_buy: True for buy, False for sell
            limit_px: Limit price as string
            sz: Size as string
            user: User address
            reduce_only: Whether order is reduce-only
            order_type: Optional order type configuration
        """
        params = {
            "coin": coin,
            "isBuy": is_buy,
            "limitPx": limit_px,
            "sz": sz,
            "user": user,
            "reduceOnly": reduce_only,
        }
        if order_type:
            params["orderType"] = order_type
        return self._rpc("hl_preflight", params)

    # ═══════════════════════════════════════════════════════════════════════════
    # BUILDER FEE
    # ═══════════════════════════════════════════════════════════════════════════

    def get_max_builder_fee(self, user: str, builder: str) -> Dict[str, Any]:
        """Get maximum builder fee for a user-builder pair."""
        return self._rpc("hl_getMaxBuilderFee", {"user": user, "builder": builder})

    # ═══════════════════════════════════════════════════════════════════════════
    # ORDER BUILDING (Returns unsigned actions for signing)
    # ═══════════════════════════════════════════════════════════════════════════

    def build_order(
        self,
        coin: str,
        is_buy: bool,
        limit_px: str,
        sz: str,
        user: str,
        *,
        reduce_only: bool = False,
        order_type: Optional[Dict[str, Any]] = None,
        cloid: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Build an order for signing.

        Args:
            coin: Trading pair (e.g., "BTC")
            is_buy: True for buy, False for sell
            limit_px: Limit price as string
            sz: Size as string
            user: User address
            reduce_only: Whether order is reduce-only
            order_type: Optional order type configuration
            cloid: Optional client order ID
        """
        params: Dict[str, Any] = {
            "coin": coin,
            "isBuy": is_buy,
            "limitPx": limit_px,
            "sz": sz,
            "user": user,
            "reduceOnly": reduce_only,
        }
        if order_type:
            params["orderType"] = order_type
        if cloid:
            params["cloid"] = cloid
        return self._rpc("hl_buildOrder", params)

    def build_cancel(self, coin: str, oid: int, user: str) -> Dict[str, Any]:
        """
        Build a cancel action for signing.

        Args:
            coin: Trading pair
            oid: Order ID to cancel
            user: User address
        """
        return self._rpc("hl_buildCancel", {"coin": coin, "oid": oid, "user": user})

    def build_modify(
        self,
        coin: str,
        oid: int,
        user: str,
        *,
        limit_px: Optional[str] = None,
        sz: Optional[str] = None,
        is_buy: Optional[bool] = None,
    ) -> Dict[str, Any]:
        """
        Build a modify action for signing.

        Args:
            coin: Trading pair
            oid: Order ID to modify
            user: User address
            limit_px: New limit price
            sz: New size
            is_buy: New side
        """
        params: Dict[str, Any] = {"coin": coin, "oid": oid, "user": user}
        if limit_px is not None:
            params["limitPx"] = limit_px
        if sz is not None:
            params["sz"] = sz
        if is_buy is not None:
            params["isBuy"] = is_buy
        return self._rpc("hl_buildModify", params)

    def build_approve_builder_fee(
        self, user: str, builder: str, max_fee_rate: str, nonce: int
    ) -> Dict[str, Any]:
        """
        Build a builder fee approval for signing.

        Args:
            user: User address
            builder: Builder address
            max_fee_rate: Maximum fee rate (e.g., "0.001" for 0.1%)
            nonce: Nonce for the action
        """
        return self._rpc("hl_buildApproveBuilderFee", {
            "user": user,
            "builder": builder,
            "maxFeeRate": max_fee_rate,
            "nonce": nonce,
        })

    def build_revoke_builder_fee(self, user: str, builder: str, nonce: int) -> Dict[str, Any]:
        """
        Build a builder fee revocation for signing.

        Args:
            user: User address
            builder: Builder address to revoke
            nonce: Nonce for the action
        """
        return self._rpc("hl_buildRevokeBuilderFee", {
            "user": user,
            "builder": builder,
            "nonce": nonce,
        })

    # ═══════════════════════════════════════════════════════════════════════════
    # SENDING SIGNED ACTIONS
    # ═══════════════════════════════════════════════════════════════════════════

    def send_order(self, action: Dict[str, Any], signature: str, nonce: int) -> Dict[str, Any]:
        """
        Send a signed order.

        Args:
            action: The order action from build_order
            signature: EIP-712 signature
            nonce: Nonce used in signing
        """
        return self._rpc("hl_sendOrder", {"action": action, "signature": signature, "nonce": nonce})

    def send_cancel(self, action: Dict[str, Any], signature: str, nonce: int) -> Dict[str, Any]:
        """
        Send a signed cancel.

        Args:
            action: The cancel action from build_cancel
            signature: EIP-712 signature
            nonce: Nonce used in signing
        """
        return self._rpc("hl_sendCancel", {"action": action, "signature": signature, "nonce": nonce})

    def send_modify(self, action: Dict[str, Any], signature: str, nonce: int) -> Dict[str, Any]:
        """
        Send a signed modify.

        Args:
            action: The modify action from build_modify
            signature: EIP-712 signature
            nonce: Nonce used in signing
        """
        return self._rpc("hl_sendModify", {"action": action, "signature": signature, "nonce": nonce})

    def send_approval(self, action: Dict[str, Any], signature: str) -> Dict[str, Any]:
        """
        Send a signed builder fee approval.

        Args:
            action: The approval action from build_approve_builder_fee
            signature: EIP-712 signature
        """
        return self._rpc("hl_sendApproval", {"action": action, "signature": signature})

    def send_revocation(self, action: Dict[str, Any], signature: str) -> Dict[str, Any]:
        """
        Send a signed builder fee revocation.

        Args:
            action: The revocation action from build_revoke_builder_fee
            signature: EIP-712 signature
        """
        return self._rpc("hl_sendRevocation", {"action": action, "signature": signature})

    # ═══════════════════════════════════════════════════════════════════════════
    # WEBSOCKET SUBSCRIPTIONS (via JSON-RPC)
    # ═══════════════════════════════════════════════════════════════════════════

    def subscribe(self, subscription: Dict[str, Any]) -> Dict[str, Any]:
        """
        Subscribe to a WebSocket stream via JSON-RPC.

        Args:
            subscription: Subscription parameters (type, coin, user, etc.)

        Example:
            hc.subscribe({"type": "trades", "coin": "BTC"})
        """
        return self._rpc("hl_subscribe", {"subscription": subscription})

    def unsubscribe(self, subscription: Dict[str, Any]) -> Dict[str, Any]:
        """
        Unsubscribe from a WebSocket stream.

        Args:
            subscription: Subscription parameters to unsubscribe from
        """
        return self._rpc("hl_unsubscribe", {"subscription": subscription})

    def __enter__(self) -> HyperCore:
        return self

    def __exit__(self, *args) -> None:
        self._session.close()

    def __repr__(self) -> str:
        return f"<HyperCore {self._hypercore_url[:40]}...>"
