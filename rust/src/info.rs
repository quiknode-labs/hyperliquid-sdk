//! Info API client for Hyperliquid.
//!
//! Provides read-only queries for market data, account state, and more.

use serde_json::{json, Value};
use std::sync::Arc;

use crate::client::HyperliquidSDKInner;
use crate::error::Result;

/// Info API client
pub struct Info {
    inner: Arc<HyperliquidSDKInner>,
}

impl Info {
    pub(crate) fn new(inner: Arc<HyperliquidSDKInner>) -> Self {
        Self { inner }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Market Metadata
    // ──────────────────────────────────────────────────────────────────────────

    /// Get exchange metadata (perp assets, decimals, etc.)
    pub async fn meta(&self) -> Result<Value> {
        self.inner.query_info(&json!({"type": "meta"})).await
    }

    /// Get spot metadata
    pub async fn spot_meta(&self) -> Result<Value> {
        self.inner.query_info(&json!({"type": "spotMeta"})).await
    }

    /// Get metadata with asset contexts (real-time funding, open interest)
    pub async fn meta_and_asset_ctxs(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "metaAndAssetCtxs"}))
            .await
    }

    /// Get spot metadata with contexts
    pub async fn spot_meta_and_asset_ctxs(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "spotMetaAndAssetCtxs"}))
            .await
    }

    /// Get exchange status
    pub async fn exchange_status(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "exchangeStatus"}))
            .await
    }

    /// Get all perp DEXes (HIP-3)
    pub async fn perp_dexes(&self) -> Result<Value> {
        self.inner.query_info(&json!({"type": "perpDexs"})).await
    }

    /// Get perp categories
    pub async fn perp_categories(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "perpCategories"}))
            .await
    }

    /// Get perp annotation for an asset
    pub async fn perp_annotation(&self, asset: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "perpAnnotation", "asset": asset}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Pricing
    // ──────────────────────────────────────────────────────────────────────────

    /// Get all mid prices
    pub async fn all_mids(&self, dex: Option<&str>) -> Result<Value> {
        let mut body = json!({"type": "allMids"});
        if let Some(d) = dex {
            body["dex"] = json!(d);
        }
        self.inner.query_info(&body).await
    }

    /// Get mid price for a single asset
    pub async fn get_mid(&self, asset: &str) -> Result<f64> {
        let mids = self.all_mids(None).await?;
        mids.get(asset)
            .and_then(|v| v.as_str())
            .and_then(|s| s.parse::<f64>().ok())
            .ok_or_else(|| crate::Error::ValidationError(format!("No price for {}", asset)))
    }

    /// Get L2 order book
    pub async fn l2_book(
        &self,
        coin: &str,
        n_sig_figs: Option<u32>,
        mantissa: Option<u32>,
    ) -> Result<Value> {
        let mut body = json!({"type": "l2Book", "coin": coin});
        if let Some(n) = n_sig_figs {
            body["nSigFigs"] = json!(n);
        }
        if let Some(m) = mantissa {
            body["mantissa"] = json!(m);
        }
        self.inner.query_info(&body).await
    }

    /// Get recent trades
    pub async fn recent_trades(&self, coin: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "recentTrades", "coin": coin}))
            .await
    }

    /// Get candlestick data
    pub async fn candles(
        &self,
        coin: &str,
        interval: &str,
        start_time: u64,
        end_time: Option<u64>,
    ) -> Result<Value> {
        let mut body = json!({
            "type": "candleSnapshot",
            "req": {
                "coin": coin,
                "interval": interval,
                "startTime": start_time,
            }
        });
        if let Some(end) = end_time {
            body["req"]["endTime"] = json!(end);
        }
        self.inner.query_info(&body).await
    }

    /// Get funding history
    pub async fn funding_history(
        &self,
        coin: &str,
        start_time: u64,
        end_time: Option<u64>,
    ) -> Result<Value> {
        let mut body = json!({
            "type": "fundingHistory",
            "coin": coin,
            "startTime": start_time,
        });
        if let Some(end) = end_time {
            body["endTime"] = json!(end);
        }
        self.inner.query_info(&body).await
    }

    /// Get predicted fundings
    pub async fn predicted_fundings(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "predictedFundings"}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // User Account Data
    // ──────────────────────────────────────────────────────────────────────────

    /// Get clearinghouse state (positions, margin)
    pub async fn clearinghouse_state(&self, user: &str, dex: Option<&str>) -> Result<Value> {
        let mut body = json!({"type": "clearinghouseState", "user": user});
        if let Some(d) = dex {
            body["dex"] = json!(d);
        }
        self.inner.query_info(&body).await
    }

    /// Get spot clearinghouse state (token balances)
    pub async fn spot_clearinghouse_state(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "spotClearinghouseState", "user": user}))
            .await
    }

    /// Get open orders
    pub async fn open_orders(&self, user: &str, dex: Option<&str>) -> Result<Value> {
        let mut body = json!({"type": "openOrders", "user": user});
        if let Some(d) = dex {
            body["dex"] = json!(d);
        }
        self.inner.query_info(&body).await
    }

    /// Get frontend open orders (enhanced)
    pub async fn frontend_open_orders(&self, user: &str, dex: Option<&str>) -> Result<Value> {
        let mut body = json!({"type": "frontendOpenOrders", "user": user});
        if let Some(d) = dex {
            body["dex"] = json!(d);
        }
        self.inner.query_info(&body).await
    }

    /// Get order status
    pub async fn order_status(&self, user: &str, oid: u64, dex: Option<&str>) -> Result<Value> {
        let mut body = json!({"type": "orderStatus", "user": user, "oid": oid});
        if let Some(d) = dex {
            body["dex"] = json!(d);
        }
        self.inner.query_info(&body).await
    }

    /// Get historical orders
    pub async fn historical_orders(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "historicalOrders", "user": user}))
            .await
    }

    /// Get user fills
    pub async fn user_fills(&self, user: &str, aggregate_by_time: bool) -> Result<Value> {
        self.inner
            .query_info(&json!({
                "type": "userFills",
                "user": user,
                "aggregateByTime": aggregate_by_time,
            }))
            .await
    }

    /// Get user fills by time range
    pub async fn user_fills_by_time(
        &self,
        user: &str,
        start_time: u64,
        end_time: Option<u64>,
    ) -> Result<Value> {
        let mut body = json!({
            "type": "userFillsByTime",
            "user": user,
            "startTime": start_time,
        });
        if let Some(end) = end_time {
            body["endTime"] = json!(end);
        }
        self.inner.query_info(&body).await
    }

    /// Get user funding payments
    pub async fn user_funding(
        &self,
        user: &str,
        start_time: Option<u64>,
        end_time: Option<u64>,
    ) -> Result<Value> {
        let mut body = json!({"type": "userFunding", "user": user});
        if let Some(start) = start_time {
            body["startTime"] = json!(start);
        }
        if let Some(end) = end_time {
            body["endTime"] = json!(end);
        }
        self.inner.query_info(&body).await
    }

    /// Get user fees
    pub async fn user_fees(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userFees", "user": user}))
            .await
    }

    /// Get user rate limit
    pub async fn user_rate_limit(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userRateLimit", "user": user}))
            .await
    }

    /// Get user role
    pub async fn user_role(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userRole", "user": user}))
            .await
    }

    /// Get sub-accounts
    pub async fn sub_accounts(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "subAccounts", "user": user}))
            .await
    }

    /// Get extra agents (API keys)
    pub async fn extra_agents(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "extraAgents", "user": user}))
            .await
    }

    /// Get portfolio history
    pub async fn portfolio(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "portfolio", "user": user}))
            .await
    }

    /// Get comprehensive web data
    pub async fn web_data2(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "webData2", "user": user}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Batch Queries
    // ──────────────────────────────────────────────────────────────────────────

    /// Batch query clearinghouse states for multiple users
    pub async fn batch_clearinghouse_states(&self, users: &[&str]) -> Result<Value> {
        self.inner
            .query_info(&json!({
                "type": "batchClearinghouseStates",
                "users": users,
            }))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Vaults
    // ──────────────────────────────────────────────────────────────────────────

    /// Get all vault summaries
    pub async fn vault_summaries(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "vaultSummaries"}))
            .await
    }

    /// Get vault details
    pub async fn vault_details(&self, vault_address: &str, user: Option<&str>) -> Result<Value> {
        let mut body = json!({"type": "vaultDetails", "vaultAddress": vault_address});
        if let Some(u) = user {
            body["user"] = json!(u);
        }
        self.inner.query_info(&body).await
    }

    /// Get user vault equities
    pub async fn user_vault_equities(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userVaultEquities", "user": user}))
            .await
    }

    /// Get leading vaults
    pub async fn leading_vaults(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "leadingVaults", "user": user}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Delegation & Staking
    // ──────────────────────────────────────────────────────────────────────────

    /// Get delegations
    pub async fn delegations(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "delegations", "user": user}))
            .await
    }

    /// Get delegator summary
    pub async fn delegator_summary(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "delegatorSummary", "user": user}))
            .await
    }

    /// Get delegator history
    pub async fn delegator_history(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "delegatorHistory", "user": user}))
            .await
    }

    /// Get delegator rewards
    pub async fn delegator_rewards(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "delegatorRewards", "user": user}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // TWAP
    // ──────────────────────────────────────────────────────────────────────────

    /// Get user TWAP slice fills
    pub async fn user_twap_slice_fills(&self, user: &str, limit: Option<u32>) -> Result<Value> {
        let mut body = json!({"type": "userTwapSliceFills", "user": user});
        if let Some(l) = limit {
            body["limit"] = json!(l);
        }
        self.inner.query_info(&body).await
    }

    /// Get user TWAP history
    pub async fn user_twap_history(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userTwapHistory", "user": user}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Borrow/Lend
    // ──────────────────────────────────────────────────────────────────────────

    /// Get borrow/lend user state
    pub async fn borrow_lend_user_state(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "borrowLendUserState", "user": user}))
            .await
    }

    /// Get borrow/lend reserve state
    pub async fn borrow_lend_reserve_state(&self, token: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "borrowLendReserveState", "token": token}))
            .await
    }

    /// Get all borrow/lend reserve states
    pub async fn all_borrow_lend_reserve_states(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "allBorrowLendReserveStates"}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Account Abstraction
    // ──────────────────────────────────────────────────────────────────────────

    /// Get user abstraction mode
    pub async fn user_abstraction(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userAbstraction", "user": user}))
            .await
    }

    /// Get user DEX abstraction mode
    pub async fn user_dex_abstraction(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userDexAbstraction", "user": user}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Misc
    // ──────────────────────────────────────────────────────────────────────────

    /// Get liquidatable positions
    pub async fn liquidatable(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "liquidatable"}))
            .await
    }

    /// Get max market order notionals
    pub async fn max_market_order_ntls(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "maxMarketOrderNtls"}))
            .await
    }

    /// Get max builder fee
    pub async fn max_builder_fee(&self, user: &str, builder: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({
                "type": "maxBuilderFee",
                "user": user,
                "builder": builder,
            }))
            .await
    }

    /// Get active asset data
    pub async fn active_asset_data(&self, user: &str, asset: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({
                "type": "activeAssetData",
                "user": user,
                "asset": asset,
            }))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Tokens / Spot
    // ──────────────────────────────────────────────────────────────────────────

    /// Get token details
    pub async fn token_details(&self, token_id: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "tokenDetails", "tokenId": token_id}))
            .await
    }

    /// Get spot deployment state for user
    pub async fn spot_deploy_state(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "spotDeployState", "user": user}))
            .await
    }

    /// Get spot pair deploy auction status
    pub async fn spot_pair_deploy_auction_status(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "spotPairDeployAuctionStatus"}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Additional Methods
    // ──────────────────────────────────────────────────────────────────────────

    /// Get user's referral information
    pub async fn referral(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "referral", "user": user}))
            .await
    }

    /// Get user's non-funding ledger updates (deposits, withdrawals, transfers)
    pub async fn user_non_funding_ledger_updates(
        &self,
        user: &str,
        start_time: Option<u64>,
        end_time: Option<u64>,
    ) -> Result<Value> {
        let mut body = json!({"type": "userNonFundingLedgerUpdates", "user": user});
        if let Some(start) = start_time {
            body["startTime"] = json!(start);
        }
        if let Some(end) = end_time {
            body["endTime"] = json!(end);
        }
        self.inner.query_info(&body).await
    }

    /// Get multi-sig signers for a user
    pub async fn user_to_multi_sig_signers(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "userToMultiSigSigners", "user": user}))
            .await
    }

    /// Get gossip root IPs for the network
    pub async fn gossip_root_ips(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "gossipRootIps"}))
            .await
    }

    /// Get perpetual deploy auction status
    pub async fn perp_deploy_auction_status(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "perpDeployAuctionStatus"}))
            .await
    }

    /// Get perps that are at their open interest cap
    pub async fn perps_at_open_interest_cap(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "perpsAtOpenInterestCap"}))
            .await
    }

    /// Get L1 validator votes
    pub async fn validator_l1_votes(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "validatorL1Votes"}))
            .await
    }

    /// Get list of approved builders for a user
    pub async fn approved_builders(&self, user: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "approvedBuilders", "user": user}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Extended Perp DEX Info
    // ──────────────────────────────────────────────────────────────────────────

    /// Get consolidated universe, margin tables, asset contexts across all DEXs
    pub async fn all_perp_metas(&self) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "allPerpMetas"}))
            .await
    }

    /// Get OI caps and transfer limits for builder-deployed markets
    pub async fn perp_dex_limits(&self, dex: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "perpDexLimits", "dex": dex}))
            .await
    }

    /// Get total net deposits for builder-deployed markets
    pub async fn perp_dex_status(&self, dex: &str) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "perpDexStatus", "dex": dex}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Aligned Quote Token
    // ──────────────────────────────────────────────────────────────────────────

    /// Get aligned quote token information
    pub async fn aligned_quote_token_info(&self, token: u32) -> Result<Value> {
        self.inner
            .query_info(&json!({"type": "alignedQuoteTokenInfo", "token": token}))
            .await
    }
}
