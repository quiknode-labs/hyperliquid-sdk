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
    """Builder fee approval required or fee exceeds approved amount."""

    def __init__(
        self,
        message: str = "Builder fee approval required",
        approval_data: Optional[Dict[str, Any]] = None,
        **kwargs,
    ):
        self.approval_data = approval_data or {}
        super().__init__(message, **kwargs)


class ValidationError(HyperliquidError):
    """Invalid order parameters (price tick, size decimals, etc.)."""
    pass


class SignatureError(HyperliquidError):
    """Signature verification failed."""
    pass


class NoPositionError(HyperliquidError):
    """No position to close."""

    def __init__(self, asset: str, user: Optional[str] = None):
        super().__init__(
            f"No open position for {asset}",
            code="NO_POSITION",
            guidance=f"You don't have a {asset} position to close. "
                     "Check your positions via sdk.positions() or the Hyperliquid app.",
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
    """Access denied due to geographic restrictions.

    Hyperliquid blocks access from certain jurisdictions including the US.
    The SDK detects this and raises a clear GeoBlockedError.
    """

    def __init__(self, data: Dict[str, Any]):
        error_info = data.get("error", {})
        if isinstance(error_info, dict):
            message = error_info.get("message", "Access denied from restricted jurisdiction")
            jurisdictions = error_info.get("restricted_jurisdictions", "")
            note = error_info.get("note", "")
        else:
            message = str(error_info) if error_info else "Access denied from restricted jurisdiction"
            jurisdictions = ""
            note = ""

        # Build clear guidance (no workaround suggestions)
        guidance_parts = [
            "Your IP is blocked by Hyperliquid's geo-restrictions.",
        ]
        if jurisdictions:
            # Show first few restricted locations
            restricted_list = str(jurisdictions).split(", ")[:5]
            guidance_parts.append(f"Blocked regions include: {', '.join(restricted_list)}...")
        if note:
            guidance_parts.append(f"Note: {note}")

        super().__init__(
            message,
            code="GEO_BLOCKED",
            guidance=" ".join(guidance_parts),
            raw=data,
        )
        self.jurisdictions = jurisdictions
        self.note = note


class InsufficientMarginError(HyperliquidError):
    """Not enough margin for this order."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Insufficient margin for this order",
            code="INSUFFICIENT_MARGIN",
            guidance="Deposit more collateral or reduce order size. "
                     "Check your available margin at https://app.hyperliquid.xyz",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class LeverageError(HyperliquidError):
    """Leverage configuration conflict."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Leverage configuration incompatible with this order",
            code="LEVERAGE_CONFLICT",
            guidance="Close your existing position or update leverage first. "
                     "You may need to switch between cross/isolated margin modes.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class RateLimitError(HyperliquidError):
    """Rate limit exceeded."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Rate limit exceeded",
            code="RATE_LIMITED",
            guidance="Wait a moment and retry. Consider batching multiple orders.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class MaxOrdersError(HyperliquidError):
    """Maximum open orders exceeded."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Maximum open orders exceeded",
            code="MAX_ORDERS_EXCEEDED",
            guidance="Cancel some existing orders before placing new ones. "
                     "Use sdk.open_orders() to see your orders and sdk.cancel_all() to clear them.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class ReduceOnlyError(HyperliquidError):
    """Reduce-only order would increase position."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Reduce-only order would increase position",
            code="REDUCE_ONLY_VIOLATION",
            guidance="Remove reduce_only flag or ensure the order direction "
                     "actually reduces your current position.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class DuplicateOrderError(HyperliquidError):
    """Duplicate client order ID."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Duplicate order (client order ID already exists)",
            code="DUPLICATE_ORDER",
            guidance="Use a unique cloid for each order, or omit it to auto-generate.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class UserNotFoundError(HyperliquidError):
    """User/wallet not recognized by Hyperliquid."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Wallet not recognized by Hyperliquid",
            code="USER_NOT_FOUND",
            guidance="Deposit USDC to your Hyperliquid account first at "
                     "https://app.hyperliquid.xyz — deposits go through the Arbitrum bridge.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class MustDepositError(HyperliquidError):
    """Account needs a deposit before trading."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Your account needs a deposit before trading",
            code="MUST_DEPOSIT_FIRST",
            guidance="Go to https://app.hyperliquid.xyz and deposit USDC, then try again.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


class InvalidNonceError(HyperliquidError):
    """Nonce invalid or expired."""

    def __init__(self, raw_error: str = ""):
        super().__init__(
            "Nonce invalid or expired",
            code="INVALID_NONCE",
            guidance="The SDK handles nonces automatically. "
                     "If you see this error, your request may have timed out. Retry.",
            raw={"rawHlError": raw_error} if raw_error else {},
        )


def parse_api_error(data: Dict[str, Any], status_code: int = 0) -> HyperliquidError:
    """Parse API error response into appropriate exception."""
    error = data.get("error", "UNKNOWN_ERROR")
    message = data.get("message", str(error) if error else "Unknown error")
    guidance = data.get("guidance")
    raw = data
    raw_hl_error = data.get("rawHlError", "")

    # Handle nested error object (geo-blocking returns {"error": {...}})
    if isinstance(error, dict):
        if error.get("code") == 403 or "restricted jurisdiction" in str(error.get("message", "")).lower():
            return GeoBlockedError(data)
        message = error.get("message", message)
        error = str(error.get("code", "UNKNOWN_ERROR"))

    # Normalize error code
    error_code = str(error).upper().replace("-", "_").replace(" ", "_")

    # Check for translated HL errors (these have errorCode field)
    hl_error_code = data.get("errorCode", "")
    if hl_error_code:
        error_code = hl_error_code

    # Map error codes to specific exceptions

    # Approval errors
    if error_code in ("NOT_APPROVED", "BUILDER_APPROVAL_REQUIRED"):
        return ApprovalError(
            message=message,
            guidance=guidance or "Builder fee not approved. Run sdk.approve_builder_fee('1%') "
                                 "or use HyperliquidSDK(auto_approve=True).",
            approval_data=data.get("approvalRequired"),
            code=error_code,
            raw=raw,
        )

    if error_code == "FEE_EXCEEDS_APPROVED":
        return ApprovalError(
            message=message,
            guidance=guidance or "Your approved max fee is too low. "
                                 "Re-approve with a higher rate: sdk.approve_builder_fee('1%')",
            code=error_code,
            raw=raw,
        )

    # Validation errors
    if error_code in ("INVALID_JSON", "MISSING_FIELD", "INVALID_PARAMS", "INVALID_ORDER_PARAMS"):
        return ValidationError(message, code=error_code, guidance=guidance, raw=raw)

    if error_code in ("INVALID_PRICE_TICK", "INVALID_SIZE"):
        return ValidationError(
            message,
            code=error_code,
            guidance=guidance or "Use sdk.preflight() to validate orders before placing them.",
            raw=raw
        )

    # Signature errors
    if error_code == "SIGNATURE_INVALID":
        return SignatureError(
            message,
            code=error_code,
            guidance=guidance or "Signature verification failed. This is usually an SDK bug — please report it.",
            raw=raw,
        )

    # Position errors
    if error_code == "NO_POSITION":
        asset = data.get("asset", "unknown")
        return NoPositionError(asset)

    # Geo-blocking
    if status_code == 403 or "restricted" in message.lower() or "jurisdiction" in message.lower():
        return GeoBlockedError(data)

    # Translated HL errors
    if error_code == "INSUFFICIENT_MARGIN":
        return InsufficientMarginError(raw_hl_error)

    if error_code == "LEVERAGE_CONFLICT":
        return LeverageError(raw_hl_error)

    if error_code == "RATE_LIMITED":
        return RateLimitError(raw_hl_error)

    if error_code == "MAX_ORDERS_EXCEEDED":
        return MaxOrdersError(raw_hl_error)

    if error_code == "REDUCE_ONLY_VIOLATION":
        return ReduceOnlyError(raw_hl_error)

    if error_code == "DUPLICATE_ORDER":
        return DuplicateOrderError(raw_hl_error)

    if error_code == "USER_NOT_FOUND":
        return UserNotFoundError(raw_hl_error)

    if error_code == "MUST_DEPOSIT_FIRST":
        return MustDepositError(raw_hl_error)

    if error_code == "INVALID_NONCE":
        return InvalidNonceError(raw_hl_error)

    # Generic categorization based on phase
    phase = str(data.get("phase", "")).lower()
    if "build" in phase:
        return BuildError(message, code=error_code, guidance=guidance, raw=raw)

    # Default to SendError for exchange-related errors
    return SendError(message, code=error_code, guidance=guidance, raw=raw)
