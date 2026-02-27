"""
Order Builder — Fluent, type-safe, beautiful.

Build orders the way you think about them:

    Order.buy("BTC").size(0.001).price(67000).gtc()
    Order.sell("ETH").size(1.5).market()
    Order.buy("xyz:SILVER").notional(100).ioc()
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
