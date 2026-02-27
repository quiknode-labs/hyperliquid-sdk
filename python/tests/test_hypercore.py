"""Tests for the HyperCore JSON-RPC client."""

import pytest
from unittest.mock import Mock, patch
import requests

from hyperliquid_sdk.hypercore import HyperCore
from hyperliquid_sdk.errors import HyperliquidError


class TestHyperCoreUrlBuilder:
    """Test URL building for HyperCore client."""

    def test_quicknode_url(self):
        """Test QuickNode URL conversion."""
        hc = HyperCore("https://endpoint.hype-mainnet.quiknode.pro/abc123")
        assert hc._hypercore_url == "https://endpoint.hype-mainnet.quiknode.pro/abc123/hypercore"

    def test_quicknode_with_path(self):
        """Test QuickNode URL with existing path."""
        hc = HyperCore("https://endpoint.quiknode.pro/abc123/info")
        assert hc._hypercore_url == "https://endpoint.quiknode.pro/abc123/hypercore"


class TestHyperCoreMethods:
    """Test HyperCore API methods with mocks."""

    @pytest.fixture
    def hc(self):
        """Create HyperCore instance with mocked session."""
        hc = HyperCore("https://test.quiknode.pro/token")
        hc._session = Mock()
        return hc

    def test_latest_block_number(self, hc):
        """Test latest_block_number method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"jsonrpc": "2.0", "result": 12345, "id": 1}
        hc._session.post.return_value = mock_response

        result = hc.latest_block_number()

        assert result == 12345
        call_args = hc._session.post.call_args
        body = call_args[1]["json"]
        assert body["method"] == "hl_getLatestBlockNumber"

    def test_get_block(self, hc):
        """Test get_block method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "result": {"blockNumber": 100, "events": []},
            "id": 1
        }
        hc._session.post.return_value = mock_response

        result = hc.get_block(100)

        assert result["blockNumber"] == 100
        call_args = hc._session.post.call_args
        body = call_args[1]["json"]
        assert body["method"] == "hl_getBlock"
        assert body["params"] == ["trades", 100]

    def test_latest_trades(self, hc):
        """Test latest_trades method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "result": {
                "blocks": [
                    {
                        "events": [
                            ["0x1234", {"coin": "BTC", "px": "67000", "sz": "0.1"}],
                            ["0x5678", {"coin": "ETH", "px": "3500", "sz": "1.0"}]
                        ]
                    }
                ]
            },
            "id": 1
        }
        hc._session.post.return_value = mock_response

        result = hc.latest_trades(count=10)

        assert len(result) == 2
        assert result[0]["user"] == "0x1234"
        assert result[0]["coin"] == "BTC"

    def test_latest_trades_with_coin_filter(self, hc):
        """Test latest_trades with coin filter."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "result": {
                "blocks": [
                    {
                        "events": [
                            ["0x1234", {"coin": "BTC", "px": "67000", "sz": "0.1"}],
                            ["0x5678", {"coin": "ETH", "px": "3500", "sz": "1.0"}]
                        ]
                    }
                ]
            },
            "id": 1
        }
        hc._session.post.return_value = mock_response

        result = hc.latest_trades(count=10, coin="BTC")

        assert len(result) == 1
        assert result[0]["coin"] == "BTC"

    def test_open_orders(self, hc):
        """Test open_orders method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "result": [{"oid": 123, "coin": "BTC"}],
            "id": 1
        }
        hc._session.post.return_value = mock_response

        result = hc.open_orders("0x1234")

        assert len(result) == 1
        call_args = hc._session.post.call_args
        body = call_args[1]["json"]
        assert body["params"]["user"] == "0x1234"

    def test_build_order(self, hc):
        """Test build_order method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "result": {"action": {"type": "order"}},
            "id": 1
        }
        hc._session.post.return_value = mock_response

        result = hc.build_order(
            coin="BTC",
            is_buy=True,
            limit_px="67000",
            sz="0.001",
            user="0x1234"
        )

        assert "action" in result
        call_args = hc._session.post.call_args
        body = call_args[1]["json"]
        assert body["method"] == "hl_buildOrder"
        params = body["params"]
        assert params["coin"] == "BTC"
        assert params["isBuy"] is True


class TestHyperCoreErrorHandling:
    """Test error handling in HyperCore client."""

    @pytest.fixture
    def hc(self):
        """Create HyperCore instance with mocked session."""
        hc = HyperCore("https://test.quiknode.pro/token")
        hc._session = Mock()
        return hc

    def test_http_error(self, hc):
        """Test HTTP error handling."""
        mock_response = Mock()
        mock_response.status_code = 500
        mock_response.text = "Internal Server Error"
        hc._session.post.return_value = mock_response

        with pytest.raises(HyperliquidError) as exc_info:
            hc.latest_block_number()

        assert exc_info.value.code == "HTTP_ERROR"

    def test_rpc_error_dict(self, hc):
        """Test JSON-RPC error handling with dict error."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "error": {"code": -32600, "message": "Invalid request"},
            "id": 1
        }
        hc._session.post.return_value = mock_response

        with pytest.raises(HyperliquidError) as exc_info:
            hc.latest_block_number()

        assert "Invalid request" in str(exc_info.value)

    def test_rpc_error_string(self, hc):
        """Test JSON-RPC error handling with string error."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jsonrpc": "2.0",
            "error": "Something went wrong",
            "id": 1
        }
        hc._session.post.return_value = mock_response

        with pytest.raises(HyperliquidError) as exc_info:
            hc.latest_block_number()

        assert "Something went wrong" in str(exc_info.value)

    def test_timeout_error(self, hc):
        """Test timeout error handling."""
        hc._session.post.side_effect = requests.exceptions.Timeout()

        with pytest.raises(HyperliquidError) as exc_info:
            hc.latest_block_number()

        assert exc_info.value.code == "TIMEOUT"

    def test_connection_error(self, hc):
        """Test connection error handling."""
        hc._session.post.side_effect = requests.exceptions.ConnectionError()

        with pytest.raises(HyperliquidError) as exc_info:
            hc.latest_block_number()

        assert exc_info.value.code == "CONNECTION_ERROR"

    def test_invalid_json_response(self, hc):
        """Test invalid JSON response handling."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.side_effect = ValueError()
        mock_response.text = "Not JSON"
        hc._session.post.return_value = mock_response

        with pytest.raises(HyperliquidError) as exc_info:
            hc.latest_block_number()

        assert exc_info.value.code == "PARSE_ERROR"
