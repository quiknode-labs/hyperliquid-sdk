"""Tests for the Info client."""

import pytest
from unittest.mock import Mock, patch, MagicMock
import requests

from hyperliquid_sdk.info import Info
from hyperliquid_sdk.errors import HyperliquidError


class TestInfoUrlBuilder:
    """Test URL building for Info client."""

    def test_quicknode_url(self):
        """Test QuickNode URL conversion."""
        info = Info("https://endpoint.hype-mainnet.quiknode.pro/abc123")
        assert info._info_url == "https://endpoint.hype-mainnet.quiknode.pro/abc123/info"

    def test_quicknode_with_path(self):
        """Test QuickNode URL with existing path."""
        info = Info("https://endpoint.quiknode.pro/abc123/hypercore")
        assert info._info_url == "https://endpoint.quiknode.pro/abc123/info"

    def test_public_hyperliquid_url(self):
        """Test public Hyperliquid URL."""
        info = Info("https://api.hyperliquid.xyz")
        assert info._info_url == "https://api.hyperliquid.xyz/info"

    def test_already_has_info_path(self):
        """Test URL that already ends with /info."""
        info = Info("https://api.hyperliquid.xyz/info")
        assert info._info_url == "https://api.hyperliquid.xyz/info"


class TestInfoMethods:
    """Test Info API methods with mocks."""

    @pytest.fixture
    def info(self):
        """Create Info instance with mocked session."""
        info = Info("https://test.quiknode.pro/token")
        info._session = Mock()
        return info

    def test_all_mids(self, info):
        """Test all_mids method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"BTC": "67000.5", "ETH": "3500.25"}
        info._session.post.return_value = mock_response

        result = info.all_mids()

        assert result == {"BTC": "67000.5", "ETH": "3500.25"}
        # Should use proxy URL for allMids
        call_args = info._session.post.call_args
        assert "send.hyperliquidapi.com" in call_args[0][0]

    def test_meta(self, info):
        """Test meta method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"universe": []}
        info._session.post.return_value = mock_response

        result = info.meta()

        assert result == {"universe": []}
        # Should use direct URL for meta
        call_args = info._session.post.call_args
        assert "test.quiknode.pro" in call_args[0][0]

    def test_clearinghouse_state(self, info):
        """Test clearinghouse_state method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "marginSummary": {"accountValue": "10000"},
            "assetPositions": []
        }
        info._session.post.return_value = mock_response

        result = info.clearinghouse_state("0x1234")

        assert "marginSummary" in result
        call_args = info._session.post.call_args
        body = call_args[1]["json"]
        assert body["type"] == "clearinghouseState"
        assert body["user"] == "0x1234"

    def test_l2_book(self, info):
        """Test l2_book method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "coin": "BTC",
            "levels": [
                [{"px": "67000", "sz": "1.5", "n": 3}],
                [{"px": "67001", "sz": "0.5", "n": 1}]
            ]
        }
        info._session.post.return_value = mock_response

        result = info.l2_book("BTC")

        assert result["coin"] == "BTC"
        # l2Book should use proxy
        call_args = info._session.post.call_args
        assert "send.hyperliquidapi.com" in call_args[0][0]

    def test_open_orders(self, info):
        """Test open_orders method."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = [
            {"oid": 123, "coin": "BTC", "side": "A", "sz": "0.1"}
        ]
        info._session.post.return_value = mock_response

        result = info.open_orders("0x1234")

        assert len(result) == 1
        assert result[0]["oid"] == 123


class TestInfoErrorHandling:
    """Test error handling in Info client."""

    @pytest.fixture
    def info(self):
        """Create Info instance with mocked session."""
        info = Info("https://test.quiknode.pro/token")
        info._session = Mock()
        return info

    def test_http_error(self, info):
        """Test HTTP error handling."""
        mock_response = Mock()
        mock_response.status_code = 500
        mock_response.text = "Internal Server Error"
        info._session.post.return_value = mock_response

        with pytest.raises(HyperliquidError) as exc_info:
            info.meta()

        assert exc_info.value.code == "HTTP_ERROR"
        assert "500" in str(exc_info.value)

    def test_timeout_error(self, info):
        """Test timeout error handling."""
        info._session.post.side_effect = requests.exceptions.Timeout()

        with pytest.raises(HyperliquidError) as exc_info:
            info.meta()

        assert exc_info.value.code == "TIMEOUT"

    def test_connection_error(self, info):
        """Test connection error handling."""
        info._session.post.side_effect = requests.exceptions.ConnectionError("Connection refused")

        with pytest.raises(HyperliquidError) as exc_info:
            info.meta()

        assert exc_info.value.code == "CONNECTION_ERROR"

    def test_invalid_json_response(self, info):
        """Test invalid JSON response handling."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.side_effect = ValueError("Invalid JSON")
        mock_response.text = "Not JSON"
        info._session.post.return_value = mock_response

        with pytest.raises(HyperliquidError) as exc_info:
            info.meta()

        assert exc_info.value.code == "PARSE_ERROR"
