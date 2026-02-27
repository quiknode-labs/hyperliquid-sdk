"""Pytest configuration and shared fixtures."""

import pytest
from unittest.mock import Mock, MagicMock, patch
import sys


@pytest.fixture(autouse=True)
def mock_websocket():
    """Mock websocket module for all tests."""
    mock_ws = MagicMock()
    with patch.dict(sys.modules, {'websocket': mock_ws}):
        yield mock_ws


@pytest.fixture
def mock_response():
    """Create a mock HTTP response factory."""
    def _mock_response(status_code=200, json_data=None, text=""):
        response = Mock()
        response.status_code = status_code
        response.text = text
        if json_data is not None:
            response.json.return_value = json_data
        return response
    return _mock_response


@pytest.fixture
def mock_session(mock_response):
    """Create a mock requests session."""
    session = Mock()
    return session
