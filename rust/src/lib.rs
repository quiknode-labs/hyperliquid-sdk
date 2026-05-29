//! Hyperliquid SDK for Rust
//!
//! A simple, performant SDK for trading on Hyperliquid.
//!
//! # Quick Start
//!
//! ```rust,no_run
//! use hyperliquid_sdk::{HyperliquidSDK, Side, TIF};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     // Initialize with private key (or set PRIVATE_KEY env var)
//!     let sdk = HyperliquidSDK::new()
//!         .endpoint("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
//!         .private_key("0x...")
//!         .build()
//!         .await?;
//!
//!     // Market buy $100 worth of BTC
//!     let order = sdk.market_buy("BTC").await.notional(100.0).await?;
//!     println!("Order placed: {:?}", order.oid);
//!
//!     // Or use the fluent builder
//!     use hyperliquid_sdk::Order;
//!     let order = sdk.order(
//!         Order::buy("BTC").size(0.001).price(65000.0).gtc()
//!     ).await?;
//!
//!     Ok(())
//! }
//! ```
//!
//! # Features
//!
//! - **Trading**: Market/limit orders, stop-loss, take-profit, TWAP
//! - **Order Management**: Cancel, modify, batch operations
//! - **Info API**: Market data, positions, open orders, account state
//! - **HyperCore**: Real-time block data, trades, order book updates
//! - **Streaming**: WebSocket and gRPC (optional) for real-time data
//! - **HyperEVM**: Ethereum JSON-RPC compatibility
//!
//! # External signing (KMS/HSM/remote signer)
//!
//! Implement [`HyperliquidSigner`] and pass it via
//! [`HyperliquidSDKBuilder::signer`] to sign action hashes without giving the
//! SDK a raw private key. It takes precedence over `private_key`/`PRIVATE_KEY`,
//! the acting address is taken from the signer, and builder-fee auto-approval is
//! skipped (call `approve_builder_fee` yourself if you need it). A signer failure
//! surfaces as [`Error::SignerError`] (code `SIGNER_FAILED`), distinct from a
//! venue rejection ([`Error::ApiError`]). Bound each call with
//! [`HyperliquidSDKBuilder::signer_deadline`] if your remote signer can stall.

pub mod types;
pub mod signing;
pub mod order;
pub mod error;
pub mod client;
pub mod info;
pub mod hypercore;
pub mod evm;
pub mod stream;
pub mod evm_stream;

pub mod grpc;

// Re-export main types for convenience
pub use types::{
    Chain, Side, TIF, TpSl, OrderGrouping, Signature,
    OrderRequest, OrderTypePlacement, TimeInForce,
    Action, ActionRequest, PredictionMarket, PredictionMarketFilter, PredictionSide,
};
pub use order::{Order, TriggerOrder, PlacedOrder};
pub use error::{Error, ErrorCode, Result};
pub use signing::{HyperliquidSigner, LocalSigner};
pub use client::{HyperliquidSDK, HyperliquidSDKBuilder, EndpointInfo};
pub use info::Info;
pub use hypercore::HyperCore;
pub use evm::EVM;
pub use stream::Stream;
pub use evm_stream::{EVMStream, EVMSubscriptionType, EVMConnectionState};

pub use grpc::{GRPCStream, GRPCSubscriptionOptions};

// Re-export serde_json::Value for convenience since many API methods return it
pub use serde_json::Value;
