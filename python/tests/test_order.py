"""Tests for the Order builder."""

import pytest
from hyperliquid_sdk.order import Order, Side, TIF, PlacedOrder
from hyperliquid_sdk.errors import ValidationError


class TestOrderBuilder:
    """Test the fluent order builder."""

    def test_buy_order_basic(self):
        """Test creating a basic buy order."""
        order = Order.buy("BTC").size(0.001).price(67000).gtc()
        assert order.asset == "BTC"
        assert order.side == Side.BUY
        assert order._size == "0.001"
        assert order._price == "67000"
        assert order._tif == TIF.GTC

    def test_sell_order_basic(self):
        """Test creating a basic sell order."""
        order = Order.sell("ETH").size(1.5).price(3500).ioc()
        assert order.asset == "ETH"
        assert order.side == Side.SELL
        assert order._size == "1.5"
        assert order._price == "3500"
        assert order._tif == TIF.IOC

    def test_long_short_aliases(self):
        """Test long/short aliases for buy/sell."""
        long_order = Order.long("BTC").size(1)
        short_order = Order.short("BTC").size(1)
        assert long_order.side == Side.BUY
        assert short_order.side == Side.SELL

    def test_market_order(self):
        """Test market order creation."""
        order = Order.buy("BTC").size(0.01).market()
        assert order._tif == TIF.MARKET
        assert order._price is None

    def test_alo_order(self):
        """Test add-liquidity-only order."""
        order = Order.buy("BTC").size(0.01).price(65000).alo()
        assert order._tif == TIF.ALO

    def test_notional_order(self):
        """Test order with notional value."""
        order = Order.buy("BTC").notional(1000)
        assert order._notional == 1000
        assert order._size is None

    def test_reduce_only(self):
        """Test reduce-only flag."""
        order = Order.sell("BTC").size(0.01).price(70000).reduce_only()
        assert order._reduce_only is True

    def test_cloid(self):
        """Test client order ID."""
        order = Order.buy("BTC").size(0.01).price(67000).cloid("my-order-123")
        assert order._cloid == "my-order-123"

    def test_to_action(self):
        """Test converting order to API action format."""
        order = Order.buy("BTC").size(0.001).price(67000).gtc()
        action = order.to_action()
        
        assert action["type"] == "order"
        assert len(action["orders"]) == 1
        order_spec = action["orders"][0]
        assert order_spec["asset"] == "BTC"
        assert order_spec["side"] == "buy"
        assert order_spec["size"] == "0.001"
        assert order_spec["price"] == "67000"
        assert order_spec["tif"] == "gtc"

    def test_market_order_action(self):
        """Test market order action format."""
        order = Order.buy("BTC").size(0.01).market()
        action = order.to_action()
        
        order_spec = action["orders"][0]
        assert order_spec["tif"] == "market"
        assert "price" not in order_spec

    def test_reduce_only_action(self):
        """Test reduce-only in action."""
        order = Order.sell("BTC").size(0.01).price(70000).reduce_only()
        action = order.to_action()
        
        assert action["orders"][0]["reduceOnly"] is True

    def test_repr(self):
        """Test string representation."""
        order = Order.buy("BTC").size(0.001).price(67000).gtc()
        repr_str = repr(order)
        assert "BTC" in repr_str
        assert "buy" in repr_str
        assert "0.001" in repr_str


class TestOrderValidation:
    """Test order validation."""

    def test_missing_asset(self):
        """Test validation fails without asset."""
        order = Order(asset="", side=Side.BUY)
        order._size = "1"
        order._price = "100"
        with pytest.raises(ValidationError, match="Asset is required"):
            order.validate()

    def test_missing_size_and_notional(self):
        """Test validation fails without size or notional."""
        order = Order.buy("BTC").price(67000)
        with pytest.raises(ValidationError, match="Either size or notional"):
            order.validate()

    def test_negative_size(self):
        """Test validation fails with negative size."""
        order = Order.buy("BTC").size(-0.001).price(67000)
        with pytest.raises(ValidationError, match="Size must be positive"):
            order.validate()

    def test_zero_size(self):
        """Test validation fails with zero size."""
        order = Order.buy("BTC").size(0).price(67000)
        with pytest.raises(ValidationError, match="Size must be positive"):
            order.validate()

    def test_negative_notional(self):
        """Test validation fails with negative notional."""
        order = Order.buy("BTC").notional(-100)
        with pytest.raises(ValidationError, match="Notional must be positive"):
            order.validate()

    def test_missing_price_limit_order(self):
        """Test validation fails for limit order without price."""
        order = Order.buy("BTC").size(0.001).gtc()
        with pytest.raises(ValidationError, match="Price is required"):
            order.validate()

    def test_negative_price(self):
        """Test validation fails with negative price."""
        order = Order.buy("BTC").size(0.001).price(-100)
        with pytest.raises(ValidationError, match="Price must be positive"):
            order.validate()

    def test_market_order_no_price_required(self):
        """Test market orders don't require price."""
        order = Order.buy("BTC").size(0.001).market()
        order.validate()  # Should not raise

    def test_notional_no_price_required(self):
        """Test notional orders don't require price for limit."""
        order = Order.buy("BTC").notional(100).gtc()
        order.validate()  # Should not raise (price derived from notional)

    def test_valid_order(self):
        """Test valid order passes validation."""
        order = Order.buy("BTC").size(0.001).price(67000).gtc()
        order.validate()  # Should not raise


class TestPlacedOrder:
    """Test PlacedOrder response parsing."""

    def test_from_response_resting(self):
        """Test parsing a resting order response."""
        response = {
            "status": "ok",
            "response": {
                "type": "order",
                "data": {
                    "statuses": [{"resting": {"oid": 12345}}]
                }
            }
        }
        order = Order.buy("BTC").size(0.001).price(67000)
        placed = PlacedOrder.from_response(response, order)
        
        assert placed.oid == 12345
        assert placed.status == "resting"
        assert placed.asset == "BTC"
        assert placed.side == "buy"
        assert placed.is_resting
        assert not placed.is_filled

    def test_from_response_filled(self):
        """Test parsing a filled order response."""
        response = {
            "status": "ok",
            "response": {
                "type": "order",
                "data": {
                    "statuses": [{
                        "filled": {
                            "oid": 12345,
                            "totalSz": "0.001",
                            "avgPx": "67000.5"
                        }
                    }]
                }
            }
        }
        order = Order.buy("BTC").size(0.001).price(67000)
        placed = PlacedOrder.from_response(response, order)
        
        assert placed.oid == 12345
        assert placed.status == "filled"
        assert placed.filled_size == "0.001"
        assert placed.avg_price == "67000.5"
        assert placed.is_filled
        assert not placed.is_resting

    def test_from_response_error(self):
        """Test parsing an error response."""
        response = {
            "status": "ok",
            "response": {
                "type": "order",
                "data": {
                    "statuses": [{"error": "Insufficient margin"}]
                }
            }
        }
        order = Order.buy("BTC").size(0.001).price(67000)
        placed = PlacedOrder.from_response(response, order)
        
        assert placed.status == "error: Insufficient margin"
        assert placed.is_error

    def test_repr(self):
        """Test string representation of PlacedOrder."""
        placed = PlacedOrder(
            oid=12345,
            status="resting",
            asset="BTC",
            side="buy",
            size="0.001",
            price="67000",
        )
        repr_str = repr(placed)
        assert "BUY" in repr_str
        assert "BTC" in repr_str
        assert "12345" in repr_str
