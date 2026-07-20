//! gRPC streaming client for Hyperliquid.
//!
//! Provides low-latency real-time data streaming via gRPC.
//! Authentication is via x-token header with your QuickNode API token.
//!
//! Example:
//! ```ignore
//! use hyperliquid_sdk::GRPCStream;
//!
//! let mut stream = GRPCStream::new(Some("https://your-endpoint.quiknode.pro/TOKEN".to_string()));
//! stream.trades(&["BTC", "ETH"], |data| {
//!     println!("Trade: {:?}", data);
//! });
//! stream.start().await?;
//! ```

// tonic interceptors must return `Result<Request<()>, tonic::Status>`, and
// `Status` is large by clippy's default threshold; the signature is not ours
// to change.
#![allow(clippy::result_large_err)]

use parking_lot::RwLock;
use serde_json::Value;
use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicU32, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::time::sleep;
use tonic::metadata::MetadataValue;
use tonic::transport::{Channel, ClientTlsConfig};
use tonic::Request;

use crate::error::Result;
use crate::stream::ConnectionState;

fn is_permanent_stream_error(status: &tonic::Status) -> bool {
    matches!(
        status.code(),
        tonic::Code::InvalidArgument
            | tonic::Code::Unimplemented
            | tonic::Code::PermissionDenied
            | tonic::Code::Unauthenticated
    )
}

// Include generated protobuf code
pub mod proto {
    tonic::include_proto!("hyperliquid");
}

use proto::block_streaming_client::BlockStreamingClient;
use proto::order_book_streaming_client::OrderBookStreamingClient;
use proto::streaming_client::StreamingClient;
use proto::{
    BboBookRequest, FilterValues, L2BookDiffRequest, L2BookRequest, L4BookRequest,
    L4BookUpdatesRequest, Ping, PingRequest, StreamBytesResponse, StreamSubscribe,
    SubscribeRequest, Timestamp, TpslUpdatesRequest,
};

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Constants
// ══════════════════════════════════════════════════════════════════════════════

const GRPC_PORT: u16 = 10000;
const INITIAL_RECONNECT_DELAY: Duration = Duration::from_secs(1);
const MAX_RECONNECT_DELAY: Duration = Duration::from_secs(60);
const RECONNECT_BACKOFF_FACTOR: f64 = 2.0;
const KEEPALIVE_TIME: Duration = Duration::from_secs(30);
const KEEPALIVE_TIMEOUT: Duration = Duration::from_secs(10);

type ValueCallbackMap = HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>;
type BytesCallbackMap = HashMap<u32, Box<dyn Fn(StreamBytesResponse) + Send + Sync>>;
/// Per-subscription high-water mark of delivered `block_number`s (0 = none seen).
type LastSeenBlockMap = HashMap<u32, Arc<AtomicU64>>;

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Stream Types
// ══════════════════════════════════════════════════════════════════════════════

/// gRPC stream types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum GRPCStreamType {
    Trades,
    Orders,
    BookUpdates,
    Twap,
    Events,
    Blocks,
    WriterActions,
    MempoolTxs,
    OrderPriority,
    GossipPriority,
    L2Book,
    L4Book,
    BboBook,
    L2BookDiff,
    L4BookUpdates,
    TpslUpdates,
    L2BookPacked,
    BboBookPacked,
    L4BookBytes,
}

impl GRPCStreamType {
    /// Get the stream name
    pub fn as_str(&self) -> &'static str {
        match self {
            GRPCStreamType::Trades => "trades",
            GRPCStreamType::Orders => "orders",
            GRPCStreamType::BookUpdates => "book_updates",
            GRPCStreamType::Twap => "twap",
            GRPCStreamType::Events => "events",
            GRPCStreamType::Blocks => "blocks",
            GRPCStreamType::WriterActions => "writer_actions",
            GRPCStreamType::MempoolTxs => "mempool_txs",
            GRPCStreamType::OrderPriority => "order_priority",
            GRPCStreamType::GossipPriority => "gossip_priority",
            GRPCStreamType::L2Book => "l2_book",
            GRPCStreamType::L4Book => "l4_book",
            GRPCStreamType::BboBook => "bbo_book",
            GRPCStreamType::L2BookDiff => "l2_book_diff",
            GRPCStreamType::L4BookUpdates => "l4_book_updates",
            GRPCStreamType::TpslUpdates => "tpsl_updates",
            GRPCStreamType::L2BookPacked => "l2_book_packed",
            GRPCStreamType::BboBookPacked => "bbo_book_packed",
            GRPCStreamType::L4BookBytes => "l4_book_bytes",
        }
    }

    /// Convert to proto enum value
    fn to_proto(self) -> i32 {
        match self {
            GRPCStreamType::Trades => 1,
            GRPCStreamType::Orders => 2,
            GRPCStreamType::BookUpdates => 3,
            GRPCStreamType::Twap => 4,
            GRPCStreamType::Events => 5,
            GRPCStreamType::Blocks => 6,
            GRPCStreamType::WriterActions => 7,
            GRPCStreamType::MempoolTxs => 8,
            GRPCStreamType::OrderPriority => 9,
            GRPCStreamType::GossipPriority => 10,
            GRPCStreamType::L2Book
            | GRPCStreamType::L4Book
            | GRPCStreamType::BboBook
            | GRPCStreamType::L2BookDiff
            | GRPCStreamType::L4BookUpdates
            | GRPCStreamType::TpslUpdates
            | GRPCStreamType::L2BookPacked
            | GRPCStreamType::BboBookPacked
            | GRPCStreamType::L4BookBytes => 0,
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Subscription
// ══════════════════════════════════════════════════════════════════════════════

/// A gRPC subscription handle
#[derive(Debug, Clone)]
pub struct GRPCSubscription {
    pub id: u32,
    pub stream_type: GRPCStreamType,
}

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Stream Configuration
// ══════════════════════════════════════════════════════════════════════════════

/// gRPC stream configuration
#[derive(Clone)]
pub struct GRPCStreamConfig {
    pub endpoint: Option<String>,
    pub reconnect: bool,
    pub max_reconnect_attempts: Option<u32>,
    pub keepalive_interval: Duration,
    pub keepalive_timeout: Duration,
}

impl Default for GRPCStreamConfig {
    fn default() -> Self {
        Self {
            endpoint: None,
            reconnect: true,
            max_reconnect_attempts: None,
            keepalive_interval: KEEPALIVE_TIME,
            keepalive_timeout: KEEPALIVE_TIMEOUT,
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Subscription Info
// ══════════════════════════════════════════════════════════════════════════════

#[derive(Clone)]
struct GRPCSubscriptionInfo {
    stream_type: GRPCStreamType,
    coins: Vec<String>,
    users: Vec<String>,
    coin: Option<String>,
    n_levels: Option<u32>,
    n_sig_figs: Option<u32>,
    mantissa: Option<u64>,
    skip_initial_snapshot: bool,
    start_block: Option<u64>,
    raw: bool,
    bytes: bool,
}

/// Options for resumable gRPC data subscriptions.
#[derive(Debug, Clone, Copy, Default)]
pub struct GRPCSubscriptionOptions {
    /// Start streaming from this Hyperliquid block number when supported.
    pub start_block: Option<u64>,
}

/// Options for gRPC L2 book diff subscriptions.
#[derive(Debug, Clone, Copy)]
pub struct GRPCL2BookDiffOptions {
    /// Max tracked levels per side (default 20, max 100).
    pub n_levels: u32,
    /// Significant figures for price bucketing (2-5).
    pub n_sig_figs: Option<u32>,
    /// Mantissa for bucketing (1, 2, or 5).
    pub mantissa: Option<u64>,
    /// If false, the first update per coin contains the current levels.
    pub skip_initial_snapshot: bool,
}

impl Default for GRPCL2BookDiffOptions {
    fn default() -> Self {
        Self {
            n_levels: 20,
            n_sig_figs: None,
            mantissa: None,
            skip_initial_snapshot: false,
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Stream
// ══════════════════════════════════════════════════════════════════════════════

/// gRPC stream client for Hyperliquid real-time data
pub struct GRPCStream {
    config: GRPCStreamConfig,
    host: String,
    token: String,
    state: Arc<RwLock<ConnectionState>>,
    running: Arc<AtomicBool>,
    reconnect_attempts: Arc<AtomicU32>,
    subscription_id: Arc<AtomicU32>,
    subscriptions: Arc<RwLock<HashMap<u32, GRPCSubscriptionInfo>>>,
    callbacks: Arc<RwLock<ValueCallbackMap>>,
    bytes_callbacks: Arc<RwLock<BytesCallbackMap>>,
    last_seen_blocks: Arc<RwLock<LastSeenBlockMap>>,
    on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
    on_close: Option<Arc<dyn Fn() + Send + Sync>>,
    on_connect: Option<Arc<dyn Fn() + Send + Sync>>,
    on_reconnect: Option<Arc<dyn Fn(u32) + Send + Sync>>,
    on_state_change: Option<Arc<dyn Fn(ConnectionState) + Send + Sync>>,
    stop_tx: Option<mpsc::Sender<()>>,
}

impl GRPCStream {
    /// Create a new gRPC stream client
    pub fn new(endpoint: Option<String>) -> Self {
        let (host, token) = endpoint
            .as_ref()
            .map(|ep| parse_endpoint(ep))
            .unwrap_or_default();

        Self {
            config: GRPCStreamConfig {
                endpoint,
                ..Default::default()
            },
            host,
            token,
            state: Arc::new(RwLock::new(ConnectionState::Disconnected)),
            running: Arc::new(AtomicBool::new(false)),
            reconnect_attempts: Arc::new(AtomicU32::new(0)),
            subscription_id: Arc::new(AtomicU32::new(0)),
            subscriptions: Arc::new(RwLock::new(HashMap::new())),
            callbacks: Arc::new(RwLock::new(HashMap::new())),
            bytes_callbacks: Arc::new(RwLock::new(HashMap::new())),
            last_seen_blocks: Arc::new(RwLock::new(HashMap::new())),
            on_error: None,
            on_close: None,
            on_connect: None,
            on_reconnect: None,
            on_state_change: None,
            stop_tx: None,
        }
    }

    /// Configure stream options
    pub fn configure(mut self, config: GRPCStreamConfig) -> Self {
        if let Some(ref ep) = config.endpoint {
            let (host, token) = parse_endpoint(ep);
            self.host = host;
            self.token = token;
        }
        self.config = config;
        self
    }

    /// Set error callback
    pub fn on_error<F>(mut self, f: F) -> Self
    where
        F: Fn(String) + Send + Sync + 'static,
    {
        self.on_error = Some(Arc::new(f));
        self
    }

    /// Set close callback
    pub fn on_close<F>(mut self, f: F) -> Self
    where
        F: Fn() + Send + Sync + 'static,
    {
        self.on_close = Some(Arc::new(f));
        self
    }

    /// Set connect callback
    pub fn on_connect<F>(mut self, f: F) -> Self
    where
        F: Fn() + Send + Sync + 'static,
    {
        self.on_connect = Some(Arc::new(f));
        self
    }

    /// Set reconnect callback
    pub fn on_reconnect<F>(mut self, f: F) -> Self
    where
        F: Fn(u32) + Send + Sync + 'static,
    {
        self.on_reconnect = Some(Arc::new(f));
        self
    }

    /// Set state change callback
    pub fn on_state_change<F>(mut self, f: F) -> Self
    where
        F: Fn(ConnectionState) + Send + Sync + 'static,
    {
        self.on_state_change = Some(Arc::new(f));
        self
    }

    /// Get current connection state
    pub fn state(&self) -> ConnectionState {
        *self.state.read()
    }

    /// Check if connected
    pub fn connected(&self) -> bool {
        *self.state.read() == ConnectionState::Connected
    }

    fn set_state(&self, state: ConnectionState) {
        let mut s = self.state.write();
        if *s != state {
            *s = state;
            if let Some(ref cb) = self.on_state_change {
                cb(state);
            }
        }
    }

    fn next_subscription_id(&self) -> u32 {
        self.subscription_id.fetch_add(1, Ordering::SeqCst)
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Subscriptions
    // ──────────────────────────────────────────────────────────────────────────

    /// Subscribe to trades
    pub fn trades<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.trades_with_options(coins, GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw trade blocks.
    pub fn raw_trades<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Trades,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Trades,
        }
    }

    /// Subscribe to trades with options.
    pub fn trades_with_options<F>(
        &mut self,
        coins: &[&str],
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Trades,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Trades,
        }
    }

    /// Subscribe to orders
    pub fn orders<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.orders_with_options(coins, GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw order blocks.
    pub fn raw_orders<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Orders,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Orders,
        }
    }

    /// Subscribe to orders with options.
    pub fn orders_with_options<F>(
        &mut self,
        coins: &[&str],
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Orders,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Orders,
        }
    }

    /// Subscribe to book updates
    pub fn book_updates<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.book_updates_with_options(coins, GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw book update blocks.
    pub fn raw_book_updates<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::BookUpdates,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::BookUpdates,
        }
    }

    /// Subscribe to book updates with options.
    pub fn book_updates_with_options<F>(
        &mut self,
        coins: &[&str],
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::BookUpdates,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::BookUpdates,
        }
    }

    /// Subscribe to L2 order book
    pub fn l2_book<F>(&mut self, coin: &str, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.l2_book_with_options(coin, 20, None, callback)
    }

    /// Subscribe to L2 order book with options
    pub fn l2_book_with_options<F>(
        &mut self,
        coin: &str,
        n_levels: u32,
        n_sig_figs: Option<u32>,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::L2Book,
                coins: vec![],
                users: vec![],
                coin: Some(coin.to_string()),
                n_levels: Some(n_levels),
                n_sig_figs,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::L2Book,
        }
    }

    /// Subscribe to L4 order book (individual orders with OIDs)
    pub fn l4_book<F>(&mut self, coin: &str, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::L4Book,
                coins: vec![],
                users: vec![],
                coin: Some(coin.to_string()),
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::L4Book,
        }
    }

    /// Subscribe to blocks
    pub fn blocks<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Blocks,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Blocks,
        }
    }

    /// Subscribe to TWAP updates
    pub fn twap<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.twap_with_options(coins, GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw TWAP execution blocks.
    pub fn raw_twap<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Twap,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Twap,
        }
    }

    /// Subscribe to TWAP updates with options.
    pub fn twap_with_options<F>(
        &mut self,
        coins: &[&str],
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Twap,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Twap,
        }
    }

    /// Subscribe to events
    pub fn events<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.events_with_options(GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to events with options.
    pub fn events_with_options<F>(
        &mut self,
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Events,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Events,
        }
    }

    /// Subscribe to raw event blocks.
    pub fn raw_events<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::Events,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::Events,
        }
    }

    /// Subscribe to writer actions
    pub fn writer_actions<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.writer_actions_with_options(GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw writer action blocks.
    pub fn raw_writer_actions<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::WriterActions,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::WriterActions,
        }
    }

    /// Subscribe to writer actions with options.
    pub fn writer_actions_with_options<F>(
        &mut self,
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::WriterActions,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::WriterActions,
        }
    }

    /// Subscribe to pre-consensus mempool transactions.
    /// Pass an empty `coins` slice for the unfiltered stream.
    pub fn mempool_txs<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.mempool_txs_with_options(coins, GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw mempool transaction blocks.
    pub fn raw_mempool_txs<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::MempoolTxs,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::MempoolTxs,
        }
    }

    /// Subscribe to mempool transactions with options.
    pub fn mempool_txs_with_options<F>(
        &mut self,
        coins: &[&str],
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::MempoolTxs,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::MempoolTxs,
        }
    }

    /// Subscribe to derived order/write priority actions.
    /// Events carry server-enriched fields `coin`, `market_type`, and `sz_decimals`.
    pub fn order_priority<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.order_priority_with_options(GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw order priority blocks.
    pub fn raw_order_priority<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::OrderPriority,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::OrderPriority,
        }
    }

    /// Subscribe to order priority actions with options.
    pub fn order_priority_with_options<F>(
        &mut self,
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::OrderPriority,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::OrderPriority,
        }
    }

    /// Subscribe to derived gossip/read priority bid actions.
    /// Events carry server-enriched fields `coin`, `market_type`, and `sz_decimals`.
    pub fn gossip_priority<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.gossip_priority_with_options(GRPCSubscriptionOptions::default(), callback)
    }

    /// Subscribe to raw gossip priority blocks.
    pub fn raw_gossip_priority<F>(&mut self, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::GossipPriority,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: true,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::GossipPriority,
        }
    }

    /// Subscribe to gossip priority actions with options.
    pub fn gossip_priority_with_options<F>(
        &mut self,
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::GossipPriority,
                coins: vec![],
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::GossipPriority,
        }
    }

    /// Subscribe to the low-level bytes variant of a data stream (`StreamDataBytes` RPC).
    /// The callback receives raw [`StreamBytesResponse`] messages (block number,
    /// server ingress timestamp, and the undecoded payload bytes). Only data
    /// stream types (trades, orders, ..., gossip_priority) are valid here.
    pub fn raw_bytes<F>(
        &mut self,
        stream_type: GRPCStreamType,
        coins: &[&str],
        options: GRPCSubscriptionOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(StreamBytesResponse) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: options.start_block,
                raw: true,
                bytes: true,
            },
        );
        self.bytes_callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription { id, stream_type }
    }

    /// Subscribe to best bid/offer updates. Empty `coins` means all coins.
    pub fn bbo_book<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::BboBook,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::BboBook,
        }
    }

    /// Subscribe to best bid/offer updates with fixed-point prices/sizes
    /// (scaled by 1e8). Empty `coins` means all coins.
    pub fn bbo_book_packed<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::BboBookPacked,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::BboBookPacked,
        }
    }

    /// Subscribe to incremental L2 price-level changes. Empty `coins` means all coins.
    pub fn l2_book_diff<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.l2_book_diff_with_options(coins, GRPCL2BookDiffOptions::default(), callback)
    }

    /// Subscribe to incremental L2 price-level changes with options.
    /// A level with `sz == "0"` means the level was removed.
    pub fn l2_book_diff_with_options<F>(
        &mut self,
        coins: &[&str],
        options: GRPCL2BookDiffOptions,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::L2BookDiff,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: Some(options.n_levels),
                n_sig_figs: options.n_sig_figs,
                mantissa: options.mantissa,
                skip_initial_snapshot: options.skip_initial_snapshot,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::L2BookDiff,
        }
    }

    /// Subscribe to typed L4 order book updates. Empty `coins` means all coins.
    pub fn l4_book_updates<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::L4BookUpdates,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::L4BookUpdates,
        }
    }

    /// Subscribe to trigger/TP-SL order updates. Empty `coins` means all perp coins.
    pub fn tpsl_updates<F>(&mut self, coins: &[&str], callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::TpslUpdates,
                coins: coins.iter().map(|s| s.to_string()).collect(),
                users: vec![],
                coin: None,
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::TpslUpdates,
        }
    }

    /// Subscribe to fast-path L2 order book updates with fixed-point
    /// prices/sizes (scaled by 1e8).
    pub fn l2_book_packed<F>(&mut self, coin: &str, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.l2_book_packed_with_options(coin, 20, None, callback)
    }

    /// Subscribe to fast-path L2 order book updates with options.
    pub fn l2_book_packed_with_options<F>(
        &mut self,
        coin: &str,
        n_levels: u32,
        n_sig_figs: Option<u32>,
        callback: F,
    ) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::L2BookPacked,
                coins: vec![],
                users: vec![],
                coin: Some(coin.to_string()),
                n_levels: Some(n_levels),
                n_sig_figs,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::L2BookPacked,
        }
    }

    /// Subscribe to the fast-path L4 order book stream. Same payload shape as
    /// `l4_book`, but diffs are transported as JSON bytes on the wire.
    pub fn l4_book_bytes<F>(&mut self, coin: &str, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        self.subscriptions.write().insert(
            id,
            GRPCSubscriptionInfo {
                stream_type: GRPCStreamType::L4BookBytes,
                coins: vec![],
                users: vec![],
                coin: Some(coin.to_string()),
                n_levels: None,
                n_sig_figs: None,
                mantissa: None,
                skip_initial_snapshot: false,
                start_block: None,
                raw: false,
                bytes: false,
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::L4BookBytes,
        }
    }

    /// Unsubscribe
    pub fn unsubscribe(&mut self, subscription: &GRPCSubscription) {
        self.subscriptions.write().remove(&subscription.id);
        self.callbacks.write().remove(&subscription.id);
        self.bytes_callbacks.write().remove(&subscription.id);
        self.last_seen_blocks.write().remove(&subscription.id);
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Lifecycle
    // ──────────────────────────────────────────────────────────────────────────

    /// Start the stream in background (non-blocking)
    pub fn start(&mut self) -> Result<()> {
        if self.running.load(Ordering::SeqCst) {
            return Ok(());
        }

        self.running.store(true, Ordering::SeqCst);

        let (stop_tx, stop_rx) = mpsc::channel(1);
        self.stop_tx = Some(stop_tx);

        let host = self.host.clone();
        let token = self.token.clone();
        let state = self.state.clone();
        let running = self.running.clone();
        let reconnect_attempts = self.reconnect_attempts.clone();
        let subscriptions = self.subscriptions.clone();
        let callbacks = self.callbacks.clone();
        let bytes_callbacks = self.bytes_callbacks.clone();
        let last_seen_blocks = self.last_seen_blocks.clone();
        let config = self.config.clone();
        let on_error = self.on_error.clone();
        let on_close = self.on_close.clone();
        let on_connect = self.on_connect.clone();
        let on_reconnect = self.on_reconnect.clone();
        let on_state_change = self.on_state_change.clone();

        tokio::spawn(async move {
            Self::run_loop(
                host,
                token,
                state,
                running,
                reconnect_attempts,
                subscriptions,
                callbacks,
                bytes_callbacks,
                last_seen_blocks,
                config,
                on_error,
                on_close,
                on_connect,
                on_reconnect,
                on_state_change,
                stop_rx,
            )
            .await;
        });

        Ok(())
    }

    /// Run the stream (blocking)
    pub async fn run(&mut self) -> Result<()> {
        self.start()?;

        while self.running.load(Ordering::SeqCst) {
            sleep(Duration::from_millis(100)).await;
        }

        Ok(())
    }

    /// Stop the stream
    pub fn stop(&mut self) {
        self.running.store(false, Ordering::SeqCst);
        if let Some(tx) = self.stop_tx.take() {
            let _ = tx.try_send(());
        }
        self.set_state(ConnectionState::Disconnected);

        if let Some(ref cb) = self.on_close {
            cb();
        }
    }

    /// Ping the server
    pub async fn ping(&self) -> bool {
        if self.host.is_empty() {
            return false;
        }

        let target = format!("https://{}:{}", self.host, GRPC_PORT);

        let channel = match Channel::from_shared(target)
            .unwrap()
            .tls_config(ClientTlsConfig::new().with_native_roots())
            .unwrap()
            .connect()
            .await
        {
            Ok(c) => c,
            Err(_) => return false,
        };

        let token: MetadataValue<_> = self.token.parse().unwrap();
        let mut client = StreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
            req.metadata_mut().insert("x-token", token.clone());
            Ok(req)
        });

        match client.ping(PingRequest { count: 1 }).await {
            Ok(resp) => resp.into_inner().count == 1,
            Err(_) => false,
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn run_loop(
        host: String,
        token: String,
        state: Arc<RwLock<ConnectionState>>,
        running: Arc<AtomicBool>,
        reconnect_attempts: Arc<AtomicU32>,
        subscriptions: Arc<RwLock<HashMap<u32, GRPCSubscriptionInfo>>>,
        callbacks: Arc<RwLock<ValueCallbackMap>>,
        bytes_callbacks: Arc<RwLock<BytesCallbackMap>>,
        last_seen_blocks: Arc<RwLock<LastSeenBlockMap>>,
        config: GRPCStreamConfig,
        on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
        on_close: Option<Arc<dyn Fn() + Send + Sync>>,
        _on_connect: Option<Arc<dyn Fn() + Send + Sync>>,
        on_reconnect: Option<Arc<dyn Fn(u32) + Send + Sync>>,
        on_state_change: Option<Arc<dyn Fn(ConnectionState) + Send + Sync>>,
        mut stop_rx: mpsc::Receiver<()>,
    ) {
        let mut backoff = INITIAL_RECONNECT_DELAY;
        // False only for the very first connection attempt of this run;
        // reconnects adjust start_block / snapshot semantics (see
        // build_subscribe_request and build_l2_book_diff_request).
        let mut is_reconnect = false;

        while running.load(Ordering::SeqCst) {
            // Check for stop signal
            if stop_rx.try_recv().is_ok() {
                break;
            }

            // Update state
            {
                let mut s = state.write();
                if *s == ConnectionState::Reconnecting {
                    if let Some(ref cb) = on_reconnect {
                        cb(reconnect_attempts.load(Ordering::SeqCst));
                    }
                }
                *s = ConnectionState::Connecting;
            }
            if let Some(ref cb) = on_state_change {
                cb(ConnectionState::Connecting);
            }

            // Try to connect and stream
            let result = Self::connect_and_stream(
                &host,
                &token,
                &subscriptions,
                &callbacks,
                &bytes_callbacks,
                &last_seen_blocks,
                &running,
                is_reconnect,
                &mut stop_rx,
            )
            .await;
            is_reconnect = true;

            if let Err(e) = result {
                if let Some(ref cb) = on_error {
                    cb(e.to_string());
                }
            }

            if !running.load(Ordering::SeqCst) {
                break;
            }

            if !config.reconnect {
                break;
            }

            let attempts = reconnect_attempts.fetch_add(1, Ordering::SeqCst) + 1;
            if let Some(max) = config.max_reconnect_attempts {
                if attempts >= max {
                    break;
                }
            }

            {
                *state.write() = ConnectionState::Reconnecting;
            }
            if let Some(ref cb) = on_state_change {
                cb(ConnectionState::Reconnecting);
            }

            // Wait before reconnecting
            tokio::select! {
                _ = sleep(backoff) => {}
                _ = stop_rx.recv() => { break; }
            }

            backoff = Duration::from_secs_f64(
                (backoff.as_secs_f64() * RECONNECT_BACKOFF_FACTOR)
                    .min(MAX_RECONNECT_DELAY.as_secs_f64()),
            );
        }

        {
            *state.write() = ConnectionState::Disconnected;
        }
        if let Some(ref cb) = on_state_change {
            cb(ConnectionState::Disconnected);
        }
        if let Some(ref cb) = on_close {
            cb();
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn connect_and_stream(
        host: &str,
        token: &str,
        subscriptions: &Arc<RwLock<HashMap<u32, GRPCSubscriptionInfo>>>,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        bytes_callbacks: &Arc<RwLock<BytesCallbackMap>>,
        last_seen_blocks: &Arc<RwLock<LastSeenBlockMap>>,
        running: &Arc<AtomicBool>,
        is_reconnect: bool,
        stop_rx: &mut mpsc::Receiver<()>,
    ) -> Result<()> {
        if host.is_empty() {
            return Err(crate::error::Error::ConfigError(
                "No gRPC endpoint configured".to_string(),
            ));
        }

        let target = format!("https://{}:{}", host, GRPC_PORT);

        // Create channel with TLS using native system root certificates
        // (like Python's ssl_channel_credentials() and TypeScript's grpc.credentials.createSsl())
        let channel = Channel::from_shared(target)
            .map_err(|e| crate::error::Error::NetworkError(e.to_string()))?
            .tls_config(ClientTlsConfig::new().with_native_roots())
            .map_err(|e: tonic::transport::Error| crate::error::Error::NetworkError(e.to_string()))?
            .connect()
            .await
            .map_err(|e| crate::error::Error::NetworkError(format!("Failed to connect: {}", e)))?;

        // Get subscriptions snapshot
        let subs: Vec<(u32, GRPCSubscriptionInfo)> = {
            let guard = subscriptions.read();
            guard.iter().map(|(k, v)| (*k, v.clone())).collect()
        };

        // Start each subscription stream
        let mut handles = Vec::new();
        for (sub_id, sub_info) in subs {
            let channel = channel.clone();
            let token = token.to_string();
            let callbacks = callbacks.clone();
            let bytes_callbacks = bytes_callbacks.clone();
            let running = running.clone();
            // Per-subscription highest-block cursor, shared across reconnects so
            // resumable streams don't replay already-delivered blocks.
            let last_seen = {
                let mut map = last_seen_blocks.write();
                map.entry(sub_id)
                    .or_insert_with(|| Arc::new(AtomicU64::new(0)))
                    .clone()
            };

            let handle = tokio::spawn(async move {
                match sub_info.stream_type {
                    GRPCStreamType::L2Book => {
                        Self::stream_l2_book(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::L4Book => {
                        Self::stream_l4_book(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::BboBook => {
                        Self::stream_bbo_book(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::BboBookPacked => {
                        Self::stream_bbo_book_packed(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::L2BookDiff => {
                        Self::stream_l2_book_diff(
                            channel,
                            &token,
                            sub_id,
                            &sub_info,
                            &callbacks,
                            &running,
                            is_reconnect,
                        )
                        .await
                    }
                    GRPCStreamType::L4BookUpdates => {
                        Self::stream_l4_book_updates(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::TpslUpdates => {
                        Self::stream_tpsl_updates(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::L2BookPacked => {
                        Self::stream_l2_book_packed(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::L4BookBytes => {
                        Self::stream_l4_book_bytes(
                            channel, &token, sub_id, &sub_info, &callbacks, &running,
                        )
                        .await
                    }
                    GRPCStreamType::Blocks => {
                        Self::stream_blocks(channel, &token, sub_id, &callbacks, &running).await
                    }
                    _ if sub_info.bytes => {
                        Self::stream_data_bytes(
                            channel,
                            &token,
                            sub_id,
                            &sub_info,
                            &bytes_callbacks,
                            &running,
                            &last_seen,
                            is_reconnect,
                        )
                        .await
                    }
                    _ => {
                        Self::stream_data(
                            channel,
                            &token,
                            sub_id,
                            &sub_info,
                            &callbacks,
                            &running,
                            &last_seen,
                            is_reconnect,
                        )
                        .await
                    }
                }
            });
            handles.push(handle);
        }

        // Wait for stop signal or any stream to end
        loop {
            tokio::select! {
                _ = stop_rx.recv() => { break; }
                _ = sleep(Duration::from_secs(1)) => {
                    if !running.load(Ordering::SeqCst) {
                        break;
                    }
                    // Check if any handles finished
                    let mut all_done = true;
                    for h in &handles {
                        if !h.is_finished() {
                            all_done = false;
                            break;
                        }
                    }
                    if all_done && !handles.is_empty() {
                        break;
                    }
                }
            }
        }

        let mut stream_error = None;
        for handle in handles {
            match handle.await {
                Ok(Err(e)) if stream_error.is_none() => stream_error = Some(e),
                Ok(_) => {}
                Err(e) if stream_error.is_none() => {
                    stream_error = Some(crate::error::Error::NetworkError(format!(
                        "gRPC stream task failed: {e}"
                    )));
                }
                Err(_) => {}
            }
        }

        if let Some(err) = stream_error {
            Err(err)
        } else {
            Ok(())
        }
    }

    /// Build the StreamSubscribe request. On the first connect the user's
    /// original `start_block` (or unset) is sent verbatim. On reconnects, if a
    /// start_block was set, the cursor advances past the highest block already
    /// delivered so transient disconnects don't replay processed blocks. An
    /// unset start_block always stays unset (tip-following semantics).
    fn build_subscribe_request(
        sub_info: &GRPCSubscriptionInfo,
        is_reconnect: bool,
        last_seen_block: u64,
    ) -> SubscribeRequest {
        let mut filters = HashMap::new();
        if !sub_info.coins.is_empty() {
            filters.insert(
                "coin".to_string(),
                FilterValues {
                    values: sub_info.coins.clone(),
                },
            );
        }
        if !sub_info.users.is_empty() {
            filters.insert(
                "user".to_string(),
                FilterValues {
                    values: sub_info.users.clone(),
                },
            );
        }

        let start_block = match sub_info.start_block {
            Some(orig) if is_reconnect && last_seen_block > 0 => {
                orig.max(last_seen_block.saturating_add(1))
            }
            Some(orig) => orig,
            None => 0,
        };

        SubscribeRequest {
            request: Some(proto::subscribe_request::Request::Subscribe(
                StreamSubscribe {
                    stream_type: sub_info.stream_type.to_proto(),
                    start_block,
                    filters,
                    filter_name: String::new(),
                },
            )),
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn stream_data(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
        last_seen: &Arc<AtomicU64>,
        is_reconnect: bool,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client = StreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
            req.metadata_mut().insert("x-token", token_value.clone());
            Ok(req)
        });

        // Build subscribe request
        let subscribe_req =
            Self::build_subscribe_request(sub_info, is_reconnect, last_seen.load(Ordering::Relaxed));

        // Create bidirectional stream
        let (tx, rx) = tokio::sync::mpsc::channel(16);
        let outbound = tokio_stream::wrappers::ReceiverStream::new(rx);

        // Send initial subscribe
        if tx.send(subscribe_req).await.is_err() {
            return Ok(());
        }

        // Start ping task
        let tx_ping = tx.clone();
        let running_ping = running.clone();
        tokio::spawn(async move {
            loop {
                sleep(Duration::from_secs(30)).await;
                if !running_ping.load(Ordering::SeqCst) {
                    break;
                }
                let ping_req = SubscribeRequest {
                    request: Some(proto::subscribe_request::Request::Ping(Ping {
                        timestamp: chrono::Utc::now().timestamp_millis(),
                    })),
                };
                if tx_ping.send(ping_req).await.is_err() {
                    break;
                }
            }
        });

        // Call StreamData
        let response = match client.stream_data(outbound).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamData error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamData error: {e}"
                )));
            }
        };

        let mut inbound = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match inbound.message().await {
                Ok(Some(update)) => {
                    if let Some(proto::subscribe_update::Update::Data(data)) = update.update {
                        // Parse the JSON data. The resume cursor advances only
                        // after a successful parse, so a corrupt block is
                        // re-requested on reconnect instead of skipped.
                        if let Ok(parsed) = serde_json::from_str::<Value>(&data.data) {
                            last_seen.fetch_max(data.block_number, Ordering::Relaxed);
                            if sub_info.raw {
                                let mut data_with_meta =
                                    parsed.as_object().cloned().unwrap_or_default();
                                data_with_meta.insert(
                                    "_block_number".to_string(),
                                    Value::Number(data.block_number.into()),
                                );
                                data_with_meta.insert(
                                    "_timestamp".to_string(),
                                    Value::Number(data.timestamp.into()),
                                );

                                if let Some(cb) = callbacks.read().get(&sub_id) {
                                    cb(Value::Object(data_with_meta));
                                }
                                continue;
                            }

                            // Extract events if present
                            if let Some(events) = parsed.get("events").and_then(|e| e.as_array()) {
                                let mut emitted_events = false;
                                for (index, event) in events.iter().enumerate() {
                                    let mut user: Option<Value> = None;
                                    let mut event_data = None;

                                    if let Some(arr) = event.as_array() {
                                        if arr.len() >= 2 {
                                            user = Some(arr[0].clone());
                                            event_data = arr[1].as_object();
                                        }
                                    } else {
                                        event_data = event.as_object();
                                    }

                                    if let Some(event_data) = event_data {
                                        let mut data_with_meta = serde_json::Map::new();
                                        for (k, v) in event_data {
                                            data_with_meta.insert(k.clone(), v.clone());
                                        }
                                        data_with_meta.insert(
                                            "_block_number".to_string(),
                                            Value::Number(data.block_number.into()),
                                        );
                                        data_with_meta.insert(
                                            "_timestamp".to_string(),
                                            Value::Number(data.timestamp.into()),
                                        );
                                        data_with_meta.insert(
                                            "_event_index".to_string(),
                                            Value::Number((index as u64).into()),
                                        );
                                        if let Some(user) = user {
                                            data_with_meta.insert("_user".to_string(), user);
                                        }

                                        if let Some(cb) = callbacks.read().get(&sub_id) {
                                            cb(Value::Object(data_with_meta));
                                        }
                                        emitted_events = true;
                                    }
                                }
                                if !emitted_events {
                                    let mut data_with_meta =
                                        parsed.as_object().cloned().unwrap_or_default();
                                    data_with_meta.insert(
                                        "_block_number".to_string(),
                                        Value::Number(data.block_number.into()),
                                    );
                                    data_with_meta.insert(
                                        "_timestamp".to_string(),
                                        Value::Number(data.timestamp.into()),
                                    );

                                    if let Some(cb) = callbacks.read().get(&sub_id) {
                                        cb(Value::Object(data_with_meta));
                                    }
                                }
                            } else {
                                // No events, return raw data
                                let mut data_with_meta =
                                    parsed.as_object().cloned().unwrap_or_default();
                                data_with_meta.insert(
                                    "_block_number".to_string(),
                                    Value::Number(data.block_number.into()),
                                );
                                data_with_meta.insert(
                                    "_timestamp".to_string(),
                                    Value::Number(data.timestamp.into()),
                                );

                                if let Some(cb) = callbacks.read().get(&sub_id) {
                                    cb(Value::Object(data_with_meta));
                                }
                            }
                        }
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("Stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "Stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    #[allow(clippy::too_many_arguments)]
    async fn stream_data_bytes(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        bytes_callbacks: &Arc<RwLock<BytesCallbackMap>>,
        running: &Arc<AtomicBool>,
        last_seen: &Arc<AtomicU64>,
        is_reconnect: bool,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client = StreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
            req.metadata_mut().insert("x-token", token_value.clone());
            Ok(req)
        });

        // Build subscribe request
        let subscribe_req =
            Self::build_subscribe_request(sub_info, is_reconnect, last_seen.load(Ordering::Relaxed));

        // Create bidirectional stream
        let (tx, rx) = tokio::sync::mpsc::channel(16);
        let outbound = tokio_stream::wrappers::ReceiverStream::new(rx);

        // Send initial subscribe
        if tx.send(subscribe_req).await.is_err() {
            return Ok(());
        }

        // Start ping task
        let tx_ping = tx.clone();
        let running_ping = running.clone();
        tokio::spawn(async move {
            loop {
                sleep(Duration::from_secs(30)).await;
                if !running_ping.load(Ordering::SeqCst) {
                    break;
                }
                let ping_req = SubscribeRequest {
                    request: Some(proto::subscribe_request::Request::Ping(Ping {
                        timestamp: chrono::Utc::now().timestamp_millis(),
                    })),
                };
                if tx_ping.send(ping_req).await.is_err() {
                    break;
                }
            }
        });

        // Call StreamDataBytes
        let response = match client.stream_data_bytes(outbound).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamDataBytes error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamDataBytes error: {e}"
                )));
            }
        };

        let mut inbound = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match inbound.message().await {
                Ok(Some(update)) => {
                    if let Some(proto::subscribe_bytes_update::Update::Data(data)) = update.update
                    {
                        let block_number = data.block_number;
                        if let Some(cb) = bytes_callbacks.read().get(&sub_id) {
                            cb(data);
                        }
                        // Cursor advances only after delivery so reconnects
                        // never skip an undelivered block.
                        last_seen.fetch_max(block_number, Ordering::Relaxed);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("Bytes stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "Bytes stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_blocks(
        channel: Channel,
        token: &str,
        sub_id: u32,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            BlockStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = Timestamp {
            timestamp: chrono::Utc::now().timestamp_millis(),
        };

        let response = match client.stream_blocks(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamBlocks error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamBlocks error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(block)) => {
                    if let Ok(data) = serde_json::from_str::<Value>(&block.data_json) {
                        if let Some(cb) = callbacks.read().get(&sub_id) {
                            cb(data);
                        }
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("Block stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "Block stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_l2_book(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = L2BookRequest {
            coin: sub_info.coin.clone().unwrap_or_default(),
            n_levels: sub_info.n_levels.unwrap_or(20),
            n_sig_figs: sub_info.n_sig_figs,
            mantissa: None,
        };

        let response = match client.stream_l2_book(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamL2Book error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamL2Book error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let bids: Vec<Value> = update
                        .bids
                        .iter()
                        .map(|l| serde_json::json!([l.px, l.sz, l.n]))
                        .collect();
                    let asks: Vec<Value> = update
                        .asks
                        .iter()
                        .map(|l| serde_json::json!([l.px, l.sz, l.n]))
                        .collect();

                    let data = serde_json::json!({
                        "coin": update.coin,
                        "time": update.time,
                        "block_number": update.block_number,
                        "bids": bids,
                        "asks": asks,
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("L2 book stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "L2 book stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_l4_book(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = L4BookRequest {
            coin: sub_info.coin.clone().unwrap_or_default(),
        };

        let response = match client.stream_l4_book(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamL4Book error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamL4Book error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let Some(update) = update.update else {
                        continue;
                    };
                    let data = match update {
                        proto::l4_book_update::Update::Snapshot(snapshot) => {
                            let bids: Vec<Value> =
                                snapshot.bids.iter().map(l4_order_to_json).collect();
                            let asks: Vec<Value> =
                                snapshot.asks.iter().map(l4_order_to_json).collect();

                            serde_json::json!({
                                "type": "snapshot",
                                "coin": snapshot.coin,
                                "time": snapshot.time,
                                "height": snapshot.height,
                                "bids": bids,
                                "asks": asks,
                            })
                        }
                        proto::l4_book_update::Update::Diff(diff) => {
                            let diff_data: Value =
                                serde_json::from_str(&diff.data).unwrap_or(Value::Null);
                            serde_json::json!({
                                "type": "diff",
                                "time": diff.time,
                                "height": diff.height,
                                "data": diff_data,
                            })
                        }
                    };

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("L4 book stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "L4 book stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_bbo_book(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = BboBookRequest {
            coins: sub_info.coins.clone(),
        };

        let response = match client.stream_bbo_book(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamBboBook error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamBboBook error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let data = serde_json::json!({
                        "coin": update.coin,
                        "time": update.time,
                        "block_number": update.block_number,
                        "bid": update.bid.map(|l| serde_json::json!([l.px, l.sz, l.n])),
                        "ask": update.ask.map(|l| serde_json::json!([l.px, l.sz, l.n])),
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("BBO book stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "BBO book stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_bbo_book_packed(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = BboBookRequest {
            coins: sub_info.coins.clone(),
        };

        let response = match client.stream_bbo_book_packed(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamBboBookPacked error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamBboBookPacked error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let data = serde_json::json!({
                        "coin": update.coin,
                        "time": update.time,
                        "block_number": update.block_number,
                        "bid": update.bid.map(|l| serde_json::json!([l.px, l.sz, l.n])),
                        "ask": update.ask.map(|l| serde_json::json!([l.px, l.sz, l.n])),
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("Packed BBO book stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "Packed BBO book stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    #[allow(clippy::too_many_arguments)]
    async fn stream_l2_book_diff(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
        is_reconnect: bool,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = L2BookDiffRequest {
            coins: sub_info.coins.clone(),
            n_levels: sub_info.n_levels.unwrap_or(20),
            n_sig_figs: sub_info.n_sig_figs,
            mantissa: sub_info.mantissa,
            // skip_initial_snapshot only applies to the first connect: after a
            // reconnect the snapshot is required so the local book can resync.
            skip_initial_snapshot: if is_reconnect {
                false
            } else {
                sub_info.skip_initial_snapshot
            },
        };

        let response = match client.stream_l2_book_diff(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamL2BookDiff error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamL2BookDiff error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let diffs: Vec<Value> = update
                        .diffs
                        .iter()
                        .map(|d| {
                            let bids: Vec<Value> = d
                                .bids
                                .iter()
                                .map(|l| serde_json::json!([l.px, l.sz, l.n]))
                                .collect();
                            let asks: Vec<Value> = d
                                .asks
                                .iter()
                                .map(|l| serde_json::json!([l.px, l.sz, l.n]))
                                .collect();
                            serde_json::json!({
                                "coin": d.coin,
                                "seq": d.seq,
                                "prev_seq": d.prev_seq,
                                "bids": bids,
                                "asks": asks,
                                "snapshot": d.snapshot,
                            })
                        })
                        .collect();

                    let data = serde_json::json!({
                        "time": update.time,
                        "height": update.height,
                        "snapshot": update.snapshot,
                        "diffs": diffs,
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("L2 book diff stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "L2 book diff stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_l4_book_updates(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = L4BookUpdatesRequest {
            coins: sub_info.coins.clone(),
        };

        let response = match client.stream_l4_book_updates(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamL4BookUpdates error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamL4BookUpdates error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let diffs: Vec<Value> = update
                        .diffs
                        .iter()
                        .map(|d| {
                            serde_json::json!({
                                "diff_type": l4_order_diff_type_str(d.diff_type),
                                "coin": d.coin,
                                "oid": d.oid,
                                "user": d.user,
                                "side": d.side,
                                "px": d.px,
                                "sz": d.sz,
                            })
                        })
                        .collect();

                    let data = serde_json::json!({
                        "time": update.time,
                        "height": update.height,
                        "snapshot": update.snapshot,
                        "diffs": diffs,
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("L4 book updates stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "L4 book updates stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_tpsl_updates(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = TpslUpdatesRequest {
            coins: sub_info.coins.clone(),
        };

        let response = match client.stream_tpsl_updates(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamTpslUpdates error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamTpslUpdates error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let diffs: Vec<Value> = update
                        .diffs
                        .iter()
                        .map(|d| {
                            serde_json::json!({
                                "diff_type": tpsl_diff_type_str(d.diff_type),
                                "oid": d.oid,
                                "coin": d.coin,
                                "user": d.user,
                                "side": d.side,
                                "trigger_px": d.trigger_px,
                                "limit_px": d.limit_px,
                                "sz": d.sz,
                                "trigger_condition": d.trigger_condition,
                                "order_type": d.order_type,
                                "is_position_tpsl": d.is_position_tpsl,
                                "reduce_only": d.reduce_only,
                                "timestamp": d.timestamp,
                                "reason": d.reason,
                            })
                        })
                        .collect();

                    let data = serde_json::json!({
                        "time": update.time,
                        "height": update.height,
                        "snapshot": update.snapshot,
                        "diffs": diffs,
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("TPSL updates stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "TPSL updates stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_l2_book_packed(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = L2BookRequest {
            coin: sub_info.coin.clone().unwrap_or_default(),
            n_levels: sub_info.n_levels.unwrap_or(20),
            n_sig_figs: sub_info.n_sig_figs,
            mantissa: None,
        };

        let response = match client.stream_l2_book_packed(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamL2BookPacked error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamL2BookPacked error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let bids: Vec<Value> = update
                        .bids
                        .iter()
                        .map(|l| serde_json::json!([l.px, l.sz, l.n]))
                        .collect();
                    let asks: Vec<Value> = update
                        .asks
                        .iter()
                        .map(|l| serde_json::json!([l.px, l.sz, l.n]))
                        .collect();

                    let data = serde_json::json!({
                        "coin": update.coin,
                        "time": update.time,
                        "block_number": update.block_number,
                        "bids": bids,
                        "asks": asks,
                    });

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("Packed L2 book stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "Packed L2 book stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }

    async fn stream_l4_book_bytes(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<ValueCallbackMap>>,
        running: &Arc<AtomicBool>,
    ) -> Result<()> {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client =
            OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut().insert("x-token", token_value.clone());
                Ok(req)
            });

        let request = L4BookRequest {
            coin: sub_info.coin.clone().unwrap_or_default(),
        };

        let response = match client.stream_l4_book_bytes(request).await {
            Ok(r) => r,
            Err(e) => {
                tracing::error!("StreamL4BookBytes error: {}", e);
                if is_permanent_stream_error(&e) {
                    running.store(false, Ordering::SeqCst);
                }
                return Err(crate::error::Error::NetworkError(format!(
                    "StreamL4BookBytes error: {e}"
                )));
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let Some(update) = update.update else {
                        continue;
                    };
                    let data = match update {
                        proto::l4_book_bytes_update::Update::Snapshot(snapshot) => {
                            let bids: Vec<Value> =
                                snapshot.bids.iter().map(l4_order_to_json).collect();
                            let asks: Vec<Value> =
                                snapshot.asks.iter().map(l4_order_to_json).collect();

                            serde_json::json!({
                                "type": "snapshot",
                                "coin": snapshot.coin,
                                "time": snapshot.time,
                                "height": snapshot.height,
                                "bids": bids,
                                "asks": asks,
                            })
                        }
                        proto::l4_book_bytes_update::Update::Diff(diff) => {
                            let diff_data: Value =
                                serde_json::from_slice(&diff.data).unwrap_or(Value::Null);
                            serde_json::json!({
                                "type": "diff",
                                "time": diff.time,
                                "height": diff.height,
                                "data": diff_data,
                            })
                        }
                    };

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("L4 book bytes stream error: {}", e);
                    if is_permanent_stream_error(&e) {
                        running.store(false, Ordering::SeqCst);
                    }
                    return Err(crate::error::Error::NetworkError(format!(
                        "L4 book bytes stream error: {e}"
                    )));
                }
            }
        }

        Ok(())
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Helper Functions
// ══════════════════════════════════════════════════════════════════════════════

fn parse_endpoint(url: &str) -> (String, String) {
    let parsed = match url::Url::parse(url) {
        Ok(u) => u,
        Err(_) => return (String::new(), String::new()),
    };

    let host = parsed.host_str().unwrap_or("").to_string();

    // Extract token from path
    let path_parts: Vec<&str> = parsed.path().trim_matches('/').split('/').collect();
    let mut token = String::new();
    for part in path_parts {
        if !part.is_empty()
            && part != "info"
            && part != "hypercore"
            && part != "evm"
            && part != "nanoreth"
            && part != "ws"
        {
            token = part.to_string();
            break;
        }
    }

    (host, token)
}

fn l4_order_diff_type_str(diff_type: i32) -> &'static str {
    match proto::L4OrderDiffType::try_from(diff_type) {
        Ok(proto::L4OrderDiffType::New) => "new",
        Ok(proto::L4OrderDiffType::Update) => "update",
        Ok(proto::L4OrderDiffType::Remove) => "remove",
        _ => "unspecified",
    }
}

fn tpsl_diff_type_str(diff_type: i32) -> &'static str {
    match proto::TpslDiffType::try_from(diff_type) {
        Ok(proto::TpslDiffType::Add) => "add",
        Ok(proto::TpslDiffType::Remove) => "remove",
        _ => "unspecified",
    }
}

fn l4_order_to_json(order: &proto::L4Order) -> Value {
    serde_json::json!({
        "user": order.user,
        "coin": order.coin,
        "side": order.side,
        "limit_px": order.limit_px,
        "sz": order.sz,
        "oid": order.oid,
        "timestamp": order.timestamp,
        "trigger_condition": order.trigger_condition,
        "is_trigger": order.is_trigger,
        "trigger_px": order.trigger_px,
        "is_position_tpsl": order.is_position_tpsl,
        "reduce_only": order.reduce_only,
        "order_type": order.order_type,
        "tif": order.tif,
        "cloid": order.cloid,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stream_type_proto_values() {
        assert_eq!(GRPCStreamType::Trades.to_proto(), 1);
        assert_eq!(GRPCStreamType::Orders.to_proto(), 2);
        assert_eq!(GRPCStreamType::BookUpdates.to_proto(), 3);
        assert_eq!(GRPCStreamType::Twap.to_proto(), 4);
        assert_eq!(GRPCStreamType::Events.to_proto(), 5);
        assert_eq!(GRPCStreamType::Blocks.to_proto(), 6);
        assert_eq!(GRPCStreamType::WriterActions.to_proto(), 7);
        assert_eq!(GRPCStreamType::MempoolTxs.to_proto(), 8);
        assert_eq!(GRPCStreamType::OrderPriority.to_proto(), 9);
        assert_eq!(GRPCStreamType::GossipPriority.to_proto(), 10);

        // Order book streams go through dedicated RPCs, not StreamData.
        for pseudo in [
            GRPCStreamType::L2Book,
            GRPCStreamType::L4Book,
            GRPCStreamType::BboBook,
            GRPCStreamType::L2BookDiff,
            GRPCStreamType::L4BookUpdates,
            GRPCStreamType::TpslUpdates,
            GRPCStreamType::L2BookPacked,
            GRPCStreamType::BboBookPacked,
            GRPCStreamType::L4BookBytes,
        ] {
            assert_eq!(pseudo.to_proto(), 0);
        }
    }

    #[test]
    fn stream_type_names() {
        assert_eq!(GRPCStreamType::MempoolTxs.as_str(), "mempool_txs");
        assert_eq!(GRPCStreamType::OrderPriority.as_str(), "order_priority");
        assert_eq!(GRPCStreamType::GossipPriority.as_str(), "gossip_priority");
        assert_eq!(GRPCStreamType::BboBook.as_str(), "bbo_book");
        assert_eq!(GRPCStreamType::L2BookDiff.as_str(), "l2_book_diff");
        assert_eq!(GRPCStreamType::L4BookUpdates.as_str(), "l4_book_updates");
        assert_eq!(GRPCStreamType::TpslUpdates.as_str(), "tpsl_updates");
        assert_eq!(GRPCStreamType::L2BookPacked.as_str(), "l2_book_packed");
        assert_eq!(GRPCStreamType::BboBookPacked.as_str(), "bbo_book_packed");
        assert_eq!(GRPCStreamType::L4BookBytes.as_str(), "l4_book_bytes");
    }

    #[test]
    fn mempool_txs_registers_coin_filter_subscription() {
        let mut stream = GRPCStream::new(None);
        let sub = stream.mempool_txs(&["BTC", "ETH"], |_| {});

        assert_eq!(sub.stream_type, GRPCStreamType::MempoolTxs);
        let subs = stream.subscriptions.read();
        let info = subs.get(&sub.id).unwrap();
        assert_eq!(info.coins, vec!["BTC".to_string(), "ETH".to_string()]);
        assert!(!info.raw);
        assert!(!info.bytes);

        // Coin filter is carried via the generic filters map.
        let req = GRPCStream::build_subscribe_request(info, false, 0);
        let Some(proto::subscribe_request::Request::Subscribe(sub_msg)) = req.request else {
            panic!("expected subscribe request");
        };
        assert_eq!(sub_msg.stream_type, 8);
        assert_eq!(
            sub_msg.filters.get("coin").unwrap().values,
            vec!["BTC".to_string(), "ETH".to_string()]
        );
    }

    #[test]
    fn start_block_is_plumbed_into_subscribe_request() {
        let mut stream = GRPCStream::new(None);
        let sub = stream.trades_with_options(
            &["BTC"],
            GRPCSubscriptionOptions {
                start_block: Some(123_456),
            },
            |_| {},
        );

        let subs = stream.subscriptions.read();
        let info = subs.get(&sub.id).unwrap();
        assert_eq!(info.start_block, Some(123_456));

        let req = GRPCStream::build_subscribe_request(info, false, 0);
        let Some(proto::subscribe_request::Request::Subscribe(sub_msg)) = req.request else {
            panic!("expected subscribe request");
        };
        assert_eq!(sub_msg.start_block, 123_456);
    }

    #[test]
    fn reconnect_advances_start_block_cursor() {
        let mut stream = GRPCStream::new(None);
        let sub = stream.trades_with_options(
            &["BTC"],
            GRPCSubscriptionOptions {
                start_block: Some(100),
            },
            |_| {},
        );

        let subs = stream.subscriptions.read();
        let info = subs.get(&sub.id).unwrap();

        let extract = |req: SubscribeRequest| -> u64 {
            let Some(proto::subscribe_request::Request::Subscribe(sub_msg)) = req.request else {
                panic!("expected subscribe request");
            };
            sub_msg.start_block
        };

        // First connect: the user's original start_block, even if data flowed before.
        assert_eq!(extract(GRPCStream::build_subscribe_request(info, false, 500)), 100);
        // Reconnect after seeing block 500: resume past it.
        assert_eq!(extract(GRPCStream::build_subscribe_request(info, true, 500)), 501);
        // Reconnect before any data: keep the original.
        assert_eq!(extract(GRPCStream::build_subscribe_request(info, true, 0)), 100);
        // Reconnect where the original is still ahead of last-seen: keep the original.
        assert_eq!(extract(GRPCStream::build_subscribe_request(info, true, 50)), 100);
    }

    #[test]
    fn reconnect_keeps_unset_start_block_unset() {
        let mut stream = GRPCStream::new(None);
        let sub = stream.trades(&["BTC"], |_| {});

        let subs = stream.subscriptions.read();
        let info = subs.get(&sub.id).unwrap();
        assert_eq!(info.start_block, None);

        // Tip-following subscriptions never grow a cursor on reconnect.
        let req = GRPCStream::build_subscribe_request(info, true, 12_345);
        let Some(proto::subscribe_request::Request::Subscribe(sub_msg)) = req.request else {
            panic!("expected subscribe request");
        };
        assert_eq!(sub_msg.start_block, 0);
    }

    #[test]
    fn raw_bytes_registers_bytes_subscription() {
        let mut stream = GRPCStream::new(None);
        let sub = stream.raw_bytes(
            GRPCStreamType::Trades,
            &["BTC"],
            GRPCSubscriptionOptions::default(),
            |_| {},
        );

        let subs = stream.subscriptions.read();
        let info = subs.get(&sub.id).unwrap();
        assert!(info.bytes);
        assert!(stream.bytes_callbacks.read().contains_key(&sub.id));
    }

    #[test]
    fn l2_book_diff_options_defaults() {
        let options = GRPCL2BookDiffOptions::default();
        assert_eq!(options.n_levels, 20);
        assert_eq!(options.n_sig_figs, None);
        assert_eq!(options.mantissa, None);
        assert!(!options.skip_initial_snapshot);
    }

    #[test]
    fn diff_type_strings() {
        assert_eq!(l4_order_diff_type_str(1), "new");
        assert_eq!(l4_order_diff_type_str(2), "update");
        assert_eq!(l4_order_diff_type_str(3), "remove");
        assert_eq!(l4_order_diff_type_str(0), "unspecified");
        assert_eq!(tpsl_diff_type_str(1), "add");
        assert_eq!(tpsl_diff_type_str(2), "remove");
        assert_eq!(tpsl_diff_type_str(0), "unspecified");
    }
}
