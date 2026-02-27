"""
HyperCore Info API Client — Market data, positions, orders, and more.

50+ methods for querying Hyperliquid's info endpoint.

Methods are automatically routed:
- QuickNode: meta, clearinghouseState, vaults, delegations, etc.
- Proxied: allMids, l2Book, recentTrades, predictedFundings, metaAndAssetCtxs, candleSnapshot

The SDK handles routing automatically — you don't need to think about it.

Example:
    >>> from hyperliquid_sdk import Info
    >>> info = Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> print(info.meta())  # Exchange metadata
    >>> print(info.clearinghouse_state("0x..."))  # User positions
    >>> print(info.all_mids())  # Real-time mid prices (auto-proxied)
"""

from __future__ import annotations
from typing import Optional, List, Dict, Any
from urllib.parse import urlparse, urljoin

import requests

from .errors import HyperliquidError, GeoBlockedError

# Proxy URL for Info methods not available on QuickNode
_PROXY_INFO_URL = "https://send.hyperliquidapi.com/info"

# Types that require proxying (not available on QuickNode endpoints)
_PROXIED_TYPES = frozenset([
    "allMids",
    "l2Book",
    "recentTrades",
    "predictedFundings",
    "metaAndAssetCtxs",
    "candleSnapshot",
    "orderStatus",
])


class Info:
    """
    HyperCore Info API — Market data, user accounts, positions, orders.

    50+ methods for querying Hyperliquid data.

    Examples:
        info = Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

        # Market data
        info.all_mids()
        info.l2_book("BTC")
        info.recent_trades("ETH")
        info.candles("BTC", "1h", start, end)

        # User data
        info.clearinghouse_state("0x...")
        info.open_orders("0x...")
        info.user_fills("0x...")
    """

    def __init__(self, endpoint: str, *, timeout: int = 30):
        """
        Initialize the Info client.

        Args:
            endpoint: Hyperliquid endpoint URL (e.g., https://your-endpoint.quiknode.pro/TOKEN)
            timeout: Request timeout in seconds (default: 30)
        """
        self._info_url = self._build_info_url(endpoint)
        self._timeout = timeout
        self._session = requests.Session()

    def _build_info_url(self, url: str) -> str:
        """Build the /info endpoint URL."""
        parsed = urlparse(url)
        base = f"{parsed.scheme}://{parsed.netloc}"
        path_parts = [p for p in parsed.path.strip("/").split("/") if p]

        # Check if this is the public Hyperliquid API
        if "hyperliquid.xyz" in parsed.netloc or "api.hyperliquid" in parsed.netloc:
            return f"{base}/info"

        # Check if URL already ends with /info
        if path_parts and path_parts[-1] == "info":
            return url.rstrip("/")

        # Find the token (not a known path like info, evm, etc.)
        token = None
        for part in path_parts:
            if part not in ("info", "hypercore", "evm", "nanoreth", "ws"):
                token = part
                break

        if token:
            return f"{base}/{token}/info"
        return f"{base}/info"

    def _post(self, body: Dict[str, Any]) -> Any:
        """POST to /info endpoint, routing to proxy for unsupported types."""
        req_type = body.get("type", "")
        url = _PROXY_INFO_URL if req_type in _PROXIED_TYPES else self._info_url

        try:
            resp = self._session.post(url, json=body, timeout=self._timeout)
        except requests.exceptions.Timeout:
            raise HyperliquidError(
                f"Request timed out after {self._timeout}s",
                code="TIMEOUT",
                raw={"type": req_type, "timeout": self._timeout},
            )
        except requests.exceptions.ConnectionError as e:
            raise HyperliquidError(
                f"Connection failed: {e}",
                code="CONNECTION_ERROR",
                raw={"type": req_type, "error": str(e)},
            ) from e

        if resp.status_code != 200:
            # Check for geo-blocking (403 with specific message)
            if resp.status_code == 403:
                try:
                    error_data = resp.json()
                    # Check the parsed JSON data for geo-blocking indicators
                    error_str = str(error_data).lower()
                    if "restricted" in error_str or "jurisdiction" in error_str:
                        raise GeoBlockedError(error_data)
                except ValueError:
                    # If JSON parsing fails, check raw text
                    if "restricted" in resp.text.lower() or "jurisdiction" in resp.text.lower():
                        raise GeoBlockedError({"error": resp.text})
            raise HyperliquidError(
                f"Request failed with status {resp.status_code}",
                code="HTTP_ERROR",
                raw={"status": resp.status_code, "body": resp.text},
            )

        try:
            return resp.json()
        except ValueError:
            raise HyperliquidError(
                "Invalid JSON response",
                code="PARSE_ERROR",
                raw={"body": resp.text[:500]},
            )

    # ═══════════════════════════════════════════════════════════════════════════
    # MARKET DATA
    # ═══════════════════════════════════════════════════════════════════════════

    def all_mids(self) -> Dict[str, str]:
        """Get all asset mid prices."""
        return self._post({"type": "allMids"})

    def l2_book(
        self,
        coin: str,
        *,
        n_sig_figs: Optional[int] = None,
        mantissa: Optional[int] = None,
    ) -> Dict[str, Any]:
        """
        Get Level 2 order book for an asset.

        Args:
            coin: Asset name ("BTC", "ETH")
            n_sig_figs: Number of significant figures for price bucketing (2-5)
            mantissa: Bucketing mantissa multiplier (1, 2, or 5)
        """
        body: Dict[str, Any] = {"type": "l2Book", "coin": coin}
        if n_sig_figs is not None:
            body["nSigFigs"] = n_sig_figs
        if mantissa is not None:
            body["mantissa"] = mantissa
        return self._post(body)

    def recent_trades(self, coin: str) -> List[Dict[str, Any]]:
        """Get recent trades for an asset."""
        return self._post({"type": "recentTrades", "coin": coin})

    def candles(self, coin: str, interval: str, start_time: int, end_time: int) -> List[Dict[str, Any]]:
        """
        Get historical OHLCV candlestick data.

        Args:
            coin: Asset name ("BTC", "ETH")
            interval: Candle interval ("1m", "5m", "15m", "1h", "4h", "1d")
            start_time: Start timestamp in milliseconds
            end_time: End timestamp in milliseconds
        """
        return self._post({
            "type": "candleSnapshot",
            "req": {"coin": coin, "interval": interval, "startTime": start_time, "endTime": end_time},
        })

    def funding_history(self, coin: str, start_time: int, end_time: Optional[int] = None) -> List[Dict[str, Any]]:
        """Get historical funding rates for an asset."""
        body: Dict[str, Any] = {"type": "fundingHistory", "coin": coin, "startTime": start_time}
        if end_time is not None:
            body["endTime"] = end_time
        return self._post(body)

    def predicted_fundings(self) -> List[Dict[str, Any]]:
        """Get predicted funding rates for all assets."""
        return self._post({"type": "predictedFundings"})

    # ═══════════════════════════════════════════════════════════════════════════
    # METADATA
    # ═══════════════════════════════════════════════════════════════════════════

    def meta(self) -> Dict[str, Any]:
        """Get exchange metadata including assets and margin configurations."""
        return self._post({"type": "meta"})

    def spot_meta(self) -> Dict[str, Any]:
        """Get spot trading metadata."""
        return self._post({"type": "spotMeta"})

    def meta_and_asset_ctxs(self) -> Dict[str, Any]:
        """Get metadata + real-time asset context (funding rates, open interest)."""
        return self._post({"type": "metaAndAssetCtxs"})

    def spot_meta_and_asset_ctxs(self) -> Dict[str, Any]:
        """Get spot metadata + real-time asset context."""
        return self._post({"type": "spotMetaAndAssetCtxs"})

    def exchange_status(self) -> Dict[str, Any]:
        """Get current exchange status."""
        return self._post({"type": "exchangeStatus"})

    def perp_dexs(self) -> List[Dict[str, Any]]:
        """Get perpetual DEX information."""
        return self._post({"type": "perpDexs"})

    # ═══════════════════════════════════════════════════════════════════════════
    # USER ACCOUNT
    # ═══════════════════════════════════════════════════════════════════════════

    def clearinghouse_state(self, user: str, *, dex: Optional[str] = None) -> Dict[str, Any]:
        """
        Get user's perpetual positions and margin info.

        Args:
            user: User address
            dex: The perp dex name. Defaults to empty string for first perp dex.
        """
        body: Dict[str, Any] = {"type": "clearinghouseState", "user": user}
        if dex is not None:
            body["dex"] = dex
        return self._post(body)

    def spot_clearinghouse_state(self, user: str) -> Dict[str, Any]:
        """Get user's spot token balances."""
        return self._post({"type": "spotClearinghouseState", "user": user})

    def open_orders(self, user: str) -> List[Dict[str, Any]]:
        """Get user's open orders."""
        return self._post({"type": "openOrders", "user": user})

    def frontend_open_orders(self, user: str) -> List[Dict[str, Any]]:
        """Get user's open orders with enhanced info."""
        return self._post({"type": "frontendOpenOrders", "user": user})

    def order_status(self, user: str, oid: int) -> Dict[str, Any]:
        """Get status of a specific order."""
        return self._post({"type": "orderStatus", "user": user, "oid": oid})

    def historical_orders(self, user: str) -> List[Dict[str, Any]]:
        """Get user's historical orders."""
        return self._post({"type": "historicalOrders", "user": user})

    def user_fills(self, user: str, *, aggregate_by_time: bool = False) -> List[Dict[str, Any]]:
        """Get user's trade fills."""
        body: Dict[str, Any] = {"type": "userFills", "user": user}
        if aggregate_by_time:
            body["aggregateByTime"] = True
        return self._post(body)

    def user_fills_by_time(self, user: str, start_time: int, end_time: Optional[int] = None) -> List[Dict[str, Any]]:
        """Get user's trade fills within a time range."""
        body: Dict[str, Any] = {"type": "userFillsByTime", "user": user, "startTime": start_time}
        if end_time is not None:
            body["endTime"] = end_time
        return self._post(body)

    def user_funding(self, user: str, start_time: Optional[int] = None, end_time: Optional[int] = None) -> List[Dict[str, Any]]:
        """Get user's funding payments."""
        body: Dict[str, Any] = {"type": "userFunding", "user": user}
        if start_time is not None:
            body["startTime"] = start_time
        if end_time is not None:
            body["endTime"] = end_time
        return self._post(body)

    def user_fees(self, user: str) -> Dict[str, Any]:
        """Get user's fee structure (maker/taker rates)."""
        return self._post({"type": "userFees", "user": user})

    def user_rate_limit(self, user: str) -> Dict[str, Any]:
        """Get user's rate limit status."""
        return self._post({"type": "userRateLimit", "user": user})

    def portfolio(self, user: str) -> Dict[str, Any]:
        """Get user's portfolio history."""
        return self._post({"type": "portfolio", "user": user})

    def web_data2(self, user: str) -> Dict[str, Any]:
        """Get comprehensive account snapshot."""
        return self._post({"type": "webData2", "user": user})

    def sub_accounts(self, user: str) -> List[Dict[str, Any]]:
        """Get user's sub-accounts."""
        return self._post({"type": "subAccounts", "user": user})

    def extra_agents(self, user: str) -> List[Dict[str, Any]]:
        """Get user's extra agents (API keys)."""
        return self._post({"type": "extraAgents", "user": user})

    # ═══════════════════════════════════════════════════════════════════════════
    # BATCH QUERIES
    # ═══════════════════════════════════════════════════════════════════════════

    def batch_clearinghouse_states(self, users: List[str]) -> List[Dict[str, Any]]:
        """Get clearinghouse states for multiple users in one call."""
        return self._post({"type": "batchClearinghouseStates", "users": users})

    # ═══════════════════════════════════════════════════════════════════════════
    # VAULTS
    # ═══════════════════════════════════════════════════════════════════════════

    def vault_summaries(self) -> List[Dict[str, Any]]:
        """Get summaries of all vaults."""
        return self._post({"type": "vaultSummaries"})

    def vault_details(self, vault_address: str, user: Optional[str] = None) -> Dict[str, Any]:
        """Get vault details."""
        body: Dict[str, Any] = {"type": "vaultDetails", "vaultAddress": vault_address}
        if user:
            body["user"] = user
        return self._post(body)

    def leading_vaults(self, user: str) -> List[Dict[str, Any]]:
        """Get vaults that user is leading."""
        return self._post({"type": "leadingVaults", "user": user})

    def user_vault_equities(self, user: str) -> Dict[str, Any]:
        """Get user's vault equities."""
        return self._post({"type": "userVaultEquities", "user": user})

    # ═══════════════════════════════════════════════════════════════════════════
    # DELEGATION / STAKING
    # ═══════════════════════════════════════════════════════════════════════════

    def delegations(self, user: str) -> List[Dict[str, Any]]:
        """Get user's delegations."""
        return self._post({"type": "delegations", "user": user})

    def delegator_history(self, user: str) -> List[Dict[str, Any]]:
        """Get user's delegation history."""
        return self._post({"type": "delegatorHistory", "user": user})

    def delegator_rewards(self, user: str) -> Dict[str, Any]:
        """Get user's delegation rewards."""
        return self._post({"type": "delegatorRewards", "user": user})

    def delegator_summary(self, user: str) -> Dict[str, Any]:
        """Get user's delegation summary."""
        return self._post({"type": "delegatorSummary", "user": user})

    # ═══════════════════════════════════════════════════════════════════════════
    # TOKENS / SPOT
    # ═══════════════════════════════════════════════════════════════════════════

    def token_details(self, token_id: str) -> Dict[str, Any]:
        """Get token details."""
        return self._post({"type": "tokenDetails", "tokenId": token_id})

    def spot_deploy_state(self, user: str) -> Dict[str, Any]:
        """Get spot deployment state for user."""
        return self._post({"type": "spotDeployState", "user": user})

    def liquidatable(self) -> List[Dict[str, Any]]:
        """Get list of liquidatable positions."""
        return self._post({"type": "liquidatable"})

    # ═══════════════════════════════════════════════════════════════════════════
    # OTHER
    # ═══════════════════════════════════════════════════════════════════════════

    def max_builder_fee(self, user: str, builder: str) -> Dict[str, Any]:
        """Get maximum builder fee for a user-builder pair."""
        return self._post({"type": "maxBuilderFee", "user": user, "builder": builder})

    def referral(self, user: str) -> Dict[str, Any]:
        """Get user's referral information."""
        return self._post({"type": "referral", "user": user})

    # ═══════════════════════════════════════════════════════════════════════════
    # ADDITIONAL METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def active_asset_data(self, user: str, coin: str) -> Dict[str, Any]:
        """Get user's active asset trading parameters."""
        return self._post({"type": "activeAssetData", "user": user, "coin": coin})

    def user_role(self, user: str) -> Dict[str, Any]:
        """Get account type (user, agent, vault, or sub-account)."""
        return self._post({"type": "userRole", "user": user})

    def user_non_funding_ledger_updates(
        self, user: str, start_time: Optional[int] = None, end_time: Optional[int] = None
    ) -> List[Dict[str, Any]]:
        """Get user's non-funding ledger updates (deposits, withdrawals, transfers)."""
        body: Dict[str, Any] = {"type": "userNonFundingLedgerUpdates", "user": user}
        if start_time is not None:
            body["startTime"] = start_time
        if end_time is not None:
            body["endTime"] = end_time
        return self._post(body)

    def user_twap_slice_fills(self, user: str, *, limit: int = 500) -> List[Dict[str, Any]]:
        """Get user's TWAP slice fills."""
        return self._post({"type": "userTwapSliceFills", "user": user, "limit": limit})

    def user_to_multi_sig_signers(self, user: str) -> Dict[str, Any]:
        """Get multi-sig signers for a user."""
        return self._post({"type": "userToMultiSigSigners", "user": user})

    def gossip_root_ips(self) -> List[str]:
        """Get gossip root IPs for the network."""
        return self._post({"type": "gossipRootIps"})

    def max_market_order_ntls(self) -> Dict[str, Any]:
        """Get maximum market order notionals per asset."""
        return self._post({"type": "maxMarketOrderNtls"})

    def perp_deploy_auction_status(self) -> Dict[str, Any]:
        """Get perpetual deploy auction status."""
        return self._post({"type": "perpDeployAuctionStatus"})

    def perps_at_open_interest_cap(self) -> List[str]:
        """Get perps that are at their open interest cap."""
        return self._post({"type": "perpsAtOpenInterestCap"})

    def validator_l1_votes(self) -> Dict[str, Any]:
        """Get L1 validator votes."""
        return self._post({"type": "validatorL1Votes"})

    def approved_builders(self, user: str) -> List[Dict[str, Any]]:
        """Get list of approved builders for a user."""
        return self._post({"type": "approvedBuilders", "user": user})

    def __enter__(self) -> Info:
        return self

    def __exit__(self, *args) -> None:
        self._session.close()

    def __repr__(self) -> str:
        return f"<Info {self._info_url[:40]}...>"
