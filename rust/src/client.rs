//! Main SDK client for Hyperliquid.
//!
//! Provides a unified interface for all Hyperliquid operations.

use alloy::primitives::Address;
use alloy::signers::local::PrivateKeySigner;
use dashmap::DashMap;
use parking_lot::RwLock;
use reqwest::Client;
use rust_decimal::Decimal;
use serde_json::{json, Value};
use std::collections::HashMap;
use std::str::FromStr;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use crate::error::{Error, Result};
use crate::order::{Order, PlacedOrder, TriggerOrder};
use crate::signing::sign_hash;
use crate::types::*;

// ══════════════════════════════════════════════════════════════════════════════
// Constants
// ══════════════════════════════════════════════════════════════════════════════

const DEFAULT_WORKER_URL: &str = "https://send.hyperliquidapi.com";
const DEFAULT_WORKER_INFO_URL: &str = "https://send.hyperliquidapi.com/info";

/// Known path segments that are NOT tokens
const KNOWN_PATHS: &[&str] = &["info", "hypercore", "evm", "nanoreth", "ws", "send"];
const HL_INFO_URL: &str = "https://api.hyperliquid.xyz/info";
#[allow(dead_code)]
const HL_EXCHANGE_URL: &str = "https://api.hyperliquid.xyz/exchange";
const DEFAULT_SLIPPAGE: f64 = 0.03; // 3%
const DEFAULT_TIMEOUT_SECS: u64 = 30;
const METADATA_CACHE_TTL_SECS: u64 = 300; // 5 minutes

// QuickNode-supported info query types
const QN_SUPPORTED_INFO_TYPES: &[&str] = &[
    "meta",
    "spotMeta",
    "clearinghouseState",
    "spotClearinghouseState",
    "openOrders",
    "exchangeStatus",
    "frontendOpenOrders",
    "liquidatable",
    "activeAssetData",
    "maxMarketOrderNtls",
    "vaultSummaries",
    "userVaultEquities",
    "leadingVaults",
    "extraAgents",
    "subAccounts",
    "userFees",
    "userRateLimit",
    "spotDeployState",
    "perpDeployAuctionStatus",
    "delegations",
    "delegatorSummary",
    "maxBuilderFee",
    "userToMultiSigSigners",
    "userRole",
    "perpsAtOpenInterestCap",
    "validatorL1Votes",
    "marginTable",
    "perpDexs",
    "webData2",
];

// ══════════════════════════════════════════════════════════════════════════════
// Asset Metadata
// ══════════════════════════════════════════════════════════════════════════════

/// Asset metadata information
#[derive(Debug, Clone)]
pub struct AssetInfo {
    pub index: usize,
    pub name: String,
    pub sz_decimals: u8,
    pub is_spot: bool,
}

/// Metadata cache
#[derive(Debug, Default)]
pub struct MetadataCache {
    assets: RwLock<HashMap<String, AssetInfo>>,
    assets_by_index: RwLock<HashMap<usize, AssetInfo>>,
    dexes: RwLock<Vec<String>>,
    last_update: RwLock<Option<SystemTime>>,
}

impl MetadataCache {
    /// Get asset info by name
    pub fn get_asset(&self, name: &str) -> Option<AssetInfo> {
        self.assets.read().get(name).cloned()
    }

    /// Get asset info by index
    pub fn get_asset_by_index(&self, index: usize) -> Option<AssetInfo> {
        self.assets_by_index.read().get(&index).cloned()
    }

    /// Resolve asset name to index
    pub fn resolve_asset(&self, name: &str) -> Option<usize> {
        self.assets.read().get(name).map(|a| a.index)
    }

    /// Get all DEX names
    pub fn get_dexes(&self) -> Vec<String> {
        self.dexes.read().clone()
    }

    /// Check if cache is valid
    pub fn is_valid(&self) -> bool {
        if let Some(last) = *self.last_update.read() {
            if let Ok(elapsed) = last.elapsed() {
                return elapsed.as_secs() < METADATA_CACHE_TTL_SECS;
            }
        }
        false
    }

    /// Update cache from API response
    pub fn update(&self, meta: &Value, spot_meta: Option<&Value>, dexes: &[String]) {
        let mut assets = HashMap::new();
        let mut assets_by_index = HashMap::new();

        // Parse perp assets
        if let Some(universe) = meta.get("universe").and_then(|u| u.as_array()) {
            for (i, asset) in universe.iter().enumerate() {
                if let Some(name) = asset.get("name").and_then(|n| n.as_str()) {
                    let sz_decimals = asset
                        .get("szDecimals")
                        .and_then(|d| d.as_u64())
                        .unwrap_or(8) as u8;

                    let info = AssetInfo {
                        index: i,
                        name: name.to_string(),
                        sz_decimals,
                        is_spot: false,
                    };
                    assets.insert(name.to_string(), info.clone());
                    assets_by_index.insert(i, info);
                }
            }
        }

        // Parse spot assets
        if let Some(spot) = spot_meta {
            if let Some(tokens) = spot.get("tokens").and_then(|t| t.as_array()) {
                for token in tokens {
                    if let (Some(name), Some(index)) = (
                        token.get("name").and_then(|n| n.as_str()),
                        token.get("index").and_then(|i| i.as_u64()),
                    ) {
                        let sz_decimals = token
                            .get("szDecimals")
                            .and_then(|d| d.as_u64())
                            .unwrap_or(8) as u8;

                        let info = AssetInfo {
                            index: index as usize,
                            name: name.to_string(),
                            sz_decimals,
                            is_spot: true,
                        };
                        assets.insert(name.to_string(), info.clone());
                        assets_by_index.insert(index as usize, info);
                    }
                }
            }
        }

        *self.assets.write() = assets;
        *self.assets_by_index.write() = assets_by_index;
        *self.dexes.write() = dexes.to_vec();
        *self.last_update.write() = Some(SystemTime::now());
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// SDK Inner (shared state)
// ══════════════════════════════════════════════════════════════════════════════

/// Parsed endpoint information
#[derive(Debug, Clone)]
pub struct EndpointInfo {
    /// Base URL (scheme + host)
    pub base: String,
    /// Token extracted from URL path (if any)
    pub token: Option<String>,
    /// Whether this is a QuickNode endpoint
    pub is_quicknode: bool,
}

impl EndpointInfo {
    /// Parse endpoint URL and extract token
    ///
    /// Handles URLs like:
    /// - `https://x.quiknode.pro/TOKEN/evm` -> base=`https://x.quiknode.pro`, token=`TOKEN`
    /// - `https://x.quiknode.pro/TOKEN` -> base=`https://x.quiknode.pro`, token=`TOKEN`
    /// - `https://api.hyperliquid.xyz/info` -> base=`https://api.hyperliquid.xyz`, token=None
    pub fn parse(url: &str) -> Self {
        let parsed = url::Url::parse(url).ok();

        if let Some(parsed) = parsed {
            let base = format!("{}://{}", parsed.scheme(), parsed.host_str().unwrap_or(""));
            let is_quicknode = parsed.host_str().map(|h| h.contains("quiknode.pro")).unwrap_or(false);

            // Extract path segments
            let path_parts: Vec<&str> = parsed.path()
                .trim_matches('/')
                .split('/')
                .filter(|p| !p.is_empty())
                .collect();

            // Find the token (first segment that's not a known path)
            let token = path_parts.iter()
                .find(|&part| !KNOWN_PATHS.contains(part))
                .map(|s| s.to_string());

            Self { base, token, is_quicknode }
        } else {
            // Fallback for unparseable URLs
            Self {
                base: url.to_string(),
                token: None,
                is_quicknode: url.contains("quiknode.pro"),
            }
        }
    }

    /// Build URL for a specific path suffix (e.g., "info", "hypercore", "evm")
    pub fn build_url(&self, suffix: &str) -> String {
        if let Some(ref token) = self.token {
            format!("{}/{}/{}", self.base, token, suffix)
        } else {
            format!("{}/{}", self.base, suffix)
        }
    }

    /// Build WebSocket URL
    pub fn build_ws_url(&self) -> String {
        let ws_base = self.base.replace("https://", "wss://").replace("http://", "ws://");
        if let Some(ref token) = self.token {
            format!("{}/{}/hypercore/ws", ws_base, token)
        } else {
            format!("{}/ws", ws_base)
        }
    }

    /// Build gRPC URL (uses port 10000)
    pub fn build_grpc_url(&self) -> String {
        // gRPC uses the same host but port 10000
        if let Some(ref token) = self.token {
            let grpc_base = self.base.replace(":443", "").replace("https://", "");
            format!("https://{}:10000/{}", grpc_base, token)
        } else {
            self.base.replace(":443", ":10000")
        }
    }
}

/// Shared SDK state
pub struct HyperliquidSDKInner {
    pub(crate) http_client: Client,
    pub(crate) signer: Option<PrivateKeySigner>,
    pub(crate) address: Option<Address>,
    pub(crate) chain: Chain,
    pub(crate) endpoint: Option<String>,
    pub(crate) endpoint_info: Option<EndpointInfo>,
    pub(crate) slippage: f64,
    pub(crate) metadata: MetadataCache,
    pub(crate) mid_prices: DashMap<String, f64>,
}

impl std::fmt::Debug for HyperliquidSDKInner {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("HyperliquidSDKInner")
            .field("address", &self.address)
            .field("chain", &self.chain)
            .field("endpoint", &self.endpoint)
            .field("slippage", &self.slippage)
            .finish_non_exhaustive()
    }
}

/// Exchange URL (worker handles ALL trading operations)
const DEFAULT_EXCHANGE_URL: &str = "https://send.hyperliquidapi.com/exchange";

impl HyperliquidSDKInner {
    /// Get the exchange endpoint URL (for sending orders)
    ///
    /// ALL trading/exchange operations go through the worker at
    /// `send.hyperliquidapi.com/exchange`. The QuickNode `/send` endpoint
    /// is NOT used - QuickNode endpoints are only for info/hypercore/evm APIs.
    fn exchange_url(&self) -> String {
        DEFAULT_EXCHANGE_URL.to_string()
    }

    /// Get the info endpoint URL for a query type
    fn info_url(&self, query_type: &str) -> String {
        if let Some(ref info) = self.endpoint_info {
            // QuickNode endpoint - check if query type is supported
            if info.is_quicknode && QN_SUPPORTED_INFO_TYPES.contains(&query_type) {
                return info.build_url("info");
            }
        }
        // Fall back to worker for unsupported methods (worker proxies to public HL endpoint)
        DEFAULT_WORKER_INFO_URL.to_string()
    }

    /// Get the HyperCore endpoint URL
    pub fn hypercore_url(&self) -> String {
        if let Some(ref info) = self.endpoint_info {
            if info.is_quicknode {
                return info.build_url("hypercore");
            }
        }
        // No public HyperCore endpoint - fall back to info
        HL_INFO_URL.to_string()
    }

    /// Get the EVM endpoint URL
    pub fn evm_url(&self, use_nanoreth: bool) -> String {
        if let Some(ref info) = self.endpoint_info {
            if info.is_quicknode {
                let suffix = if use_nanoreth { "nanoreth" } else { "evm" };
                return info.build_url(suffix);
            }
        }
        // Public EVM endpoints
        match self.chain {
            Chain::Mainnet => "https://rpc.hyperliquid.xyz/evm".to_string(),
            Chain::Testnet => "https://rpc.hyperliquid-testnet.xyz/evm".to_string(),
        }
    }

    /// Get the WebSocket URL
    pub fn ws_url(&self) -> String {
        if let Some(ref info) = self.endpoint_info {
            return info.build_ws_url();
        }
        // Public WebSocket
        "wss://api.hyperliquid.xyz/ws".to_string()
    }

    /// Get the gRPC URL
    pub fn grpc_url(&self) -> String {
        if let Some(ref info) = self.endpoint_info {
            if info.is_quicknode {
                return info.build_grpc_url();
            }
        }
        // No public gRPC endpoint
        String::new()
    }

    /// Make a POST request to the info endpoint
    pub async fn query_info(&self, body: &Value) -> Result<Value> {
        let query_type = body.get("type").and_then(|t| t.as_str()).unwrap_or("");
        let url = self.info_url(query_type);

        let response = self
            .http_client
            .post(&url)
            .json(body)
            .send()
            .await?;

        let status = response.status();
        let text = response.text().await?;

        if !status.is_success() {
            return Err(Error::NetworkError(format!(
                "Info endpoint returned {}: {}",
                status, text
            )));
        }

        serde_json::from_str(&text).map_err(|e| Error::JsonError(e.to_string()))
    }

    /// Build an action (get hash without sending)
    pub async fn build_action(&self, action: &Value, slippage: Option<f64>) -> Result<BuildResponse> {
        let url = self.exchange_url();

        let mut body = json!({ "action": action });
        if let Some(s) = slippage {
            body["slippage"] = json!(s);
        }

        let response = self
            .http_client
            .post(url)
            .json(&body)
            .send()
            .await?;

        let status = response.status();
        let text = response.text().await?;

        if !status.is_success() {
            return Err(Error::NetworkError(format!(
                "Build request failed {}: {}",
                status, text
            )));
        }

        let result: Value = serde_json::from_str(&text)?;

        // Check for error
        if let Some(error) = result.get("error") {
            return Err(Error::from_api_error(
                error.as_str().unwrap_or("Unknown error"),
            ));
        }

        Ok(BuildResponse {
            hash: result
                .get("hash")
                .and_then(|h| h.as_str())
                .unwrap_or("")
                .to_string(),
            nonce: result.get("nonce").and_then(|n| n.as_u64()).unwrap_or(0),
            action: result.get("action").cloned().unwrap_or(action.clone()),
        })
    }

    /// Send a signed action
    pub async fn send_action(
        &self,
        action: &Value,
        nonce: u64,
        signature: &Signature,
    ) -> Result<Value> {
        let url = self.exchange_url();

        let body = json!({
            "action": action,
            "nonce": nonce,
            "signature": signature,
        });

        let response = self
            .http_client
            .post(url)
            .json(&body)
            .send()
            .await?;

        let status = response.status();
        let text = response.text().await?;

        if !status.is_success() {
            return Err(Error::NetworkError(format!(
                "Send request failed {}: {}",
                status, text
            )));
        }

        let result: Value = serde_json::from_str(&text)?;

        // Check for API error
        if let Some(hl_status) = result.get("status") {
            if hl_status.as_str() == Some("err") {
                if let Some(response) = result.get("response") {
                    let raw = response.as_str()
                        .map(|s| s.to_string())
                        .unwrap_or_else(|| response.to_string());
                    return Err(Error::from_api_error(&raw));
                }
            }
        }

        Ok(result)
    }

    /// Build, sign, and send an action
    ///
    /// If `slippage` is `Some`, it is included in the build payload for the worker
    /// to apply when computing market order prices. When `None`, the constructor-level
    /// default slippage is used (if > 0).
    pub async fn build_sign_send(&self, action: &Value, slippage: Option<f64>) -> Result<Value> {
        let signer = self
            .signer
            .as_ref()
            .ok_or_else(|| Error::ConfigError("No private key configured".to_string()))?;

        // Resolve effective slippage: per-call override > constructor default > omit
        let effective_slippage = slippage.or_else(|| {
            if self.slippage > 0.0 {
                Some(self.slippage)
            } else {
                None
            }
        });

        // Step 1: Build
        let build_result = self.build_action(action, effective_slippage).await?;

        // Step 2: Sign
        let hash_bytes = hex::decode(build_result.hash.trim_start_matches("0x"))
            .map_err(|e| Error::SigningError(format!("Invalid hash: {}", e)))?;

        let hash = alloy::primitives::B256::from_slice(&hash_bytes);
        let signature = sign_hash(signer, hash).await?;

        // Step 3: Send
        self.send_action(&build_result.action, build_result.nonce, &signature)
            .await
    }

    /// Refresh metadata cache
    pub async fn refresh_metadata(&self) -> Result<()> {
        // Fetch perp meta
        let meta = self.query_info(&json!({"type": "meta"})).await?;

        // Fetch spot meta
        let spot_meta = self.query_info(&json!({"type": "spotMeta"})).await.ok();

        // Fetch DEXes
        let dexes_result = self.query_info(&json!({"type": "perpDexs"})).await.ok();
        let dexes: Vec<String> = dexes_result
            .and_then(|v| {
                v.as_array().map(|arr| {
                    arr.iter()
                        .filter_map(|d| d.get("name").and_then(|n| n.as_str()).map(|s| s.to_string()))
                        .collect()
                })
            })
            .unwrap_or_default();

        self.metadata.update(&meta, spot_meta.as_ref(), &dexes);

        Ok(())
    }

    /// Fetch all mid prices
    pub async fn fetch_all_mids(&self) -> Result<HashMap<String, f64>> {
        let result = self.query_info(&json!({"type": "allMids"})).await?;

        let mut mids = HashMap::new();
        if let Some(obj) = result.as_object() {
            for (coin, price_val) in obj {
                let price_str = price_val.as_str().unwrap_or("");
                if let Ok(price) = price_str.parse::<f64>() {
                    mids.insert(coin.clone(), price);
                    self.mid_prices.insert(coin.clone(), price);
                }
            }
        }

        // Also fetch HIP-3 mids
        for dex in self.metadata.get_dexes() {
            if let Ok(dex_result) = self.query_info(&json!({"type": "allMids", "dex": dex})).await {
                if let Some(obj) = dex_result.as_object() {
                    for (coin, price_val) in obj {
                        let price_str = price_val.as_str().unwrap_or("");
                        if let Ok(price) = price_str.parse::<f64>() {
                            mids.insert(coin.clone(), price);
                            self.mid_prices.insert(coin.clone(), price);
                        }
                    }
                }
            }
        }

        Ok(mids)
    }

    /// Get mid price for an asset (from cache or fetch)
    pub async fn get_mid_price(&self, asset: &str) -> Result<f64> {
        if let Some(price) = self.mid_prices.get(asset) {
            return Ok(*price);
        }

        // Fetch all mids
        let mids = self.fetch_all_mids().await?;
        mids.get(asset)
            .copied()
            .ok_or_else(|| Error::ValidationError(format!("No price found for {}", asset)))
    }

    /// Resolve asset name to index
    pub fn resolve_asset(&self, name: &str) -> Option<usize> {
        self.metadata.resolve_asset(name)
    }

    /// Cancel an order by OID
    pub async fn cancel_by_oid(&self, oid: u64, asset: &str) -> Result<Value> {
        let asset_index = self
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let action = json!({
            "type": "cancel",
            "cancels": [{
                "a": asset_index,
                "o": oid,
            }]
        });

        self.build_sign_send(&action, None).await
    }

    /// Modify an order by OID
    pub async fn modify_by_oid(
        &self,
        oid: u64,
        asset: &str,
        side: Side,
        price: Decimal,
        size: Decimal,
    ) -> Result<PlacedOrder> {
        let asset_index = self
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let action = json!({
            "type": "batchModify",
            "modifies": [{
                "oid": oid,
                "order": {
                    "a": asset_index,
                    "b": side.is_buy(),
                    "p": price.normalize().to_string(),
                    "s": size.normalize().to_string(),
                    "r": false,
                    "t": {"limit": {"tif": "Gtc"}},
                    "c": "0x00000000000000000000000000000000",
                }
            }]
        });

        let response = self.build_sign_send(&action, None).await?;

        Ok(PlacedOrder::from_response(
            response,
            asset.to_string(),
            side,
            size,
            Some(price),
            None,
        ))
    }
}

/// Build response from the server
#[derive(Debug)]
pub struct BuildResponse {
    pub hash: String,
    pub nonce: u64,
    pub action: Value,
}

// ══════════════════════════════════════════════════════════════════════════════
// SDK Builder
// ══════════════════════════════════════════════════════════════════════════════

/// Builder for HyperliquidSDK
#[derive(Default)]
pub struct HyperliquidSDKBuilder {
    endpoint: Option<String>,
    private_key: Option<String>,
    testnet: bool,
    auto_approve: bool,
    max_fee: String,
    slippage: f64,
    timeout: Duration,
}

impl HyperliquidSDKBuilder {
    /// Create a new builder
    pub fn new() -> Self {
        Self {
            endpoint: None,
            private_key: None,
            testnet: false,
            auto_approve: true,
            max_fee: "1%".to_string(),
            slippage: DEFAULT_SLIPPAGE,
            timeout: Duration::from_secs(DEFAULT_TIMEOUT_SECS),
        }
    }

    /// Set the QuickNode endpoint
    pub fn endpoint(mut self, endpoint: impl Into<String>) -> Self {
        self.endpoint = Some(endpoint.into());
        self
    }

    /// Set the private key
    pub fn private_key(mut self, key: impl Into<String>) -> Self {
        self.private_key = Some(key.into());
        self
    }

    /// Use testnet
    pub fn testnet(mut self, testnet: bool) -> Self {
        self.testnet = testnet;
        self
    }

    /// Auto-approve builder fee on first trade
    pub fn auto_approve(mut self, auto: bool) -> Self {
        self.auto_approve = auto;
        self
    }

    /// Set maximum builder fee
    pub fn max_fee(mut self, fee: impl Into<String>) -> Self {
        self.max_fee = fee.into();
        self
    }

    /// Set slippage for market orders
    pub fn slippage(mut self, slippage: f64) -> Self {
        self.slippage = slippage;
        self
    }

    /// Set request timeout
    pub fn timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Build the SDK
    pub async fn build(self) -> Result<HyperliquidSDK> {
        // Get private key from builder or environment
        let private_key = self
            .private_key
            .or_else(|| std::env::var("PRIVATE_KEY").ok());

        // Parse signer if key provided
        let (signer, address) = if let Some(key) = private_key {
            let key = key.trim_start_matches("0x");
            let signer = PrivateKeySigner::from_str(key)?;
            let address = signer.address();
            (Some(signer), Some(address))
        } else {
            (None, None)
        };

        // Build HTTP client
        let http_client = Client::builder()
            .timeout(self.timeout)
            .build()
            .map_err(|e| Error::ConfigError(format!("Failed to create HTTP client: {}", e)))?;

        let chain = if self.testnet {
            Chain::Testnet
        } else {
            Chain::Mainnet
        };

        // Parse endpoint info for URL routing
        let endpoint_info = self.endpoint.as_ref().map(|ep| EndpointInfo::parse(ep));

        let inner = Arc::new(HyperliquidSDKInner {
            http_client,
            signer,
            address,
            chain,
            endpoint: self.endpoint,
            endpoint_info,
            slippage: self.slippage,
            metadata: MetadataCache::default(),
            mid_prices: DashMap::new(),
        });

        // Refresh metadata
        if let Err(e) = inner.refresh_metadata().await {
            tracing::warn!("Failed to fetch initial metadata: {}", e);
        }

        Ok(HyperliquidSDK {
            inner,
            auto_approve: self.auto_approve,
            max_fee: self.max_fee,
        })
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Main SDK
// ══════════════════════════════════════════════════════════════════════════════

/// Main Hyperliquid SDK client
pub struct HyperliquidSDK {
    inner: Arc<HyperliquidSDKInner>,
    #[allow(dead_code)]
    auto_approve: bool,
    max_fee: String,
}

impl HyperliquidSDK {
    /// Create a new SDK builder
    pub fn new() -> HyperliquidSDKBuilder {
        HyperliquidSDKBuilder::new()
    }

    /// Get the user's address
    pub fn address(&self) -> Option<Address> {
        self.inner.address
    }

    /// Get the chain
    pub fn chain(&self) -> Chain {
        self.inner.chain
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Info API (lazy accessor)
    // ──────────────────────────────────────────────────────────────────────────

    /// Access the Info API
    pub fn info(&self) -> crate::info::Info {
        crate::info::Info::new(self.inner.clone())
    }

    /// Access the HyperCore API
    pub fn core(&self) -> crate::hypercore::HyperCore {
        crate::hypercore::HyperCore::new(self.inner.clone())
    }

    /// Access the EVM API
    pub fn evm(&self) -> crate::evm::EVM {
        crate::evm::EVM::new(self.inner.clone())
    }

    /// Create a WebSocket stream
    pub fn stream(&self) -> crate::stream::Stream {
        crate::stream::Stream::new(self.inner.endpoint.clone())
    }

    /// Create a gRPC stream
    pub fn grpc(&self) -> crate::grpc::GRPCStream {
        crate::grpc::GRPCStream::new(self.inner.endpoint.clone())
    }

    /// Access the EVM WebSocket stream
    pub fn evm_stream(&self) -> crate::evm_stream::EVMStream {
        crate::evm_stream::EVMStream::new(self.inner.endpoint.clone())
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Quick Queries
    // ──────────────────────────────────────────────────────────────────────────

    /// Get all available markets
    pub async fn markets(&self) -> Result<Value> {
        self.inner.query_info(&json!({"type": "meta"})).await
    }

    /// Get all DEXes (HIP-3)
    pub async fn dexes(&self) -> Result<Value> {
        self.inner.query_info(&json!({"type": "perpDexs"})).await
    }

    /// Get open orders for the current user
    pub async fn open_orders(&self) -> Result<Value> {
        let address = self
            .inner
            .address
            .ok_or_else(|| Error::ConfigError("No address configured".to_string()))?;

        self.inner
            .query_info(&json!({
                "type": "openOrders",
                "user": format!("{:?}", address),
            }))
            .await
    }

    /// Get status of a specific order
    pub async fn order_status(&self, oid: u64, dex: Option<&str>) -> Result<Value> {
        let address = self
            .inner
            .address
            .ok_or_else(|| Error::ConfigError("No address configured".to_string()))?;

        let mut req = json!({
            "type": "orderStatus",
            "user": format!("{:?}", address),
            "oid": oid,
        });

        if let Some(d) = dex {
            req["dex"] = json!(d);
        }

        self.inner.query_info(&req).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Order Placement
    // ──────────────────────────────────────────────────────────────────────────

    /// Place a market buy order
    pub async fn market_buy(&self, asset: &str) -> MarketOrderBuilder {
        MarketOrderBuilder::new(self.inner.clone(), asset.to_string(), Side::Buy)
    }

    /// Place a market sell order
    pub async fn market_sell(&self, asset: &str) -> MarketOrderBuilder {
        MarketOrderBuilder::new(self.inner.clone(), asset.to_string(), Side::Sell)
    }

    /// Place a limit buy order
    pub async fn buy(
        &self,
        asset: &str,
        size: f64,
        price: f64,
        tif: TIF,
    ) -> Result<PlacedOrder> {
        self.place_order(asset, Side::Buy, size, Some(price), tif, false, false, None)
            .await
    }

    /// Place a limit sell order
    pub async fn sell(
        &self,
        asset: &str,
        size: f64,
        price: f64,
        tif: TIF,
    ) -> Result<PlacedOrder> {
        self.place_order(asset, Side::Sell, size, Some(price), tif, false, false, None)
            .await
    }

    /// Place an order using the fluent builder
    pub async fn order(&self, order: Order) -> Result<PlacedOrder> {
        order.validate()?;

        let asset = order.get_asset();
        let side = order.get_side();
        let tif = order.get_tif();

        // Resolve size from notional if needed
        let size = if let Some(s) = order.get_size() {
            s
        } else if let Some(notional) = order.get_notional() {
            let mid = self.inner.get_mid_price(asset).await?;
            Decimal::from_f64_retain(notional.to_string().parse::<f64>().unwrap_or(0.0) / mid)
                .unwrap_or_default()
        } else {
            return Err(Error::ValidationError(
                "Order must have size or notional".to_string(),
            ));
        };

        // For market orders, delegate price computation to the worker.
        // For limit orders, use the user-specified price.
        let is_market = order.is_market();
        let price = if is_market {
            None // worker computes price from mid + slippage
        } else {
            order
                .get_price()
                .map(|p| p.to_string().parse::<f64>().unwrap_or(0.0))
        };

        self.place_order(
            asset,
            side,
            size.to_string().parse::<f64>().unwrap_or(0.0),
            price,
            if is_market { TIF::Market } else { tif },
            order.is_reduce_only(),
            is_market,
            None, // use constructor-level default slippage
        )
        .await
    }

    /// Place a trigger order (stop-loss / take-profit)
    pub async fn trigger_order(&self, order: TriggerOrder) -> Result<PlacedOrder> {
        order.validate()?;

        let asset = order.get_asset();
        let asset_index = self
            .inner
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        // Get size decimals for rounding
        let sz_decimals = self.inner.metadata.get_asset(asset)
            .map(|a| a.sz_decimals)
            .unwrap_or(5) as u32;

        let trigger_px = order
            .get_trigger_price()
            .ok_or_else(|| Error::ValidationError("Trigger price required".to_string()))?;

        let size = order
            .get_size()
            .ok_or_else(|| Error::ValidationError("Size required".to_string()))?;

        // Round size to allowed decimals
        let size_rounded = size.round_dp(sz_decimals);

        // Get execution price, rounded to valid tick
        let limit_px = if order.is_market() {
            let mid = self.inner.get_mid_price(asset).await?;
            let slippage = self.inner.slippage;
            let price = if order.get_side().is_buy() {
                mid * (1.0 + slippage)
            } else {
                mid * (1.0 - slippage)
            };
            Decimal::from_f64_retain(price.round()).unwrap_or_default()
        } else {
            order.get_limit_price().unwrap_or(trigger_px).round()
        };

        // Round trigger price
        let trigger_px_rounded = trigger_px.round();

        // Generate random cloid (Hyperliquid requires nonzero cloid)
        let cloid = {
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default();
            let nanos = now.as_nanos() as u64;
            let hi = nanos.wrapping_mul(0x517cc1b727220a95);
            format!("0x{:016x}{:016x}", nanos, hi)
        };

        let action = json!({
            "type": "order",
            "orders": [{
                "a": asset_index,
                "b": order.get_side().is_buy(),
                "p": limit_px.normalize().to_string(),
                "s": size_rounded.normalize().to_string(),
                "r": order.is_reduce_only(),
                "t": {
                    "trigger": {
                        "isMarket": order.is_market(),
                        "triggerPx": trigger_px_rounded.normalize().to_string(),
                        "tpsl": order.get_tpsl().to_string(),
                    }
                },
                "c": cloid,
            }],
            "grouping": "na",
        });

        let response = self.inner.build_sign_send(&action, None).await?;

        Ok(PlacedOrder::from_response(
            response,
            asset.to_string(),
            order.get_side(),
            size,
            Some(limit_px),
            Some(self.inner.clone()),
        ))
    }

    /// Stop-loss helper
    pub async fn stop_loss(
        &self,
        asset: &str,
        size: f64,
        trigger_price: f64,
    ) -> Result<PlacedOrder> {
        self.trigger_order(
            TriggerOrder::stop_loss(asset)
                .size(size)
                .trigger_price(trigger_price)
                .market(),
        )
        .await
    }

    /// Take-profit helper
    pub async fn take_profit(
        &self,
        asset: &str,
        size: f64,
        trigger_price: f64,
    ) -> Result<PlacedOrder> {
        self.trigger_order(
            TriggerOrder::take_profit(asset)
                .size(size)
                .trigger_price(trigger_price)
                .market(),
        )
        .await
    }

    /// Internal order placement
    ///
    /// For market orders (`is_market = true`), uses the human-readable format
    /// (`asset`, `side`, `size`, `tif: "market"`) and delegates price computation
    /// to the worker. For limit orders, uses the wire format (`a`, `b`, `p`, `s`).
    async fn place_order(
        &self,
        asset: &str,
        side: Side,
        size: f64,
        price: Option<f64>,
        tif: TIF,
        reduce_only: bool,
        is_market: bool,
        slippage: Option<f64>,
    ) -> Result<PlacedOrder> {
        // Get size decimals for rounding
        let sz_decimals = self.inner.metadata.get_asset(asset)
            .map(|a| a.sz_decimals)
            .unwrap_or(5) as i32;

        // Round size to allowed decimals
        let size_rounded = (size * 10f64.powi(sz_decimals)).round() / 10f64.powi(sz_decimals);

        let (action, effective_slippage) = if is_market {
            // Market orders: use human-readable format, let worker compute price
            let action = json!({
                "type": "order",
                "orders": [{
                    "asset": asset,
                    "side": if side.is_buy() { "buy" } else { "sell" },
                    "size": format!("{}", size_rounded),
                    "tif": "market",
                }],
            });
            (action, slippage)
        } else {
            // Limit orders: use wire format with asset index
            let asset_index = self
                .inner
                .resolve_asset(asset)
                .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

            let resolved_price = price.map(|p| p.round()).unwrap_or(0.0);

            let tif_wire = match tif {
                TIF::Ioc => "Ioc",
                TIF::Gtc => "Gtc",
                TIF::Alo => "Alo",
                TIF::Market => "Ioc",
            };

            // Generate random cloid (Hyperliquid requires nonzero cloid)
            let cloid = {
                let now = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default();
                let nanos = now.as_nanos() as u64;
                let hi = nanos.wrapping_mul(0x517cc1b727220a95);
                format!("0x{:016x}{:016x}", nanos, hi)
            };

            let action = json!({
                "type": "order",
                "orders": [{
                    "a": asset_index,
                    "b": side.is_buy(),
                    "p": format!("{}", resolved_price),
                    "s": format!("{}", size_rounded),
                    "r": reduce_only,
                    "t": {"limit": {"tif": tif_wire}},
                    "c": cloid,
                }],
                "grouping": "na",
            });
            (action, None) // no slippage for limit orders
        };

        let response = self.inner.build_sign_send(&action, effective_slippage).await?;

        Ok(PlacedOrder::from_response(
            response,
            asset.to_string(),
            side,
            Decimal::from_f64_retain(size_rounded).unwrap_or_default(),
            price.map(|p| Decimal::from_f64_retain(p).unwrap_or_default()),
            Some(self.inner.clone()),
        ))
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Order Management
    // ──────────────────────────────────────────────────────────────────────────

    /// Modify an existing order
    ///
    /// The order is identified by OID, which is included in the returned order.
    pub async fn modify(
        &self,
        oid: u64,
        asset: &str,
        is_buy: bool,
        size: f64,
        price: f64,
        tif: TIF,
        reduce_only: bool,
        cloid: Option<&str>,
    ) -> Result<PlacedOrder> {
        let asset_idx = self
            .inner
            .metadata
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let sz_decimals = self.inner.metadata.get_asset(asset)
            .map(|a| a.sz_decimals)
            .unwrap_or(8) as i32;
        let size_rounded = (size * 10f64.powi(sz_decimals)).round() / 10f64.powi(sz_decimals);

        let order_type = match tif {
            TIF::Gtc => json!({"limit": {"tif": "Gtc"}}),
            TIF::Ioc | TIF::Market => json!({"limit": {"tif": "Ioc"}}),
            TIF::Alo => json!({"limit": {"tif": "Alo"}}),
        };

        let cloid_val = cloid
            .map(|s| s.to_string())
            .unwrap_or_else(|| {
                let now = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default();
                let nanos = now.as_nanos() as u64;
                let hi = nanos.wrapping_mul(0x517cc1b727220a95);
                format!("0x{:016x}{:016x}", nanos, hi)
            });

        let action = json!({
            "type": "batchModify",
            "modifies": [{
                "oid": oid,
                "order": {
                    "a": asset_idx,
                    "b": is_buy,
                    "p": format!("{:.8}", price).trim_end_matches('0').trim_end_matches('.'),
                    "s": format!("{:.8}", size_rounded).trim_end_matches('0').trim_end_matches('.'),
                    "r": reduce_only,
                    "t": order_type,
                    "c": cloid_val,
                }
            }]
        });

        let response = self.inner.build_sign_send(&action, None).await?;

        Ok(PlacedOrder::from_response(
            response,
            asset.to_string(),
            if is_buy { Side::Buy } else { Side::Sell },
            Decimal::from_f64_retain(size_rounded).unwrap_or_default(),
            Some(Decimal::from_f64_retain(price).unwrap_or_default()),
            Some(self.inner.clone()),
        ))
    }

    /// Cancel an order by OID
    pub async fn cancel(&self, oid: u64, asset: &str) -> Result<Value> {
        self.inner.cancel_by_oid(oid, asset).await
    }

    /// Cancel all orders (optionally for a specific asset)
    pub async fn cancel_all(&self, asset: Option<&str>) -> Result<Value> {
        // Ensure we have an address configured
        if self.inner.address.is_none() {
            return Err(Error::ConfigError("No address configured".to_string()));
        }

        // Get open orders
        let open_orders = self.open_orders().await?;

        let cancels: Vec<Value> = open_orders
            .as_array()
            .unwrap_or(&vec![])
            .iter()
            .filter(|order| {
                if let Some(asset) = asset {
                    order.get("coin").and_then(|c| c.as_str()) == Some(asset)
                } else {
                    true
                }
            })
            .filter_map(|order| {
                let oid = order.get("oid").and_then(|o| o.as_u64())?;
                let coin = order.get("coin").and_then(|c| c.as_str())?;
                let asset_index = self.inner.resolve_asset(coin)?;
                Some(json!({"a": asset_index, "o": oid}))
            })
            .collect();

        if cancels.is_empty() {
            return Ok(json!({"status": "ok", "message": "No orders to cancel"}));
        }

        let action = json!({
            "type": "cancel",
            "cancels": cancels,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Close position for an asset
    ///
    /// Delegates position lookup and counter-order building to the worker using
    /// the `closePosition` action type. Optionally accepts a per-call slippage
    /// override.
    pub async fn close_position(&self, asset: &str, slippage: Option<f64>) -> Result<PlacedOrder> {
        let address = self
            .inner
            .address
            .ok_or_else(|| Error::ConfigError("No address configured".to_string()))?;

        let action = json!({
            "type": "closePosition",
            "asset": asset,
            "user": format!("{:?}", address),
        });

        let response = self.inner.build_sign_send(&action, slippage).await?;

        // Extract position info from close context
        let ctx = response.get("closePositionContext").cloned().unwrap_or_default();
        let side_str = ctx.get("closeSide").and_then(|s| s.as_str()).unwrap_or("sell");
        let side = if side_str == "buy" { Side::Buy } else { Side::Sell };
        let size_str = ctx.get("closeSize").and_then(|s| s.as_str()).unwrap_or("0");
        let size = Decimal::from_str(size_str).unwrap_or_default();

        Ok(PlacedOrder::from_response(
            response,
            asset.to_string(),
            side,
            size,
            None,
            Some(self.inner.clone()),
        ))
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Leverage & Margin
    // ──────────────────────────────────────────────────────────────────────────

    /// Update leverage for an asset
    pub async fn update_leverage(
        &self,
        asset: &str,
        leverage: i32,
        is_cross: bool,
    ) -> Result<Value> {
        let asset_index = self
            .inner
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let action = json!({
            "type": "updateLeverage",
            "asset": asset_index,
            "isCross": is_cross,
            "leverage": leverage,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Update isolated margin
    pub async fn update_isolated_margin(
        &self,
        asset: &str,
        is_buy: bool,
        amount_usd: f64,
    ) -> Result<Value> {
        let asset_index = self
            .inner
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let action = json!({
            "type": "updateIsolatedMargin",
            "asset": asset_index,
            "isBuy": is_buy,
            "ntli": (amount_usd * 1_000_000.0) as i64, // Convert to USDC with 6 decimals
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // TWAP Orders
    // ──────────────────────────────────────────────────────────────────────────

    /// Place a TWAP order
    pub async fn twap_order(
        &self,
        asset: &str,
        size: f64,
        is_buy: bool,
        duration_minutes: i64,
        reduce_only: bool,
        randomize: bool,
    ) -> Result<Value> {
        let action = json!({
            "type": "twapOrder",
            "twap": {
                "a": asset,
                "b": is_buy,
                "s": format!("{}", size),
                "r": reduce_only,
                "m": duration_minutes,
                "t": randomize,
            }
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Cancel a TWAP order
    pub async fn twap_cancel(&self, asset: &str, twap_id: i64) -> Result<Value> {
        let action = json!({
            "type": "twapCancel",
            "a": asset,
            "t": twap_id,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Transfers
    // ──────────────────────────────────────────────────────────────────────────

    /// Transfer USD to another address
    pub async fn transfer_usd(&self, destination: &str, amount: f64) -> Result<Value> {
        let time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "usdSend",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "destination": destination,
            "amount": format!("{}", amount),
            "time": time,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Transfer spot token to another address
    pub async fn transfer_spot(
        &self,
        token: &str,
        destination: &str,
        amount: f64,
    ) -> Result<Value> {
        let time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "spotSend",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "token": token,
            "destination": destination,
            "amount": format!("{}", amount),
            "time": time,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Withdraw to Arbitrum
    pub async fn withdraw(&self, amount: f64, destination: Option<&str>) -> Result<Value> {
        let time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let dest = destination
            .map(|s| s.to_string())
            .or_else(|| self.inner.address.map(|a| format!("{:?}", a)))
            .ok_or_else(|| Error::ConfigError("No destination address".to_string()))?;

        let action = json!({
            "type": "withdraw3",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "destination": dest,
            "amount": format!("{}", amount),
            "time": time,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Transfer spot balance to perp balance
    pub async fn transfer_spot_to_perp(&self, amount: f64) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "usdClassTransfer",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "amount": format!("{}", amount),
            "toPerp": true,
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Transfer perp balance to spot balance
    pub async fn transfer_perp_to_spot(&self, amount: f64) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "usdClassTransfer",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "amount": format!("{}", amount),
            "toPerp": false,
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Vaults
    // ──────────────────────────────────────────────────────────────────────────

    /// Deposit to a vault
    pub async fn vault_deposit(&self, vault_address: &str, amount: f64) -> Result<Value> {
        let action = json!({
            "type": "vaultTransfer",
            "vaultAddress": vault_address,
            "isDeposit": true,
            "usd": amount,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Withdraw from a vault
    pub async fn vault_withdraw(&self, vault_address: &str, amount: f64) -> Result<Value> {
        let action = json!({
            "type": "vaultTransfer",
            "vaultAddress": vault_address,
            "isDeposit": false,
            "usd": amount,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Staking
    // ──────────────────────────────────────────────────────────────────────────

    /// Stake tokens
    pub async fn stake(&self, amount_tokens: f64) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let wei = (amount_tokens * 1e18) as u128;

        let action = json!({
            "type": "cDeposit",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "wei": wei.to_string(),
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Unstake tokens
    pub async fn unstake(&self, amount_tokens: f64) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let wei = (amount_tokens * 1e18) as u128;

        let action = json!({
            "type": "cWithdraw",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "wei": wei.to_string(),
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Delegate tokens to a validator
    pub async fn delegate(&self, validator: &str, amount_tokens: f64) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let wei = (amount_tokens * 1e18) as u128;

        let action = json!({
            "type": "tokenDelegate",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "validator": validator,
            "isUndelegate": false,
            "wei": wei.to_string(),
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Undelegate tokens from a validator
    pub async fn undelegate(&self, validator: &str, amount_tokens: f64) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let wei = (amount_tokens * 1e18) as u128;

        let action = json!({
            "type": "tokenDelegate",
            "hyperliquidChain": self.inner.chain.to_string(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "validator": validator,
            "isUndelegate": true,
            "wei": wei.to_string(),
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Builder Fee Approval
    // ──────────────────────────────────────────────────────────────────────────

    /// Approve builder fee
    pub async fn approve_builder_fee(&self, max_fee: Option<&str>) -> Result<Value> {
        let fee = max_fee.unwrap_or(&self.max_fee);

        let action = json!({
            "type": "approveBuilderFee",
            "maxFeeRate": fee,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Revoke builder fee approval
    pub async fn revoke_builder_fee(&self) -> Result<Value> {
        self.approve_builder_fee(Some("0%")).await
    }

    /// Check approval status
    pub async fn approval_status(&self) -> Result<Value> {
        let address = self
            .inner
            .address
            .ok_or_else(|| Error::ConfigError("No address configured".to_string()))?;

        // Use the worker's /approval endpoint
        let url = format!("{}/approval", DEFAULT_WORKER_URL);

        let response = self
            .inner
            .http_client
            .post(&url)
            .json(&json!({"user": format!("{:?}", address)}))
            .send()
            .await?;

        let text = response.text().await?;
        serde_json::from_str(&text).map_err(|e| Error::JsonError(e.to_string()))
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Misc
    // ──────────────────────────────────────────────────────────────────────────

    /// Reserve request weight (purchase rate limit capacity)
    pub async fn reserve_request_weight(&self, weight: i32) -> Result<Value> {
        let action = json!({
            "type": "reserveRequestWeight",
            "weight": weight,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// No-op (consume nonce)
    pub async fn noop(&self) -> Result<Value> {
        let action = json!({"type": "noop"});
        self.inner.build_sign_send(&action, None).await
    }

    /// Preflight validation
    pub async fn preflight(
        &self,
        asset: &str,
        side: Side,
        price: f64,
        size: f64,
    ) -> Result<Value> {
        let url = format!("{}/preflight", DEFAULT_WORKER_URL);

        let body = json!({
            "asset": asset,
            "side": side.to_string(),
            "price": price,
            "size": size,
        });

        let response = self
            .inner
            .http_client
            .post(&url)
            .json(&body)
            .send()
            .await?;

        let text = response.text().await?;
        serde_json::from_str(&text).map_err(|e| Error::JsonError(e.to_string()))
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Agent/API Key Management
    // ──────────────────────────────────────────────────────────────────────────

    /// Approve an agent (API wallet) to trade on your behalf
    pub async fn approve_agent(
        &self,
        agent_address: &str,
        name: Option<&str>,
    ) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "approveAgent",
            "hyperliquidChain": self.inner.chain.as_str(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "agentAddress": agent_address,
            "agentName": name,
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Account Abstraction
    // ──────────────────────────────────────────────────────────────────────────

    /// Set account abstraction mode
    ///
    /// Mode can be: "disabled", "unifiedAccount", or "portfolioMargin"
    pub async fn set_abstraction(&self, mode: &str, user: Option<&str>) -> Result<Value> {
        let address = self
            .inner
            .address
            .ok_or_else(|| Error::ConfigError("No address configured".to_string()))?;

        let addr_string = format!("{:?}", address);
        let user_addr = user.unwrap_or(&addr_string);
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "userSetAbstraction",
            "hyperliquidChain": self.inner.chain.as_str(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "user": user_addr,
            "abstraction": mode,
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Set account abstraction mode as an agent
    pub async fn agent_set_abstraction(&self, mode: &str) -> Result<Value> {
        // Map full mode names to short codes
        let short_mode = match mode {
            "disabled" | "i" => "i",
            "unifiedAccount" | "u" => "u",
            "portfolioMargin" | "p" => "p",
            _ => {
                return Err(Error::ValidationError(format!(
                    "Invalid mode: {}. Use 'disabled', 'unifiedAccount', or 'portfolioMargin'",
                    mode
                )))
            }
        };

        let action = json!({
            "type": "agentSetAbstraction",
            "abstraction": short_mode,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Advanced Transfers
    // ──────────────────────────────────────────────────────────────────────────

    /// Generalized asset transfer between DEXs and accounts
    pub async fn send_asset(
        &self,
        token: &str,
        amount: f64,
        destination: &str,
        source_dex: Option<&str>,
        destination_dex: Option<&str>,
        from_sub_account: Option<&str>,
    ) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "sendAsset",
            "hyperliquidChain": self.inner.chain.as_str(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "destination": destination,
            "sourceDex": source_dex.unwrap_or(""),
            "destinationDex": destination_dex.unwrap_or(""),
            "token": token,
            "amount": amount.to_string(),
            "fromSubAccount": from_sub_account.unwrap_or(""),
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Transfer tokens to HyperEVM with custom data payload
    pub async fn send_to_evm_with_data(
        &self,
        token: &str,
        amount: f64,
        destination: &str,
        data: &str,
        source_dex: &str,
        destination_chain_id: u32,
        gas_limit: u64,
    ) -> Result<Value> {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64;

        let action = json!({
            "type": "sendToEvmWithData",
            "hyperliquidChain": self.inner.chain.as_str(),
            "signatureChainId": self.inner.chain.signature_chain_id(),
            "token": token,
            "amount": amount.to_string(),
            "sourceDex": source_dex,
            "destinationRecipient": destination,
            "addressEncoding": "hex",
            "destinationChainId": destination_chain_id,
            "gasLimit": gas_limit,
            "data": data,
            "nonce": nonce,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Additional Margin Operations
    // ──────────────────────────────────────────────────────────────────────────

    /// Top up isolated-only margin to target a specific leverage
    pub async fn top_up_isolated_only_margin(
        &self,
        asset: &str,
        leverage: f64,
    ) -> Result<Value> {
        let asset_idx = self
            .inner
            .metadata
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let action = json!({
            "type": "topUpIsolatedOnlyMargin",
            "asset": asset_idx,
            "leverage": leverage.to_string(),
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Validator Operations
    // ──────────────────────────────────────────────────────────────────────────

    /// Submit a validator vote for the risk-free rate (validator only)
    pub async fn validator_l1_stream(&self, risk_free_rate: &str) -> Result<Value> {
        let action = json!({
            "type": "validatorL1Stream",
            "riskFreeRate": risk_free_rate,
        });

        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Cancel Operations
    // ──────────────────────────────────────────────────────────────────────────

    /// Cancel an order by client order ID (cloid)
    pub async fn cancel_by_cloid(&self, cloid: &str, asset: &str) -> Result<Value> {
        let asset_idx = self
            .inner
            .metadata
            .resolve_asset(asset)
            .ok_or_else(|| Error::ValidationError(format!("Unknown asset: {}", asset)))?;

        let action = json!({
            "type": "cancelByCloid",
            "cancels": [{"asset": asset_idx, "cloid": cloid}],
        });

        self.inner.build_sign_send(&action, None).await
    }

    /// Schedule cancellation of all orders after a delay (dead-man's switch)
    pub async fn schedule_cancel(&self, time_ms: Option<u64>) -> Result<Value> {
        let mut action = json!({"type": "scheduleCancel"});
        if let Some(t) = time_ms {
            action["time"] = json!(t);
        }
        self.inner.build_sign_send(&action, None).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // EVM Stream Access
    // ──────────────────────────────────────────────────────────────────────────
    // Convenience Queries
    // ──────────────────────────────────────────────────────────────────────────

    /// Get mid price for an asset
    pub async fn get_mid(&self, asset: &str) -> Result<f64> {
        self.inner.get_mid_price(asset).await
    }

    /// Force refresh of market metadata cache
    pub async fn refresh_markets(&self) -> Result<()> {
        self.inner.refresh_metadata().await
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Market Order Builder
// ══════════════════════════════════════════════════════════════════════════════

/// Builder for market orders with size or notional
pub struct MarketOrderBuilder {
    inner: Arc<HyperliquidSDKInner>,
    asset: String,
    side: Side,
    size: Option<f64>,
    notional: Option<f64>,
    slippage: Option<f64>,
}

impl MarketOrderBuilder {
    fn new(inner: Arc<HyperliquidSDKInner>, asset: String, side: Side) -> Self {
        Self {
            inner,
            asset,
            side,
            size: None,
            notional: None,
            slippage: None,
        }
    }

    /// Set order size (in base asset units)
    pub fn size(mut self, size: f64) -> Self {
        self.size = Some(size);
        self
    }

    /// Set notional value (in USD)
    pub fn notional(mut self, notional: f64) -> Self {
        self.notional = Some(notional);
        self
    }

    /// Set per-call slippage override (default uses constructor-level slippage)
    ///
    /// Range: 0.001 (0.1%) to 0.1 (10%)
    pub fn slippage(mut self, slippage: f64) -> Self {
        self.slippage = Some(slippage);
        self
    }

    /// Execute the market order
    ///
    /// Uses the human-readable format (`asset`, `side`, `size`, `tif: "market"`)
    /// and delegates price computation to the worker.
    pub async fn execute(self) -> Result<PlacedOrder> {
        // Get size decimals for rounding
        let sz_decimals = self.inner.metadata.get_asset(&self.asset)
            .map(|a| a.sz_decimals)
            .unwrap_or(5) as i32;

        let size = if let Some(s) = self.size {
            s
        } else if let Some(notional) = self.notional {
            let mid = self.inner.get_mid_price(&self.asset).await?;
            notional / mid
        } else {
            return Err(Error::ValidationError(
                "Market order must have size or notional".to_string(),
            ));
        };

        // Round size to allowed decimals
        let size_rounded = (size * 10f64.powi(sz_decimals)).round() / 10f64.powi(sz_decimals);

        // Use human-readable format — worker computes price from mid + slippage
        let action = json!({
            "type": "order",
            "orders": [{
                "asset": self.asset,
                "side": if self.side.is_buy() { "buy" } else { "sell" },
                "size": format!("{}", size_rounded),
                "tif": "market",
            }],
        });

        let response = self.inner.build_sign_send(&action, self.slippage).await?;

        Ok(PlacedOrder::from_response(
            response,
            self.asset,
            self.side,
            Decimal::from_f64_retain(size_rounded).unwrap_or_default(),
            None,
            Some(self.inner),
        ))
    }
}

// Implement await for MarketOrderBuilder
impl std::future::IntoFuture for MarketOrderBuilder {
    type Output = Result<PlacedOrder>;
    type IntoFuture = std::pin::Pin<Box<dyn std::future::Future<Output = Self::Output> + Send>>;

    fn into_future(self) -> Self::IntoFuture {
        Box::pin(self.execute())
    }
}
