//! WebSocket streaming client for Hyperliquid.
//!
//! Provides real-time market data and user event streaming.

use futures_util::{SinkExt, StreamExt};
use parking_lot::RwLock;
use serde_json::{json, Value};
use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicU32, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::time::sleep;
use tokio_tungstenite::{connect_async, tungstenite::Message};

use crate::error::Result;

// ══════════════════════════════════════════════════════════════════════════════
// Connection State
// ══════════════════════════════════════════════════════════════════════════════

/// WebSocket connection state
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ConnectionState {
    Disconnected,
    Connecting,
    Connected,
    Reconnecting,
}

// ══════════════════════════════════════════════════════════════════════════════
// Subscription
// ══════════════════════════════════════════════════════════════════════════════

/// A subscription handle
#[derive(Debug, Clone)]
pub struct Subscription {
    pub id: u32,
    pub channel: String,
}

// ══════════════════════════════════════════════════════════════════════════════
// Stream Configuration
// ══════════════════════════════════════════════════════════════════════════════

/// Stream configuration
#[derive(Clone)]
pub struct StreamConfig {
    pub endpoint: Option<String>,
    pub reconnect: bool,
    pub max_reconnect_attempts: Option<u32>,
    pub ping_interval: Duration,
    pub ping_timeout: Duration,
}

impl Default for StreamConfig {
    fn default() -> Self {
        Self {
            endpoint: None,
            reconnect: true,
            max_reconnect_attempts: None, // Infinite
            ping_interval: Duration::from_secs(30),
            ping_timeout: Duration::from_secs(10),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Stream
// ══════════════════════════════════════════════════════════════════════════════

/// WebSocket stream client
pub struct Stream {
    config: StreamConfig,
    is_quicknode: bool,
    jsonrpc_id: Arc<AtomicU32>,
    state: Arc<RwLock<ConnectionState>>,
    running: Arc<AtomicBool>,
    reconnect_attempts: Arc<AtomicU32>,
    subscription_id: Arc<AtomicU32>,
    subscriptions: Arc<RwLock<HashMap<u32, SubscriptionInfo>>>,
    callbacks: Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
    on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
    on_close: Option<Arc<dyn Fn() + Send + Sync>>,
    on_open: Option<Arc<dyn Fn() + Send + Sync>>,
    on_reconnect: Option<Arc<dyn Fn(u32) + Send + Sync>>,
    on_state_change: Option<Arc<dyn Fn(ConnectionState) + Send + Sync>>,
    command_tx: Option<mpsc::Sender<StreamCommand>>,
}

struct SubscriptionInfo {
    channel: String,
    params: Value,
}

#[allow(dead_code)]
enum StreamCommand {
    Subscribe { id: u32, channel: String, params: Value },
    Unsubscribe { id: u32 },
    Stop,
}

impl Stream {
    /// Create a new stream client
    pub fn new(endpoint: Option<String>) -> Self {
        let is_quicknode = endpoint.as_ref()
            .map(|e| e.contains("quiknode.pro"))
            .unwrap_or(false);
        Self {
            config: StreamConfig {
                endpoint,
                ..Default::default()
            },
            is_quicknode,
            jsonrpc_id: Arc::new(AtomicU32::new(0)),
            state: Arc::new(RwLock::new(ConnectionState::Disconnected)),
            running: Arc::new(AtomicBool::new(false)),
            reconnect_attempts: Arc::new(AtomicU32::new(0)),
            subscription_id: Arc::new(AtomicU32::new(0)),
            subscriptions: Arc::new(RwLock::new(HashMap::new())),
            callbacks: Arc::new(RwLock::new(HashMap::new())),
            on_error: None,
            on_close: None,
            on_open: None,
            on_reconnect: None,
            on_state_change: None,
            command_tx: None,
        }
    }

    /// Configure stream options
    pub fn configure(mut self, config: StreamConfig) -> Self {
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

    /// Set open callback
    pub fn on_open<F>(mut self, f: F) -> Self
    where
        F: Fn() + Send + Sync + 'static,
    {
        self.on_open = Some(Arc::new(f));
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

    /// Get reconnect attempts
    pub fn reconnect_attempts(&self) -> u32 {
        self.reconnect_attempts.load(Ordering::SeqCst)
    }

    fn set_state(&self, state: ConnectionState) {
        *self.state.write() = state;
        if let Some(ref cb) = self.on_state_change {
            cb(state);
        }
    }

    fn get_ws_url(&self) -> String {
        if let Some(ref endpoint) = self.config.endpoint {
            // Parse endpoint to extract token and build proper WebSocket URL
            let info = crate::client::EndpointInfo::parse(endpoint);
            info.build_ws_url()
        } else {
            // Public WebSocket endpoint
            "wss://api.hyperliquid.xyz/ws".to_string()
        }
    }

    fn next_subscription_id(&self) -> u32 {
        self.subscription_id.fetch_add(1, Ordering::SeqCst)
    }

    // ──────────────────────────────────────────────────────────────────────────
    // QuickNode Streams (snake_case)
    // ──────────────────────────────────────────────────────────────────────────

    /// Subscribe to trades
    pub fn trades<F>(&mut self, coins: &[&str], callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"coins": coins});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "trades".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        if let Some(tx) = &self.command_tx {
            let _ = tx.try_send(StreamCommand::Subscribe {
                id,
                channel: "trades".to_string(),
                params,
            });
        }

        Subscription {
            id,
            channel: "trades".to_string(),
        }
    }

    /// Subscribe to orders
    pub fn orders<F>(&mut self, coins: &[&str], callback: F, users: Option<&[&str]>) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let mut params = json!({"coins": coins});
        if let Some(u) = users {
            params["users"] = json!(u);
        }

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "orders".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        if let Some(tx) = &self.command_tx {
            let _ = tx.try_send(StreamCommand::Subscribe {
                id,
                channel: "orders".to_string(),
                params,
            });
        }

        Subscription {
            id,
            channel: "orders".to_string(),
        }
    }

    /// Subscribe to book updates
    pub fn book_updates<F>(&mut self, coins: &[&str], callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"coins": coins});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "book_updates".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        if let Some(tx) = &self.command_tx {
            let _ = tx.try_send(StreamCommand::Subscribe {
                id,
                channel: "book_updates".to_string(),
                params,
            });
        }

        Subscription {
            id,
            channel: "book_updates".to_string(),
        }
    }

    /// Subscribe to TWAP updates
    pub fn twap<F>(&mut self, coins: &[&str], callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"coins": coins});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "twap".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "twap".to_string(),
        }
    }

    /// Subscribe to events
    pub fn events<F>(&mut self, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "events".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "events".to_string(),
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Public API Streams (camelCase)
    // ──────────────────────────────────────────────────────────────────────────

    /// Subscribe to L2 order book
    pub fn l2_book<F>(&mut self, coin: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "l2Book", "coin": coin});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "l2Book".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "l2Book".to_string(),
        }
    }

    /// Subscribe to all mid prices
    pub fn all_mids<F>(&mut self, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "allMids"});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "allMids".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "allMids".to_string(),
        }
    }

    /// Subscribe to candles
    pub fn candle<F>(&mut self, coin: &str, interval: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "candle", "coin": coin, "interval": interval});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "candle".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "candle".to_string(),
        }
    }

    /// Subscribe to user open orders
    pub fn open_orders<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "openOrders", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "openOrders".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "openOrders".to_string(),
        }
    }

    /// Subscribe to order updates
    pub fn order_updates<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "orderUpdates", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "orderUpdates".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "orderUpdates".to_string(),
        }
    }

    /// Subscribe to user events
    pub fn user_events<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "userEvents", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "userEvents".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "userEvents".to_string(),
        }
    }

    /// Subscribe to user fills
    pub fn user_fills<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "userFills", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "userFills".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "userFills".to_string(),
        }
    }

    /// Subscribe to user fundings
    pub fn user_fundings<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "userFundings", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "userFundings".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "userFundings".to_string(),
        }
    }

    /// Subscribe to user non-funding ledger updates
    pub fn user_non_funding_ledger<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "userNonFundingLedgerUpdates", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "userNonFundingLedgerUpdates".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "userNonFundingLedgerUpdates".to_string(),
        }
    }

    /// Subscribe to clearinghouse state updates
    pub fn clearinghouse_state<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "clearinghouseState", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "clearinghouseState".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "clearinghouseState".to_string(),
        }
    }

    /// Subscribe to best bid/offer
    pub fn bbo<F>(&mut self, coin: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "bbo", "coin": coin});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "bbo".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "bbo".to_string(),
        }
    }

    /// Subscribe to active asset context
    pub fn active_asset_ctx<F>(&mut self, coin: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "activeAssetCtx", "coin": coin});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "activeAssetCtx".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "activeAssetCtx".to_string(),
        }
    }

    /// Subscribe to active asset data for a user
    pub fn active_asset_data<F>(&mut self, user: &str, coin: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "activeAssetData", "user": user, "coin": coin});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "activeAssetData".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "activeAssetData".to_string(),
        }
    }

    /// Subscribe to TWAP states
    pub fn twap_states<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "twapStates", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "twapStates".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "twapStates".to_string(),
        }
    }

    /// Subscribe to user TWAP slice fills
    pub fn user_twap_slice_fills<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "userTwapSliceFills", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "userTwapSliceFills".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "userTwapSliceFills".to_string(),
        }
    }

    /// Subscribe to user TWAP history
    pub fn user_twap_history<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "userTwapHistory", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "userTwapHistory".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "userTwapHistory".to_string(),
        }
    }

    /// Subscribe to notifications
    pub fn notification<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "notification", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "notification".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "notification".to_string(),
        }
    }

    /// Subscribe to web data 3 (aggregate user info)
    pub fn web_data_3<F>(&mut self, user: &str, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "webData3", "user": user});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "webData3".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "webData3".to_string(),
        }
    }

    /// Subscribe to writer actions (spot token transfers)
    pub fn writer_actions<F>(&mut self, callback: F) -> Subscription
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        let id = self.next_subscription_id();
        let params = json!({"type": "writer_actions"});

        self.subscriptions.write().insert(
            id,
            SubscriptionInfo {
                channel: "writer_actions".to_string(),
                params: params.clone(),
            },
        );
        self.callbacks.write().insert(id, Box::new(callback));

        Subscription {
            id,
            channel: "writer_actions".to_string(),
        }
    }

    /// Unsubscribe from a channel
    pub fn unsubscribe(&mut self, subscription: &Subscription) {
        self.subscriptions.write().remove(&subscription.id);
        self.callbacks.write().remove(&subscription.id);

        if let Some(tx) = &self.command_tx {
            let _ = tx.try_send(StreamCommand::Unsubscribe { id: subscription.id });
        }
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
        let (tx, rx) = mpsc::channel(100);
        self.command_tx = Some(tx);

        let ws_url = self.get_ws_url();
        let is_quicknode = self.is_quicknode;
        let jsonrpc_id = self.jsonrpc_id.clone();
        let state = self.state.clone();
        let running = self.running.clone();
        let reconnect_attempts = self.reconnect_attempts.clone();
        let subscriptions = self.subscriptions.clone();
        let callbacks = self.callbacks.clone();
        let config = self.config.clone();
        let on_error = self.on_error.clone();
        let on_close = self.on_close.clone();
        let on_open = self.on_open.clone();
        let on_reconnect = self.on_reconnect.clone();
        let on_state_change = self.on_state_change.clone();

        tokio::spawn(async move {
            Self::run_loop(
                ws_url,
                is_quicknode,
                jsonrpc_id,
                state,
                running,
                reconnect_attempts,
                subscriptions,
                callbacks,
                config,
                rx,
                on_error,
                on_close,
                on_open,
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

        // Wait until stopped
        while self.running.load(Ordering::SeqCst) {
            sleep(Duration::from_millis(100)).await;
        }

        Ok(())
    }

    /// Stop the stream
    pub fn stop(&mut self) {
        self.running.store(false, Ordering::SeqCst);

        if let Some(tx) = self.command_tx.take() {
            let _ = tx.try_send(StreamCommand::Stop);
        }

        self.set_state(ConnectionState::Disconnected);

        if let Some(ref cb) = self.on_close {
            cb();
        }
    }

    async fn run_loop(
        ws_url: String,
        is_quicknode: bool,
        jsonrpc_id: Arc<AtomicU32>,
        state: Arc<RwLock<ConnectionState>>,
        running: Arc<AtomicBool>,
        reconnect_attempts: Arc<AtomicU32>,
        subscriptions: Arc<RwLock<HashMap<u32, SubscriptionInfo>>>,
        callbacks: Arc<RwLock<HashMap<u32, Box<dyn Fn(Value) + Send + Sync>>>>,
        config: StreamConfig,
        mut command_rx: mpsc::Receiver<StreamCommand>,
        on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
        on_close: Option<Arc<dyn Fn() + Send + Sync>>,
        on_open: Option<Arc<dyn Fn() + Send + Sync>>,
        on_reconnect: Option<Arc<dyn Fn(u32) + Send + Sync>>,
        on_state_change: Option<Arc<dyn Fn(ConnectionState) + Send + Sync>>,
    ) {
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

            // Connect
            match connect_async(&ws_url).await {
                Ok((ws_stream, _)) => {
                    // Connected
                    {
                        *state.write() = ConnectionState::Connected;
                    }
                    if let Some(ref cb) = on_state_change {
                        cb(ConnectionState::Connected);
                    }
                    if let Some(ref cb) = on_open {
                        cb();
                    }

                    // Reset backoff
                    backoff = Duration::from_secs(1);
                    reconnect_attempts.store(0, Ordering::SeqCst);

                    let (mut ws_write, mut ws_read) = ws_stream.split();

                    // Send existing subscriptions
                    // Collect subscription data first to avoid holding lock across await
                    let sub_messages: Vec<String> = {
                        let subs = subscriptions.read();
                        subs.iter()
                            .filter_map(|(_, info)| {
                                let msg = if is_quicknode {
                                    // QuickNode JSON-RPC format
                                    let mut qn_params = json!({
                                        "streamType": info.channel
                                    });
                                    // Add filters if specified
                                    let mut filters = serde_json::Map::new();
                                    if let Some(coins) = info.params.get("coins") {
                                        filters.insert("coin".to_string(), coins.clone());
                                    }
                                    if let Some(users) = info.params.get("users") {
                                        filters.insert("user".to_string(), users.clone());
                                    }
                                    if !filters.is_empty() {
                                        qn_params["filters"] = Value::Object(filters);
                                    }
                                    json!({
                                        "jsonrpc": "2.0",
                                        "method": "hl_subscribe",
                                        "params": qn_params,
                                        "id": jsonrpc_id.fetch_add(1, Ordering::SeqCst)
                                    })
                                } else {
                                    json!({
                                        "method": "subscribe",
                                        "subscription": {
                                            "type": info.channel,
                                            "params": info.params,
                                        }
                                    })
                                };
                                serde_json::to_string(&msg).ok()
                            })
                            .collect()
                    };
                    for text in sub_messages {
                        let _ = ws_write.send(Message::Text(text.into())).await;
                    }

                    // Message loop
                    loop {
                        tokio::select! {
                            msg = ws_read.next() => {
                                match msg {
                                    Some(Ok(Message::Text(text))) => {
                                        if let Ok(data) = serde_json::from_str::<Value>(&text) {
                                            // Dispatch to callbacks
                                            let cbs = callbacks.read();
                                            for (_, cb) in cbs.iter() {
                                                cb(data.clone());
                                            }
                                        }
                                    }
                                    Some(Ok(Message::Ping(data))) => {
                                        let _ = ws_write.send(Message::Pong(data)).await;
                                    }
                                    Some(Ok(Message::Close(_))) | None => {
                                        break;
                                    }
                                    Some(Err(e)) => {
                                        if let Some(ref cb) = on_error {
                                            cb(e.to_string());
                                        }
                                        break;
                                    }
                                    _ => {}
                                }
                            }
                            cmd = command_rx.recv() => {
                                match cmd {
                                    Some(StreamCommand::Subscribe { id: _, channel, params }) => {
                                        let msg = if is_quicknode {
                                            // QuickNode JSON-RPC format
                                            let mut qn_params = json!({
                                                "streamType": channel
                                            });
                                            // Add filters if specified
                                            let mut filters = serde_json::Map::new();
                                            if let Some(coins) = params.get("coins") {
                                                filters.insert("coin".to_string(), coins.clone());
                                            }
                                            if let Some(users) = params.get("users") {
                                                filters.insert("user".to_string(), users.clone());
                                            }
                                            if !filters.is_empty() {
                                                qn_params["filters"] = Value::Object(filters);
                                            }
                                            json!({
                                                "jsonrpc": "2.0",
                                                "method": "hl_subscribe",
                                                "params": qn_params,
                                                "id": jsonrpc_id.fetch_add(1, Ordering::SeqCst)
                                            })
                                        } else {
                                            // Public API format
                                            json!({
                                                "method": "subscribe",
                                                "subscription": {
                                                    "type": channel,
                                                    "params": params,
                                                }
                                            })
                                        };
                                        if let Ok(text) = serde_json::to_string(&msg) {
                                            let _ = ws_write.send(Message::Text(text.into())).await;
                                        }
                                    }
                                    Some(StreamCommand::Unsubscribe { id }) => {
                                        let msg = if is_quicknode {
                                            json!({
                                                "jsonrpc": "2.0",
                                                "method": "hl_unsubscribe",
                                                "params": { "id": id },
                                                "id": jsonrpc_id.fetch_add(1, Ordering::SeqCst)
                                            })
                                        } else {
                                            json!({
                                                "method": "unsubscribe",
                                                "subscription": id,
                                            })
                                        };
                                        if let Ok(text) = serde_json::to_string(&msg) {
                                            let _ = ws_write.send(Message::Text(text.into())).await;
                                        }
                                    }
                                    Some(StreamCommand::Stop) | None => {
                                        break;
                                    }
                                }
                            }
                        }
                    }
                }
                Err(e) => {
                    if let Some(ref cb) = on_error {
                        cb(e.to_string());
                    }
                }
            }

            // Check if we should reconnect
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

            // Update state
            {
                *state.write() = ConnectionState::Reconnecting;
            }
            if let Some(ref cb) = on_state_change {
                cb(ConnectionState::Reconnecting);
            }

            // Wait before reconnecting
            sleep(backoff).await;
            backoff = (backoff * 2).min(max_backoff);
        }

        // Final cleanup
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
