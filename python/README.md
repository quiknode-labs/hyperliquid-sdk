# Hyperliquid SDK

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK()
order = sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

> **Community SDK** — Not affiliated with Hyperliquid Foundation.

## Installation

```bash
pip install hyperliquid-sdk
```

That's it. Everything is included: trading, Info API, WebSocket streaming, gRPC streaming, HyperCore, and EVM.

## Quick Start

### Endpoint Flexibility

The SDK automatically handles any endpoint format you provide:

```python
# All of these work - the SDK extracts the token and routes correctly
endpoint = "https://x.quiknode.pro/TOKEN"
endpoint = "https://x.quiknode.pro/TOKEN/"
endpoint = "https://x.quiknode.pro/TOKEN/info"
endpoint = "https://x.quiknode.pro/TOKEN/hypercore"
endpoint = "https://x.quiknode.pro/TOKEN/evm"
```

Just pass your endpoint - the SDK handles the rest.

### 1. Set your private key

```bash
export PRIVATE_KEY="0xYOUR_PRIVATE_KEY"
```

### 2. Start trading

```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK()

# Market orders
order = sdk.market_buy("BTC", size=0.001)
order = sdk.market_sell("ETH", notional=100)  # $100 worth

# Limit orders
order = sdk.buy("BTC", size=0.001, price=65000, tif="gtc")

# Check your order
print(order.status)  # "filled" or "resting"
print(order.oid)     # Order ID
```

---

## Data APIs

Query Hyperliquid data with clean, simple interfaces.

### Info API

50+ methods for account state, positions, market data, and metadata.

```python
from hyperliquid_sdk import Info

info = Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

# Market data
info.all_mids()                          # All mid prices
info.l2_book("BTC")                      # Order book
info.recent_trades("BTC")                # Recent trades
info.candles("BTC", "1h", start, end)    # OHLCV candles
info.funding_history("BTC", start, end)  # Funding history
info.predicted_fundings()                # Predicted funding rates

# Metadata
info.meta()                              # Exchange metadata
info.spot_meta()                         # Spot metadata
info.exchange_status()                   # Exchange status
info.perp_dexs()                         # Perpetual DEX info
info.max_market_order_ntls()             # Max market order notionals

# User data
info.clearinghouse_state("0x...")        # Positions & margin
info.spot_clearinghouse_state("0x...")   # Spot balances
info.open_orders("0x...")                # Open orders
info.frontend_open_orders("0x...")       # Enhanced open orders
info.order_status("0x...", oid)          # Specific order status
info.historical_orders("0x...")          # Order history
info.user_fills("0x...")                 # Trade history
info.user_fills_by_time("0x...", start)  # Fills by time range
info.user_funding("0x...")               # Funding payments
info.user_fees("0x...")                  # Fee structure
info.user_rate_limit("0x...")            # Rate limit status
info.user_role("0x...")                  # Account type
info.portfolio("0x...")                  # Portfolio history
info.sub_accounts("0x...")               # Sub-accounts
info.extra_agents("0x...")               # API keys/agents

# TWAP
info.user_twap_slice_fills("0x...")      # TWAP slice fills

# Batch queries
info.batch_clearinghouse_states(["0x...", "0x..."])

# Vaults
info.vault_summaries()                   # All vault summaries
info.vault_details("0x...")              # Specific vault
info.user_vault_equities("0x...")        # User's vault equities
info.leading_vaults("0x...")             # Vaults user leads

# Delegation/Staking
info.delegations("0x...")                # Active delegations
info.delegator_summary("0x...")          # Delegation summary
info.delegator_history("0x...")          # Delegation history
info.delegator_rewards("0x...")          # Delegation rewards

# Tokens
info.token_details("token_id")           # Token details
info.spot_deploy_state("0x...")          # Spot deployment state

# Other
info.referral("0x...")                   # Referral info
info.max_builder_fee("0x...", "0x...")   # Builder fee limits
info.approved_builders("0x...")          # Approved builders
info.liquidatable()                      # Liquidatable positions
```

### HyperCore API

Block data, trading operations, and real-time data via JSON-RPC.

```python
from hyperliquid_sdk import HyperCore

hc = HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

# Block data
hc.latest_block_number()                 # Latest block
hc.get_block(12345)                      # Get specific block
hc.get_batch_blocks(100, 110)            # Get block range
hc.latest_blocks(count=10)               # Latest blocks

# Recent data
hc.latest_trades(count=10)               # Recent trades (all coins)
hc.latest_trades(count=10, coin="BTC")   # Recent BTC trades
hc.latest_orders(count=10)               # Recent order events
hc.latest_book_updates(count=10)         # Recent book updates

# Discovery
hc.list_dexes()                          # All DEXes
hc.list_markets()                        # All markets
hc.list_markets(dex="hyperliquidity")    # Markets by DEX

# Order queries
hc.open_orders("0x...")                  # User's open orders
hc.order_status("0x...", oid)            # Specific order status
hc.preflight(...)                        # Validate order before signing

# Order building (for manual signing)
hc.build_order(coin, is_buy, limit_px, sz, user)
hc.build_cancel(coin, oid, user)
hc.build_modify(coin, oid, user, limit_px=..., sz=...)
hc.build_approve_builder_fee(user, builder, rate, nonce)
hc.build_revoke_builder_fee(user, builder, nonce)

# Send signed actions
hc.send_order(action, signature, nonce)
hc.send_cancel(action, signature, nonce)
hc.send_modify(action, signature, nonce)
hc.send_approval(action, signature)
hc.send_revocation(action, signature)

# Builder fees
hc.get_max_builder_fee("0x...", "0x...")

# Subscriptions
hc.subscribe({"type": "trades", "coin": "BTC"})
hc.unsubscribe({"type": "trades", "coin": "BTC"})
```

### EVM (Ethereum JSON-RPC)

50+ Ethereum JSON-RPC methods for Hyperliquid's EVM chain (chain ID 999 mainnet, 998 testnet).

```python
from hyperliquid_sdk import EVM

evm = EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

# Chain info
evm.block_number()                       # Latest block
evm.chain_id()                           # 999 mainnet, 998 testnet
evm.gas_price()                          # Current gas price
evm.max_priority_fee_per_gas()           # Priority fee
evm.net_version()                        # Network version
evm.syncing()                            # Sync status

# Accounts
evm.get_balance("0x...")                 # Account balance
evm.get_transaction_count("0x...")       # Nonce
evm.get_code("0x...")                    # Contract code
evm.get_storage_at("0x...", position)    # Storage value

# Transactions
evm.call({"to": "0x...", "data": "0x..."})
evm.estimate_gas(tx)
evm.send_raw_transaction(signed_tx)
evm.get_transaction_by_hash("0x...")
evm.get_transaction_receipt("0x...")

# Blocks
evm.get_block_by_number(12345)
evm.get_block_by_hash("0x...")
evm.get_block_receipts(12345)
evm.get_block_transaction_count_by_number(12345)

# Logs
evm.get_logs({"address": "0x...", "topics": [...]})

# HyperEVM-specific
evm.big_block_gas_price()                # Big block gas price
evm.using_big_blocks()                   # Is using big blocks?
evm.get_system_txs_by_block_number(12345)

# Debug/Trace (use EVM(endpoint, debug=True))
evm = EVM(endpoint, debug=True)
evm.debug_trace_transaction("0x...", {"tracer": "callTracer"})
evm.debug_trace_block_by_number(12345)
evm.debug_storage_range_at(block_hash, tx_idx, addr, key, max)
evm.trace_transaction("0x...")
evm.trace_block(12345)
evm.trace_call(tx, ["trace", "vmTrace"])
evm.trace_filter({"fromBlock": "0x1", "toBlock": "0x10"})
evm.trace_replay_transaction("0x...", ["trace"])
```

---

## Real-Time Streaming

### WebSocket Streaming

20+ subscription types for real-time data with automatic reconnection.

```python
from hyperliquid_sdk import Stream

stream = Stream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

# Subscribe to trades
stream.trades(["BTC", "ETH"], lambda t: print(f"Trade: {t}"))

# Subscribe to book updates
stream.book_updates(["BTC"], lambda b: print(f"Book: {b}"))

# Subscribe to orders (your orders)
stream.orders(["BTC"], lambda o: print(f"Order: {o}"), users=["0x..."])

# Run in background
stream.start()  # or stream.run_in_background()
# ... do other work ...
stream.stop()

# Or run blocking
stream.run()
```

Available streams:

**Market Data:**
- `trades(coins, callback)` — Executed trades
- `book_updates(coins, callback)` — Order book changes
- `l2_book(coin, callback)` — L2 order book snapshots
- `all_mids(callback)` — All mid price updates
- `candle(coin, interval, callback)` — Candlestick data
- `bbo(coin, callback)` — Best bid/offer updates
- `active_asset_ctx(coin, callback)` — Asset context (pricing, volume)

**User Data:**
- `orders(coins, callback, users=None)` — Order lifecycle events
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

### gRPC Streaming (High Performance)

Lower latency streaming via gRPC for high-frequency applications. gRPC is included with all QuickNode Hyperliquid endpoints - no add-on needed.

```python
from hyperliquid_sdk import GRPCStream

stream = GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

# Subscribe to trades
stream.trades(["BTC", "ETH"], lambda t: print(f"Trade: {t}"))

# Subscribe to L2 order book (aggregated by price level)
stream.l2_book("BTC", lambda b: print(f"Book: {b}"), n_sig_figs=5)

# Subscribe to L4 order book (CRITICAL: individual orders with order IDs)
stream.l4_book("BTC", lambda b: print(f"L4: {b}"))

# Subscribe to blocks
stream.blocks(lambda b: print(f"Block: {b}"))

# Run in background
stream.start()
# ... do other work ...
stream.stop()

# Or run blocking
stream.run()
```

The SDK automatically connects to port 10000 with your token.

**Available gRPC Streams:**

| Method | Parameters | Description |
|--------|-----------|-------------|
| `trades(coins, callback)` | coins: `List[str]` | Executed trades with price, size, direction |
| `orders(coins, callback, users=None)` | coins: `List[str]`, users: `List[str]` (optional) | Order lifecycle events |
| `book_updates(coins, callback)` | coins: `List[str]` | Order book changes (deltas) |
| `l2_book(coin, callback, n_sig_figs=None)` | coin: `str`, n_sig_figs: `int` (3-5) | L2 order book (aggregated by price) |
| `l4_book(coin, callback)` | coin: `str` | **L4 order book (individual orders)** |
| `blocks(callback)` | - | Block data |
| `twap(coins, callback)` | coins: `List[str]` | TWAP execution updates |
| `events(callback)` | - | System events (funding, liquidations) |

### L4 Order Book (Critical for Trading)

L4 order book shows **every individual order** with its order ID. This is essential for:

- **Market Making**: Know your exact queue position
- **Order Flow Analysis**: Detect large orders and icebergs
- **Optimal Execution**: See exactly what you're crossing
- **HFT**: Lower latency than WebSocket

```python
from hyperliquid_sdk import GRPCStream

def on_l4_book(data):
    """
    L4 book data structure:
    {
        "coin": "BTC",
        "bids": [[price, size, order_id], ...],
        "asks": [[price, size, order_id], ...]
    }
    """
    for bid in data.get("bids", [])[:3]:
        px, sz, oid = bid[0], bid[1], bid[2]
        print(f"Bid: ${float(px):,.2f} x {sz} (order: {oid})")

stream = GRPCStream("https://your-endpoint.quiknode.pro/TOKEN")
stream.l4_book("BTC", on_l4_book)
stream.run()
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

---

## Trading Features

### One-Line Orders

```python
# Market orders
sdk.market_buy("BTC", size=0.001)
sdk.market_sell("ETH", notional=100)

# Limit orders
sdk.buy("BTC", size=0.001, price=65000)
sdk.sell("ETH", size=0.5, price=4000, tif="gtc")

# Perp trader aliases
sdk.long("BTC", size=0.001, price=65000)
sdk.short("ETH", notional=500, tif="ioc")
```

### Order Management

```python
# Place, modify, cancel
order = sdk.buy("BTC", size=0.001, price=60000, tif="gtc")
order.modify(price=61000)
order.cancel()

# Cancel all
sdk.cancel_all()
sdk.cancel_all("BTC")  # Just BTC orders

# Dead-man's switch
import time
sdk.schedule_cancel(int(time.time() * 1000) + 60000)
```

### Position Management

```python
sdk.close_position("BTC")  # Close entire position
```

### Fluent Order Builder

```python
from hyperliquid_sdk import Order

order = sdk.order(
    Order.buy("BTC")
         .size(0.001)
         .price(65000)
         .gtc()
         .reduce_only()
)
```

---

## Error Handling

All errors inherit from `HyperliquidError` with a `code` and `message`.

```python
from hyperliquid_sdk import (
    HyperliquidError,
    ApprovalError,
    InsufficientMarginError,
    GeoBlockedError,
)

try:
    order = sdk.buy("BTC", size=0.001, price=65000)
except ApprovalError as e:
    print(f"Need approval: {e.guidance}")
except InsufficientMarginError as e:
    print(f"Not enough margin: {e.guidance}")
except GeoBlockedError as e:
    print(f"Geo-blocked: {e.message}")
except HyperliquidError as e:
    print(f"Error [{e.code}]: {e.message}")
```

Available error types:
- `HyperliquidError` — Base error
- `BuildError` — Order building failed
- `SendError` — Transaction send failed
- `ApprovalError` — Builder fee approval needed
- `ValidationError` — Invalid parameters
- `SignatureError` — Signature verification failed
- `NoPositionError` — No position to close
- `OrderNotFoundError` — Order not found
- `GeoBlockedError` — Region blocked
- `InsufficientMarginError` — Not enough margin
- `LeverageError` — Invalid leverage
- `RateLimitError` — Rate limited
- `MaxOrdersError` — Too many orders
- `ReduceOnlyError` — Reduce-only constraint
- `DuplicateOrderError` — Duplicate order
- `UserNotFoundError` — User not found
- `MustDepositError` — Deposit required
- `InvalidNonceError` — Invalid nonce

---

## API Reference

### HyperliquidSDK (Trading)

```python
HyperliquidSDK(
    private_key=None,      # Falls back to PRIVATE_KEY env var
    testnet=False,         # Use testnet
    auto_approve=False,    # Auto-approve builder fee
    max_fee="1%",          # Max fee for auto-approval
)
```

### Info (Account & Metadata)

```python
Info(
    endpoint,              # Endpoint URL
    timeout=30,            # Request timeout
)
```

### HyperCore (Blocks & Trades)

```python
HyperCore(
    endpoint,              # Endpoint URL
    timeout=30,            # Request timeout
)
```

### EVM (Ethereum JSON-RPC)

```python
EVM(
    endpoint,              # Endpoint URL
    timeout=30,            # Request timeout
)
```

### Stream (WebSocket)

```python
Stream(
    endpoint,              # Endpoint URL
    on_error=None,         # Error callback
    on_close=None,         # Close callback
    on_open=None,          # Open callback
    reconnect=True,        # Auto-reconnect
)
```

### GRPCStream (gRPC)

```python
GRPCStream(
    endpoint,              # Endpoint URL (token extracted)
    on_error=None,         # Error callback
    on_close=None,         # Close callback
    on_connect=None,       # Connect callback
    secure=True,           # Use TLS
)
```

---

## Examples

See the [hyperliquid-examples](https://github.com/quiknode-labs/hyperliquid-examples) repository for complete, runnable examples:

**Trading:**
- [market_order.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/market_order.py) — Place market orders
- [place_order.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/place_order.py) — Place limit orders
- [cancel_order.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/cancel_order.py) — Cancel orders
- [close_position.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/close_position.py) — Close positions

**Data APIs:**
- [info_market_data.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/info_market_data.py) — Market data and order book
- [info_user_data.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/info_user_data.py) — User positions and orders
- [hypercore_blocks.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/hypercore_blocks.py) — Block and trade data
- [evm_basics.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/evm_basics.py) — EVM chain interaction

**Streaming:**
- [stream_trades.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/stream_trades.py) — WebSocket streaming basics
- [stream_grpc.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/stream_grpc.py) — gRPC streaming basics
- [stream_l4_book.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/stream_l4_book.py) — **L4 order book (individual orders) — CRITICAL**
- [stream_l2_book.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/stream_l2_book.py) — L2 order book (gRPC vs WebSocket)
- [stream_orderbook.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/stream_orderbook.py) — L2 vs L4 comparison
- [stream_websocket_all.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/stream_websocket_all.py) — Complete WebSocket reference (20+ types)

**Complete Demo:**
- [full_demo.py](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/python/full_demo.py) — All features in one file

---

## Disclaimer

This is an **unofficial community SDK**. It is **not affiliated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing.

## License

MIT License - see [LICENSE](LICENSE) for details.
