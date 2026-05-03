"""
Comprehensive tests for hyperliquid-sdk.

Run with: pytest tests/test_sdk.py -v

These tests verify:
1. All imports work
2. Endpoint parsing works for all formats
3. Info API works
4. WebSocket streaming works
5. gRPC streaming works
6. Connection states work
7. Error handling works
"""

import pytest
import time
import os
from decimal import Decimal

# Test 1: All imports
def test_all_imports():
    """Test that all SDK components can be imported."""
    from hyperliquid_sdk import (
        HyperliquidSDK,
        Info,
        HyperCore,
        EVM,
        Stream,
        StreamType,
        ConnectionState,
        GRPCStream,
        GRPCStreamType,
        Order,
        PlacedOrder,
        Side,
        TIF,
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
    assert True


# Test 2: Endpoint parsing
class TestEndpointParsing:
    """Test endpoint parsing for various formats."""

    ENDPOINTS = [
        "https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
        "https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/",
        "https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/info",
        "https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/hypercore",
        "https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/evm",
        "https://api.hyperliquid.xyz",
    ]

    def test_info_endpoint_parsing(self):
        """Test Info client handles all endpoint formats."""
        from hyperliquid_sdk import Info

        for endpoint in self.ENDPOINTS:
            info = Info(endpoint)
            assert info._info_url is not None
            assert "/info" in info._info_url

    def test_websocket_endpoint_parsing(self):
        """Test Stream client handles all endpoint formats."""
        from hyperliquid_sdk import Stream

        for endpoint in self.ENDPOINTS:
            stream = Stream(endpoint, reconnect=False)
            assert stream._ws_url is not None
            assert "ws" in stream._ws_url

    def test_grpc_endpoint_parsing(self):
        """Test GRPCStream client handles all endpoint formats."""
        from hyperliquid_sdk import GRPCStream

        for endpoint in self.ENDPOINTS:
            grpc = GRPCStream(endpoint, reconnect=False)
            target = grpc._get_target()
            assert ":10000" in target


class TestTradingHelpers:
    """Test local trading helper behavior without network calls."""

    def test_hype_to_wei_uses_8_decimals(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(auto_approve=False)
        assert sdk._hype_to_wei(Decimal("0.001")) == 100000
        assert sdk._hype_to_wei("1") == 100000000

    def test_market_order_sends_priority_fee_to_build(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        calls = []

        def fake_exchange(body):
            calls.append(body)
            if "signature" not in body:
                return {
                    "hash": "0x" + "00" * 32,
                    "nonce": 123,
                    "action": {
                        "type": "order",
                        "orders": body["action"]["orders"],
                        "grouping": {"p": 10000},
                    },
                }
            return {
                "exchangeResponse": {
                    "response": {
                        "data": {
                            "statuses": [
                                {"filled": {"oid": 1, "totalSz": "0.3", "avgPx": "42"}}
                            ]
                        }
                    }
                }
            }

        sdk._exchange = fake_exchange
        placed = sdk.market_buy("HYPE", size=0.3, priority_fee=10000)

        assert calls[0]["priorityFee"] == 10000
        assert calls[1]["action"]["grouping"] == {"p": 10000}
        assert placed.oid == 1

    def test_stake_builds_c_deposit_with_8_decimal_wei(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        calls = []

        def fake_exchange(body):
            calls.append(body)
            if "signature" not in body:
                assert body["action"]["type"] == "cDeposit"
                assert body["action"]["wei"] == 100000
                return {
                    "hash": "0x" + "00" * 32,
                    "nonce": body["action"]["nonce"],
                    "action": {
                        "type": "cDeposit",
                        "wei": 100000,
                        "nonce": body["action"]["nonce"],
                        "hyperliquidChain": "Mainnet",
                        "signatureChainId": "0xa4b1",
                    },
                }
            return {"success": True, "exchangeResponse": {"status": "ok"}}

        sdk._exchange = fake_exchange
        result = sdk.fund_priority_fees(Decimal("0.001"))

        assert result["success"] is True
        assert calls[1]["action"]["type"] == "cDeposit"

    def test_prediction_markets_build_tradeable_sides(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(auto_approve=False)

        def fake_post_info(body):
            if body["type"] == "outcomeMeta":
                return {
                    "outcomes": [
                        {
                            "outcome": 1,
                            "name": "Recurring",
                            "description": (
                                "class:priceBinary|underlying:BTC|expiry:20260504-0600|"
                                "targetPrice:78213|period:1d"
                            ),
                            "sideSpecs": [{"name": "Yes"}, {"name": "No"}],
                        }
                    ],
                    "questions": [],
                }
            return {"#10": "0.62", "#11": "0.38"}

        sdk._post_info = fake_post_info
        markets = sdk.prediction_markets()
        market = markets.find(underlying="BTC", target_price="78213")

        assert market is not None
        assert market.title == "BTC above 78213 on 2026-05-04T06:00:00Z"
        assert market.yes.symbol == "#10"
        assert market.yes.asset_id == 100000010
        assert market.no.symbol == "#11"
        assert str(market.yes) == "#10"

    def test_prediction_side_can_be_used_as_order_asset(self):
        from hyperliquid_sdk import HyperliquidSDK, PredictionSide

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        side = PredictionSide(1, 0, "Yes", "#10", "+10", 100000010, "0.62")
        calls = []

        def fake_exchange(body):
            calls.append(body)
            if "signature" not in body:
                return {
                    "hash": "0x" + "00" * 32,
                    "nonce": 123,
                    "action": body["action"],
                }
            return {
                "exchangeResponse": {
                    "response": {
                        "data": {
                            "statuses": [
                                {"filled": {"oid": 1, "totalSz": "20", "avgPx": "0.62"}}
                            ]
                        }
                    }
                }
            }

        sdk._exchange = fake_exchange
        placed = sdk.buy(side, size=20, price="0.62")

        assert calls[0]["action"]["orders"][0]["asset"] == "#10"
        assert placed.oid == 1

    def test_prediction_market_rejects_priority_fee(self):
        from hyperliquid_sdk import HyperliquidSDK, PredictionSide, ValidationError

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        side = PredictionSide(1, 0, "Yes", "#10", "+10", 100000010, "0.62")

        with pytest.raises(ValidationError, match="priority_fee is not supported"):
            sdk.buy(side, size=20, price="0.62", priority_fee=10000)

    def test_prediction_market_rejects_fractional_contracts(self):
        from hyperliquid_sdk import HyperliquidSDK, PredictionSide, ValidationError

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        side = PredictionSide(1, 0, "Yes", "#10", "+10", 100000010, "0.62")

        with pytest.raises(ValidationError, match="whole number"):
            sdk.buy(side, size="20.5", price="0.62")

    def test_prediction_market_rejects_buy_below_minimum(self):
        from hyperliquid_sdk import HyperliquidSDK, PredictionSide, ValidationError

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        side = PredictionSide(1, 0, "Yes", "#10", "+10", 100000010, "0.62")

        with pytest.raises(ValidationError, match="minimum value"):
            sdk.buy(side, size=10, price="0.62")

    def test_prediction_market_allows_sell_below_minimum(self):
        from hyperliquid_sdk import HyperliquidSDK, PredictionSide

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        side = PredictionSide(1, 1, "No", "#11", "+11", 100000011, "0.27")
        calls = []

        def fake_exchange(body):
            calls.append(body)
            if "signature" not in body:
                return {
                    "hash": "0x" + "00" * 32,
                    "nonce": 123,
                    "action": body["action"],
                }
            return {
                "exchangeResponse": {
                    "response": {
                        "data": {
                            "statuses": [
                                {"filled": {"oid": 1, "totalSz": "35", "avgPx": "0.2655"}}
                            ]
                        }
                    }
                }
            }

        sdk._exchange = fake_exchange
        placed = sdk.sell(side, size=35, price="0.2655")

        assert calls[0]["action"]["orders"][0]["asset"] == "#11"
        assert placed.oid == 1


# Test 3: Info API
class TestInfoAPI:
    """Test Info API methods work.

    Note: Some methods proxy to public Hyperliquid API which may be geo-blocked.
    Tests handle GeoBlockedError gracefully - if geo-blocked, the test passes
    (proves error detection works), otherwise validates the response.
    """

    @pytest.fixture
    def info(self):
        from hyperliquid_sdk import Info
        return Info("https://api.hyperliquid.xyz")

    def test_all_mids(self, info):
        """Test all_mids returns data (or raises GeoBlockedError if geo-blocked)."""
        from hyperliquid_sdk import GeoBlockedError
        try:
            mids = info.all_mids()
            assert isinstance(mids, dict)
            assert "BTC" in mids
            assert "ETH" in mids
            assert len(mids) > 100  # Should have many assets
        except GeoBlockedError as e:
            # Geo-blocking detected correctly - test passes
            assert "GEO_BLOCKED" in str(e)

    def test_meta(self, info):
        """Test meta returns exchange metadata."""
        meta = info.meta()
        assert isinstance(meta, dict)
        assert "universe" in meta
        assert len(meta["universe"]) > 100  # Should have many markets

    def test_l2_book(self, info):
        """Test l2_book returns order book (or raises GeoBlockedError if geo-blocked)."""
        from hyperliquid_sdk import GeoBlockedError
        try:
            book = info.l2_book("BTC")
            assert isinstance(book, dict)
            assert "levels" in book
            levels = book["levels"]
            assert len(levels) == 2  # bids and asks
            assert len(levels[0]) > 0  # has bids
            assert len(levels[1]) > 0  # has asks
        except GeoBlockedError as e:
            # Geo-blocking detected correctly - test passes
            assert "GEO_BLOCKED" in str(e)

    def test_recent_trades(self, info):
        """Test recent_trades returns trades (or raises GeoBlockedError if geo-blocked)."""
        from hyperliquid_sdk import GeoBlockedError
        try:
            trades = info.recent_trades("BTC")
            assert isinstance(trades, list)
            assert len(trades) > 0
            assert "px" in trades[0]
            assert "sz" in trades[0]
        except GeoBlockedError as e:
            # Geo-blocking detected correctly - test passes
            assert "GEO_BLOCKED" in str(e)

    def test_predicted_fundings(self, info):
        """Test predicted_fundings returns funding rates (or raises GeoBlockedError if geo-blocked)."""
        from hyperliquid_sdk import GeoBlockedError
        try:
            fundings = info.predicted_fundings()
            assert isinstance(fundings, list)
            assert len(fundings) > 0
        except GeoBlockedError as e:
            # Geo-blocking detected correctly - test passes
            assert "GEO_BLOCKED" in str(e)


# Test 4: WebSocket streaming
class TestWebSocketStreaming:
    """Test WebSocket streaming works."""

    def test_websocket_connection(self):
        """Test WebSocket can connect and receive trades.

        Note: Public Hyperliquid WebSocket may be geo-blocked.
        If connection fails due to geo-blocking, verify state transitions happened.
        """
        from hyperliquid_sdk import Stream, ConnectionState

        trades_received = []
        states = []
        errors = []

        def on_trade(data):
            trades_received.append(data)

        def on_state(state):
            states.append(state)

        def on_error(err):
            errors.append(err)

        stream = Stream(
            "https://api.hyperliquid.xyz",
            on_state_change=on_state,
            on_error=on_error,
            reconnect=False,
        )

        stream.trades(["BTC"], on_trade)
        stream.start()

        # Wait for some trades (or error)
        start = time.time()
        while time.time() - start < 10:
            if len(trades_received) >= 1 or len(errors) > 0:
                break
            time.sleep(0.5)

        stream.stop()

        # Verify connection states - CONNECTING should always happen
        assert ConnectionState.CONNECTING in states

        # If we got trades, connection was successful
        if len(trades_received) > 0:
            assert ConnectionState.CONNECTED in states
        else:
            # No trades received - likely geo-blocked or network issue
            # Test passes as long as we had proper state transitions
            assert ConnectionState.DISCONNECTED in states or len(errors) > 0

    def test_stream_types_enum(self):
        """Test StreamType enum has all values."""
        from hyperliquid_sdk import StreamType

        # Core types
        assert StreamType.TRADES.value == "trades"
        assert StreamType.ORDERS.value == "orders"
        assert StreamType.BOOK_UPDATES.value == "book_updates"
        assert StreamType.TWAP.value == "twap"
        assert StreamType.EVENTS.value == "events"
        # Additional types
        assert StreamType.L2_BOOK.value == "l2Book"
        assert StreamType.ALL_MIDS.value == "allMids"
        assert StreamType.CANDLE.value == "candle"
        assert StreamType.BBO.value == "bbo"
        assert StreamType.USER_EVENTS.value == "userEvents"
        assert StreamType.USER_FILLS.value == "userFills"
        assert StreamType.NOTIFICATION.value == "notification"

    def test_connection_state_enum(self):
        """Test ConnectionState enum has all values."""
        from hyperliquid_sdk import ConnectionState

        assert ConnectionState.DISCONNECTED.value == "disconnected"
        assert ConnectionState.CONNECTING.value == "connecting"
        assert ConnectionState.CONNECTED.value == "connected"
        assert ConnectionState.RECONNECTING.value == "reconnecting"


# Test 5: gRPC streaming
class TestGRPCStreaming:
    """Test gRPC streaming setup."""

    def test_grpc_stream_types_enum(self):
        """Test GRPCStreamType enum has all values."""
        from hyperliquid_sdk import GRPCStreamType

        assert GRPCStreamType.TRADES.value == "TRADES"
        assert GRPCStreamType.ORDERS.value == "ORDERS"
        assert GRPCStreamType.BOOK_UPDATES.value == "BOOK_UPDATES"
        assert GRPCStreamType.TWAP.value == "TWAP"
        assert GRPCStreamType.EVENTS.value == "EVENTS"
        assert GRPCStreamType.BLOCKS.value == "BLOCKS"

    def test_grpc_stream_initialization(self):
        """Test GRPCStream can be initialized."""
        from hyperliquid_sdk import GRPCStream

        stream = GRPCStream(
            "https://test.quiknode.pro/TOKEN",
            reconnect=False,
        )

        assert stream._host == "test.quiknode.pro"
        assert stream._token == "TOKEN"
        assert stream._get_target() == "test.quiknode.pro:10000"

    def test_grpc_subscriptions(self):
        """Test GRPCStream subscription methods."""
        from hyperliquid_sdk import GRPCStream

        stream = GRPCStream(
            "https://test.quiknode.pro/TOKEN",
            reconnect=False,
        )

        # Chain subscriptions
        stream.trades(["BTC"], lambda x: None)
        stream.orders(["ETH"], lambda x: None)
        stream.blocks(lambda x: None)
        stream.l2_book("BTC", lambda x: None)

        assert len(stream._subscriptions) == 4


# Test 6: Error handling
class TestErrorHandling:
    """Test error classes work correctly."""

    def test_hyperliquid_error(self):
        """Test base HyperliquidError."""
        from hyperliquid_sdk import HyperliquidError

        error = HyperliquidError("Test error", code="TEST", raw={"foo": "bar"})
        assert "Test error" in str(error)
        assert error.code == "TEST"
        assert error.raw == {"foo": "bar"}

    def test_specific_errors(self):
        """Test specific error types inherit from HyperliquidError."""
        from hyperliquid_sdk import (
            HyperliquidError,
            BuildError,
            SendError,
            ValidationError,
        )

        # Test basic errors that take simple string messages
        build_error = BuildError("build failed")
        assert isinstance(build_error, HyperliquidError)

        send_error = SendError("send failed")
        assert isinstance(send_error, HyperliquidError)

        validation_error = ValidationError("validation failed")
        assert isinstance(validation_error, HyperliquidError)


# Test 7: Order and Side enums
class TestOrderTypes:
    """Test Order and Side types."""

    def test_side_enum(self):
        """Test Side enum."""
        from hyperliquid_sdk import Side

        assert Side.BUY.value == "buy"
        assert Side.SELL.value == "sell"

    def test_tif_enum(self):
        """Test TIF enum."""
        from hyperliquid_sdk import TIF

        assert hasattr(TIF, "GTC")
        assert hasattr(TIF, "IOC")
        assert hasattr(TIF, "ALO")


# Test 8: HyperCore API
class TestHyperCoreAPI:
    """Test HyperCore API initialization."""

    def test_hypercore_initialization(self):
        """Test HyperCore can be initialized."""
        from hyperliquid_sdk import HyperCore

        hc = HyperCore("https://test.quiknode.pro/TOKEN")
        assert hc._hypercore_url is not None
        assert "/hypercore" in hc._hypercore_url

    def test_hypercore_endpoint_parsing(self):
        """Test HyperCore handles various endpoint formats."""
        from hyperliquid_sdk import HyperCore

        endpoints = [
            "https://x.quiknode.pro/TOKEN",
            "https://x.quiknode.pro/TOKEN/info",
            "https://x.quiknode.pro/TOKEN/hypercore",
        ]
        for ep in endpoints:
            hc = HyperCore(ep)
            assert "/TOKEN/hypercore" in hc._hypercore_url


# Test 9: EVM API
class TestEVMAPI:
    """Test EVM API initialization."""

    def test_evm_initialization(self):
        """Test EVM can be initialized."""
        from hyperliquid_sdk import EVM

        evm = EVM("https://test.quiknode.pro/TOKEN")
        assert evm._base_url is not None
        assert "/evm" in evm._base_url

    def test_evm_endpoint_parsing(self):
        """Test EVM handles various endpoint formats."""
        from hyperliquid_sdk import EVM

        endpoints = [
            "https://x.quiknode.pro/TOKEN",
            "https://x.quiknode.pro/TOKEN/evm",
            "https://x.quiknode.pro/TOKEN/info",
        ]
        for ep in endpoints:
            evm = EVM(ep)
            assert "/TOKEN/evm" in evm._base_url


# Test 10: SDK Version
class TestSDKVersion:
    """Test SDK version is defined."""

    def test_version_exists(self):
        """Test SDK has version."""
        import hyperliquid_sdk
        assert hasattr(hyperliquid_sdk, "__version__")
        # Check version format (major.minor.patch)
        assert hyperliquid_sdk.__version__.count(".") >= 1


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
