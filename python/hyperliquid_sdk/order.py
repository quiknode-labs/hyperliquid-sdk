"""
Order Builder — Fluent, type-safe, beautiful.

Build orders the way you think about them:

    Order.buy("BTC").size(0.001).price(67000).gtc()
    Order.sell("ETH").size(1.5).market()
    Order.buy("xyz:SILVER").notional(100).ioc()

Trigger Orders (Stop Loss / Take Profit):

    TriggerOrder.stop_loss("BTC").size(0.001).trigger_price(60000).market()
    TriggerOrder.take_profit("ETH").size(1.0).trigger_price(5000).limit(4990)
"""

from __future__ import annotations
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional, Union, TYPE_CHECKING
from decimal import Decimal

if TYPE_CHECKING:
    from .client import HyperliquidSDK


class Side(Enum):
    """Order side."""
    BUY = "buy"
    SELL = "sell"

    # Aliases for traders
    LONG = "buy"
    SHORT = "sell"


class TIF(Enum):
    """Time in force."""
    IOC = "ioc"      # Immediate or cancel
    GTC = "gtc"      # Good till cancelled
    ALO = "alo"      # Add liquidity only (post-only)
    MARKET = "market"  # Market order (auto-price)


class TpSl(Enum):
    """Trigger order type (take profit or stop loss)."""
    TP = "tp"  # Take profit
    SL = "sl"  # Stop loss


class OrderGrouping(Enum):
    """
    Order grouping for TP/SL attachment.

    - NA: No grouping (standalone order)
    - NORMAL_TPSL: Attach TP/SL to the fill of this order
    - POSITION_TPSL: Attach TP/SL to the entire position
    """
    NA = "na"
    NORMAL_TPSL = "normalTpsl"
    POSITION_TPSL = "positionTpsl"


@dataclass
class Order:
    """
    Fluent order builder.

    Examples:
        # Simple limit buy
        Order.buy("BTC").size(0.001).price(67000)

        # Market sell by notional
        Order.sell("ETH").notional(500).market()

        # Post-only with reduce_only
        Order.buy("BTC").size(0.01).price(65000).alo().reduce_only()
    """

    asset: str
    side: Side
    _size: Optional[str] = None
    _price: Optional[str] = None
    _tif: TIF = TIF.IOC
    _reduce_only: bool = False
    _notional: Optional[float] = None
    _cloid: Optional[str] = None

    # ═══════════════ STATIC CONSTRUCTORS ═══════════════

    @classmethod
    def buy(cls, asset: str) -> Order:
        """Create a buy order."""
        return cls(asset=asset, side=Side.BUY)

    @classmethod
    def sell(cls, asset: str) -> Order:
        """Create a sell order."""
        return cls(asset=asset, side=Side.SELL)

    @classmethod
    def long(cls, asset: str) -> Order:
        """Alias for buy (perps terminology)."""
        return cls.buy(asset)

    @classmethod
    def short(cls, asset: str) -> Order:
        """Alias for sell (perps terminology)."""
        return cls.sell(asset)

    # ═══════════════ SIZE ═══════════════

    def size(self, size: Union[float, str, Decimal]) -> Order:
        """Set order size in asset units."""
        self._size = str(size)
        return self

    def notional(self, usd: float) -> Order:
        """
        Set order size by USD notional value.
        SDK will calculate size based on current price.
        """
        self._notional = usd
        return self

    # ═══════════════ PRICE ═══════════════

    def price(self, price: Union[float, int, str, Decimal]) -> Order:
        """Set limit price."""
        self._price = str(price)
        return self

    def limit(self, price: Union[float, int, str, Decimal]) -> Order:
        """Alias for price()."""
        return self.price(price)

    # ═══════════════ TIME IN FORCE ═══════════════

    def tif(self, tif: Union[TIF, str]) -> Order:
        """Set time in force."""
        if isinstance(tif, str):
            tif = TIF(tif.lower())
        self._tif = tif
        return self

    def ioc(self) -> Order:
        """Immediate or cancel."""
        self._tif = TIF.IOC
        return self

    def gtc(self) -> Order:
        """Good till cancelled (resting order)."""
        self._tif = TIF.GTC
        return self

    def alo(self) -> Order:
        """Add liquidity only (post-only, maker only)."""
        self._tif = TIF.ALO
        return self

    def market(self) -> Order:
        """Market order (price computed automatically with slippage)."""
        self._tif = TIF.MARKET
        self._price = None  # Price will be computed by API
        return self

    # ═══════════════ OPTIONS ═══════════════

    def reduce_only(self, value: bool = True) -> Order:
        """Mark as reduce-only (close position only, no new exposure)."""
        self._reduce_only = value
        return self

    def cloid(self, client_order_id: str) -> Order:
        """Set client order ID for tracking."""
        self._cloid = client_order_id
        return self

    # ═══════════════ BUILD ACTION ═══════════════

    def to_action(self) -> dict:
        """Convert to API action format."""
        order_spec = {
            "asset": self.asset,
            "side": self.side.value,
            "size": self._size,
        }

        # Price (omit for market orders)
        if self._tif == TIF.MARKET:
            order_spec["tif"] = "market"
        else:
            if self._price:
                order_spec["price"] = self._price
            order_spec["tif"] = self._tif.value

        # Optional fields
        if self._reduce_only:
            order_spec["reduceOnly"] = True

        if self._cloid:
            order_spec["cloid"] = self._cloid

        return {
            "type": "order",
            "orders": [order_spec],
        }

    # ═══════════════ VALIDATION ═══════════════

    def validate(self) -> None:
        """Validate order before sending."""
        from .errors import ValidationError

        if not self.asset:
            raise ValidationError("Asset is required")

        if self._size is None and self._notional is None:
            raise ValidationError(
                "Either size or notional is required",
                guidance="Use .size(0.001) or .notional(100)",
            )

        # Validate size is positive
        if self._size is not None:
            try:
                size_val = float(self._size)
                if size_val <= 0:
                    raise ValidationError(
                        "Size must be positive",
                        guidance=f"Got size={self._size}, use a positive value like 0.001",
                    )
            except ValueError:
                raise ValidationError(
                    f"Invalid size value: {self._size}",
                    guidance="Size must be a valid number",
                )

        # Validate notional is positive
        if self._notional is not None and self._notional <= 0:
            raise ValidationError(
                "Notional must be positive",
                guidance=f"Got notional={self._notional}, use a positive value like 100",
            )

        # Validate price for limit orders
        if self._tif != TIF.MARKET and self._price is None and self._notional is None:
            raise ValidationError(
                "Price is required for limit orders",
                guidance="Use .price(67000) or .market() for market orders",
            )

        # Validate price is positive for limit orders
        if self._price is not None:
            try:
                price_val = float(self._price)
                if price_val <= 0:
                    raise ValidationError(
                        "Price must be positive",
                        guidance=f"Got price={self._price}, use a positive value like 67000",
                    )
            except ValueError:
                raise ValidationError(
                    f"Invalid price value: {self._price}",
                    guidance="Price must be a valid number",
                )

    # ═══════════════ REPR ═══════════════

    def __repr__(self) -> str:
        parts = [f"Order.{self.side.value}('{self.asset}')"]
        if self._size:
            parts.append(f".size({self._size})")
        if self._notional:
            parts.append(f".notional({self._notional})")
        if self._price:
            parts.append(f".price({self._price})")
        if self._tif != TIF.IOC:
            parts.append(f".{self._tif.name.lower()}()")
        if self._reduce_only:
            parts.append(".reduce_only()")
        return "".join(parts)


@dataclass
class TriggerOrder:
    """
    Trigger order builder for stop-loss and take-profit orders.

    A trigger order activates when the market price reaches the trigger price:
    - Stop Loss (SL): Triggers when price moves AGAINST your position
    - Take Profit (TP): Triggers when price moves IN FAVOR of your position

    Once triggered, the order executes as either:
    - Market order: Executes immediately at best available price
    - Limit order: Rests at the specified limit price

    Examples:
        # Stop loss: sell when price drops to 60000 (market)
        TriggerOrder.stop_loss("BTC").size(0.001).trigger_price(60000).market()

        # Stop loss: sell at limit 59900 when price drops to 60000
        TriggerOrder.stop_loss("BTC").size(0.001).trigger_price(60000).limit(59900)

        # Take profit: sell when price rises to 80000 (market)
        TriggerOrder.take_profit("BTC").size(0.001).trigger_price(80000).market()

        # Take profit with limit
        TriggerOrder.take_profit("ETH").size(1.0).trigger_price(5000).limit(4990)

        # Buy stop (short squeeze protection)
        TriggerOrder.stop_loss("BTC", side=Side.BUY).size(0.001).trigger_price(75000).market()
    """

    asset: str
    tpsl: TpSl
    side: Side
    _size: Optional[str] = None
    _trigger_px: Optional[str] = None
    _limit_px: Optional[str] = None  # None = market order
    _is_market: bool = True
    _reduce_only: bool = False
    _cloid: Optional[str] = None

    # ═══════════════ STATIC CONSTRUCTORS ═══════════════

    @classmethod
    def stop_loss(cls, asset: str, *, side: Side = Side.SELL) -> TriggerOrder:
        """
        Create a stop-loss trigger order.

        Stop loss triggers when price moves against your position:
        - For longs: triggers when price FALLS to trigger_price (sell to exit)
        - For shorts: triggers when price RISES to trigger_price (buy to exit)

        Args:
            asset: Asset name ("BTC", "ETH")
            side: Order side when triggered (default: SELL for closing longs)
        """
        return cls(asset=asset, tpsl=TpSl.SL, side=side)

    @classmethod
    def take_profit(cls, asset: str, *, side: Side = Side.SELL) -> TriggerOrder:
        """
        Create a take-profit trigger order.

        Take profit triggers when price moves in favor of your position:
        - For longs: triggers when price RISES to trigger_price (sell to take profits)
        - For shorts: triggers when price FALLS to trigger_price (buy to take profits)

        Args:
            asset: Asset name ("BTC", "ETH")
            side: Order side when triggered (default: SELL for closing longs)
        """
        return cls(asset=asset, tpsl=TpSl.TP, side=side)

    # Aliases
    @classmethod
    def sl(cls, asset: str, *, side: Side = Side.SELL) -> TriggerOrder:
        """Alias for stop_loss()."""
        return cls.stop_loss(asset, side=side)

    @classmethod
    def tp(cls, asset: str, *, side: Side = Side.SELL) -> TriggerOrder:
        """Alias for take_profit()."""
        return cls.take_profit(asset, side=side)

    # ═══════════════ SIZE ═══════════════

    def size(self, size: Union[float, str, Decimal]) -> TriggerOrder:
        """Set order size in asset units."""
        self._size = str(size)
        return self

    # ═══════════════ TRIGGER PRICE ═══════════════

    def trigger_price(self, price: Union[float, int, str, Decimal]) -> TriggerOrder:
        """
        Set the trigger price.

        The order activates when the market price reaches this level.
        """
        self._trigger_px = str(price)
        return self

    def trigger(self, price: Union[float, int, str, Decimal]) -> TriggerOrder:
        """Alias for trigger_price()."""
        return self.trigger_price(price)

    # ═══════════════ ORDER TYPE ═══════════════

    def market(self) -> TriggerOrder:
        """
        Execute as market order when triggered.

        The order will fill immediately at the best available price.
        """
        self._is_market = True
        self._limit_px = None
        return self

    def limit(self, price: Union[float, int, str, Decimal]) -> TriggerOrder:
        """
        Execute as limit order when triggered.

        The order will rest at the specified limit price.

        Args:
            price: Limit price for the triggered order
        """
        self._is_market = False
        self._limit_px = str(price)
        return self

    # ═══════════════ OPTIONS ═══════════════

    def reduce_only(self, value: bool = True) -> TriggerOrder:
        """Mark as reduce-only (close position only)."""
        self._reduce_only = value
        return self

    def cloid(self, client_order_id: str) -> TriggerOrder:
        """Set client order ID for tracking."""
        self._cloid = client_order_id
        return self

    # ═══════════════ BUILD ACTION ═══════════════

    def to_action(self, grouping: OrderGrouping = OrderGrouping.NA) -> dict:
        """
        Convert to API action format.

        Args:
            grouping: Order grouping for TP/SL attachment
        """
        # For trigger orders, limit_px is always required by the API
        # For market orders, we use trigger_px as a placeholder
        limit_px = self._limit_px if not self._is_market else self._trigger_px

        order_spec = {
            "a": self.asset,  # Asset
            "b": self.side == Side.BUY,  # is_buy
            "p": limit_px,  # limit_px (required, use trigger for market)
            "s": self._size,  # size
            "r": self._reduce_only,  # reduce_only
            "t": {
                "trigger": {
                    "isMarket": self._is_market,
                    "triggerPx": self._trigger_px,
                    "tpsl": self.tpsl.value,
                }
            },
        }

        if self._cloid:
            order_spec["c"] = self._cloid

        return {
            "type": "order",
            "orders": [order_spec],
            "grouping": grouping.value,
        }

    # ═══════════════ VALIDATION ═══════════════

    def validate(self) -> None:
        """Validate trigger order before sending."""
        from .errors import ValidationError

        if not self.asset:
            raise ValidationError("Asset is required")

        if self._size is None:
            raise ValidationError(
                "Size is required for trigger orders",
                guidance="Use .size(0.001) to set the order size",
            )

        # Validate size is positive
        try:
            size_val = float(self._size)
            if size_val <= 0:
                raise ValidationError(
                    "Size must be positive",
                    guidance=f"Got size={self._size}, use a positive value",
                )
        except ValueError:
            raise ValidationError(f"Invalid size value: {self._size}")

        if self._trigger_px is None:
            raise ValidationError(
                "Trigger price is required",
                guidance="Use .trigger_price(60000) to set when the order activates",
            )

        # Validate trigger price is positive
        try:
            trigger_val = float(self._trigger_px)
            if trigger_val <= 0:
                raise ValidationError(
                    "Trigger price must be positive",
                    guidance=f"Got trigger_price={self._trigger_px}",
                )
        except ValueError:
            raise ValidationError(f"Invalid trigger price: {self._trigger_px}")

        # Validate limit price for limit orders
        if not self._is_market:
            if self._limit_px is None:
                raise ValidationError(
                    "Limit price is required for limit trigger orders",
                    guidance="Use .limit(59900) or .market() for market execution",
                )
            try:
                limit_val = float(self._limit_px)
                if limit_val <= 0:
                    raise ValidationError(
                        "Limit price must be positive",
                        guidance=f"Got limit={self._limit_px}",
                    )
            except ValueError:
                raise ValidationError(f"Invalid limit price: {self._limit_px}")

    # ═══════════════ REPR ═══════════════

    def __repr__(self) -> str:
        name = "stop_loss" if self.tpsl == TpSl.SL else "take_profit"
        parts = [f"TriggerOrder.{name}('{self.asset}')"]
        if self._size:
            parts.append(f".size({self._size})")
        if self._trigger_px:
            parts.append(f".trigger_price({self._trigger_px})")
        if self._is_market:
            parts.append(".market()")
        elif self._limit_px:
            parts.append(f".limit({self._limit_px})")
        if self._reduce_only:
            parts.append(".reduce_only()")
        return "".join(parts)


@dataclass
class PlacedOrder:
    """
    A successfully placed order with full context.

    Returned from SDK order methods, provides:
    - Order ID and status
    - Methods to modify/cancel
    - Original order details
    """

    oid: Optional[int]
    status: str
    asset: str
    side: str
    size: str
    price: Optional[str]
    filled_size: Optional[str] = None
    avg_price: Optional[str] = None
    raw_response: dict = field(default_factory=dict)
    _sdk: Optional[HyperliquidSDK] = field(default=None, repr=False)

    @classmethod
    def from_response(
        cls,
        response: dict,
        order: Order,
        sdk: Optional[HyperliquidSDK] = None,
    ) -> PlacedOrder:
        """Parse exchange response into PlacedOrder."""
        statuses = (
            response.get("response", {})
            .get("data", {})
            .get("statuses", [])
        )

        oid = None
        status = "unknown"
        filled_size = None
        avg_price = None

        if statuses:
            s = statuses[0]
            if isinstance(s, dict):
                if "resting" in s:
                    oid = s["resting"].get("oid")
                    status = "resting"
                elif "filled" in s:
                    filled = s["filled"]
                    oid = filled.get("oid")
                    status = "filled"
                    filled_size = filled.get("totalSz")
                    avg_price = filled.get("avgPx")
                elif "error" in s:
                    status = f"error: {s['error']}"
            elif s == "success":
                status = "success"

        return cls(
            oid=oid,
            status=status,
            asset=order.asset,
            side=order.side.value,
            size=order._size or "",
            price=order._price,
            filled_size=filled_size,
            avg_price=avg_price,
            raw_response=response,
            _sdk=sdk,
        )

    # ═══════════════ ORDER ACTIONS ═══════════════

    def cancel(self) -> dict:
        """Cancel this order."""
        if not self._sdk:
            raise RuntimeError("Order not linked to SDK")
        if not self.oid:
            raise RuntimeError("No OID to cancel")
        return self._sdk.cancel(self.oid, self.asset)

    def modify(
        self,
        price: Optional[Union[float, str]] = None,
        size: Optional[Union[float, str]] = None,
    ) -> PlacedOrder:
        """Modify this order's price and/or size."""
        if not self._sdk:
            raise RuntimeError("Order not linked to SDK")
        if not self.oid:
            raise RuntimeError("No OID to modify")

        return self._sdk.modify(
            oid=self.oid,
            asset=self.asset,
            side=self.side,
            price=str(price) if price else self.price,
            size=str(size) if size else self.size,
        )

    # ═══════════════ STATUS ═══════════════

    @property
    def is_resting(self) -> bool:
        return self.status == "resting"

    @property
    def is_filled(self) -> bool:
        return self.status == "filled"

    @property
    def is_error(self) -> bool:
        return self.status.startswith("error")

    def __repr__(self) -> str:
        if self.oid:
            return f"<PlacedOrder {self.side.upper()} {self.size} {self.asset} @ {self.price} | {self.status} (oid={self.oid})>"
        return f"<PlacedOrder {self.side.upper()} {self.size} {self.asset} | {self.status}>"
