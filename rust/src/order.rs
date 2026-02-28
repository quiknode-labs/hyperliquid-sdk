//! Order builders for the Hyperliquid SDK.
//!
//! Provides fluent API for building orders:
//!
//! ```rust,ignore
//! use hyperliquid_sdk::Order;
//!
//! // Simple limit order
//! let order = Order::buy("BTC").size(0.001).price(65000.0).gtc();
//!
//! // Market order with notional
//! let order = Order::sell("ETH").notional(500.0).market();
//!
//! // Stop loss
//! let trigger = TriggerOrder::stop_loss("BTC").size(0.001).trigger_price(60000.0).market();
//! ```

use alloy::primitives::B128;
use rust_decimal::Decimal;
use std::str::FromStr;
use std::sync::Arc;

use crate::types::{
    Cloid, OrderRequest, OrderTypePlacement, Side, TIF, TimeInForce, TpSl,
};

// ══════════════════════════════════════════════════════════════════════════════
// Order Builder
// ══════════════════════════════════════════════════════════════════════════════

/// Fluent order builder
#[derive(Debug, Clone)]
pub struct Order {
    asset: String,
    side: Side,
    size: Option<Decimal>,
    notional: Option<Decimal>,
    price: Option<Decimal>,
    tif: TIF,
    reduce_only: bool,
    cloid: Option<Cloid>,
}

impl Order {
    /// Create a buy order
    pub fn buy(asset: impl Into<String>) -> Self {
        Self::new(asset.into(), Side::Buy)
    }

    /// Create a sell order
    pub fn sell(asset: impl Into<String>) -> Self {
        Self::new(asset.into(), Side::Sell)
    }

    /// Alias for buy (for perps)
    pub fn long(asset: impl Into<String>) -> Self {
        Self::buy(asset)
    }

    /// Alias for sell (for perps)
    pub fn short(asset: impl Into<String>) -> Self {
        Self::sell(asset)
    }

    fn new(asset: String, side: Side) -> Self {
        Self {
            asset,
            side,
            size: None,
            notional: None,
            price: None,
            tif: TIF::Ioc,
            reduce_only: false,
            cloid: None,
        }
    }

    /// Set the order size (in base asset units)
    pub fn size(mut self, size: f64) -> Self {
        self.size = Some(Decimal::from_f64_retain(size).unwrap_or_default());
        self
    }

    /// Set the order size from a Decimal
    pub fn size_decimal(mut self, size: Decimal) -> Self {
        self.size = Some(size);
        self
    }

    /// Set the notional value (in USD)
    pub fn notional(mut self, notional: f64) -> Self {
        self.notional = Some(Decimal::from_f64_retain(notional).unwrap_or_default());
        self
    }

    /// Set the limit price
    pub fn price(mut self, price: f64) -> Self {
        self.price = Some(Decimal::from_f64_retain(price).unwrap_or_default());
        self
    }

    /// Alias for price
    pub fn limit(self, price: f64) -> Self {
        self.price(price)
    }

    /// Set as market order (IOC with slippage applied by SDK)
    pub fn market(mut self) -> Self {
        self.tif = TIF::Market;
        self
    }

    /// Set as Immediate-or-Cancel
    pub fn ioc(mut self) -> Self {
        self.tif = TIF::Ioc;
        self
    }

    /// Set as Good-Till-Cancel
    pub fn gtc(mut self) -> Self {
        self.tif = TIF::Gtc;
        self
    }

    /// Set as Add-Liquidity-Only (post-only)
    pub fn alo(mut self) -> Self {
        self.tif = TIF::Alo;
        self
    }

    /// Alias for alo
    pub fn post_only(self) -> Self {
        self.alo()
    }

    /// Set as reduce-only
    pub fn reduce_only(mut self) -> Self {
        self.reduce_only = true;
        self
    }

    /// Set a client order ID
    pub fn cloid(mut self, cloid: impl Into<String>) -> Self {
        let cloid_str = cloid.into();
        if let Ok(parsed) = cloid_str.parse::<B128>() {
            self.cloid = Some(parsed);
        }
        self
    }

    /// Set a client order ID from bytes
    pub fn cloid_bytes(mut self, cloid: [u8; 16]) -> Self {
        self.cloid = Some(B128::from(cloid));
        self
    }

    /// Generate a random client order ID
    pub fn random_cloid(mut self) -> Self {
        let mut bytes = [0u8; 16];
        // Simple random using time
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default();
        let nanos = now.as_nanos() as u64;
        bytes[0..8].copy_from_slice(&nanos.to_le_bytes());
        bytes[8..16].copy_from_slice(&(nanos.wrapping_mul(0x517cc1b727220a95)).to_le_bytes());
        self.cloid = Some(B128::from(bytes));
        self
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Getters
    // ──────────────────────────────────────────────────────────────────────────

    /// Get the asset
    pub fn get_asset(&self) -> &str {
        &self.asset
    }

    /// Get the side
    pub fn get_side(&self) -> Side {
        self.side
    }

    /// Get the size (if set)
    pub fn get_size(&self) -> Option<Decimal> {
        self.size
    }

    /// Get the notional (if set)
    pub fn get_notional(&self) -> Option<Decimal> {
        self.notional
    }

    /// Get the price (if set)
    pub fn get_price(&self) -> Option<Decimal> {
        self.price
    }

    /// Get the time-in-force
    pub fn get_tif(&self) -> TIF {
        self.tif
    }

    /// Is this a reduce-only order?
    pub fn is_reduce_only(&self) -> bool {
        self.reduce_only
    }

    /// Is this a market order?
    pub fn is_market(&self) -> bool {
        self.tif == TIF::Market || self.price.is_none()
    }

    /// Get the client order ID (if set)
    pub fn get_cloid(&self) -> Option<Cloid> {
        self.cloid
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Conversion
    // ──────────────────────────────────────────────────────────────────────────

    /// Validate the order
    pub fn validate(&self) -> crate::Result<()> {
        if self.size.is_none() && self.notional.is_none() {
            return Err(crate::Error::ValidationError(
                "Order must have either size or notional".to_string(),
            ));
        }

        if self.tif != TIF::Market && self.price.is_none() && self.notional.is_none() {
            return Err(crate::Error::ValidationError(
                "Non-market orders must have a price".to_string(),
            ));
        }

        Ok(())
    }

    /// Convert to an OrderRequest (requires asset index resolution)
    pub fn to_request(&self, asset_index: usize, resolved_price: Decimal) -> OrderRequest {
        // Generate random cloid if not set (Hyperliquid requires nonzero cloid)
        let cloid = self.cloid.unwrap_or_else(|| {
            let mut bytes = [0u8; 16];
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default();
            let nanos = now.as_nanos() as u64;
            bytes[0..8].copy_from_slice(&nanos.to_le_bytes());
            bytes[8..16].copy_from_slice(&(nanos.wrapping_mul(0x517cc1b727220a95)).to_le_bytes());
            B128::from(bytes)
        });

        OrderRequest {
            asset: asset_index,
            is_buy: self.side.is_buy(),
            limit_px: resolved_price,
            sz: self.size.unwrap_or_default(),
            reduce_only: self.reduce_only,
            order_type: OrderTypePlacement::Limit {
                tif: TimeInForce::from(self.tif),
            },
            cloid,
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Trigger Order Builder
// ══════════════════════════════════════════════════════════════════════════════

/// Fluent trigger order builder (stop-loss / take-profit)
#[derive(Debug, Clone)]
pub struct TriggerOrder {
    asset: String,
    tpsl: TpSl,
    side: Side,
    size: Option<Decimal>,
    trigger_price: Option<Decimal>,
    limit_price: Option<Decimal>,
    is_market: bool,
    reduce_only: bool,
    cloid: Option<Cloid>,
}

impl TriggerOrder {
    /// Create a stop-loss order
    pub fn stop_loss(asset: impl Into<String>) -> Self {
        Self::new(asset.into(), TpSl::Sl)
    }

    /// Alias for stop_loss
    pub fn sl(asset: impl Into<String>) -> Self {
        Self::stop_loss(asset)
    }

    /// Create a take-profit order
    pub fn take_profit(asset: impl Into<String>) -> Self {
        Self::new(asset.into(), TpSl::Tp)
    }

    /// Alias for take_profit
    pub fn tp(asset: impl Into<String>) -> Self {
        Self::take_profit(asset)
    }

    fn new(asset: String, tpsl: TpSl) -> Self {
        Self {
            asset,
            tpsl,
            side: Side::Sell, // Default to sell (closing a long)
            size: None,
            trigger_price: None,
            limit_price: None,
            is_market: true,
            reduce_only: true, // Default to reduce-only
            cloid: None,
        }
    }

    /// Set the order side
    pub fn side(mut self, side: Side) -> Self {
        self.side = side;
        self
    }

    /// Set to buy side
    pub fn buy(mut self) -> Self {
        self.side = Side::Buy;
        self
    }

    /// Set to sell side
    pub fn sell(mut self) -> Self {
        self.side = Side::Sell;
        self
    }

    /// Set the order size
    pub fn size(mut self, size: f64) -> Self {
        self.size = Some(Decimal::from_f64_retain(size).unwrap_or_default());
        self
    }

    /// Set the trigger price
    pub fn trigger_price(mut self, price: f64) -> Self {
        self.trigger_price = Some(Decimal::from_f64_retain(price).unwrap_or_default());
        self
    }

    /// Alias for trigger_price
    pub fn trigger(self, price: f64) -> Self {
        self.trigger_price(price)
    }

    /// Set as market execution (when triggered)
    pub fn market(mut self) -> Self {
        self.is_market = true;
        self.limit_price = None;
        self
    }

    /// Set limit price (when triggered)
    pub fn limit(mut self, price: f64) -> Self {
        self.is_market = false;
        self.limit_price = Some(Decimal::from_f64_retain(price).unwrap_or_default());
        self
    }

    /// Set as reduce-only
    pub fn reduce_only(mut self) -> Self {
        self.reduce_only = true;
        self
    }

    /// Allow increasing position
    pub fn not_reduce_only(mut self) -> Self {
        self.reduce_only = false;
        self
    }

    /// Set a client order ID
    pub fn cloid(mut self, cloid: impl Into<String>) -> Self {
        let cloid_str = cloid.into();
        if let Ok(parsed) = cloid_str.parse::<B128>() {
            self.cloid = Some(parsed);
        }
        self
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Getters
    // ──────────────────────────────────────────────────────────────────────────

    /// Get the asset
    pub fn get_asset(&self) -> &str {
        &self.asset
    }

    /// Get the trigger type
    pub fn get_tpsl(&self) -> TpSl {
        self.tpsl
    }

    /// Get the side
    pub fn get_side(&self) -> Side {
        self.side
    }

    /// Get the size
    pub fn get_size(&self) -> Option<Decimal> {
        self.size
    }

    /// Get the trigger price
    pub fn get_trigger_price(&self) -> Option<Decimal> {
        self.trigger_price
    }

    /// Get the limit price
    pub fn get_limit_price(&self) -> Option<Decimal> {
        self.limit_price
    }

    /// Is market execution?
    pub fn is_market(&self) -> bool {
        self.is_market
    }

    /// Is reduce-only?
    pub fn is_reduce_only(&self) -> bool {
        self.reduce_only
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Validation
    // ──────────────────────────────────────────────────────────────────────────

    /// Validate the trigger order
    pub fn validate(&self) -> crate::Result<()> {
        if self.size.is_none() {
            return Err(crate::Error::ValidationError(
                "Trigger order must have a size".to_string(),
            ));
        }

        if self.trigger_price.is_none() {
            return Err(crate::Error::ValidationError(
                "Trigger order must have a trigger price".to_string(),
            ));
        }

        Ok(())
    }

    /// Convert to an OrderRequest
    pub fn to_request(&self, asset_index: usize, execution_price: Decimal) -> OrderRequest {
        OrderRequest {
            asset: asset_index,
            is_buy: self.side.is_buy(),
            limit_px: self.limit_price.unwrap_or(execution_price),
            sz: self.size.unwrap_or_default(),
            reduce_only: self.reduce_only,
            order_type: OrderTypePlacement::Trigger {
                is_market: self.is_market,
                trigger_px: self.trigger_price.unwrap_or_default(),
                tpsl: self.tpsl,
            },
            cloid: self.cloid.unwrap_or(B128::ZERO),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Placed Order
// ══════════════════════════════════════════════════════════════════════════════

/// A placed order with methods for cancellation and modification.
#[derive(Debug, Clone)]
pub struct PlacedOrder {
    /// Order ID (None if order failed)
    pub oid: Option<u64>,
    /// Order status
    pub status: String,
    /// Asset name
    pub asset: String,
    /// Order side
    pub side: String,
    /// Order size
    pub size: String,
    /// Limit price (if applicable)
    pub price: Option<String>,
    /// Filled size
    pub filled_size: Option<String>,
    /// Average fill price
    pub avg_price: Option<String>,
    /// Error message (if failed)
    pub error: Option<String>,
    /// Raw response from the API
    pub raw_response: serde_json::Value,
    /// Reference to SDK for cancel/modify operations
    sdk: Option<Arc<crate::client::HyperliquidSDKInner>>,
}

impl PlacedOrder {
    /// Create from API response
    pub(crate) fn from_response(
        response: serde_json::Value,
        asset: String,
        side: Side,
        size: Decimal,
        price: Option<Decimal>,
        sdk: Option<Arc<crate::client::HyperliquidSDKInner>>,
    ) -> Self {
        // Parse response to extract order details
        let status = response
            .get("status")
            .and_then(|s| s.as_str())
            .unwrap_or("unknown")
            .to_string();

        let mut oid = None;
        let mut filled_size = None;
        let mut avg_price = None;
        let mut error = None;

        if status == "ok" {
            // Extract from response.response.data.statuses[0]
            if let Some(data) = response
                .get("response")
                .and_then(|r| r.get("data"))
                .and_then(|d| d.get("statuses"))
                .and_then(|s| s.get(0))
            {
                // Check for "resting" status
                if let Some(resting) = data.get("resting") {
                    oid = resting.get("oid").and_then(|o| o.as_u64());
                }
                // Check for "filled" status
                if let Some(filled) = data.get("filled") {
                    oid = filled.get("oid").and_then(|o| o.as_u64());
                    filled_size = filled
                        .get("totalSz")
                        .and_then(|s| s.as_str())
                        .map(|s| s.to_string());
                    avg_price = filled
                        .get("avgPx")
                        .and_then(|p| p.as_str())
                        .map(|s| s.to_string());
                }
                // Check for error
                if let Some(err) = data.get("error") {
                    error = err.as_str().map(|s| s.to_string());
                }
            }
        } else if status == "err" {
            error = response
                .get("response")
                .and_then(|r| r.as_str())
                .map(|s| s.to_string());
        }

        let status_str = if oid.is_some() {
            if filled_size.is_some() {
                "filled"
            } else {
                "resting"
            }
        } else if error.is_some() {
            "error"
        } else {
            "unknown"
        };

        Self {
            oid,
            status: status_str.to_string(),
            asset,
            side: side.to_string(),
            size: size.to_string(),
            price: price.map(|p| p.to_string()),
            filled_size,
            avg_price,
            error,
            raw_response: response,
            sdk,
        }
    }

    /// Create an error response
    #[allow(dead_code)]
    pub(crate) fn error(
        asset: String,
        side: Side,
        size: Decimal,
        error: String,
    ) -> Self {
        Self {
            oid: None,
            status: "error".to_string(),
            asset,
            side: side.to_string(),
            size: size.to_string(),
            price: None,
            filled_size: None,
            avg_price: None,
            error: Some(error),
            raw_response: serde_json::Value::Null,
            sdk: None,
        }
    }

    /// Is the order resting on the book?
    pub fn is_resting(&self) -> bool {
        self.status == "resting"
    }

    /// Is the order fully filled?
    pub fn is_filled(&self) -> bool {
        self.status == "filled"
    }

    /// Did the order fail?
    pub fn is_error(&self) -> bool {
        self.status == "error" || self.error.is_some()
    }

    /// Cancel this order
    pub async fn cancel(&self) -> crate::Result<serde_json::Value> {
        let oid = self.oid.ok_or_else(|| {
            crate::Error::OrderError("Cannot cancel order without OID".to_string())
        })?;

        let sdk = self.sdk.as_ref().ok_or_else(|| {
            crate::Error::OrderError("SDK reference not available for cancel".to_string())
        })?;

        sdk.cancel_by_oid(oid, &self.asset).await
    }

    /// Modify this order
    pub async fn modify(
        &self,
        price: Option<f64>,
        size: Option<f64>,
    ) -> crate::Result<PlacedOrder> {
        let oid = self.oid.ok_or_else(|| {
            crate::Error::OrderError("Cannot modify order without OID".to_string())
        })?;

        let sdk = self.sdk.as_ref().ok_or_else(|| {
            crate::Error::OrderError("SDK reference not available for modify".to_string())
        })?;

        let new_price = price
            .map(|p| Decimal::from_f64_retain(p).unwrap_or_default())
            .or_else(|| self.price.as_ref().and_then(|p| Decimal::from_str(p).ok()));

        let new_size = size
            .map(|s| Decimal::from_f64_retain(s).unwrap_or_default())
            .or_else(|| Decimal::from_str(&self.size).ok());

        sdk.modify_by_oid(
            oid,
            &self.asset,
            Side::from_str(&self.side).unwrap_or(Side::Buy),
            new_price.unwrap_or_default(),
            new_size.unwrap_or_default(),
        )
        .await
    }
}
