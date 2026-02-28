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
    Action, ActionRequest,
};
pub use order::{Order, TriggerOrder, PlacedOrder};
pub use error::{Error, Result};
pub use client::{HyperliquidSDK, HyperliquidSDKBuilder, EndpointInfo};
pub use info::Info;
pub use hypercore::HyperCore;
pub use evm::EVM;
pub use stream::Stream;
pub use evm_stream::{EVMStream, EVMSubscriptionType, EVMConnectionState};

pub use grpc::GRPCStream;

// Re-export serde_json::Value for convenience since many API methods return it
pub use serde_json::Value;
