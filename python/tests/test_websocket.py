"""Tests for the WebSocket client."""

import pytest
from unittest.mock import Mock, patch, MagicMock
import json

from hyperliquid_sdk.websocket import Stream, StreamType, ConnectionState


class TestStreamUrlBuilder:
    """Test URL building for WebSocket client."""

    def test_quicknode_url(self):
        """Test QuickNode URL conversion."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            stream = Stream("https://endpoint.hype-mainnet.quiknode.pro/abc123")
            assert "wss://endpoint.hype-mainnet.quiknode.pro/abc123/hypercore/ws" == stream._ws_url

    def test_public_hyperliquid_url(self):
        """Test public Hyperliquid URL."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            stream = Stream("https://api.hyperliquid.xyz")
            assert stream._ws_url == "wss://api.hyperliquid.xyz/ws"

    def test_wss_url_passthrough(self):
        """Test WSS URL passthrough."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            stream = Stream("wss://api.hyperliquid.xyz/ws")
            assert stream._ws_url == "wss://api.hyperliquid.xyz/ws"


class TestStreamSubscriptions:
    """Test subscription management."""

    @pytest.fixture
    def stream(self):
        """Create Stream instance."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            return Stream("https://test.quiknode.pro/token")

    def test_subscribe_creates_entry(self, stream):
        """Test subscription creates entry."""
        callback = Mock()
        sub_id = stream.subscribe("trades", callback, coins=["BTC"])
        
        assert sub_id in stream._subscriptions
        assert sub_id in stream._callbacks
        assert "trades" in stream._channel_callbacks
        assert callback in stream._channel_callbacks["trades"]

    def test_trades_subscription(self, stream):
        """Test trades subscription helper."""
        callback = Mock()
        sub_id = stream.trades(["BTC", "ETH"], callback)
        
        params = stream._subscriptions[sub_id]
        assert params["streamType"] == "trades"
        assert params["coin"] == ["BTC", "ETH"]

    def test_orders_subscription(self, stream):
        """Test orders subscription helper."""
        callback = Mock()
        sub_id = stream.orders(["BTC"], callback, users=["0x1234"])
        
        params = stream._subscriptions[sub_id]
        assert params["streamType"] == "orders"
        assert params["user"] == ["0x1234"]

    def test_candle_subscription(self, stream):
        """Test candle subscription helper."""
        callback = Mock()
        sub_id = stream.candle("BTC", "1h", callback)
        
        params = stream._subscriptions[sub_id]
        assert params["streamType"] == "candle"
        assert params["interval"] == "1h"
        # Should also be in channel callbacks
        assert "candle" in stream._channel_callbacks

    def test_unsubscribe_removes_entries(self, stream):
        """Test unsubscribe removes all entries."""
        callback = Mock()
        sub_id = stream.trades(["BTC"], callback)
        
        stream.unsubscribe(sub_id)
        
        assert sub_id not in stream._subscriptions
        assert sub_id not in stream._callbacks
        # Channel callbacks should be cleaned up
        assert "trades" not in stream._channel_callbacks or callback not in stream._channel_callbacks.get("trades", [])


class TestStreamMessageHandling:
    """Test message handling."""

    @pytest.fixture
    def stream(self):
        """Create Stream instance."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            return Stream("https://test.quiknode.pro/token")

    def test_on_message_routes_to_callback(self, stream):
        """Test message routing to correct callback."""
        callback = Mock()
        stream.subscribe("trades", callback, coins=["BTC"])
        
        message = json.dumps({
            "channel": "trades",
            "data": {"coin": "BTC", "px": "67000", "sz": "0.1"}
        })
        
        stream._on_message(None, message)
        
        callback.assert_called_once()
        call_data = callback.call_args[0][0]
        assert call_data["channel"] == "trades"

    def test_on_message_handles_pong(self, stream):
        """Test pong message handling."""
        import time
        old_pong = stream._last_pong
        
        message = json.dumps({"channel": "pong"})
        stream._on_message(None, message)
        
        assert stream._last_pong > old_pong

    def test_on_message_skips_subscription_response(self, stream):
        """Test subscription response is skipped."""
        callback = Mock()
        stream.subscribe("trades", callback)
        
        message = json.dumps({"channel": "subscriptionResponse", "data": {}})
        stream._on_message(None, message)
        
        callback.assert_not_called()

    def test_on_message_invalid_json(self, stream):
        """Test invalid JSON handling."""
        # Should not raise
        stream._on_message(None, "not valid json")

    def test_callback_error_does_not_crash(self, stream):
        """Test callback error doesn't crash the stream."""
        callback = Mock(side_effect=Exception("Callback error"))
        stream.subscribe("trades", callback)
        
        message = json.dumps({"channel": "trades", "data": {}})
        # Should not raise
        stream._on_message(None, message)


class TestStreamState:
    """Test connection state management."""

    @pytest.fixture
    def stream(self):
        """Create Stream instance."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            return Stream("https://test.quiknode.pro/token")

    def test_initial_state_disconnected(self, stream):
        """Test initial state is disconnected."""
        assert stream.state == ConnectionState.DISCONNECTED
        assert not stream.connected

    def test_state_change_callback(self, stream):
        """Test state change callback is invoked."""
        callback = Mock()
        stream._on_state_change = callback
        
        stream._set_state(ConnectionState.CONNECTING)
        
        callback.assert_called_once_with(ConnectionState.CONNECTING)

    def test_state_change_only_on_actual_change(self, stream):
        """Test state change callback only on actual change."""
        callback = Mock()
        stream._on_state_change = callback
        stream._state = ConnectionState.CONNECTING
        
        stream._set_state(ConnectionState.CONNECTING)
        
        callback.assert_not_called()


class TestStreamContext:
    """Test context manager support."""

    def test_context_manager(self):
        """Test stream can be used as context manager."""
        with patch.dict('sys.modules', {'websocket': MagicMock()}):
            with Stream("https://test.quiknode.pro/token") as stream:
                assert stream is not None
            # Should have called stop
            assert stream._state == ConnectionState.DISCONNECTED
