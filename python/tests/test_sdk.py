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

    def test_outcome_helpers_use_worker_endpoints(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        calls = []

        def fake_get(path, params=None):
            calls.append((path, params))
            return {"ok": True}

        sdk._get = fake_get

        assert sdk.outcomes() == {"ok": True}
        assert sdk.outcome_balances(10) == {"ok": True}

        assert calls[0] == ("/outcomes", None)
        assert calls[1] == (
            "/outcomes/balances",
            {"user": sdk.address, "outcome": "10"},
        )

    def test_outcome_split_builds_signed_helper_action(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        calls = []

        def fake_exchange(body):
            calls.append(body)
            if "signature" not in body:
                return {
                    "hash": "0x" + "00" * 32,
                    "nonce": 123,
                    "action": body["action"],
                }
            return {"success": True}

        sdk._exchange = fake_exchange
        result = sdk.outcome_split(10, Decimal("1.5"))

        assert result["success"] is True
        assert calls[0]["action"] == {
            "type": "outcomeSplit",
            "outcome": 10,
            "amount": "1.5",
        }
        assert calls[1]["action"]["type"] == "outcomeSplit"

    def test_outcome_merge_defaults_to_max_amount(self):
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
        calls = []

        def fake_exchange(body):
            calls.append(body)
            if "signature" not in body:
                return {
                    "hash": "0x" + "00" * 32,
                    "nonce": 123,
                    "action": body["action"],
                }
            return {"success": True}

        sdk._exchange = fake_exchange
        sdk.outcome_merge(10)

        assert calls[0]["action"] == {
            "type": "outcomeMerge",
            "outcome": 10,
            "amount": None,
        }

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


class TestVaultFastCancelExpiresAfter:
    """vaultAddress / fast cancel "f" flag / expiresAfter threading.

    vaultAddress and expiresAfter are top-level exchange-request fields that
    the worker folds into the action hash at build time, so they must appear
    in BOTH the build payload (signed hash covers them) and the send payload
    (signature recovery + forwarded to Hyperliquid). The fast flag "f" lives
    inside the cancel action itself and is only emitted when true.
    """

    @staticmethod
    def _sdk_with_fake_exchange():
        from hyperliquid_sdk import HyperliquidSDK

        sdk = HyperliquidSDK(private_key="0x" + "11" * 32, auto_approve=False)
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
                "success": True,
                "exchangeResponse": {
                    "response": {
                        "data": {
                            "statuses": [{"resting": {"oid": 7}}]
                        }
                    }
                },
            }

        sdk._exchange = fake_exchange
        return sdk, calls

    def test_order_threads_vault_and_expires_to_build_and_send(self):
        sdk, calls = self._sdk_with_fake_exchange()
        vault = "0x" + "ab" * 20
        expiry = 1752900000000

        placed = sdk.buy(
            "BTC", size=0.001, price=67000, tif="gtc",
            vault_address=vault, expires_after=expiry,
        )

        build, send = calls
        # Build: fields must be present so the returned hash covers them.
        assert build["vaultAddress"] == vault
        assert build["expiresAfter"] == expiry
        # Send: fields must ride again for signature recovery + forwarding.
        assert send["vaultAddress"] == vault
        assert send["expiresAfter"] == expiry
        assert placed.oid == 7

    def test_new_fields_never_emitted_when_unset(self):
        sdk, calls = self._sdk_with_fake_exchange()

        sdk.buy("BTC", size=0.001, price=67000, tif="gtc")
        sdk.cancel(7)

        for body in calls:
            assert "vaultAddress" not in body
            assert "expiresAfter" not in body
            assert "f" not in body["action"]

    def test_cancel_fast_flag_emitted_only_when_true(self):
        sdk, calls = self._sdk_with_fake_exchange()

        sdk.cancel(42, fast=True)
        assert calls[0]["action"] == {
            "type": "cancel",
            "cancels": [{"a": 0, "o": 42}],
            "f": True,
        }

        calls.clear()
        sdk.cancel(42, fast=False)
        assert "f" not in calls[0]["action"]

    def test_cancel_by_cloid_fast_and_vault(self):
        sdk, calls = self._sdk_with_fake_exchange()
        vault = "0x" + "cd" * 20

        sdk.cancel_by_cloid("0x" + "01" * 16, 3, fast=True, vault_address=vault)

        build, send = calls
        assert build["action"]["type"] == "cancelByCloid"
        assert build["action"]["f"] is True
        assert build["vaultAddress"] == vault
        assert send["vaultAddress"] == vault

    def test_cancel_all_fast_sets_flag(self):
        sdk, calls = self._sdk_with_fake_exchange()
        all_action = {"type": "cancel", "cancels": [{"a": 0, "o": 1}, {"a": 1, "o": 2}]}
        sdk.open_orders = lambda: {
            "orders": [{"oid": 1}, {"oid": 2}],
            "cancelActions": {"all": all_action},
        }

        sdk.cancel_all(fast=True)

        assert calls[0]["action"]["f"] is True
        # The worker-provided action dict must not be mutated in place.
        assert "f" not in all_action

    def test_modify_threads_vault_and_expires(self):
        sdk, calls = self._sdk_with_fake_exchange()
        vault = "0x" + "ef" * 20
        expiry = 1752900000000

        sdk.modify(
            9, "BTC", "buy", "67000", "0.001",
            vault_address=vault, expires_after=expiry,
        )

        build, send = calls
        assert build["action"]["type"] == "batchModify"
        assert build["vaultAddress"] == vault
        assert build["expiresAfter"] == expiry
        assert send["vaultAddress"] == vault
        assert send["expiresAfter"] == expiry

    def test_close_position_threads_vault_and_expires(self):
        sdk, calls = self._sdk_with_fake_exchange()
        vault = "0x" + "12" * 20

        sdk.close_position("BTC", vault_address=vault, expires_after=1752900000000)

        build, send = calls
        assert build["action"]["type"] == "closePosition"
        assert build["vaultAddress"] == vault
        assert send["vaultAddress"] == vault
        assert send["expiresAfter"] == 1752900000000

    def test_schedule_cancel_threads_vault_and_expires(self):
        sdk, calls = self._sdk_with_fake_exchange()
        vault = "0x" + "34" * 20

        sdk.schedule_cancel(1752900060000, vault_address=vault, expires_after=1752900000000)

        build, send = calls
        assert build["action"] == {"type": "scheduleCancel", "time": 1752900060000}
        assert build["vaultAddress"] == vault
        assert send["expiresAfter"] == 1752900000000

    def test_trigger_order_threads_vault_and_expires(self):
        sdk, calls = self._sdk_with_fake_exchange()
        vault = "0x" + "56" * 20

        sdk.stop_loss(
            "BTC", size=0.001, trigger_price=60000,
            vault_address=vault, expires_after=1752900000000,
        )

        build, send = calls
        assert build["vaultAddress"] == vault
        assert build["expiresAfter"] == 1752900000000
        assert send["vaultAddress"] == vault

    def test_expires_after_rejects_invalid_values(self):
        from hyperliquid_sdk.errors import ValidationError

        sdk, _ = self._sdk_with_fake_exchange()

        with pytest.raises(ValidationError, match="expires_after"):
            sdk.cancel(1, expires_after=-5)
        with pytest.raises(ValidationError, match="expires_after"):
            sdk.cancel(1, expires_after="soon")


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
        assert GRPCStreamType.MEMPOOL_TXS.value == "MEMPOOL_TXS"
        assert GRPCStreamType.ORDER_PRIORITY.value == "ORDER_PRIORITY"
        assert GRPCStreamType.GOSSIP_PRIORITY.value == "GOSSIP_PRIORITY"

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


class TestGRPCNewStreams:
    """Test the newer gRPC stream types and orderbook RPCs."""

    def _stream(self):
        from hyperliquid_sdk import GRPCStream

        return GRPCStream("https://test.quiknode.pro/TOKEN", reconnect=False)

    def test_stream_type_map_matches_proto(self):
        """Test _STREAM_TYPE_MAP matches the generated proto enum, incl. 8/9/10."""
        from hyperliquid_sdk import proto
        from hyperliquid_sdk.grpc_stream import _STREAM_TYPE_MAP

        for name, value in _STREAM_TYPE_MAP.items():
            assert proto.StreamType.Value(name) == value

        assert _STREAM_TYPE_MAP["MEMPOOL_TXS"] == 8
        assert _STREAM_TYPE_MAP["ORDER_PRIORITY"] == 9
        assert _STREAM_TYPE_MAP["GOSSIP_PRIORITY"] == 10

    def test_new_stream_subscriptions(self):
        """Test new subscription helpers register subscriptions."""
        stream = self._stream()

        stream.mempool_txs(lambda x: None, coins=["BTC"])
        stream.raw_mempool_txs(lambda x: None)
        stream.order_priority(lambda x: None)
        stream.raw_order_priority(lambda x: None)
        stream.gossip_priority(lambda x: None)
        stream.raw_gossip_priority(lambda x: None)
        stream.bbo_book(lambda x: None, coins=["BTC"])
        stream.l2_book_diff(lambda x: None, coins=["BTC"])
        stream.l4_book_updates(lambda x: None)
        stream.tpsl_updates(lambda x: None)
        stream.l2_book_packed("BTC", lambda x: None)
        stream.bbo_book_packed(lambda x: None)
        stream.l4_book_bytes("BTC", lambda x: None)
        stream.stream_bytes("TRADES", lambda x: None, coins=["BTC"])

        assert len(stream._subscriptions) == 14
        assert stream._subscriptions[0]["stream_type"] == "MEMPOOL_TXS"
        assert stream._subscriptions[0]["coins"] == ["BTC"]
        assert stream._subscriptions[1]["raw"] is True
        assert stream._subscriptions[2]["stream_type"] == "ORDER_PRIORITY"
        assert stream._subscriptions[4]["stream_type"] == "GOSSIP_PRIORITY"
        assert stream._subscriptions[13]["stream_type"] == "STREAM_BYTES"
        assert stream._subscriptions[13]["bytes_stream_type"] == "TRADES"

    def test_subscribe_request_start_block_and_filters(self):
        """Test start_block and coin/user filters are plumbed into StreamSubscribe."""
        stream = self._stream()
        stream.trades(["BTC", "ETH"], lambda x: None, start_block=12345)
        stream.orders(["ETH"], lambda x: None, users=["0xabc"])
        stream.mempool_txs(lambda x: None, coins=["BTC"])

        req = stream._build_subscribe_request(stream._subscriptions[0])
        assert req.subscribe.stream_type == 1
        assert req.subscribe.start_block == 12345
        assert list(req.subscribe.filters["coin"].values) == ["BTC", "ETH"]

        req = stream._build_subscribe_request(stream._subscriptions[1])
        assert req.subscribe.start_block == 0
        assert list(req.subscribe.filters["user"].values) == ["0xabc"]

        req = stream._build_subscribe_request(stream._subscriptions[2])
        assert req.subscribe.stream_type == 8
        assert list(req.subscribe.filters["coin"].values) == ["BTC"]

    def test_stream_bytes_request(self):
        """Test stream_bytes builds a SubscribeRequest for the requested stream type."""
        stream = self._stream()
        stream.stream_bytes("ORDERS", lambda x: None, users=["0xabc"], start_block=99)

        sub = stream._subscriptions[0]
        req = stream._build_subscribe_request(sub, sub["bytes_stream_type"])
        assert req.subscribe.stream_type == 2
        assert req.subscribe.start_block == 99
        assert list(req.subscribe.filters["user"].values) == ["0xabc"]

    def test_orderbook_request_builders(self):
        """Test orderbook subscription options map to the right request messages."""
        from hyperliquid_sdk import proto

        stream = self._stream()
        stream.bbo_book(lambda x: None, coins=["BTC"])
        stream.l2_book_diff(
            lambda x: None,
            coins=["BTC", "ETH"],
            n_levels=50,
            n_sig_figs=5,
            mantissa=2,
            skip_initial_snapshot=True,
        )
        stream.l4_book_updates(lambda x: None)
        stream.tpsl_updates(lambda x: None, coins=["SOL"])
        stream.l2_book_packed("BTC", lambda x: None, n_sig_figs=4, n_levels=10)
        stream.bbo_book_packed(lambda x: None)
        stream.l4_book_bytes("ETH", lambda x: None)

        subs = stream._subscriptions

        req = stream._build_orderbook_request(subs[0])
        assert isinstance(req, proto.BboBookRequest)
        assert list(req.coins) == ["BTC"]

        req = stream._build_orderbook_request(subs[1])
        assert isinstance(req, proto.L2BookDiffRequest)
        assert list(req.coins) == ["BTC", "ETH"]
        assert req.n_levels == 50
        assert req.n_sig_figs == 5
        assert req.mantissa == 2
        assert req.skip_initial_snapshot is True

        req = stream._build_orderbook_request(subs[2])
        assert isinstance(req, proto.L4BookUpdatesRequest)
        assert list(req.coins) == []

        req = stream._build_orderbook_request(subs[3])
        assert isinstance(req, proto.TpslUpdatesRequest)
        assert list(req.coins) == ["SOL"]

        req = stream._build_orderbook_request(subs[4])
        assert isinstance(req, proto.L2BookRequest)
        assert req.coin == "BTC"
        assert req.n_sig_figs == 4
        assert req.n_levels == 10

        req = stream._build_orderbook_request(subs[5])
        assert isinstance(req, proto.BboBookRequest)
        assert list(req.coins) == []

        req = stream._build_orderbook_request(subs[6])
        assert isinstance(req, proto.L4BookRequest)
        assert req.coin == "ETH"

    def test_bbo_update_conversion(self):
        """Test BboBookUpdate conversion (absent bid/ask -> None)."""
        from hyperliquid_sdk import proto

        stream = self._stream()

        update = proto.BboBookUpdate(
            coin="BTC",
            time=1000,
            block_number=42,
            bid=proto.L2Level(px="50000", sz="1.5", n=3),
        )
        data = stream._orderbook_update_to_dict("BBO_BOOK", update)
        assert data["coin"] == "BTC"
        assert data["bid"] == ["50000", "1.5", 3]
        assert data["ask"] is None

    def test_l2_book_diff_conversion(self):
        """Test L2BookDiffUpdate conversion (sz=0 level = removed)."""
        from hyperliquid_sdk import proto

        stream = self._stream()

        update = proto.L2BookDiffUpdate(
            time=1000,
            height=42,
            snapshot=False,
            diffs=[
                proto.L2CoinDiff(
                    coin="BTC",
                    seq=7,
                    prev_seq=6,
                    bids=[proto.L2Level(px="50000", sz="0", n=0)],
                    asks=[proto.L2Level(px="50001", sz="2", n=1)],
                )
            ],
        )
        data = stream._orderbook_update_to_dict("L2_BOOK_DIFF", update)
        assert data["height"] == 42
        assert data["snapshot"] is False
        assert data["diffs"][0]["coin"] == "BTC"
        assert data["diffs"][0]["seq"] == 7
        assert data["diffs"][0]["prev_seq"] == 6
        assert data["diffs"][0]["bids"] == [["50000", "0", 0]]
        assert data["diffs"][0]["asks"] == [["50001", "2", 1]]

    def test_l4_book_updates_conversion(self):
        """Test L4BookUpdatesUpdate conversion with typed diffs."""
        from hyperliquid_sdk import proto

        stream = self._stream()

        update = proto.L4BookUpdatesUpdate(
            time=1000,
            height=42,
            snapshot=True,
            diffs=[
                proto.L4OrderDiff(
                    diff_type=proto.L4OrderDiffType.L4_ORDER_DIFF_TYPE_NEW,
                    coin="BTC",
                    oid=123,
                    user="0xabc",
                    side="B",
                    px="50000",
                    sz="1",
                )
            ],
        )
        data = stream._orderbook_update_to_dict("L4_BOOK_UPDATES", update)
        assert data["snapshot"] is True
        assert data["diffs"][0]["diff_type"] == "L4_ORDER_DIFF_TYPE_NEW"
        assert data["diffs"][0]["oid"] == 123

    def test_tpsl_updates_conversion(self):
        """Test TpslUpdatesUpdate conversion with typed diffs."""
        from hyperliquid_sdk import proto

        stream = self._stream()

        update = proto.TpslUpdatesUpdate(
            time=1000,
            height=42,
            diffs=[
                proto.TpslOrderDiff(
                    diff_type=proto.TpslDiffType.TPSL_DIFF_TYPE_REMOVE,
                    oid=55,
                    coin="ETH",
                    user="0xdef",
                    side="A",
                    trigger_px="3000",
                    reason="filled",
                )
            ],
        )
        data = stream._orderbook_update_to_dict("TPSL_UPDATES", update)
        assert data["diffs"][0]["diff_type"] == "TPSL_DIFF_TYPE_REMOVE"
        assert data["diffs"][0]["oid"] == 55
        assert data["diffs"][0]["reason"] == "filled"

    def test_packed_conversion(self):
        """Test packed L2/BBO conversion keeps fixed-point integers."""
        from hyperliquid_sdk import proto

        stream = self._stream()

        update = proto.L2BookPackedUpdate(
            coin="BTC",
            time=1000,
            block_number=42,
            bids=[proto.L2LevelPacked(px=5000000000000, sz=150000000, n=3)],
        )
        data = stream._orderbook_update_to_dict("L2_BOOK_PACKED", update)
        assert data["bids"] == [[5000000000000, 150000000, 3]]

        update = proto.BboBookPackedUpdate(
            coin="BTC",
            time=1000,
            block_number=42,
            ask=proto.L2LevelPacked(px=5000100000000, sz=200000000, n=1),
        )
        data = stream._orderbook_update_to_dict("BBO_BOOK_PACKED", update)
        assert data["bid"] is None
        assert data["ask"] == [5000100000000, 200000000, 1]

    def test_l4_book_bytes_conversion(self):
        """Test L4BookBytesUpdate conversion (diff payload stays raw bytes)."""
        from hyperliquid_sdk import proto

        stream = self._stream()

        update = proto.L4BookBytesUpdate(
            diff=proto.L4BookBytesDiff(time=1000, height=42, data=b'{"order_statuses":[]}')
        )
        data = stream._orderbook_update_to_dict("L4_BOOK_BYTES", update)
        assert data["type"] == "diff"
        assert data["data"] == b'{"order_statuses":[]}'

        update = proto.L4BookBytesUpdate(
            snapshot=proto.L4BookSnapshot(coin="BTC", time=1000, height=42)
        )
        data = stream._orderbook_update_to_dict("L4_BOOK_BYTES", update)
        assert data["type"] == "snapshot"
        assert data["coin"] == "BTC"

    def test_stubs_have_new_rpcs(self):
        """Test generated stubs expose the new RPC methods."""
        from hyperliquid_sdk import proto

        class _FakeChannel:
            def unary_unary(self, *args, **kwargs):
                return lambda *a, **k: None

            def unary_stream(self, *args, **kwargs):
                return lambda *a, **k: None

            def stream_stream(self, *args, **kwargs):
                return lambda *a, **k: None

        streaming = proto.StreamingStub(_FakeChannel())
        assert hasattr(streaming, "StreamDataBytes")

        orderbook = proto.OrderBookStreamingStub(_FakeChannel())
        for rpc in (
            "StreamBboBook",
            "StreamL2BookDiff",
            "StreamL4BookUpdates",
            "StreamTpslUpdates",
            "StreamL2BookPacked",
            "StreamBboBookPacked",
            "StreamL4BookBytes",
        ):
            assert hasattr(orderbook, rpc)

    def test_start_streams_dispatch(self):
        """Test _start_streams routes new stream types to the right workers."""
        from hyperliquid_sdk.grpc_stream import _ORDERBOOK_RPC_MAP

        assert set(_ORDERBOOK_RPC_MAP) == {
            "BBO_BOOK",
            "L2_BOOK_DIFF",
            "L4_BOOK_UPDATES",
            "TPSL_UPDATES",
            "L2_BOOK_PACKED",
            "BBO_BOOK_PACKED",
            "L4_BOOK_BYTES",
        }


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
