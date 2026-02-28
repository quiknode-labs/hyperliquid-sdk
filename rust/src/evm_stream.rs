//! EVM WebSocket streaming client for HyperEVM.
//!
//! Stream EVM events via WebSocket on the /nanoreth namespace:
//! - newHeads: New block headers
//! - logs: Contract event logs
//! - newPendingTransactions: Pending transaction hashes

use futures_util::{SinkExt, StreamExt};
use parking_lot::RwLock;
use serde_json::{json, Value};
use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicU32, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::time::sleep;
use tokio_tungstenite::{connect_async, tungstenite::Message};

use crate::error::Result;

// ══════════════════════════════════════════════════════════════════════════════
// EVM Subscription Types
// ══════════════════════════════════════════════════════════════════════════════

/// EVM WebSocket subscription types (eth_subscribe)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum EVMSubscriptionType {
    /// New block headers
    NewHeads,
    /// Contract event logs
    Logs,
    /// Pending transaction hashes
    NewPendingTransactions,
}

impl EVMSubscriptionType {
    /// Get the subscription type string for eth_subscribe
    pub fn as_str(&self) -> &'static str {
        match self {
            EVMSubscriptionType::NewHeads => "newHeads",
            EVMSubscriptionType::Logs => "logs",
            EVMSubscriptionType::NewPendingTransactions => "newPendingTransactions",
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// Connection State
// ══════════════════════════════════════════════════════════════════════════════

/// Connection state for EVM WebSocket
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EVMConnectionState {
    Disconnected,
    Connecting,
    Connected,
    Reconnecting,
}

// ══════════════════════════════════════════════════════════════════════════════
// EVM Subscription
// ══════════════════════════════════════════════════════════════════════════════

/// An EVM subscription handle
#[derive(Debug, Clone)]
pub struct EVMSubscription {
    pub id: u32,
    pub sub_type: EVMSubscriptionType,
}

// ══════════════════════════════════════════════════════════════════════════════
// EVM Stream Configuration
// ══════════════════════════════════════════════════════════════════════════════

/// EVM stream configuration
#[derive(Clone)]
pub struct EVMStreamConfig {
    pub endpoint: Option<String>,
    pub reconnect: bool,
    pub max_reconnect_attempts: Option<u32>,
    pub reconnect_delay: Duration,
    pub ping_interval: Duration,
    pub ping_timeout: Duration,
}

impl Default for EVMStreamConfig {
    fn default() -> Self {
        Self {
            endpoint: None,
            reconnect: true,
            max_reconnect_attempts: Some(10),
            reconnect_delay: Duration::from_secs(1),
            ping_interval: Duration::from_secs(30),
            ping_timeout: Duration::from_secs(10),
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════════
// EVM Stream
// ══════════════════════════════════════════════════════════════════════════════

/// EVM WebSocket streaming client for HyperEVM
///
/// Stream EVM events via WebSocket on the /nanoreth namespace.
///
/// # Example
///
/// ```rust,no_run
/// use hyperliquid_sdk::EVMStream;
///
/// #[tokio::main]
/// async fn main() -> Result<(), Box<dyn std::error::Error>> {
///     let mut stream = EVMStream::new(Some("https://your-endpoint.quiknode.pro/TOKEN".to_string()));
///
///     // Subscribe to new block headers
///     stream.new_heads(|header| {
///         println!("New block: {:?}", header);
///     });
///
///     // Subscribe to contract logs
///     stream.logs(
///         Some(serde_json::json!({"address": "0x..."})),
///         |log| println!("Log: {:?}", log),
///     );
///
///     stream.start()?;
///     Ok(())
/// }
/// ```
pub struct EVMStream {
    config: EVMStreamConfig,
    state: Arc<RwLock<EVMConnectionState>>,
    running: Arc<AtomicBool>,
    reconnect_count: Arc<AtomicU32>,
    request_id: Arc<AtomicU32>,
    pending_subscriptions: Arc<RwLock<Vec<PendingSubscription>>>,
    active_subscriptions: Arc<RwLock<HashMap<String, SubscriptionInfo>>>,
    callbacks: Arc<RwLock<HashMap<String, Box<dyn Fn(Value) + Send + Sync>>>>,
    on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
    on_close: Option<Arc<dyn Fn() + Send + Sync>>,
    on_open: Option<Arc<dyn Fn() + Send + Sync>>,
    on_state_change: Option<Arc<dyn Fn(EVMConnectionState) + Send + Sync>>,
}

struct PendingSubscription {
    sub_type: EVMSubscriptionType,
    params: Option<Value>,
    callback: Box<dyn Fn(Value) + Send + Sync>,
}

struct SubscriptionInfo {
    #[allow(dead_code)]
    sub_type: EVMSubscriptionType,
}

impl EVMStream {
    /// Create a new EVM stream client
    pub fn new(endpoint: Option<String>) -> Self {
        Self {
            config: EVMStreamConfig {
                endpoint,
                ..Default::default()
            },
            state: Arc::new(RwLock::new(EVMConnectionState::Disconnected)),
            running: Arc::new(AtomicBool::new(false)),
            reconnect_count: Arc::new(AtomicU32::new(0)),
            request_id: Arc::new(AtomicU32::new(0)),
            pending_subscriptions: Arc::new(RwLock::new(Vec::new())),
            active_subscriptions: Arc::new(RwLock::new(HashMap::new())),
            callbacks: Arc::new(RwLock::new(HashMap::new())),
            on_error: None,
            on_close: None,
            on_open: None,
            on_state_change: None,
        }
    }

    /// Configure stream options
    pub fn configure(mut self, config: EVMStreamConfig) -> Self {
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

    /// Set state change callback
    pub fn on_state_change<F>(mut self, f: F) -> Self
    where
        F: Fn(EVMConnectionState) + Send + Sync + 'static,
    {
        self.on_state_change = Some(Arc::new(f));
        self
    }

    /// Get current connection state
    pub fn state(&self) -> EVMConnectionState {
        *self.state.read()
    }

    /// Check if connected
    pub fn connected(&self) -> bool {
        *self.state.read() == EVMConnectionState::Connected
    }

    fn set_state(&self, state: EVMConnectionState) {
        *self.state.write() = state;
        if let Some(ref cb) = self.on_state_change {
            cb(state);
        }
    }

    fn get_ws_url(&self) -> String {
        if let Some(ref endpoint) = self.config.endpoint {
            // QuickNode endpoint - build /nanoreth WebSocket URL
            let base = endpoint
                .trim_end_matches('/')
                .replace("https://", "wss://")
                .replace("http://", "ws://")
                .replace("/info", "")
                .replace("/evm", "")
                .replace("/hypercore", "");

            // Extract token from path
            if let Ok(url) = url::Url::parse(&base) {
                if let Some(host) = url.host_str() {
                    let path = url.path().trim_matches('/');
                    let parts: Vec<&str> = path.split('/').collect();
                    // Find the token (first part that's not a known path)
                    for part in parts {
                        if !part.is_empty()
                            && !["info", "hypercore", "evm", "nanoreth", "ws"].contains(&part)
                        {
                            let scheme = if base.starts_with("wss") { "wss" } else { "ws" };
                            return format!("{}://{}/{}/nanoreth", scheme, host, part);
                        }
                    }
                }
            }
            format!("{}/nanoreth", base)
        } else {
            // No endpoint available
            "wss://api.hyperliquid.xyz/nanoreth".to_string()
        }
    }

    #[allow(dead_code)]
    fn next_request_id(&self) -> u32 {
        self.request_id.fetch_add(1, Ordering::SeqCst)
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Subscription Methods
    // ──────────────────────────────────────────────────────────────────────────

    /// Subscribe to new block headers
    ///
    /// Fires a notification each time a new header is appended to the chain,
    /// including during chain reorganizations.
    pub fn new_heads<F>(&mut self, callback: F) -> &mut Self
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.pending_subscriptions.write().push(PendingSubscription {
            sub_type: EVMSubscriptionType::NewHeads,
            params: None,
            callback: Box::new(callback),
        });
        self
    }

    /// Subscribe to contract event logs
    ///
    /// Returns logs that are included in new imported blocks and match
    /// the given filter criteria.
    ///
    /// # Arguments
    ///
    /// * `filter` - Filter parameters with optional `address` and `topics`
    /// * `callback` - Function called with each matching log
    pub fn logs<F>(&mut self, filter: Option<Value>, callback: F) -> &mut Self
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.pending_subscriptions.write().push(PendingSubscription {
            sub_type: EVMSubscriptionType::Logs,
            params: filter,
            callback: Box::new(callback),
        });
        self
    }

    /// Subscribe to pending transaction hashes
    ///
    /// Returns the hash for all transactions that are added to the pending state.
    pub fn new_pending_transactions<F>(&mut self, callback: F) -> &mut Self
    where
        F: Fn(Value) + Send + Sync + 'static,
    {
        self.pending_subscriptions
            .write()
            .push(PendingSubscription {
                sub_type: EVMSubscriptionType::NewPendingTransactions,
                params: None,
                callback: Box::new(callback),
            });
        self
    }

    /// Get list of active subscription IDs
    pub fn subscriptions(&self) -> Vec<String> {
        self.active_subscriptions.read().keys().cloned().collect()
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

        let ws_url = self.get_ws_url();
        let state = self.state.clone();
        let running = self.running.clone();
        let reconnect_count = self.reconnect_count.clone();
        let request_id = self.request_id.clone();
        let pending_subscriptions = self.pending_subscriptions.clone();
        let active_subscriptions = self.active_subscriptions.clone();
        let callbacks = self.callbacks.clone();
        let config = self.config.clone();
        let on_error = self.on_error.clone();
        let on_close = self.on_close.clone();
        let on_open = self.on_open.clone();
        let on_state_change = self.on_state_change.clone();

        tokio::spawn(async move {
            Self::run_loop(
                ws_url,
                state,
                running,
                reconnect_count,
                request_id,
                pending_subscriptions,
                active_subscriptions,
                callbacks,
                config,
                on_error,
                on_close,
                on_open,
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
        self.set_state(EVMConnectionState::Disconnected);

        if let Some(ref cb) = self.on_close {
            cb();
        }
    }

    async fn run_loop(
        ws_url: String,
        state: Arc<RwLock<EVMConnectionState>>,
        running: Arc<AtomicBool>,
        reconnect_count: Arc<AtomicU32>,
        request_id: Arc<AtomicU32>,
        pending_subscriptions: Arc<RwLock<Vec<PendingSubscription>>>,
        active_subscriptions: Arc<RwLock<HashMap<String, SubscriptionInfo>>>,
        callbacks: Arc<RwLock<HashMap<String, Box<dyn Fn(Value) + Send + Sync>>>>,
        config: EVMStreamConfig,
        on_error: Option<Arc<dyn Fn(String) + Send + Sync>>,
        on_close: Option<Arc<dyn Fn() + Send + Sync>>,
        on_open: Option<Arc<dyn Fn() + Send + Sync>>,
        on_state_change: Option<Arc<dyn Fn(EVMConnectionState) + Send + Sync>>,
    ) {
        let mut backoff = config.reconnect_delay;
        let max_backoff = Duration::from_secs(30);

        while running.load(Ordering::SeqCst) {
            // Update state
            {
                *state.write() = EVMConnectionState::Connecting;
            }
            if let Some(ref cb) = on_state_change {
                cb(EVMConnectionState::Connecting);
            }

            // Connect
            match connect_async(&ws_url).await {
                Ok((ws_stream, _)) => {
                    {
                        *state.write() = EVMConnectionState::Connected;
                    }
                    if let Some(ref cb) = on_state_change {
                        cb(EVMConnectionState::Connected);
                    }
                    if let Some(ref cb) = on_open {
                        cb();
                    }

                    reconnect_count.store(0, Ordering::SeqCst);
                    backoff = config.reconnect_delay;

                    let (mut write, mut read) = ws_stream.split();

                    // Send pending subscriptions
                    let pending: Vec<_> = {
                        let mut pending = pending_subscriptions.write();
                        pending.drain(..).collect()
                    };

                    // Track request IDs for callback mapping
                    let mut req_to_callback: HashMap<u32, Box<dyn Fn(Value) + Send + Sync>> =
                        HashMap::new();

                    for sub in pending {
                        let req_id = request_id.fetch_add(1, Ordering::SeqCst);
                        let mut params = vec![json!(sub.sub_type.as_str())];
                        if let Some(p) = sub.params {
                            params.push(p);
                        }

                        let msg = json!({
                            "jsonrpc": "2.0",
                            "method": "eth_subscribe",
                            "params": params,
                            "id": req_id,
                        });

                        req_to_callback.insert(req_id, sub.callback);

                        if write.send(Message::Text(msg.to_string().into())).await.is_err() {
                            break;
                        }
                    }

                    // Message loop
                    while running.load(Ordering::SeqCst) {
                        match tokio::time::timeout(config.ping_timeout, read.next()).await {
                            Ok(Some(Ok(Message::Text(text)))) => {
                                if let Ok(data) = serde_json::from_str::<Value>(&text) {
                                    // Check for subscription confirmation
                                    if let (Some(id), Some(result)) =
                                        (data.get("id"), data.get("result"))
                                    {
                                        if let Some(id_num) = id.as_u64() {
                                            if let Some(callback) =
                                                req_to_callback.remove(&(id_num as u32))
                                            {
                                                if let Some(sub_id) = result.as_str() {
                                                    callbacks
                                                        .write()
                                                        .insert(sub_id.to_string(), callback);
                                                    active_subscriptions.write().insert(
                                                        sub_id.to_string(),
                                                        SubscriptionInfo {
                                                            sub_type: EVMSubscriptionType::NewHeads,
                                                        },
                                                    );
                                                }
                                            }
                                        }
                                    }

                                    // Check for subscription data
                                    if data.get("method") == Some(&json!("eth_subscription")) {
                                        if let Some(params) = data.get("params") {
                                            if let Some(sub_id) =
                                                params.get("subscription").and_then(|s| s.as_str())
                                            {
                                                if let Some(result) = params.get("result") {
                                                    let callbacks_read = callbacks.read();
                                                    if let Some(callback) =
                                                        callbacks_read.get(sub_id)
                                                    {
                                                        callback(result.clone());
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                            Ok(Some(Ok(Message::Close(_)))) => {
                                break;
                            }
                            Ok(Some(Err(e))) => {
                                if let Some(ref cb) = on_error {
                                    cb(e.to_string());
                                }
                                break;
                            }
                            Ok(None) => {
                                break;
                            }
                            Err(_) => {
                                // Timeout - send ping
                                if write.send(Message::Ping(vec![].into())).await.is_err() {
                                    break;
                                }
                            }
                            _ => {}
                        }
                    }
                }
                Err(e) => {
                    if let Some(ref cb) = on_error {
                        cb(format!("Connection failed: {}", e));
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

            let attempts = reconnect_count.fetch_add(1, Ordering::SeqCst) + 1;
            if let Some(max) = config.max_reconnect_attempts {
                if attempts >= max {
                    break;
                }
            }

            {
                *state.write() = EVMConnectionState::Reconnecting;
            }
            if let Some(ref cb) = on_state_change {
                cb(EVMConnectionState::Reconnecting);
            }

            sleep(backoff).await;
            backoff = (backoff * 2).min(max_backoff);
        }

        {
            *state.write() = EVMConnectionState::Disconnected;
        }
        if let Some(ref cb) = on_state_change {
            cb(EVMConnectionState::Disconnected);
        }
        if let Some(ref cb) = on_close {
            cb();
        }
    }
}
