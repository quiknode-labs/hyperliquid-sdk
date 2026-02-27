"""
Hyperliquid SDK Errors — Clear, actionable, helpful.

Every error tells you:
1. What went wrong
2. Why it happened
3. How to fix it
"""

from typing import Optional, Dict, Any


class HyperliquidError(Exception):
    """Base error for all SDK errors."""

    def __init__(
        self,
        message: str,
        code: Optional[str] = None,
        guidance: Optional[str] = None,
        raw: Optional[Dict[str, Any]] = None,
    ):
        self.message = message
        self.code = code
        self.guidance = guidance
        self.raw = raw or {}
        super().__init__(self._format_message())

    def _format_message(self) -> str:
        parts = [self.message]
        if self.code:
            parts.insert(0, f"[{self.code}]")
        if self.guidance:
            parts.append(f"\n  Hint: {self.guidance}")
        return " ".join(parts)


class BuildError(HyperliquidError):
    """Error during order/action building phase."""
    pass


class SendError(HyperliquidError):
    """Error when sending signed transaction to Hyperliquid."""
    pass


class ApprovalError(HyperliquidError):
    """Builder fee approval required or invalid."""

    def __init__(
        self,
        message: str = "Builder fee approval required",
        approval_hash: Optional[str] = None,
        **kwargs,
    ):
        self.approval_hash = approval_hash
        super().__init__(message, **kwargs)


class ValidationError(HyperliquidError):
    """Client-side validation error (before sending to API)."""
    pass


class NoPositionError(HyperliquidError):
    """No position to close."""

    def __init__(self, asset: str):
        super().__init__(
            f"No open position for {asset}",
            code="NO_POSITION",
            guidance=f"You don't have a {asset} position to close.",
        )


class OrderNotFoundError(HyperliquidError):
    """Order not found for cancel/modify."""

    def __init__(self, oid: int):
        super().__init__(
            f"Order {oid} not found",
            code="ORDER_NOT_FOUND",
            guidance="The order may have already been filled or cancelled.",
        )


class GeoBlockedError(HyperliquidError):
    """Access denied due to geographic restrictions."""

    def __init__(self, data: Dict[str, Any]):
        error_info = data.get("error", {})
        if isinstance(error_info, dict):
            message = error_info.get("message", "Access denied from restricted jurisdiction")
            jurisdictions = error_info.get("restricted_jurisdictions", "")
        else:
            message = "Access denied from restricted jurisdiction"
            jurisdictions = ""

        super().__init__(
            message,
            code="GEO_BLOCKED",
            guidance=f"Use a VPN to connect from a non-restricted location. Restricted: {jurisdictions[:100]}...",
            raw=data,
        )


def parse_api_error(data: Dict[str, Any], status_code: int = 0) -> HyperliquidError:
    """Parse API error response into appropriate exception."""
    error = data.get("error", "UNKNOWN_ERROR")
    message = data.get("message", "Unknown error")
    guidance = data.get("guidance")
    raw = data

    # Handle nested error object (geo-blocking returns {"error": {...}})
    if isinstance(error, dict):
        if error.get("code") == 403 or "restricted jurisdiction" in str(error.get("message", "")):
            return GeoBlockedError(data)
        message = error.get("message", message)
        error = str(error.get("code", "UNKNOWN_ERROR"))

    # Map error codes to specific exceptions
    if error == "NOT_APPROVED":
        return ApprovalError(
            message=message,
            guidance=guidance,
            approval_hash=data.get("approvalHash"),
            code=error,
            raw=raw,
        )

    if error in ("INVALID_JSON", "MISSING_FIELD", "INVALID_PARAMS"):
        return ValidationError(message, code=error, guidance=guidance, raw=raw)

    if error == "NO_POSITION":
        asset = data.get("asset", "unknown")
        return NoPositionError(asset)

    # Generic categorization
    if "build" in str(data.get("phase", "")).lower():
        return BuildError(message, code=error, guidance=guidance, raw=raw)

    return SendError(message, code=error, guidance=guidance, raw=raw)
