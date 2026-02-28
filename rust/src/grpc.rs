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

use parking_lot::RwLock;
use serde_json::Value;
use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicU32, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::time::sleep;
use tonic::metadata::MetadataValue;
use tonic::transport::{Channel, ClientTlsConfig};
use tonic::Request;

use crate::error::Result;
use crate::stream::ConnectionState;

// Include generated protobuf code
pub mod proto {
    tonic::include_proto!("hyperliquid");
}

use proto::streaming_client::StreamingClient;
use proto::block_streaming_client::BlockStreamingClient;
use proto::order_book_streaming_client::OrderBookStreamingClient;
use proto::{
    FilterValues, L2BookRequest, L4BookRequest, Ping, PingRequest, StreamSubscribe,
    SubscribeRequest, Timestamp,
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
    L2Book,
    L4Book,
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
            GRPCStreamType::L2Book => "l2_book",
            GRPCStreamType::L4Book => "l4_book",
        }
    }

    /// Convert to proto enum value
    fn to_proto(&self) -> i32 {
        match self {
            GRPCStreamType::Trades => 1,
            GRPCStreamType::Orders => 2,
            GRPCStreamType::BookUpdates => 3,
            GRPCStreamType::Twap => 4,
            GRPCStreamType::Events => 5,
            GRPCStreamType::Blocks => 6,
            GRPCStreamType::WriterActions => 7,
            GRPCStreamType::L2Book => 0,
            GRPCStreamType::L4Book => 0,
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

struct GRPCSubscriptionInfo {
    stream_type: GRPCStreamType,
    coins: Vec<String>,
    users: Vec<String>,
    coin: Option<String>,
    n_levels: Option<u32>,
    n_sig_figs: Option<u32>,
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
    callbacks: Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
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
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        GRPCSubscription {
            id,
            stream_type: GRPCStreamType::WriterActions,
        }
    }

    /// Unsubscribe
    pub fn unsubscribe(&mut self, subscription: &GRPCSubscription) {
        self.subscriptions.write().remove(&subscription.id);
        self.callbacks.write().remove(&subscription.id);
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
        let mut client =
            StreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
                req.metadata_mut()
                    .insert("x-token", token.clone());
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
        callbacks: Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        config: GRPCStreamConfig,
        on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
        on_close: Option<Arc<dyn Fn() + Send + Sync>>,
        _on_connect: Option<Arc<dyn Fn() + Send + Sync>>,
        on_reconnect: Option<Arc<dyn Fn(u32) + Send + Sync>>,
        on_state_change: Option<Arc<dyn Fn(ConnectionState) + Send + Sync>>,
        mut stop_rx: mpsc::Receiver<()>,
    ) {
        let mut backoff = INITIAL_RECONNECT_DELAY;

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
                &running,
                &mut stop_rx,
            )
            .await;

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
                (backoff.as_secs_f64() * RECONNECT_BACKOFF_FACTOR).min(MAX_RECONNECT_DELAY.as_secs_f64())
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

    async fn connect_and_stream(
        host: &str,
        token: &str,
        subscriptions: &Arc<RwLock<HashMap<u32, GRPCSubscriptionInfo>>>,
        callbacks: &Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        running: &Arc<AtomicBool>,
        stop_rx: &mut mpsc::Receiver<()>,
    ) -> Result<()> {
        if host.is_empty() {
            return Err(crate::error::Error::ConfigError("No gRPC endpoint configured".to_string()));
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
            guard
                .iter()
                .map(|(k, v)| {
                    (
                        *k,
                        GRPCSubscriptionInfo {
                            stream_type: v.stream_type,
                            coins: v.coins.clone(),
                            users: v.users.clone(),
                            coin: v.coin.clone(),
                            n_levels: v.n_levels,
                            n_sig_figs: v.n_sig_figs,
                        },
                    )
                })
                .collect()
        };

        // Start each subscription stream
        let mut handles = Vec::new();
        for (sub_id, sub_info) in subs {
            let channel = channel.clone();
            let token = token.to_string();
            let callbacks = callbacks.clone();
            let running = running.clone();

            let handle = tokio::spawn(async move {
                match sub_info.stream_type {
                    GRPCStreamType::L2Book => {
                        Self::stream_l2_book(channel, &token, sub_id, &sub_info, &callbacks, &running).await;
                    }
                    GRPCStreamType::L4Book => {
                        Self::stream_l4_book(channel, &token, sub_id, &sub_info, &callbacks, &running).await;
                    }
                    GRPCStreamType::Blocks => {
                        Self::stream_blocks(channel, &token, sub_id, &callbacks, &running).await;
                    }
                    _ => {
                        Self::stream_data(channel, &token, sub_id, &sub_info, &callbacks, &running).await;
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

        Ok(())
    }

    async fn stream_data(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        running: &Arc<AtomicBool>,
    ) {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client = StreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
            req.metadata_mut().insert("x-token", token_value.clone());
            Ok(req)
        });

        // Build subscribe request
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

        let subscribe_req = SubscribeRequest {
            request: Some(proto::subscribe_request::Request::Subscribe(StreamSubscribe {
                stream_type: sub_info.stream_type.to_proto(),
                filters,
                filter_name: String::new(),
            })),
        };

        // Create bidirectional stream
        let (tx, rx) = tokio::sync::mpsc::channel(16);
        let outbound = tokio_stream::wrappers::ReceiverStream::new(rx);

        // Send initial subscribe
        if tx.send(subscribe_req).await.is_err() {
            return;
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
                return;
            }
        };

        let mut inbound = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match inbound.message().await {
                Ok(Some(update)) => {
                    if let Some(proto::subscribe_update::Update::Data(data)) = update.update {
                        // Parse the JSON data
                        if let Ok(parsed) = serde_json::from_str::<Value>(&data.data) {
                            // Extract events if present
                            if let Some(events) = parsed.get("events").and_then(|e| e.as_array()) {
                                for event in events {
                                    if let Some(arr) = event.as_array() {
                                        if arr.len() >= 2 {
                                            let user = arr[0].as_str().unwrap_or("");
                                            if let Some(event_data) = arr[1].as_object() {
                                                let mut data_with_meta = serde_json::Map::new();
                                                for (k, v) in event_data {
                                                    data_with_meta.insert(k.clone(), v.clone());
                                                }
                                                data_with_meta.insert("_block_number".to_string(), Value::Number(data.block_number.into()));
                                                data_with_meta.insert("_timestamp".to_string(), Value::Number(data.timestamp.into()));
                                                data_with_meta.insert("_user".to_string(), Value::String(user.to_string()));

                                                if let Some(cb) = callbacks.read().get(&sub_id) {
                                                    cb(Value::Object(data_with_meta));
                                                }
                                            }
                                        }
                                    }
                                }
                            } else {
                                // No events, return raw data
                                let mut data_with_meta = parsed.as_object().cloned().unwrap_or_default();
                                data_with_meta.insert("_block_number".to_string(), Value::Number(data.block_number.into()));
                                data_with_meta.insert("_timestamp".to_string(), Value::Number(data.timestamp.into()));

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
                    break;
                }
            }
        }
    }

    async fn stream_blocks(
        channel: Channel,
        token: &str,
        sub_id: u32,
        callbacks: &Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        running: &Arc<AtomicBool>,
    ) {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client = BlockStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
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
                return;
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
                    break;
                }
            }
        }
    }

    async fn stream_l2_book(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        running: &Arc<AtomicBool>,
    ) {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client = OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
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
                return;
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
                    break;
                }
            }
        }
    }

    async fn stream_l4_book(
        channel: Channel,
        token: &str,
        sub_id: u32,
        sub_info: &GRPCSubscriptionInfo,
        callbacks: &Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        running: &Arc<AtomicBool>,
    ) {
        let token_value: MetadataValue<_> = token.parse().unwrap();
        let mut client = OrderBookStreamingClient::with_interceptor(channel, move |mut req: Request<()>| {
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
                return;
            }
        };

        let mut stream = response.into_inner();

        while running.load(Ordering::SeqCst) {
            match stream.message().await {
                Ok(Some(update)) => {
                    let data = if let Some(proto::l4_book_update::Update::Snapshot(snapshot)) = update.update {
                        let bids: Vec<Value> = snapshot.bids.iter().map(l4_order_to_json).collect();
                        let asks: Vec<Value> = snapshot.asks.iter().map(l4_order_to_json).collect();

                        serde_json::json!({
                            "type": "snapshot",
                            "coin": snapshot.coin,
                            "time": snapshot.time,
                            "height": snapshot.height,
                            "bids": bids,
                            "asks": asks,
                        })
                    } else if let Some(proto::l4_book_update::Update::Diff(diff)) = update.update {
                        let diff_data: Value = serde_json::from_str(&diff.data).unwrap_or(Value::Null);
                        serde_json::json!({
                            "type": "diff",
                            "time": diff.time,
                            "height": diff.height,
                            "data": diff_data,
                        })
                    } else {
                        continue;
                    };

                    if let Some(cb) = callbacks.read().get(&sub_id) {
                        cb(data);
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("L4 book stream error: {}", e);
                    break;
                }
            }
        }
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
