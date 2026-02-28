//! HyperCore API client for Hyperliquid.
//!
//! Provides access to block-level data, trades, orders, and book updates.

use serde_json::{json, Value};
use std::sync::Arc;

use crate::client::HyperliquidSDKInner;
use crate::error::Result;

/// HyperCore API client
pub struct HyperCore {
    inner: Arc<HyperliquidSDKInner>,
}

impl HyperCore {
    pub(crate) fn new(inner: Arc<HyperliquidSDKInner>) -> Self {
        Self { inner }
    }

    /// Get the HyperCore endpoint URL
    fn hypercore_url(&self) -> String {
        self.inner.hypercore_url()
    }

    /// Make a JSON-RPC request to HyperCore
    async fn rpc(&self, method: &str, params: Value) -> Result<Value> {
        let url = self.hypercore_url();

        let body = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1,
        });

        let response = self
            .inner
            .http_client
            .post(&url)
            .json(&body)
            .send()
            .await?;

        let status = response.status();
        let text = response.text().await?;

        if !status.is_success() {
            return Err(crate::Error::NetworkError(format!(
                "HyperCore request failed {}: {}",
                status, text
            )));
        }

        let result: Value = serde_json::from_str(&text)?;

        // Check for JSON-RPC error
        if let Some(error) = result.get("error") {
            let message = error
                .get("message")
                .and_then(|m| m.as_str())
                .unwrap_or("Unknown error");
            return Err(crate::Error::ApiError {
                code: crate::error::ErrorCode::Unknown,
                message: message.to_string(),
                guidance: "Check your HyperCore request parameters.".to_string(),
                raw: Some(error.to_string()),
            });
        }

        Ok(result.get("result").cloned().unwrap_or(Value::Null))
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Block Data
    // ──────────────────────────────────────────────────────────────────────────

    /// Get the latest block number for a stream
    pub async fn latest_block_number(&self, stream: Option<&str>) -> Result<u64> {
        let stream = stream.unwrap_or("trades");
        let result = self.rpc("hl_getLatestBlockNumber", json!({"stream": stream})).await?;
        result
            .as_u64()
            .ok_or_else(|| crate::Error::JsonError("Invalid block number".to_string()))
    }

    /// Get a specific block
    pub async fn get_block(&self, block_number: u64, stream: Option<&str>) -> Result<Value> {
        let stream = stream.unwrap_or("trades");
        self.rpc("hl_getBlock", json!([stream, block_number])).await
    }

    /// Get a batch of blocks
    pub async fn get_batch_blocks(
        &self,
        from_block: u64,
        to_block: u64,
        stream: Option<&str>,
    ) -> Result<Value> {
        let stream = stream.unwrap_or("trades");
        self.rpc("hl_getBatchBlocks", json!({"stream": stream, "from": from_block, "to": to_block}))
            .await
    }

    /// Get latest blocks
    pub async fn latest_blocks(&self, stream: Option<&str>, count: Option<u32>) -> Result<Value> {
        let stream = stream.unwrap_or("trades");
        let count = count.unwrap_or(10);
        self.rpc("hl_getLatestBlocks", json!({"stream": stream, "count": count})).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Recent Data
    // ──────────────────────────────────────────────────────────────────────────

    /// Get latest trades from recent blocks
    /// This is an alternative to Info.recent_trades() for QuickNode endpoints
    pub async fn latest_trades(&self, count: Option<u32>, coin: Option<&str>) -> Result<Value> {
        let count = count.unwrap_or(10);
        let blocks = self.latest_blocks(Some("trades"), Some(count)).await?;

        let mut trades = Vec::new();
        if let Some(blocks_arr) = blocks.get("blocks").and_then(|b| b.as_array()) {
            for block in blocks_arr {
                if let Some(events) = block.get("events").and_then(|e| e.as_array()) {
                    for event in events {
                        if let Some(arr) = event.as_array() {
                            if arr.len() >= 2 {
                                let user = &arr[0];
                                if let Some(trade) = arr.get(1) {
                                    // Apply coin filter if specified
                                    if let Some(c) = coin {
                                        if trade.get("coin").and_then(|tc| tc.as_str()) != Some(c) {
                                            continue;
                                        }
                                    }
                                    let mut trade_obj = trade.clone();
                                    if let Some(obj) = trade_obj.as_object_mut() {
                                        obj.insert("user".to_string(), user.clone());
                                    }
                                    trades.push(trade_obj);
                                }
                            }
                        }
                    }
                }
            }
        }
        Ok(json!(trades))
    }

    /// Get latest orders from recent blocks
    pub async fn latest_orders(&self, count: Option<u32>) -> Result<Value> {
        let count = count.unwrap_or(10);
        let blocks = self.latest_blocks(Some("orders"), Some(count)).await?;

        let mut orders = Vec::new();
        if let Some(blocks_arr) = blocks.get("blocks").and_then(|b| b.as_array()) {
            for block in blocks_arr {
                if let Some(events) = block.get("events").and_then(|e| e.as_array()) {
                    for event in events {
                        if let Some(arr) = event.as_array() {
                            if arr.len() >= 2 {
                                let user = &arr[0];
                                if let Some(order) = arr.get(1) {
                                    let mut order_obj = order.clone();
                                    if let Some(obj) = order_obj.as_object_mut() {
                                        obj.insert("user".to_string(), user.clone());
                                    }
                                    orders.push(order_obj);
                                }
                            }
                        }
                    }
                }
            }
        }
        Ok(json!(orders))
    }

    /// Get latest book updates from recent blocks
    pub async fn latest_book_updates(&self, count: Option<u32>, coin: Option<&str>) -> Result<Value> {
        let count = count.unwrap_or(10);
        let blocks = self.latest_blocks(Some("book_updates"), Some(count)).await?;

        let mut updates = Vec::new();
        if let Some(blocks_arr) = blocks.get("blocks").and_then(|b| b.as_array()) {
            for block in blocks_arr {
                if let Some(events) = block.get("events").and_then(|e| e.as_array()) {
                    for event in events {
                        // Apply coin filter if specified
                        if let Some(c) = coin {
                            if event.get("coin").and_then(|ec| ec.as_str()) != Some(c) {
                                continue;
                            }
                        }
                        updates.push(event.clone());
                    }
                }
            }
        }
        Ok(json!(updates))
    }

    /// Get latest TWAP updates
    pub async fn latest_twap(&self, count: Option<u32>) -> Result<Value> {
        let count = count.unwrap_or(10);
        self.latest_blocks(Some("twap"), Some(count)).await
    }

    /// Get latest events
    pub async fn latest_events(&self, count: Option<u32>) -> Result<Value> {
        let count = count.unwrap_or(10);
        self.latest_blocks(Some("events"), Some(count)).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Discovery
    // ──────────────────────────────────────────────────────────────────────────

    /// List all DEXes
    pub async fn list_dexes(&self) -> Result<Value> {
        self.rpc("hl_listDexes", json!({})).await
    }

    /// List all markets (optionally for a specific DEX)
    pub async fn list_markets(&self, dex: Option<&str>) -> Result<Value> {
        if let Some(d) = dex {
            self.rpc("hl_listMarkets", json!({"dex": d})).await
        } else {
            self.rpc("hl_listMarkets", json!({})).await
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Order Queries
    // ──────────────────────────────────────────────────────────────────────────

    /// Get open orders for a user
    pub async fn open_orders(&self, user: &str) -> Result<Value> {
        self.rpc("hl_openOrders", json!({"user": user})).await
    }

    /// Get status of a specific order
    pub async fn order_status(&self, user: &str, oid: u64) -> Result<Value> {
        self.rpc("hl_orderStatus", json!({"user": user, "oid": oid}))
            .await
    }

    /// Validate an order before signing (preflight check)
    pub async fn preflight(
        &self,
        coin: &str,
        is_buy: bool,
        limit_px: &str,
        sz: &str,
        user: &str,
        reduce_only: bool,
        order_type: Option<&Value>,
    ) -> Result<Value> {
        let mut params = json!({
            "coin": coin,
            "isBuy": is_buy,
            "limitPx": limit_px,
            "sz": sz,
            "user": user,
            "reduceOnly": reduce_only,
        });
        if let Some(ot) = order_type {
            params["orderType"] = ot.clone();
        }
        self.rpc("hl_preflight", params).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Builder Fee
    // ──────────────────────────────────────────────────────────────────────────

    /// Get maximum builder fee for a user-builder pair
    pub async fn get_max_builder_fee(&self, user: &str, builder: &str) -> Result<Value> {
        self.rpc("hl_getMaxBuilderFee", json!({"user": user, "builder": builder}))
            .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Order Building (Returns unsigned actions for signing)
    // ──────────────────────────────────────────────────────────────────────────

    /// Build an order for signing
    pub async fn build_order(
        &self,
        coin: &str,
        is_buy: bool,
        limit_px: &str,
        sz: &str,
        user: &str,
        reduce_only: bool,
        order_type: Option<&Value>,
        cloid: Option<&str>,
    ) -> Result<Value> {
        let mut params = json!({
            "coin": coin,
            "isBuy": is_buy,
            "limitPx": limit_px,
            "sz": sz,
            "user": user,
            "reduceOnly": reduce_only,
        });
        if let Some(ot) = order_type {
            params["orderType"] = ot.clone();
        }
        if let Some(c) = cloid {
            params["cloid"] = json!(c);
        }
        self.rpc("hl_buildOrder", params).await
    }

    /// Build a cancel action for signing
    pub async fn build_cancel(&self, coin: &str, oid: u64, user: &str) -> Result<Value> {
        self.rpc("hl_buildCancel", json!({"coin": coin, "oid": oid, "user": user}))
            .await
    }

    /// Build a modify action for signing
    pub async fn build_modify(
        &self,
        coin: &str,
        oid: u64,
        user: &str,
        limit_px: Option<&str>,
        sz: Option<&str>,
        is_buy: Option<bool>,
    ) -> Result<Value> {
        let mut params = json!({"coin": coin, "oid": oid, "user": user});
        if let Some(px) = limit_px {
            params["limitPx"] = json!(px);
        }
        if let Some(s) = sz {
            params["sz"] = json!(s);
        }
        if let Some(buy) = is_buy {
            params["isBuy"] = json!(buy);
        }
        self.rpc("hl_buildModify", params).await
    }

    /// Build a builder fee approval for signing
    pub async fn build_approve_builder_fee(
        &self,
        user: &str,
        builder: &str,
        max_fee_rate: &str,
        nonce: u64,
    ) -> Result<Value> {
        self.rpc(
            "hl_buildApproveBuilderFee",
            json!({
                "user": user,
                "builder": builder,
                "maxFeeRate": max_fee_rate,
                "nonce": nonce,
            }),
        )
        .await
    }

    /// Build a builder fee revocation for signing
    pub async fn build_revoke_builder_fee(
        &self,
        user: &str,
        builder: &str,
        nonce: u64,
    ) -> Result<Value> {
        self.rpc(
            "hl_buildRevokeBuilderFee",
            json!({
                "user": user,
                "builder": builder,
                "nonce": nonce,
            }),
        )
        .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Sending Signed Actions
    // ──────────────────────────────────────────────────────────────────────────

    /// Send a signed order
    pub async fn send_order(
        &self,
        action: &Value,
        signature: &str,
        nonce: u64,
    ) -> Result<Value> {
        self.rpc(
            "hl_sendOrder",
            json!({"action": action, "signature": signature, "nonce": nonce}),
        )
        .await
    }

    /// Send a signed cancel
    pub async fn send_cancel(
        &self,
        action: &Value,
        signature: &str,
        nonce: u64,
    ) -> Result<Value> {
        self.rpc(
            "hl_sendCancel",
            json!({"action": action, "signature": signature, "nonce": nonce}),
        )
        .await
    }

    /// Send a signed modify
    pub async fn send_modify(
        &self,
        action: &Value,
        signature: &str,
        nonce: u64,
    ) -> Result<Value> {
        self.rpc(
            "hl_sendModify",
            json!({"action": action, "signature": signature, "nonce": nonce}),
        )
        .await
    }

    /// Send a signed builder fee approval
    pub async fn send_approval(&self, action: &Value, signature: &str) -> Result<Value> {
        self.rpc(
            "hl_sendApproval",
            json!({"action": action, "signature": signature}),
        )
        .await
    }

    /// Send a signed builder fee revocation
    pub async fn send_revocation(&self, action: &Value, signature: &str) -> Result<Value> {
        self.rpc(
            "hl_sendRevocation",
            json!({"action": action, "signature": signature}),
        )
        .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // WebSocket Subscriptions (via JSON-RPC)
    // ──────────────────────────────────────────────────────────────────────────

    /// Subscribe to a WebSocket stream via JSON-RPC
    pub async fn subscribe(&self, subscription: &Value) -> Result<Value> {
        self.rpc("hl_subscribe", json!({"subscription": subscription}))
            .await
    }

    /// Unsubscribe from a WebSocket stream
    pub async fn unsubscribe(&self, subscription: &Value) -> Result<Value> {
        self.rpc("hl_unsubscribe", json!({"subscription": subscription}))
            .await
    }
}
