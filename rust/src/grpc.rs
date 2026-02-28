//! gRPC streaming client for Hyperliquid.
//!
//! Provides low-latency real-time data streaming via gRPC.
//! Enable with the `grpc` feature.

use parking_lot::RwLock;
use serde_json::Value;
use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicU32, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::time::sleep;

use crate::error::Result;
use crate::stream::ConnectionState;

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
            keepalive_interval: Duration::from_secs(30),
            keepalive_timeout: Duration::from_secs(10),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// gRPC Stream
// ══════════════════════════════════════════════════════════════════════════════

/// gRPC stream client
pub struct GRPCStream {
    config: GRPCStreamConfig,
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
}

struct GRPCSubscriptionInfo {
    #[allow(dead_code)]
    stream_type: GRPCStreamType,
    #[allow(dead_code)]
    coins: Vec<String>,
}

impl GRPCStream {
    /// Create a new gRPC stream client
    pub fn new(endpoint: Option<String>) -> Self {
        Self {
            config: GRPCStreamConfig {
                endpoint,
                ..Default::default()
            },
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
        }
    }

    /// Configure stream options
    pub fn configure(mut self, config: GRPCStreamConfig) -> Self {
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
        *self.state.write() = state;
        if let Some(ref cb) = self.on_state_change {
            cb(state);
        }
    }

    fn get_grpc_url(&self) -> String {
        if let Some(ref endpoint) = self.config.endpoint {
            // Parse endpoint to extract token and build proper gRPC URL
            let info = crate::client::EndpointInfo::parse(endpoint);
            info.build_grpc_url()
        } else {
            // No public gRPC endpoint available
            String::new()
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
        self.book_updates(&[coin], callback)
    }

    /// Subscribe to L4 order book (individual orders with OIDs)
    pub fn l4_book<F>(&mut self, coin: &str, callback: F) -> GRPCSubscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        // L4 is delivered through orders stream with full order details
        self.orders(&[coin], callback)
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

        let grpc_url = self.get_grpc_url();
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
                grpc_url,
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
        self.set_state(ConnectionState::Disconnected);

        if let Some(ref cb) = self.on_close {
            cb();
        }
    }

    async fn run_loop(
        _grpc_url: String,
        state: Arc<RwLock<ConnectionState>>,
        running: Arc<AtomicBool>,
        reconnect_attempts: Arc<AtomicU32>,
        _subscriptions: Arc<RwLock<HashMap<u32, GRPCSubscriptionInfo>>>,
        _callbacks: Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        config: GRPCStreamConfig,
        on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
        on_close: Option<Arc<dyn Fn() + Send + Sync>>,
        _on_connect: Option<Arc<dyn Fn() + Send + Sync>>,
        on_reconnect: Option<Arc<dyn Fn(u32) + Send + Sync>>,
        on_state_change: Option<Arc<dyn Fn(ConnectionState) + Send + Sync>>,
    ) {
        // Note: Full gRPC implementation requires generated protobuf code
        // This is a placeholder that demonstrates the structure.
        // In production, you would use tonic with generated stubs.

        let mut backoff = Duration::from_secs(1);
        let max_backoff = Duration::from_secs(60);

        while running.load(Ordering::SeqCst) {
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

            // TODO: Implement actual gRPC connection using tonic
            // For now, we'll just simulate connection failure to trigger reconnect
            if let Some(ref cb) = on_error {
                cb("gRPC client not fully implemented - requires protobuf stubs".to_string());
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

            sleep(backoff).await;
            backoff = (backoff * 2).min(max_backoff);
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
}
