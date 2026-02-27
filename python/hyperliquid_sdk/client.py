"""
Hyperliquid SDK Client — The magic happens here.

One class to rule them all:
- Auto build → sign → send (no ceremony)
- Auto approval management
- Smart defaults everywhere
- Full power when you need it
- Unified access: sdk.info, sdk.core, sdk.evm, sdk.stream, sdk.grpc
"""

from __future__ import annotations
import os
import time as time_module
from typing import Optional, Union, List, Dict, Any, TYPE_CHECKING
from decimal import Decimal

import requests
from eth_account import Account

from .order import Order, PlacedOrder, Side, TIF
from .errors import (
    HyperliquidError,
    BuildError,
    SendError,
    ApprovalError,
    ValidationError,
    NoPositionError,
    parse_api_error,
)

if TYPE_CHECKING:
    from .info import Info
    from .hypercore import HyperCore
    from .evm import EVM
    from .websocket import Stream
    from .grpc_stream import GRPCStream
    from .evm_stream import EVMStream


class HyperliquidSDK:
    """
    Hyperliquid SDK — Stupidly simple, insanely powerful.

    Unified access to ALL Hyperliquid APIs through a single SDK instance.

    Examples:
        # QUICKNODE USERS — Full power with one line:
        sdk = HyperliquidSDK("https://your-endpoint.quiknode.pro/TOKEN")

        # Access everything through the SDK:
        sdk.info.meta()                    # Info API
        sdk.info.clearinghouse_state("0x...")
        sdk.core.latest_block_number()     # HyperCore
        sdk.evm.block_number()             # HyperEVM
        sdk.stream.trades(["BTC"], cb)     # WebSocket
        sdk.grpc.trades(["BTC"], cb)       # gRPC

        # TRADING — Add private key:
        sdk = HyperliquidSDK("https://...", private_key="0x...")

        # Or use environment variable:
        # export PRIVATE_KEY=0x...
        sdk = HyperliquidSDK("https://...")

        # Then trade:
        sdk.market_buy("BTC", size=0.001)
        sdk.buy("BTC", size=0.001, price=67000)
        sdk.close_position("BTC")
        sdk.cancel_all()

        # PUBLIC API ONLY (no QuickNode endpoint):
        sdk = HyperliquidSDK()
        sdk.markets()     # Get all markets
        sdk.get_mid("BTC")  # Get mid price
    """

    DEFAULT_API_URL = "https://send.hyperliquidapi.com"
    DEFAULT_INFO_URL = "https://api.hyperliquid.xyz/info"
    DEFAULT_SLIPPAGE = 0.03  # 3% for market orders
    DEFAULT_TIMEOUT = 30  # seconds
    CACHE_TTL = 300  # 5 minutes cache TTL for market metadata

    def __init__(
        self,
        endpoint: Optional[str] = None,
        *,
        private_key: Optional[str] = None,
        auto_approve: bool = True,
        max_fee: str = "1%",
        slippage: float = DEFAULT_SLIPPAGE,
        timeout: int = DEFAULT_TIMEOUT,
    ):
        """
        Initialize the SDK.

        Args:
            endpoint: QuickNode endpoint URL. Enables: sdk.info, sdk.core, sdk.evm, sdk.stream, sdk.grpc
            private_key: Hex private key (with or without 0x). Falls back to PRIVATE_KEY env var.
                        Required for trading. Optional for read-only operations.
            auto_approve: Automatically approve builder fee for trading (default: True)
            max_fee: Max builder fee to approve (default: "1%")
            slippage: Default slippage for market orders (default: 3%)
            timeout: Request timeout in seconds (default: 30)
        """
        # Store endpoint for lazy initialization of sub-clients
        self._endpoint = endpoint
        self._timeout = timeout
        self._slippage = slippage
        self._max_fee = max_fee
        self._auto_approve = auto_approve

        # Get private key (optional for read-only operations)
        pk = private_key or os.environ.get("PRIVATE_KEY")

        # Initialize wallet if key provided
        if pk:
            self._wallet = Account.from_key(pk)
            self.address = self._wallet.address
        else:
            self._wallet = None
            self.address = None

        # Config for builder API (trading)
        self._api_url = self.DEFAULT_API_URL
        self._info_url = self.DEFAULT_INFO_URL
        self._session = requests.Session()

        # Cache for market metadata with TTL (reduces API calls)
        self._markets_cache: Optional[dict] = None
        self._markets_cache_time: float = 0
        self._sz_decimals_cache: Dict[str, int] = {}

        # Lazy-initialized sub-clients
        self._info: Optional[Info] = None
        self._core: Optional[HyperCore] = None
        self._evm: Optional[EVM] = None
        self._stream: Optional[Stream] = None
        self._grpc: Optional[GRPCStream] = None
        self._evm_stream: Optional[EVMStream] = None

        # Auto-approve if requested and wallet available
        if auto_approve and self._wallet:
            self._ensure_approved(max_fee)

    # ═══════════════════════════════════════════════════════════════════════════
    # UNIFIED SUB-CLIENTS — Lazy initialization
    # ═══════════════════════════════════════════════════════════════════════════

    @property
    def info(self) -> "Info":
        """
        Info API — Market data, positions, orders, vaults.

        Examples:
            sdk.info.meta()
            sdk.info.all_mids()
            sdk.info.clearinghouse_state("0x...")
            sdk.info.open_orders("0x...")
        """
        if self._info is None:
            if not self._endpoint:
                raise ValueError(
                    "Endpoint required for Info API. "
                    "Pass endpoint to HyperliquidSDK() constructor."
                )
            from .info import Info
            self._info = Info(self._endpoint, timeout=self._timeout)
        return self._info

    @property
    def core(self) -> "HyperCore":
        """
        HyperCore API — Blocks, trades, orders from the L1.

        Examples:
            sdk.core.latest_block_number()
            sdk.core.latest_trades(count=10)
            sdk.core.block_by_number(12345)
        """
        if self._core is None:
            if not self._endpoint:
                raise ValueError(
                    "Endpoint required for HyperCore API. "
                    "Pass endpoint to HyperliquidSDK() constructor."
                )
            from .hypercore import HyperCore
            self._core = HyperCore(self._endpoint, timeout=self._timeout)
        return self._core

    @property
    def evm(self) -> "EVM":
        """
        HyperEVM API — Ethereum JSON-RPC for HyperEVM.

        Examples:
            sdk.evm.block_number()
            sdk.evm.get_balance("0x...")
            sdk.evm.call(to, data)
        """
        if self._evm is None:
            if not self._endpoint:
                raise ValueError(
                    "Endpoint required for EVM API. "
                    "Pass endpoint to HyperliquidSDK() constructor."
                )
            from .evm import EVM
            self._evm = EVM(self._endpoint, timeout=self._timeout)
        return self._evm

    @property
    def stream(self) -> "Stream":
        """
        WebSocket Streaming — Real-time trades, orderbook, orders.

        Examples:
            sdk.stream.trades(["BTC"], lambda t: print(t))
            sdk.stream.l2_book("ETH", callback)
            sdk.stream.run()
        """
        if self._stream is None:
            if not self._endpoint:
                raise ValueError(
                    "Endpoint required for WebSocket streaming. "
                    "Pass endpoint to HyperliquidSDK() constructor."
                )
            from .websocket import Stream
            self._stream = Stream(self._endpoint)
        return self._stream

    @property
    def grpc(self) -> "GRPCStream":
        """
        gRPC Streaming — High-performance real-time data.

        Examples:
            sdk.grpc.trades(["BTC"], lambda t: print(t))
            sdk.grpc.l2_book("ETH", callback)
            sdk.grpc.run()
        """
        if self._grpc is None:
            if not self._endpoint:
                raise ValueError(
                    "Endpoint required for gRPC streaming. "
                    "Pass endpoint to HyperliquidSDK() constructor."
                )
            from .grpc_stream import GRPCStream
            self._grpc = GRPCStream(self._endpoint)
        return self._grpc

    @property
    def evm_stream(self) -> "EVMStream":
        """
        EVM WebSocket Streaming — eth_subscribe/eth_unsubscribe.

        Stream EVM events via WebSocket on the /nanoreth namespace:
        - newHeads: New block headers
        - logs: Contract event logs
        - newPendingTransactions: Pending transaction hashes

        Examples:
            sdk.evm_stream.new_heads(lambda h: print(f"Block: {h['number']}"))
            sdk.evm_stream.logs({"address": "0x..."}, callback)
            sdk.evm_stream.new_pending_transactions(callback)
            sdk.evm_stream.start()
        """
        if self._evm_stream is None:
            if not self._endpoint:
                raise ValueError(
                    "Endpoint required for EVM WebSocket streaming. "
                    "Pass endpoint to HyperliquidSDK() constructor."
                )
            from .evm_stream import EVMStream
            self._evm_stream = EVMStream(self._endpoint)
        return self._evm_stream

    # ═══════════════════════════════════════════════════════════════════════════
    # ORDER PLACEMENT — The Simple Way
    # ═══════════════════════════════════════════════════════════════════════════

    def buy(
        self,
        asset: str,
        *,
        size: Optional[Union[float, str]] = None,
        notional: Optional[float] = None,
        price: Optional[Union[float, int, str]] = None,
        tif: Union[str, TIF] = TIF.IOC,
        reduce_only: bool = False,
    ) -> PlacedOrder:
        """
        Place a buy order.

        Args:
            asset: Asset to buy ("BTC", "ETH", "xyz:SILVER")
            size: Size in asset units
            notional: Size in USD (alternative to size)
            price: Limit price (omit for market order)
            tif: Time in force ("ioc", "gtc", "alo", "market")
            reduce_only: Close position only, no new exposure

        Returns:
            PlacedOrder with oid, status, and cancel/modify methods

        Examples:
            sdk.buy("BTC", size=0.001, price=67000)
            sdk.buy("ETH", notional=100, tif="market")
        """
        return self._place_order(
            asset=asset,
            side=Side.BUY,
            size=size,
            notional=notional,
            price=price,
            tif=tif,
            reduce_only=reduce_only,
        )

    def sell(
        self,
        asset: str,
        *,
        size: Optional[Union[float, str]] = None,
        notional: Optional[float] = None,
        price: Optional[Union[float, int, str]] = None,
        tif: Union[str, TIF] = TIF.IOC,
        reduce_only: bool = False,
    ) -> PlacedOrder:
        """
        Place a sell order.

        Args:
            asset: Asset to sell ("BTC", "ETH", "xyz:SILVER")
            size: Size in asset units
            notional: Size in USD (alternative to size)
            price: Limit price (omit for market order)
            tif: Time in force ("ioc", "gtc", "alo", "market")
            reduce_only: Close position only, no new exposure

        Returns:
            PlacedOrder with oid, status, and cancel/modify methods
        """
        return self._place_order(
            asset=asset,
            side=Side.SELL,
            size=size,
            notional=notional,
            price=price,
            tif=tif,
            reduce_only=reduce_only,
        )

    # Aliases for perp traders
    long = buy
    short = sell

    def market_buy(
        self,
        asset: str,
        *,
        size: Optional[Union[float, str]] = None,
        notional: Optional[float] = None,
    ) -> PlacedOrder:
        """
        Market buy — executes immediately at best available price.

        Args:
            asset: Asset to buy
            size: Size in asset units
            notional: Size in USD (alternative to size)

        Example:
            sdk.market_buy("BTC", size=0.001)
            sdk.market_buy("ETH", notional=100)  # $100 worth
        """
        return self.buy(asset, size=size, notional=notional, tif="market")

    def market_sell(
        self,
        asset: str,
        *,
        size: Optional[Union[float, str]] = None,
        notional: Optional[float] = None,
    ) -> PlacedOrder:
        """
        Market sell — executes immediately at best available price.

        Args:
            asset: Asset to sell
            size: Size in asset units
            notional: Size in USD (alternative to size)
        """
        return self.sell(asset, size=size, notional=notional, tif="market")

    def order(self, order: Order) -> PlacedOrder:
        """
        Place an order using the fluent Order builder.

        Args:
            order: Order built with Order.buy() or Order.sell()

        Example:
            sdk.order(Order.buy("BTC").size(0.001).price(67000).gtc())
        """
        order.validate()

        # Handle notional orders
        if order._notional and not order._size:
            mid = self.get_mid(order.asset)
            if mid == 0:
                raise ValidationError(
                    f"Could not fetch price for {order.asset}",
                    guidance="Check the asset name or try again.",
                )
            size = round(order._notional / mid, 6)
            order._size = str(size)

        return self._execute_order(order)

    # ═══════════════════════════════════════════════════════════════════════════
    # POSITION MANAGEMENT
    # ═══════════════════════════════════════════════════════════════════════════

    def close_position(self, asset: str) -> PlacedOrder:
        """
        Close an open position completely.

        The API queries your position and builds the counter-order automatically.

        Args:
            asset: Asset to close ("BTC", "ETH")

        Returns:
            PlacedOrder for the closing trade

        Example:
            sdk.close_position("BTC")
        """
        action = {
            "type": "closePosition",
            "asset": asset,
            "user": self.address,
        }

        result = self._build_sign_send(action)

        # Build a pseudo PlacedOrder from the response
        order = Order.sell(asset)  # Direction determined by API
        order._size = "0"  # Will be filled from response
        return PlacedOrder.from_response(
            result.get("exchangeResponse", {}),
            order,
            sdk=self,
        )

    # ═══════════════════════════════════════════════════════════════════════════
    # ORDER MANAGEMENT
    # ═══════════════════════════════════════════════════════════════════════════

    def cancel(
        self,
        oid: int,
        asset: Optional[str] = None,
    ) -> dict:
        """
        Cancel an order by OID.

        Args:
            oid: Order ID to cancel
            asset: Asset (optional, but speeds up cancellation)

        Returns:
            Exchange response
        """
        cancel_action = {
            "type": "cancel",
            "cancels": [{"a": asset or 0, "o": oid}],
        }
        return self._build_sign_send(cancel_action)

    def cancel_all(self, asset: Optional[str] = None) -> dict:
        """
        Cancel all open orders.

        Args:
            asset: Only cancel orders for this asset (optional)

        Returns:
            Exchange response
        """
        orders = self.open_orders()

        if not orders["orders"]:
            return {"message": "No orders to cancel"}

        if asset:
            # Get cancel action for specific asset
            cancel_actions = orders.get("cancelActions", {}).get("byAsset", {})
            if asset not in cancel_actions:
                return {"message": f"No {asset} orders to cancel"}
            cancel_action = cancel_actions[asset]
        else:
            # Cancel all
            cancel_action = orders.get("cancelActions", {}).get("all")
            if not cancel_action:
                return {"message": "No orders to cancel"}

        return self._build_sign_send(cancel_action)

    def cancel_by_cloid(
        self,
        cloid: str,
        asset: str,
    ) -> dict:
        """
        Cancel an order by client order ID (cloid).

        Args:
            cloid: Client order ID (hex string, e.g., "0x...")
            asset: Asset name

        Returns:
            Exchange response
        """
        cancel_action = {
            "type": "cancelByCloid",
            "cancels": [{"asset": asset, "cloid": cloid}],
        }
        return self._build_sign_send(cancel_action)

    def schedule_cancel(
        self,
        time: Optional[int] = None,
    ) -> dict:
        """
        Schedule cancellation of all orders after a delay.

        This is a dead-man's switch — if you don't send another scheduleCancel
        before the time expires, all orders are cancelled.

        Args:
            time: Unix timestamp (ms) when to cancel. If None, cancels the scheduled cancel.

        Returns:
            Exchange response

        Example:
            # Cancel all orders in 60 seconds
            sdk.schedule_cancel(int(time.time() * 1000) + 60000)

            # Cancel the scheduled cancel
            sdk.schedule_cancel(None)
        """
        cancel_action: dict = {"type": "scheduleCancel"}
        if time is not None:
            cancel_action["time"] = time
        return self._build_sign_send(cancel_action)

    def modify(
        self,
        oid: int,
        asset: str,
        side: Union[str, Side],
        price: str,
        size: str,
        *,
        tif: Union[str, TIF] = TIF.GTC,
        reduce_only: bool = False,
    ) -> PlacedOrder:
        """
        Modify an existing order.

        Args:
            oid: Order ID to modify
            asset: Asset
            side: "buy" or "sell"
            price: New price
            size: New size
            tif: Time in force (default: GTC)
            reduce_only: Reduce only flag

        Returns:
            PlacedOrder with new details

        Raises:
            ValidationError: If inputs are invalid
        """
        # Input validation
        if not isinstance(oid, int) or oid <= 0:
            raise ValidationError(f"oid must be a positive integer, got: {oid}")
        if not asset or not isinstance(asset, str):
            raise ValidationError(f"asset must be a non-empty string, got: {asset}")

        if isinstance(side, Side):
            side = side.value
        if side not in ("buy", "sell"):
            raise ValidationError(f"side must be 'buy' or 'sell', got: {side}")

        if isinstance(tif, TIF):
            tif = tif.value
        if tif not in ("gtc", "ioc", "alo"):
            raise ValidationError(f"tif must be 'gtc', 'ioc', or 'alo', got: {tif}")

        # Validate price and size are convertible to numbers
        try:
            float(price)
        except (ValueError, TypeError):
            raise ValidationError(f"price must be a valid number, got: {price}")
        try:
            float(size)
        except (ValueError, TypeError):
            raise ValidationError(f"size must be a valid number, got: {size}")

        modify_action = {
            "type": "batchModify",
            "modifies": [{
                "oid": oid,
                "order": {
                    "a": asset,
                    "b": side == "buy",
                    "p": str(price),
                    "s": str(size),
                    "r": reduce_only,
                    "t": {"limit": {"tif": tif.capitalize()}},
                },
            }],
        }

        result = self._build_sign_send(modify_action)

        order = Order(asset=asset, side=Side.BUY if side == "buy" else Side.SELL)
        order._price = price
        order._size = size

        return PlacedOrder.from_response(
            result.get("exchangeResponse", {}),
            order,
            sdk=self,
        )

    # ═══════════════════════════════════════════════════════════════════════════
    # QUERIES
    # ═══════════════════════════════════════════════════════════════════════════

    def open_orders(self, user: Optional[str] = None) -> dict:
        """
        Get open orders with enriched info and pre-built cancel actions.

        Args:
            user: User address (default: self, requires private key)

        Returns:
            {
                "count": 5,
                "orders": [...],
                "cancelActions": {
                    "byAsset": {"BTC": {...}, "ETH": {...}},
                    "all": {...}
                }
            }
        """
        if user is None:
            self._require_wallet()
            user = self.address
        return self._post("/openOrders", {"user": user})

    def order_status(self, oid: int, user: Optional[str] = None) -> dict:
        """
        Get detailed status for an order.

        Args:
            oid: Order ID
            user: User address (default: self, requires private key)

        Returns:
            Order status with explanation
        """
        if user is None:
            self._require_wallet()
            user = self.address
        return self._post("/orderStatus", {"user": user, "oid": oid})

    def markets(self) -> dict:
        """
        Get all available markets.

        Returns:
            {
                "perps": [...],
                "spot": [...],
                "hip3": {...}  # Grouped by DEX
            }
        """
        return self._get("/markets")

    def dexes(self) -> dict:
        """Get all HIP-3 DEXes."""
        return self._get("/dexes")

    def preflight(
        self,
        asset: str,
        side: Union[str, Side],
        price: Union[float, int, str],
        size: Union[float, str],
        *,
        tif: Union[str, TIF] = TIF.GTC,
        reduce_only: bool = False,
    ) -> dict:
        """
        Validate an order before signing (preflight check).

        Catches invalid prices (wrong tick size) and sizes (too many decimals)
        BEFORE you sign, saving failed transactions.

        Args:
            asset: Asset name ("BTC", "ETH", "xyz:SILVER")
            side: "buy" or "sell"
            price: Order price
            size: Order size
            tif: Time in force (default: GTC)
            reduce_only: Reduce only flag

        Returns:
            {"valid": true} or {"valid": false, "error": "...", "suggestion": "..."}

        Example:
            result = sdk.preflight("BTC", "buy", price=67000.123456, size=0.001)
            if not result["valid"]:
                print(f"Invalid: {result['error']}")
                print(f"Try: {result['suggestion']}")
        """
        if isinstance(side, Side):
            side = side.value
        if isinstance(tif, TIF):
            tif = tif.value

        order = {
            "a": asset,
            "b": side == "buy",
            "p": str(price),
            "s": str(size),
            "r": reduce_only,
            "t": {"limit": {"tif": tif.capitalize()}},
        }

        return self._post("/preflight", {"action": {"type": "order", "orders": [order]}})

    def approval_status(self, user: Optional[str] = None) -> dict:
        """
        Check builder fee approval status.

        Args:
            user: User address (default: self, requires private key)

        Returns:
            {"approved": true, "maxFeeRate": "1%", ...}
        """
        if user is None:
            self._require_wallet()
            user = self.address
        return self._get("/approval", params={"user": user})

    def get_mid(self, asset: str) -> float:
        """
        Get current mid price for an asset.

        Args:
            asset: Asset name ("BTC", "ETH", "xyz:SILVER")

        Returns:
            Mid price as float
        """
        # HIP-3 markets need dex parameter
        if ":" in asset:
            dex = asset.split(":")[0]
            data = self._post_info({"type": "allMids", "dex": dex})
        else:
            data = self._post_info({"type": "allMids"})

        return float(data.get(asset, 0))

    def refresh_markets(self) -> dict:
        """Force refresh of market metadata cache."""
        self._markets_cache = self.markets()
        self._markets_cache_time = time_module.time()
        return self._markets_cache

    def _get_size_decimals(self, asset: str) -> int:
        """Get the maximum decimal places for order size on this market (cached)."""
        # Check cache first
        if asset in self._sz_decimals_cache:
            return self._sz_decimals_cache[asset]

        try:
            # Use cached markets or fetch fresh (with TTL)
            now = time_module.time()
            if self._markets_cache is None or (now - self._markets_cache_time) > self.CACHE_TTL:
                self._markets_cache = self.markets()
                self._markets_cache_time = now

            markets = self._markets_cache

            # Check perps
            for m in markets.get("perps", []):
                if m.get("name") == asset:
                    decimals = m.get("szDecimals", 5)
                    self._sz_decimals_cache[asset] = decimals
                    return decimals
            # Check spot
            for m in markets.get("spot", []):
                if m.get("name") == asset:
                    decimals = m.get("szDecimals", 5)
                    self._sz_decimals_cache[asset] = decimals
                    return decimals
            # Check HIP-3 markets
            for dex, dex_markets in markets.get("hip3", {}).items():
                for m in dex_markets:
                    if m.get("name") == asset:
                        decimals = m.get("szDecimals", 5)
                        self._sz_decimals_cache[asset] = decimals
                        return decimals
        except Exception:
            pass

        # Default to 5 decimals (safe for most markets)
        return 5

    # ═══════════════════════════════════════════════════════════════════════════
    # APPROVAL MANAGEMENT
    # ═══════════════════════════════════════════════════════════════════════════

    def approve_builder_fee(self, max_fee: str = "1%") -> dict:
        """
        Approve builder fee for trading.

        Args:
            max_fee: Maximum fee rate (e.g., "1%")

        Returns:
            Exchange response
        """
        action = {"type": "approveBuilderFee", "maxFeeRate": max_fee}
        return self._build_sign_send(action)

    def revoke_builder_fee(self) -> dict:
        """Revoke builder fee approval."""
        action = {"type": "approveBuilderFee", "maxFeeRate": "0%"}
        return self._build_sign_send(action)

    # ═══════════════════════════════════════════════════════════════════════════
    # INTERNAL METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def _place_order(
        self,
        asset: str,
        side: Side,
        size: Optional[Union[float, str]],
        notional: Optional[float],
        price: Optional[Union[float, int, str]],
        tif: Union[str, TIF],
        reduce_only: bool,
    ) -> PlacedOrder:
        """Internal order placement logic."""
        # Build order
        order = Order(asset=asset, side=side)

        # Handle size
        if notional:
            mid = self.get_mid(asset)
            if mid == 0:
                raise ValidationError(
                    f"Could not fetch price for {asset}",
                    guidance="Check the asset name or try again.",
                )
            # Get size decimals for this market (default 5 for most)
            sz_decimals = self._get_size_decimals(asset)
            size = round(notional / mid, sz_decimals)

        if size is None:
            raise ValidationError(
                "Either size or notional is required",
                guidance="Use size=0.001 or notional=100",
            )

        order._size = str(size)

        # Handle TIF
        if isinstance(tif, str):
            tif = TIF(tif.lower())
        order._tif = tif

        # Handle price (not needed for market orders)
        if tif != TIF.MARKET and price is not None:
            order._price = str(price)

        if reduce_only:
            order._reduce_only = True

        return self._execute_order(order)

    def _execute_order(self, order: Order) -> PlacedOrder:
        """Execute an order through build→sign→send."""
        action = order.to_action()
        result = self._build_sign_send(action)
        return PlacedOrder.from_response(
            result.get("exchangeResponse", {}),
            order,
            sdk=self,
        )

    def _require_wallet(self) -> None:
        """Ensure a wallet is configured for signing operations."""
        if self._wallet is None:
            raise ValueError(
                "Private key required for this operation. "
                "Pass private_key to HyperliquidSDK() or set PRIVATE_KEY env var."
            )

    def _build_sign_send(self, action: dict) -> dict:
        """
        The magic ceremony — build, sign, send in one call.

        1. Build: POST action to get hash
        2. Sign: Sign the hash locally
        3. Send: POST action + nonce + signature
        """
        self._require_wallet()

        # Step 1: Build
        build_result = self._exchange({"action": action})

        if "hash" not in build_result:
            raise BuildError(
                "Build response missing hash",
                raw=build_result,
            )

        # Step 2: Sign
        sig = self._sign_hash(build_result["hash"])

        # Step 3: Send
        send_payload = {
            "action": build_result.get("action", action),
            "nonce": build_result["nonce"],
            "signature": sig,
        }

        return self._exchange(send_payload)

    def _sign_hash(self, hash_hex: str) -> dict:
        """Sign a hash with the wallet."""
        hash_bytes = bytes.fromhex(hash_hex.removeprefix("0x"))
        signed = self._wallet.unsafe_sign_hash(hash_bytes)
        return {
            "r": f"0x{signed.r:064x}",
            "s": f"0x{signed.s:064x}",
            "v": signed.v,
        }

    def _exchange(self, body: dict) -> dict:
        """POST to /exchange endpoint."""
        try:
            resp = self._session.post(f"{self._api_url}/exchange", json=body, timeout=self._timeout)
        except requests.exceptions.Timeout:
            raise HyperliquidError(
                f"Exchange request timed out after {self._timeout}s",
                code="TIMEOUT",
            )
        except requests.exceptions.ConnectionError as e:
            raise HyperliquidError(f"Connection failed: {e}", code="CONNECTION_ERROR") from e

        try:
            data = resp.json()
        except ValueError:
            raise HyperliquidError("Invalid JSON response", code="PARSE_ERROR")

        if isinstance(data, dict) and data.get("error"):
            raise parse_api_error(data, resp.status_code)

        return data

    def _get(self, path: str, params: Optional[dict] = None) -> Any:
        """GET request to API."""
        try:
            resp = self._session.get(f"{self._api_url}{path}", params=params, timeout=self._timeout)
        except requests.exceptions.Timeout:
            raise HyperliquidError(
                f"Request timed out after {self._timeout}s",
                code="TIMEOUT",
            )
        except requests.exceptions.ConnectionError as e:
            raise HyperliquidError(f"Connection failed: {e}", code="CONNECTION_ERROR") from e

        try:
            data = resp.json()
        except ValueError:
            raise HyperliquidError("Invalid JSON response", code="PARSE_ERROR")

        if isinstance(data, dict) and data.get("error"):
            raise parse_api_error(data, resp.status_code)
        return data

    def _post(self, path: str, body: dict) -> Any:
        """POST request to API."""
        try:
            resp = self._session.post(f"{self._api_url}{path}", json=body, timeout=self._timeout)
        except requests.exceptions.Timeout:
            raise HyperliquidError(
                f"Request timed out after {self._timeout}s",
                code="TIMEOUT",
            )
        except requests.exceptions.ConnectionError as e:
            raise HyperliquidError(f"Connection failed: {e}", code="CONNECTION_ERROR") from e

        try:
            data = resp.json()
        except ValueError:
            raise HyperliquidError("Invalid JSON response", code="PARSE_ERROR")

        if isinstance(data, dict) and data.get("error"):
            raise parse_api_error(data, resp.status_code)
        return data

    def _post_info(self, body: dict) -> dict:
        """POST to Hyperliquid info API."""
        try:
            resp = self._session.post(self._info_url, json=body, timeout=self._timeout)
        except requests.exceptions.Timeout:
            raise HyperliquidError(
                f"Info request timed out after {self._timeout}s",
                code="TIMEOUT",
            )
        except requests.exceptions.ConnectionError as e:
            raise HyperliquidError(f"Connection failed: {e}", code="CONNECTION_ERROR") from e

        try:
            data = resp.json()
        except ValueError:
            raise HyperliquidError("Invalid JSON response", code="PARSE_ERROR")

        # Check for error responses (same pattern as _post)
        if isinstance(data, dict) and data.get("error"):
            raise parse_api_error(data, resp.status_code)
        return data

    def _ensure_approved(self, max_fee: str) -> None:
        """Ensure builder fee is approved."""
        status = self.approval_status()
        if not status.get("approved"):
            self.approve_builder_fee(max_fee)

    # ═══════════════════════════════════════════════════════════════════════════
    # CONTEXT MANAGER
    # ═══════════════════════════════════════════════════════════════════════════

    def __enter__(self) -> HyperliquidSDK:
        return self

    def __exit__(self, *args) -> None:
        self._session.close()

    def __repr__(self) -> str:
        if self.address:
            return f"<HyperliquidSDK {self.address[:10]}...>"
        return "<HyperliquidSDK (read-only)>"
