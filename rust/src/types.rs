//! Core types for the Hyperliquid SDK.
//!
//! These types mirror the Hyperliquid API exactly for byte-identical serialization.

use alloy::primitives::{Address, B128, U256};
use alloy::sol_types::{eip712_domain, Eip712Domain};
use either::Either;
use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};
use std::fmt;
use std::str::FromStr;

// ══════════════════════════════════════════════════════════════════════════════
// Type Aliases
// ══════════════════════════════════════════════════════════════════════════════

/// Client Order ID - 128-bit unique identifier
pub type Cloid = B128;

/// Either an order ID (u64) or a client order ID (Cloid)
pub type OidOrCloid = Either<u64, Cloid>;

// ══════════════════════════════════════════════════════════════════════════════
// Chain
// ══════════════════════════════════════════════════════════════════════════════

/// Hyperliquid chain (Mainnet or Testnet)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub enum Chain {
    #[default]
    Mainnet,
    Testnet,
}

impl Chain {
    /// Returns true if this is mainnet
    pub fn is_mainnet(&self) -> bool {
        matches!(self, Chain::Mainnet)
    }

    /// Returns the chain as a string ("Mainnet" or "Testnet")
    pub fn as_str(&self) -> &'static str {
        match self {
            Chain::Mainnet => "Mainnet",
            Chain::Testnet => "Testnet",
        }
    }

    /// Returns the signature chain ID for EIP-712 signing
    pub fn signature_chain_id(&self) -> &'static str {
        match self {
            Chain::Mainnet => "0xa4b1", // Arbitrum One
            Chain::Testnet => "0x66eee", // Arbitrum Sepolia
        }
    }

    /// Returns the EVM chain ID
    pub fn evm_chain_id(&self) -> u64 {
        match self {
            Chain::Mainnet => 999,
            Chain::Testnet => 998,
        }
    }
}

impl fmt::Display for Chain {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Chain::Mainnet => write!(f, "Mainnet"),
            Chain::Testnet => write!(f, "Testnet"),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Side
// ══════════════════════════════════════════════════════════════════════════════

/// Order side
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Side {
    Buy,
    Sell,
}

impl Side {
    /// Returns true if this is a buy side
    pub fn is_buy(&self) -> bool {
        matches!(self, Side::Buy)
    }

    /// Converts to bool for API (true = buy, false = sell)
    pub fn as_bool(&self) -> bool {
        self.is_buy()
    }
}

impl fmt::Display for Side {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Side::Buy => write!(f, "buy"),
            Side::Sell => write!(f, "sell"),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// HIP-4 Prediction Markets
// ══════════════════════════════════════════════════════════════════════════════

/// A tradeable HIP-4 outcome side.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PredictionSide {
    pub outcome: u64,
    pub side: usize,
    pub name: String,
    pub symbol: String,
    pub token: String,
    pub asset_id: usize,
    pub mid: Option<String>,
    pub sz_decimals: u8,
    pub supports_priority_fee: bool,
}

impl fmt::Display for PredictionSide {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.symbol)
    }
}

impl From<PredictionSide> for String {
    fn from(side: PredictionSide) -> Self {
        side.symbol
    }
}

impl From<&PredictionSide> for String {
    fn from(side: &PredictionSide) -> Self {
        side.symbol.clone()
    }
}

/// A HIP-4 prediction market with yes/no tradeable sides.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PredictionMarket {
    pub outcome: u64,
    pub name: String,
    pub description: String,
    pub title: String,
    pub slug: String,
    pub underlying: Option<String>,
    pub target_price: Option<String>,
    pub expiry: Option<String>,
    pub period: Option<String>,
    pub collateral: String,
    pub min_order_value: String,
    pub aliases: Vec<String>,
    pub yes: PredictionSide,
    pub no: PredictionSide,
    pub sides: Vec<PredictionSide>,
}

impl PredictionMarket {
    pub fn matches(&self, query: &str) -> bool {
        let normalized = query.to_lowercase();
        let mut values = vec![
            self.slug.clone(),
            self.title.to_lowercase(),
            self.name.to_lowercase(),
            self.underlying.clone().unwrap_or_default().to_lowercase(),
            self.yes.symbol.to_lowercase(),
            self.no.symbol.to_lowercase(),
            self.yes.token.to_lowercase(),
            self.no.token.to_lowercase(),
        ];
        values.extend(self.aliases.iter().map(|alias| alias.to_lowercase()));
        values.iter().any(|value| value == &normalized || value.contains(&normalized))
    }
}

/// Filter for selecting an active HIP-4 prediction market.
#[derive(Debug, Clone, Default)]
pub struct PredictionMarketFilter {
    pub query: Option<String>,
    pub underlying: Option<String>,
    pub target_price: Option<String>,
    pub expiry: Option<String>,
}

impl FromStr for Side {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "buy" | "b" | "long" => Ok(Side::Buy),
            "sell" | "s" | "short" => Ok(Side::Sell),
            _ => Err(format!("invalid side: {}", s)),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Time In Force (TIF)
// ══════════════════════════════════════════════════════════════════════════════

/// Time in force for orders
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TIF {
    /// Immediate or Cancel - fill immediately or cancel
    #[default]
    Ioc,
    /// Good Till Cancel - stays on book until filled or cancelled
    Gtc,
    /// Add Liquidity Only (post-only) - rejected if would cross
    Alo,
    /// Market order (converted to IOC with slippage)
    Market,
}

impl fmt::Display for TIF {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            TIF::Ioc => write!(f, "ioc"),
            TIF::Gtc => write!(f, "gtc"),
            TIF::Alo => write!(f, "alo"),
            TIF::Market => write!(f, "market"),
        }
    }
}

impl FromStr for TIF {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "ioc" => Ok(TIF::Ioc),
            "gtc" => Ok(TIF::Gtc),
            "alo" | "post_only" => Ok(TIF::Alo),
            "market" => Ok(TIF::Market),
            _ => Err(format!("invalid tif: {}", s)),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// TimeInForce (API format)
// ══════════════════════════════════════════════════════════════════════════════

/// Time in force for the wire format (PascalCase)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TimeInForce {
    Alo,
    Ioc,
    Gtc,
    FrontendMarket,
}

impl From<TIF> for TimeInForce {
    fn from(tif: TIF) -> Self {
        match tif {
            TIF::Ioc => TimeInForce::Ioc,
            TIF::Gtc => TimeInForce::Gtc,
            TIF::Alo => TimeInForce::Alo,
            TIF::Market => TimeInForce::Ioc, // Market orders use IOC with slippage
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// TpSl (Take Profit / Stop Loss)
// ══════════════════════════════════════════════════════════════════════════════

/// Take profit or stop loss trigger type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TpSl {
    /// Take profit
    Tp,
    /// Stop loss
    Sl,
}

impl fmt::Display for TpSl {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            TpSl::Tp => write!(f, "tp"),
            TpSl::Sl => write!(f, "sl"),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Order Grouping
// ══════════════════════════════════════════════════════════════════════════════

/// Order grouping for TP/SL attachment
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum OrderGrouping {
    /// No grouping
    #[default]
    Na,
    /// Normal TP/SL grouping
    NormalTpsl,
    /// Position-based TP/SL grouping
    PositionTpsl,
}

impl fmt::Display for OrderGrouping {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            OrderGrouping::Na => write!(f, "na"),
            OrderGrouping::NormalTpsl => write!(f, "normalTpsl"),
            OrderGrouping::PositionTpsl => write!(f, "positionTpsl"),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// EIP-712 Domain
// ══════════════════════════════════════════════════════════════════════════════

/// EIP-712 domain for Hyperliquid signing
pub const CORE_MAINNET_EIP712_DOMAIN: Eip712Domain = eip712_domain! {
    name: "Exchange",
    version: "1",
    chain_id: 1337,
    verifying_contract: Address::ZERO,
};

// ══════════════════════════════════════════════════════════════════════════════
// Signature
// ══════════════════════════════════════════════════════════════════════════════

/// ECDSA signature (r, s, v format)
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct Signature {
    #[serde(
        serialize_with = "serialize_u256_hex",
        deserialize_with = "deserialize_u256_hex"
    )]
    pub r: U256,
    #[serde(
        serialize_with = "serialize_u256_hex",
        deserialize_with = "deserialize_u256_hex"
    )]
    pub s: U256,
    pub v: u64,
}

impl From<alloy::signers::Signature> for Signature {
    fn from(sig: alloy::signers::Signature) -> Self {
        Self {
            r: sig.r(),
            s: sig.s(),
            v: if sig.v() { 28 } else { 27 },
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Order Type Placement
// ══════════════════════════════════════════════════════════════════════════════

/// Order type for placement (limit or trigger)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum OrderTypePlacement {
    /// Limit order
    Limit {
        tif: TimeInForce,
    },
    /// Trigger order (stop loss / take profit)
    #[serde(rename_all = "camelCase")]
    Trigger {
        is_market: bool,
        #[serde(with = "decimal_normalized")]
        trigger_px: Decimal,
        tpsl: TpSl,
    },
}

// ══════════════════════════════════════════════════════════════════════════════
// Order Request
// ══════════════════════════════════════════════════════════════════════════════

/// Order request for the API
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OrderRequest {
    /// Asset index
    #[serde(rename = "a")]
    pub asset: usize,
    /// Is buy (true) or sell (false)
    #[serde(rename = "b")]
    pub is_buy: bool,
    /// Limit price
    #[serde(rename = "p", with = "decimal_normalized")]
    pub limit_px: Decimal,
    /// Size
    #[serde(rename = "s", with = "decimal_normalized")]
    pub sz: Decimal,
    /// Reduce only
    #[serde(rename = "r")]
    pub reduce_only: bool,
    /// Order type
    #[serde(rename = "t")]
    pub order_type: OrderTypePlacement,
    /// Client order ID
    #[serde(
        rename = "c",
        serialize_with = "serialize_cloid_hex",
        deserialize_with = "deserialize_cloid_hex"
    )]
    pub cloid: Cloid,
}

// ══════════════════════════════════════════════════════════════════════════════
// Batch Order
// ══════════════════════════════════════════════════════════════════════════════

/// Batch of orders
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BatchOrder {
    pub orders: Vec<OrderRequest>,
    pub grouping: OrderGrouping,
}

// ══════════════════════════════════════════════════════════════════════════════
// Modify
// ══════════════════════════════════════════════════════════════════════════════

/// Order modification
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Modify {
    #[serde(with = "oid_or_cloid")]
    pub oid: OidOrCloid,
    pub order: OrderRequest,
}

/// Batch modification
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BatchModify {
    pub modifies: Vec<Modify>,
}

// ══════════════════════════════════════════════════════════════════════════════
// Cancel
// ══════════════════════════════════════════════════════════════════════════════

/// Cancel request
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Cancel {
    #[serde(rename = "a")]
    pub asset: usize,
    #[serde(rename = "o")]
    pub oid: u64,
}

/// Batch cancel
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BatchCancel {
    pub cancels: Vec<Cancel>,
    /// Fast cancel flag (`"f": true` on the wire; omitted unless true)
    #[serde(
        rename = "f",
        default,
        skip_serializing_if = "is_not_true_flag",
        deserialize_with = "deserialize_true_only_flag"
    )]
    pub fast: Option<bool>,
}

/// Cancel by client order ID
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CancelByCloid {
    pub asset: u32,
    #[serde(with = "const_hex_b128")]
    pub cloid: B128,
}

/// Batch cancel by client order ID
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BatchCancelCloid {
    pub cancels: Vec<CancelByCloid>,
    /// Fast cancel flag (`"f": true` on the wire; omitted unless true)
    #[serde(
        rename = "f",
        default,
        skip_serializing_if = "is_not_true_flag",
        deserialize_with = "deserialize_true_only_flag"
    )]
    pub fast: Option<bool>,
}

fn is_not_true_flag(value: &Option<bool>) -> bool {
    !matches!(value, Some(true))
}

fn deserialize_true_only_flag<'de, D>(deserializer: D) -> Result<Option<bool>, D::Error>
where
    D: serde::Deserializer<'de>,
{
    Option::<bool>::deserialize(deserializer).map(|value| value.filter(|flag| *flag))
}

/// Schedule cancel (dead-man's switch)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ScheduleCancel {
    pub time: Option<u64>,
}

// ══════════════════════════════════════════════════════════════════════════════
// TWAP Orders
// ══════════════════════════════════════════════════════════════════════════════

/// TWAP order specification
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TwapSpec {
    #[serde(rename = "a")]
    pub asset: String,
    #[serde(rename = "b")]
    pub is_buy: bool,
    #[serde(rename = "s")]
    pub sz: String,
    #[serde(rename = "r")]
    pub reduce_only: bool,
    #[serde(rename = "m")]
    pub duration_minutes: i64,
    #[serde(rename = "t")]
    pub randomize: bool,
}

/// TWAP order
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TwapOrder {
    pub twap: TwapSpec,
}

/// TWAP cancel
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TwapCancel {
    #[serde(rename = "a")]
    pub asset: String,
    #[serde(rename = "t")]
    pub twap_id: i64,
}

// ══════════════════════════════════════════════════════════════════════════════
// Leverage Management
// ══════════════════════════════════════════════════════════════════════════════

/// Update leverage
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateLeverage {
    pub asset: u32,
    pub is_cross: bool,
    pub leverage: i32,
}

/// Update isolated margin
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateIsolatedMargin {
    pub asset: u32,
    pub is_buy: bool,
    pub ntli: i64,
}

/// Top up isolated only margin
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TopUpIsolatedOnlyMargin {
    pub asset: u32,
    pub leverage: String,
}

// ══════════════════════════════════════════════════════════════════════════════
// Transfer Operations
// ══════════════════════════════════════════════════════════════════════════════

/// USD transfer
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UsdSend {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub destination: String,
    pub amount: String,
    pub time: u64,
}

/// Spot token transfer
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SpotSend {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub token: String,
    pub destination: String,
    pub amount: String,
    pub time: u64,
}

/// Withdraw to Arbitrum
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Withdraw3 {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub destination: String,
    pub amount: String,
    pub time: u64,
}

/// USD class transfer (perp <-> spot)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UsdClassTransfer {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub amount: String,
    pub to_perp: bool,
    pub nonce: u64,
}

/// Send asset
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SendAsset {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub destination: String,
    pub source_dex: String,
    pub destination_dex: String,
    pub token: String,
    pub amount: String,
    pub from_sub_account: String,
    pub nonce: u64,
}

// ══════════════════════════════════════════════════════════════════════════════
// Vault Operations
// ══════════════════════════════════════════════════════════════════════════════

/// Vault transfer (deposit/withdraw)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct VaultTransfer {
    pub vault_address: String,
    pub is_deposit: bool,
    pub usd: f64,
}

// ══════════════════════════════════════════════════════════════════════════════
// Agent/API Key Management
// ══════════════════════════════════════════════════════════════════════════════

/// Approve agent (API key)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ApproveAgent {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub agent_address: String,
    pub agent_name: Option<String>,
    pub nonce: u64,
}

/// Approve builder fee
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ApproveBuilderFee {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub max_fee_rate: String,
    pub builder: String,
    pub nonce: u64,
}

// ══════════════════════════════════════════════════════════════════════════════
// Account Abstraction
// ══════════════════════════════════════════════════════════════════════════════

/// User set abstraction
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UserSetAbstraction {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub user: String,
    pub abstraction: String,
    pub nonce: u64,
}

/// Agent set abstraction
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AgentSetAbstraction {
    pub abstraction: String,
}

// ══════════════════════════════════════════════════════════════════════════════
// Staking Operations
// ══════════════════════════════════════════════════════════════════════════════

/// Stake (cDeposit)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CDeposit {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub wei: u128,
    pub nonce: u64,
}

/// Unstake (cWithdraw)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CWithdraw {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub wei: u128,
    pub nonce: u64,
}

/// Delegate tokens
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TokenDelegate {
    pub hyperliquid_chain: Chain,
    pub signature_chain_id: String,
    pub validator: String,
    pub is_undelegate: bool,
    pub wei: u128,
    pub nonce: u64,
}

// ══════════════════════════════════════════════════════════════════════════════
// Misc Operations
// ══════════════════════════════════════════════════════════════════════════════

/// Reserve request weight (purchase rate limit capacity)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReserveRequestWeight {
    pub weight: i32,
}

/// No-op (consume nonce)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Noop {}

/// Validator L1 stream (vote on risk-free rate)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ValidatorL1Stream {
    pub risk_free_rate: String,
}

/// Close position
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ClosePosition {
    pub asset: String,
    pub user: String,
}

// ══════════════════════════════════════════════════════════════════════════════
// Action (all possible actions)
// ══════════════════════════════════════════════════════════════════════════════

/// All possible actions that can be sent to the exchange
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
#[serde(rename_all = "camelCase")]
pub enum Action {
    // Trading actions (require builder fees)
    Order(BatchOrder),
    BatchModify(BatchModify),

    // Cancel actions (no builder fees)
    Cancel(BatchCancel),
    CancelByCloid(BatchCancelCloid),
    ScheduleCancel(ScheduleCancel),

    // TWAP orders
    TwapOrder(TwapOrder),
    TwapCancel(TwapCancel),

    // Leverage management
    UpdateLeverage(UpdateLeverage),
    UpdateIsolatedMargin(UpdateIsolatedMargin),
    TopUpIsolatedOnlyMargin(TopUpIsolatedOnlyMargin),

    // Transfer operations
    UsdSend(UsdSend),
    SpotSend(SpotSend),
    Withdraw3(Withdraw3),
    UsdClassTransfer(UsdClassTransfer),
    SendAsset(SendAsset),

    // Vault operations
    VaultTransfer(VaultTransfer),

    // Agent/API key management
    ApproveAgent(ApproveAgent),
    ApproveBuilderFee(ApproveBuilderFee),

    // Account abstraction
    UserSetAbstraction(UserSetAbstraction),
    AgentSetAbstraction(AgentSetAbstraction),

    // Staking operations
    CDeposit(CDeposit),
    CWithdraw(CWithdraw),
    TokenDelegate(TokenDelegate),

    // Rate limiting
    ReserveRequestWeight(ReserveRequestWeight),

    // Noop
    Noop(Noop),

    // Validator operations
    ValidatorL1Stream(ValidatorL1Stream),

    // Close position
    ClosePosition(ClosePosition),
}

impl Action {
    /// Compute the MessagePack hash of this action for signing
    pub fn hash(
        &self,
        nonce: u64,
        vault_address: Option<Address>,
        expires_after: Option<u64>,
    ) -> Result<alloy::primitives::B256, rmp_serde::encode::Error> {
        crate::signing::rmp_hash(self, nonce, vault_address, expires_after)
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Action Request
// ══════════════════════════════════════════════════════════════════════════════

/// Signed action request
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ActionRequest {
    pub action: Action,
    pub nonce: u64,
    pub signature: Signature,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub vault_address: Option<Address>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_after: Option<u64>,
}

// ══════════════════════════════════════════════════════════════════════════════
// Exchange Options
// ══════════════════════════════════════════════════════════════════════════════

/// Optional per-action trading parameters for order/modify/closePosition actions.
///
/// Both fields are sent as top-level exchange-body fields (`vaultAddress`,
/// `expiresAfter` on the wire) so the worker folds them into the signed action
/// hash. Neither field is emitted when unset.
#[derive(Debug, Clone, Default)]
pub struct ExchangeOptions {
    /// Trade on behalf of a vault or subaccount
    pub vault_address: Option<String>,
    /// Action TTL: millisecond timestamp after which the action is rejected
    pub expires_after: Option<u64>,
}

/// Optional per-call cancel parameters.
#[derive(Debug, Clone, Default)]
pub struct CancelOptions {
    /// Fast cancel: emits `"f": true` on the cancel action (omitted when false)
    pub fast: bool,
    /// Cancel on behalf of a vault or subaccount
    pub vault_address: Option<String>,
    /// Action TTL: millisecond timestamp after which the action is rejected
    pub expires_after: Option<u64>,
}

impl CancelOptions {
    /// The vault/TTL subset of these options, for threading into build/send
    pub fn exchange_options(&self) -> ExchangeOptions {
        ExchangeOptions {
            vault_address: self.vault_address.clone(),
            expires_after: self.expires_after,
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Builder
// ══════════════════════════════════════════════════════════════════════════════

/// Builder fee information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Builder {
    /// Builder address
    #[serde(rename = "b")]
    pub address: String,
    /// Fee in tenths of basis points (40 = 0.04%)
    #[serde(rename = "f")]
    pub fee: u16,
}

// ══════════════════════════════════════════════════════════════════════════════
// Serde Helpers
// ══════════════════════════════════════════════════════════════════════════════

/// Normalized decimal serialization (removes trailing zeros)
pub mod decimal_normalized {
    use rust_decimal::Decimal;
    use serde::{de, Deserialize, Deserializer, Serializer};
    use std::str::FromStr;

    pub fn serialize<S>(value: &Decimal, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        let normalized = value.normalize();
        serializer.serialize_str(&normalized.to_string())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Decimal, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        Decimal::from_str(&s)
            .map(|d| d.normalize())
            .map_err(de::Error::custom)
    }
}

/// Serde module for OidOrCloid
pub mod oid_or_cloid {
    use super::Cloid;
    use either::Either;
    use serde::{de, Deserializer, Serializer};

    pub fn serialize<S>(value: &Either<u64, Cloid>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match value {
            Either::Left(oid) => serializer.serialize_u64(*oid),
            Either::Right(cloid) => serializer.serialize_str(&format!("{:#x}", cloid)),
        }
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Either<u64, Cloid>, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct Visitor;

        impl<'de> serde::de::Visitor<'de> for Visitor {
            type Value = Either<u64, Cloid>;

            fn expecting(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
                f.write_str("a u64 oid or a hex string cloid")
            }

            fn visit_u64<E: de::Error>(self, v: u64) -> Result<Self::Value, E> {
                Ok(Either::Left(v))
            }

            fn visit_str<E: de::Error>(self, v: &str) -> Result<Self::Value, E> {
                v.parse::<Cloid>().map(Either::Right).map_err(de::Error::custom)
            }
        }

        deserializer.deserialize_any(Visitor)
    }
}

/// B128 hex serialization
pub mod const_hex_b128 {
    use alloy::primitives::B128;
    use serde::{Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(value: &B128, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&format!("{:#x}", value))
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<B128, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        s.parse::<B128>().map_err(serde::de::Error::custom)
    }
}

fn serialize_cloid_hex<S>(value: &Cloid, serializer: S) -> Result<S::Ok, S::Error>
where
    S: serde::Serializer,
{
    serializer.serialize_str(&format!("{:#x}", value))
}

fn deserialize_cloid_hex<'de, D>(deserializer: D) -> Result<Cloid, D::Error>
where
    D: serde::Deserializer<'de>,
{
    let s = String::deserialize(deserializer)?;
    s.parse::<Cloid>().map_err(serde::de::Error::custom)
}

fn serialize_u256_hex<S>(value: &U256, serializer: S) -> Result<S::Ok, S::Error>
where
    S: serde::Serializer,
{
    serializer.serialize_str(&format!("{:#x}", value))
}

fn deserialize_u256_hex<'de, D>(deserializer: D) -> Result<U256, D::Error>
where
    D: serde::Deserializer<'de>,
{
    let s = String::deserialize(deserializer)?;
    let s = s.strip_prefix("0x").unwrap_or(&s);
    U256::from_str_radix(s, 16).map_err(serde::de::Error::custom)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn batch_cancel(fast: Option<bool>) -> BatchCancel {
        BatchCancel {
            cancels: vec![Cancel { asset: 5, oid: 123 }],
            fast,
        }
    }

    /// The fast flag is only emitted when true — unset and explicit false both
    /// produce the legacy wire shape (wire compat with existing servers).
    #[test]
    fn test_batch_cancel_fast_flag_serialization() {
        let unset = serde_json::to_string(&batch_cancel(None)).unwrap();
        assert_eq!(unset, r#"{"cancels":[{"a":5,"o":123}]}"#);

        let explicit_false = serde_json::to_string(&batch_cancel(Some(false))).unwrap();
        assert_eq!(explicit_false, unset);

        let fast = serde_json::to_string(&batch_cancel(Some(true))).unwrap();
        assert_eq!(fast, r#"{"cancels":[{"a":5,"o":123}],"f":true}"#);
    }

    /// Deserialization strips `f: false` (matching the backend, which only
    /// keeps the flag when true).
    #[test]
    fn test_batch_cancel_fast_flag_deserialization() {
        let with_true: BatchCancel =
            serde_json::from_str(r#"{"cancels":[{"a":5,"o":123}],"f":true}"#).unwrap();
        assert_eq!(with_true.fast, Some(true));

        let with_false: BatchCancel =
            serde_json::from_str(r#"{"cancels":[{"a":5,"o":123}],"f":false}"#).unwrap();
        assert_eq!(with_false.fast, None);

        let without: BatchCancel =
            serde_json::from_str(r#"{"cancels":[{"a":5,"o":123}]}"#).unwrap();
        assert_eq!(without.fast, None);
    }

    /// Same behavior for the cloid variant.
    #[test]
    fn test_batch_cancel_cloid_fast_flag_serialization() {
        let entry = CancelByCloid {
            asset: 5,
            cloid: B128::repeat_byte(0x11),
        };
        let unset = serde_json::to_string(&BatchCancelCloid {
            cancels: vec![entry.clone()],
            fast: None,
        })
        .unwrap();
        assert!(!unset.contains("\"f\""), "got: {unset}");

        let fast = serde_json::to_string(&BatchCancelCloid {
            cancels: vec![entry],
            fast: Some(true),
        })
        .unwrap();
        assert!(fast.ends_with(r#"],"f":true}"#), "got: {fast}");
    }

    /// The fast flag participates in the signed action payload: the MessagePack
    /// hash of a cancel action changes when `f: true` is set.
    #[test]
    fn test_fast_flag_changes_action_hash() {
        let plain = Action::Cancel(batch_cancel(None));
        let fast = Action::Cancel(batch_cancel(Some(true)));

        let hash_plain = plain.hash(1000, None, None).unwrap();
        let hash_fast = fast.hash(1000, None, None).unwrap();
        assert_ne!(hash_plain, hash_fast);

        // Explicit false hashes identically to unset (it is skipped entirely).
        let explicit_false = Action::Cancel(batch_cancel(Some(false)));
        assert_eq!(explicit_false.hash(1000, None, None).unwrap(), hash_plain);
    }
}
