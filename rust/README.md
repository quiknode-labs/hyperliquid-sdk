# Hyperliquid SDK for Rust

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```rust
use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let sdk = HyperliquidSDK::new().build().await?;
    let order = sdk.market_buy("BTC").await.notional(100.0).await?;  // Buy $100 of BTC
    Ok(())
}
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

> **Community SDK** — Not affiliated with Hyperliquid Foundation.

## Installation

Add to your `Cargo.toml`:

```toml
[dependencies]
quicknode-hyperliquid-sdk = "0.1"
tokio = { version = "1", features = ["full"] }
```

That's it. Everything is included: trading, Info API, WebSocket streaming, gRPC streaming, HyperCore, and EVM.

## Quick Start

### Endpoint Flexibility

The SDK automatically handles any endpoint format you provide:

```rust
// All of these work - the SDK extracts the token and routes correctly
let endpoint = "https://x.quiknode.pro/TOKEN";
let endpoint = "https://x.quiknode.pro/TOKEN/";
let endpoint = "https://x.quiknode.pro/TOKEN/info";
let endpoint = "https://x.quiknode.pro/TOKEN/hypercore";
let endpoint = "https://x.quiknode.pro/TOKEN/evm";
```

Just pass your endpoint - the SDK handles the rest.

### 1. Set your private key

```bash
export PRIVATE_KEY="0xYOUR_PRIVATE_KEY"
```

### 2. Start trading

```rust
use hyperliquid_sdk::{HyperliquidSDK, Order, TIF};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let sdk = HyperliquidSDK::new()
        .endpoint("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
        .build()
        .await?;

    // Market orders
    let order = sdk.market_buy("BTC").await.size(0.001).await?;
    let order = sdk.market_sell("ETH").await.notional(100.0).await?;  // $100 worth

    // Limit orders
    let order = sdk.buy("BTC", 0.001, 65000.0, TIF::Gtc).await?;

    // Check your order
    println!("Status: {:?}", order.status());  // "filled" or "resting"
    println!("Order ID: {:?}", order.oid);

    Ok(())
}
```

---

## Data APIs

Query Hyperliquid data with clean, simple interfaces.

### Info API

50+ methods for account state, positions, market data, and metadata.

```rust
let info = sdk.info();

// Market data
info.all_mids(None).await?;                          // All mid prices
info.l2_book("BTC", None, None).await?;              // Order book
info.recent_trades("BTC").await?;                    // Recent trades
info.candles("BTC", "1h", start, None).await?;       // OHLCV candles
info.funding_history("BTC", start, end).await?;      // Funding history
info.predicted_fundings().await?;                    // Predicted funding rates

// Metadata
info.meta().await?;                                  // Exchange metadata
info.spot_meta().await?;                             // Spot metadata
info.exchange_status().await?;                       // Exchange status
info.perp_dexs().await?;                             // Perpetual DEX info
info.max_market_order_ntls().await?;                 // Max market order notionals

// User data
info.clearinghouse_state("0x...", None).await?;      // Positions & margin
info.spot_clearinghouse_state("0x...", None).await?; // Spot balances
info.open_orders("0x...", None).await?;              // Open orders
info.frontend_open_orders("0x...", None).await?;     // Enhanced open orders
info.order_status("0x...", oid).await?;              // Specific order status
info.historical_orders("0x...").await?;              // Order history
info.user_fills("0x...", false).await?;              // Trade history
info.user_fills_by_time("0x...", start).await?;      // Fills by time range
info.user_funding("0x...").await?;                   // Funding payments
info.user_fees("0x...").await?;                      // Fee structure
info.user_rate_limit("0x...").await?;                // Rate limit status
info.user_role("0x...").await?;                      // Account type
info.portfolio("0x...").await?;                      // Portfolio history
info.sub_accounts("0x...").await?;                   // Sub-accounts
info.extra_agents("0x...").await?;                   // API keys/agents

// TWAP
info.user_twap_slice_fills("0x...").await?;          // TWAP slice fills

// Batch queries
info.batch_clearinghouse_states(&["0x...", "0x..."]).await?;

// Vaults
info.vault_summaries().await?;                       // All vault summaries
info.vault_details("0x...", None).await?;            // Specific vault
info.user_vault_equities("0x...").await?;            // User's vault equities
info.leading_vaults("0x...").await?;                 // Vaults user leads

// Delegation/Staking
info.delegations("0x...").await?;                    // Active delegations
info.delegator_summary("0x...").await?;              // Delegation summary
info.delegator_history("0x...").await?;              // Delegation history
info.delegator_rewards("0x...").await?;              // Delegation rewards

// Tokens
info.token_details("token_id").await?;               // Token details
info.spot_deploy_state("0x...").await?;              // Spot deployment state

// Other
info.referral("0x...").await?;                       // Referral info
info.max_builder_fee("0x...", "0x...").await?;       // Builder fee limits
info.approved_builders("0x...").await?;              // Approved builders
info.liquidatable().await?;                          // Liquidatable positions
```

### HyperCore API

Block data, trading operations, and real-time data via JSON-RPC.

```rust
let core = sdk.core();

// Block data
core.latest_block_number(Some("trades")).await?;     // Latest block
core.get_block(12345, Some("trades")).await?;        // Get specific block
core.get_batch_blocks(100, 110, Some("trades")).await?; // Get block range
core.latest_blocks(Some("trades"), Some(10)).await?; // Latest blocks

// Recent data
core.latest_trades(Some(10), None).await?;           // Recent trades (all coins)
core.latest_trades(Some(10), Some("BTC")).await?;    // Recent BTC trades
core.latest_orders(Some(10)).await?;                 // Recent order events
core.latest_book_updates(Some(10)).await?;           // Recent book updates

// Discovery
core.list_dexes().await?;                            // All DEXes
core.list_markets(None).await?;                      // All markets
core.list_markets(Some("hyperliquidity")).await?;    // Markets by DEX

// Order queries
core.open_orders("0x...").await?;                    // User's open orders
core.order_status("0x...", oid).await?;              // Specific order status
core.preflight(...).await?;                          // Validate order before signing

// Order building (for manual signing)
core.build_order(coin, is_buy, limit_px, sz, user).await?;
core.build_cancel(coin, oid, user).await?;
core.build_modify(coin, oid, user, limit_px, sz).await?;
core.build_approve_builder_fee(user, builder, rate, nonce).await?;
core.build_revoke_builder_fee(user, builder, nonce).await?;

// Send signed actions
core.send_order(&action, &signature, nonce).await?;
core.send_cancel(&action, &signature, nonce).await?;
core.send_modify(&action, &signature, nonce).await?;
core.send_approval(&action, &signature).await?;
core.send_revocation(&action, &signature).await?;

// Builder fees
core.get_max_builder_fee("0x...", "0x...").await?;

// Subscriptions
core.subscribe(&json!({"type": "trades", "coin": "BTC"})).await?;
core.unsubscribe(&json!({"type": "trades", "coin": "BTC"})).await?;
```

### EVM (Ethereum JSON-RPC)

50+ Ethereum JSON-RPC methods for Hyperliquid's EVM chain (chain ID 999 mainnet, 998 testnet).

```rust
let evm = sdk.evm();

// Chain info
evm.block_number().await?;                           // Latest block
evm.chain_id().await?;                               // 999 mainnet, 998 testnet
evm.gas_price().await?;                              // Current gas price
evm.max_priority_fee_per_gas().await?;               // Priority fee
evm.net_version().await?;                            // Network version
evm.syncing().await?;                                // Sync status

// Accounts
evm.get_balance("0x...", None).await?;               // Account balance
evm.get_transaction_count("0x...", None).await?;     // Nonce
evm.get_code("0x...", None).await?;                  // Contract code
evm.get_storage_at("0x...", "0x0", None).await?;     // Storage value

// Transactions
evm.call(&tx, None).await?;
evm.estimate_gas(&tx).await?;
evm.send_raw_transaction(&signed_tx).await?;
evm.get_transaction_by_hash("0x...").await?;
evm.get_transaction_receipt("0x...").await?;

// Blocks
evm.get_block_by_number("latest", false).await?;
evm.get_block_by_hash("0x...", false).await?;
evm.get_block_receipts("latest").await?;
evm.get_block_transaction_count_by_number("latest").await?;
evm.get_block_transaction_count_by_hash("0x...").await?;
evm.get_transaction_by_block_number_and_index("latest", 0).await?;
evm.get_transaction_by_block_hash_and_index("0x...", 0).await?;

// Logs
evm.get_logs(&json!({"address": "0x...", "topics": [...]})).await?;

// Fees
evm.fee_history(10, "latest", Some(&[25.0, 50.0, 75.0])).await?;

// Debug/Trace (use .with_debug(true))
let evm = sdk.evm().with_debug(true);
evm.debug_trace_transaction("0x...", Some(&json!({"tracer": "callTracer"}))).await?;
evm.trace_transaction("0x...").await?;
evm.trace_block("latest").await?;
```

---

## Real-Time Streaming

### WebSocket Streaming

20+ subscription types for real-time data with automatic reconnection.

```rust
use hyperliquid_sdk::Stream;

let mut stream = Stream::new(Some(endpoint))
    .on_open(|| println!("Connected!"))
    .on_error(|e| eprintln!("Error: {}", e))
    .on_reconnect(|n| println!("Reconnecting... attempt {}", n));

// Subscribe to trades
stream.trades(&["BTC", "ETH"], |t| println!("Trade: {:?}", t));

// Subscribe to book updates
stream.book_updates(&["BTC"], |b| println!("Book: {:?}", b));

// Subscribe to orders (your orders)
stream.orders(&["BTC"], |o| println!("Order: {:?}", o), Some(&["0x..."]));

// Run in background
stream.start()?;
// ... do other work ...
stream.stop()?;

// Or run blocking
stream.run()?;
```

**Available WebSocket Streams:**

**Market Data:**
- `trades(coins, callback)` — Executed trades
- `book_updates(coins, callback)` — Order book changes
- `l2_book(coin, callback)` — L2 order book snapshots
- `all_mids(callback)` — All mid price updates
- `candle(coin, interval, callback)` — Candlestick data
- `bbo(coin, callback)` — Best bid/offer updates
- `active_asset_ctx(coin, callback)` — Asset context (pricing, volume)

**User Data:**
- `orders(coins, callback, users)` — Order lifecycle events
- `open_orders(user, callback)` — User's open orders
- `order_updates(user, callback)` — Order status changes
- `user_events(user, callback)` — All user events
- `user_fills(user, callback)` — Trade fills
- `user_fundings(user, callback)` — Funding payments
- `user_non_funding_ledger(user, callback)` — Ledger changes
- `clearinghouse_state(user, callback)` — Position updates
- `active_asset_data(user, coin, callback)` — Trading parameters

**TWAP:**
- `twap(coins, callback)` — TWAP execution
- `twap_states(user, callback)` — TWAP algorithm states
- `user_twap_slice_fills(user, callback)` — TWAP slice fills
- `user_twap_history(user, callback)` — TWAP history

**System:**
- `events(callback)` — System events (funding, liquidations)
- `notification(user, callback)` — User notifications
- `web_data_3(user, callback)` — Aggregate user info
- `writer_actions(user, callback)` — Writer actions

### gRPC Streaming (High Performance)

Lower latency streaming via gRPC for high-frequency applications.

```rust
use hyperliquid_sdk::GRPCStream;

let mut stream = GRPCStream::new(endpoint)?;

// Subscribe to trades
stream.trades(&["BTC", "ETH"], |t| println!("Trade: {:?}", t));

// Subscribe to L2 order book (aggregated by price level)
stream.l2_book("BTC", |b| println!("Book: {:?}", b), Some(5));

// Subscribe to L4 order book (CRITICAL: individual orders with order IDs)
stream.l4_book("BTC", |b| println!("L4: {:?}", b));

// Subscribe to blocks
stream.blocks(|b| println!("Block: {:?}", b));

// Run in background
stream.start()?;
// ... do other work ...
stream.stop()?;

// Or run blocking
stream.run()?;
```

The SDK automatically connects to port 10000 with your token.

**Available gRPC Streams:**

| Method | Parameters | Description |
|--------|-----------|-------------|
| `trades(coins, callback)` | coins: `&[&str]` | Executed trades with price, size, direction |
| `orders(coins, callback, users)` | coins: `&[&str]`, users: `Option<&[&str]>` | Order lifecycle events |
| `book_updates(coins, callback)` | coins: `&[&str]` | Order book changes (deltas) |
| `l2_book(coin, callback, n_sig_figs)` | coin: `&str`, n_sig_figs: `Option<u8>` (3-5) | L2 order book (aggregated by price) |
| `l4_book(coin, callback)` | coin: `&str` | **L4 order book (individual orders)** |
| `blocks(callback)` | - | Block data |
| `twap(coins, callback)` | coins: `&[&str]` | TWAP execution updates |
| `events(callback)` | - | System events (funding, liquidations) |
| `writer_actions(callback)` | - | Writer actions |

### L4 Order Book (Critical for Trading)

L4 order book shows **every individual order** with its order ID. This is essential for:

- **Market Making**: Know your exact queue position
- **Order Flow Analysis**: Detect large orders and icebergs
- **Optimal Execution**: See exactly what you're crossing
- **HFT**: Lower latency than WebSocket

```rust
use hyperliquid_sdk::GRPCStream;

fn on_l4_book(data: serde_json::Value) {
    // L4 book data structure:
    // {
    //     "coin": "BTC",
    //     "bids": [[price, size, order_id], ...],
    //     "asks": [[price, size, order_id], ...]
    // }
    if let Some(bids) = data.get("bids").and_then(|b| b.as_array()) {
        for bid in bids.iter().take(3) {
            let px = bid.get(0).and_then(|v| v.as_f64()).unwrap_or(0.0);
            let sz = bid.get(1).and_then(|v| v.as_f64()).unwrap_or(0.0);
            let oid = bid.get(2).and_then(|v| v.as_i64()).unwrap_or(0);
            println!("Bid: ${:.2} x {} (order: {})", px, sz, oid);
        }
    }
}

let mut stream = GRPCStream::new(endpoint)?;
stream.l4_book("BTC", on_l4_book);
stream.run()?;
```

### L2 vs L4 Comparison

| Feature | L2 Book | L4 Book |
|---------|---------|---------|
| Aggregation | By price level | Individual orders |
| Order IDs | No | Yes |
| Queue Position | Unknown | Visible |
| Bandwidth | Lower | Higher |
| Protocol | WebSocket or gRPC | gRPC only |
| Use Case | Price monitoring | Market making, HFT |

### EVM WebSocket Streaming (eth_subscribe)

```rust
use hyperliquid_sdk::EVMStream;

let mut evm_stream = sdk.evm_stream();

// Subscribe to new block headers
evm_stream.new_heads(|header| {
    println!("New block: {:?}", header);
});

// Subscribe to contract logs
evm_stream.logs(
    Some(json!({"address": "0x..."})),
    |log| println!("Log: {:?}", log),
);

// Subscribe to pending transactions
evm_stream.new_pending_transactions(|tx_hash| {
    println!("Pending tx: {:?}", tx_hash);
});

evm_stream.start()?;
```

---

## Trading Features

### One-Line Orders

```rust
// Market orders
sdk.market_buy("BTC").await.size(0.001).await?;
sdk.market_sell("ETH").await.notional(100.0).await?;

// Limit orders
sdk.buy("BTC", 0.001, 65000.0, TIF::Gtc).await?;
sdk.sell("ETH", 0.5, 4000.0, TIF::Gtc).await?;

// Perp trader aliases
sdk.long("BTC", 0.001, 65000.0, TIF::Gtc).await?;
sdk.short("ETH", 0.5, 3000.0, TIF::Ioc).await?;
```

### Fluent Order Builder

```rust
use hyperliquid_sdk::Order;

let order = sdk.order(
    Order::buy("BTC")
         .size(0.001)
         .price(65000.0)
         .gtc()
         .reduce_only()
         .random_cloid()
).await?;
```

### Order Management

```rust
// Place, modify, cancel
let order = sdk.buy("BTC", 0.001, 60000.0, TIF::Gtc).await?;
order.modify(Some(61000.0), None, None).await?;
order.cancel().await?;

// Cancel all
sdk.cancel_all(None).await?;
sdk.cancel_all(Some("BTC")).await?;  // Just BTC orders

// Dead-man's switch
sdk.schedule_cancel(Some(timestamp_ms)).await?;
```

### Position Management

```rust
sdk.close_position("BTC").await?;  // Close entire position
```

### Leverage & Margin

```rust
// Update leverage
sdk.update_leverage("BTC", 10, true).await?;   // 10x cross
sdk.update_leverage("ETH", 5, false).await?;   // 5x isolated

// Isolated margin management
sdk.update_isolated_margin("BTC", true, 100.0).await?;   // Add margin to long
sdk.update_isolated_margin("ETH", false, -50.0).await?;  // Remove from short
sdk.top_up_isolated_only_margin("BTC", 100.0).await?;    // Special maintenance mode
```

### Trigger Orders (Stop Loss / Take Profit)

```rust
use hyperliquid_sdk::{TriggerOrder, Side};

// Stop loss (market order when triggered)
sdk.stop_loss("BTC", 0.001, 60000.0).await?;

// Stop loss (limit order when triggered)
let trigger = TriggerOrder::stop_loss("BTC")
    .size(0.001)
    .trigger_price(60000.0)
    .limit(59500.0);
sdk.trigger_order(trigger).await?;

// Take profit
sdk.take_profit("BTC", 0.001, 70000.0).await?;

// Buy-side (closing shorts)
let trigger = TriggerOrder::stop_loss("BTC")
    .size(0.001)
    .trigger_price(70000.0)
    .side(Side::Buy)
    .market();
sdk.trigger_order(trigger).await?;
```

### TWAP Orders

```rust
// Time-weighted average price order
let result = sdk.twap_order(
    "BTC",
    0.01,           // size
    true,           // is_buy
    60,             // duration_minutes
    true,           // randomize
    false,          // reduce_only
).await?;
let twap_id = result["response"]["data"]["running"]["id"].as_u64().unwrap();

// Cancel TWAP
sdk.twap_cancel("BTC", twap_id).await?;
```

### Transfers

```rust
// Internal transfers
sdk.transfer_spot_to_perp(100.0).await?;
sdk.transfer_perp_to_spot(100.0).await?;

// External transfers
sdk.transfer_usd("0x...", 100.0).await?;
sdk.transfer_spot("PURR", "0x...", 100.0).await?;
sdk.send_asset("USDC:0x...", 100.0, "0x...", None, None, None).await?;

// Withdraw to L1 (Arbitrum)
sdk.withdraw(100.0, Some("0x...")).await?;
```

### Vaults

```rust
let hlp_vault = "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303";
sdk.vault_deposit(hlp_vault, 100.0).await?;
sdk.vault_withdraw(hlp_vault, 50.0).await?;
```

### Staking

```rust
// Stake/unstake HYPE
sdk.stake(1000.0).await?;
sdk.unstake(500.0).await?;  // 7-day queue

// Delegate to validators
sdk.delegate("0x...", 500.0).await?;
sdk.undelegate("0x...", 250.0).await?;
```

### Builder Fee

```rust
// Check approval
let status = sdk.approval_status().await?;

// Approve builder fee
sdk.approve_builder_fee(Some("1%")).await?;

// Revoke
sdk.revoke_builder_fee().await?;
```

### Agent/API Key Management

```rust
// Approve an agent to trade on your behalf
sdk.approve_agent("0xAgent...", Some("my-bot")).await?;
```

### Account Abstraction

```rust
// Set abstraction mode
sdk.set_abstraction("unifiedAccount", None).await?;
sdk.set_abstraction("portfolioMargin", None).await?;
sdk.set_abstraction("disabled", None).await?;

// As an agent
sdk.agent_set_abstraction("unifiedAccount").await?;
```

### Advanced Transfers

```rust
// Send asset between DEXs
sdk.send_asset("USDC:0x...", 100.0, "0xDest...", None, None, None).await?;

// Send to EVM with data
sdk.send_to_evm_with_data(
    "PURR:0x...", 100.0, "0xContract...", "0xcalldata...",
    "", 999, 100000
).await?;
```

### Additional Operations

```rust
// Cancel by client order ID
sdk.cancel_by_cloid("0xmycloid...", "BTC").await?;

// Reserve rate limit capacity
sdk.reserve_request_weight(1000).await?;

// No-op (consume nonce)
sdk.noop().await?;

// Preflight validation
sdk.preflight("BTC", Side::Buy, 67000.0, 0.001).await?;

// Refresh markets cache
sdk.refresh_markets().await?;
```

---

## Error Handling

All errors are typed with specific error codes and guidance.

```rust
use hyperliquid_sdk::{Error, ErrorCode};

match sdk.market_buy("BTC").await.notional(100.0).await {
    Ok(order) => println!("Success: {:?}", order.oid),
    Err(Error::ApprovalRequired { .. }) => {
        // Need to approve builder fee first
        sdk.approve_builder_fee(None).await?;
    }
    Err(Error::InsufficientMargin { message, guidance, .. }) => {
        eprintln!("Not enough margin: {}", message);
        eprintln!("Guidance: {}", guidance);
    }
    Err(Error::GeoBlocked { message, .. }) => {
        eprintln!("Geo-blocked: {}", message);
    }
    Err(Error::ApiError { code, message, guidance, .. }) => {
        eprintln!("API Error [{}]: {}", code, message);
        eprintln!("Guidance: {}", guidance);
    }
    Err(e) => eprintln!("Error: {}", e),
}
```

**Available Error Types:**
- `Error::ApiError` — Base API error with code and message
- `Error::BuildError` — Order building failed
- `Error::SendError` — Transaction send failed
- `Error::ApprovalRequired` — Builder fee approval needed
- `Error::ValidationError` — Invalid parameters
- `Error::SignatureError` — Signature verification failed
- `Error::NoPosition` — No position to close
- `Error::OrderNotFound` — Order not found
- `Error::GeoBlocked` — Region blocked
- `Error::InsufficientMargin` — Not enough margin
- `Error::LeverageError` — Invalid leverage
- `Error::RateLimited` — Rate limited
- `Error::MaxOrders` — Too many orders
- `Error::ReduceOnlyError` — Reduce-only constraint
- `Error::DuplicateOrder` — Duplicate order
- `Error::UserNotFound` — User not found
- `Error::MustDeposit` — Deposit required
- `Error::InvalidNonce` — Invalid nonce

---

## API Reference

### HyperliquidSDK (Trading)

```rust
HyperliquidSDK::new()
    .endpoint(endpoint)          // Endpoint URL
    .private_key(key)            // Falls back to PRIVATE_KEY env var
    .auto_approve(true)          // Auto-approve builder fee (default: true)
    .max_fee("1%")               // Max fee for auto-approval
    .slippage(0.03)              // Default slippage for market orders (3%)
    .timeout(30)                 // Request timeout in seconds
    .build()
    .await?;
```

### Info (Account & Metadata)

```rust
Info::new(endpoint);
```

### HyperCore (Blocks & Trades)

```rust
HyperCore::new(endpoint);
```

### EVM (Ethereum JSON-RPC)

```rust
EVM::new(endpoint);
EVM::new(endpoint).with_debug(true);  // Enable debug/trace methods
```

### Stream (WebSocket)

```rust
Stream::new(Some(endpoint))
    .on_error(callback)          // Error callback
    .on_close(callback)          // Close callback
    .on_open(callback)           // Open callback
    .on_reconnect(callback);     // Reconnect callback
```

### GRPCStream (gRPC)

```rust
GRPCStream::new(endpoint)?;      // Token extracted from endpoint
```

### EVMStream (EVM WebSocket)

```rust
EVMStream::new(endpoint);
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PRIVATE_KEY` | Your Ethereum private key (hex, with or without 0x) |
| `ENDPOINT` | Endpoint URL |

---

## Performance

The SDK is designed for high performance:

- Zero-copy where possible
- Connection pooling via reqwest
- Lock-free data structures (DashMap)
- Efficient MessagePack serialization
- Lazy initialization of sub-clients

---

## Examples

See the [hyperliquid-examples](https://github.com/quiknode-labs/hyperliquid-examples) repository for 44 complete, runnable examples covering trading, streaming, market data, and more.

---

## Disclaimer

This is an **unofficial community SDK**. It is **not affiliated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing.

## License

MIT License - see [LICENSE](LICENSE) for details.
