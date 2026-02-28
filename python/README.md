# Hyperliquid SDK

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK(endpoint)
order = sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

> **Community SDK** — Not affiliated with Hyperliquid Foundation.

## Installation

```bash
pip install hyperliquid-sdk
```

Everything is included: trading, market data, WebSocket streaming, gRPC streaming, HyperCore blocks, and EVM.

## Quick Start

### 1. Set your private key

```bash
export PRIVATE_KEY="0xYOUR_PRIVATE_KEY"
```

### 2. Start trading

```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK(endpoint)

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

All data APIs are accessed through the SDK instance.

### Info API

50+ methods for account state, positions, market data, and metadata.

```python
sdk = HyperliquidSDK(endpoint)

# Market data
sdk.info().all_mids()                          # All mid prices
sdk.info().l2_book("BTC")                      # Order book
sdk.info().recent_trades("BTC")                # Recent trades
sdk.info().candles("BTC", "1h", start, end)    # OHLCV candles
sdk.info().funding_history("BTC", start, end)  # Funding history
sdk.info().predicted_fundings()                # Predicted funding rates

# Metadata
sdk.info().meta()                              # Exchange metadata
sdk.info().spot_meta()                         # Spot metadata
sdk.info().exchange_status()                   # Exchange status
sdk.info().perp_dexs()                         # Perpetual DEX info
sdk.info().max_market_order_ntls()             # Max market order notionals

# User data
sdk.info().clearinghouse_state("0x...")        # Positions & margin
sdk.info().spot_clearinghouse_state("0x...")   # Spot balances
sdk.info().open_orders("0x...")                # Open orders
sdk.info().frontend_open_orders("0x...")       # Enhanced open orders
sdk.info().order_status("0x...", oid)          # Specific order status
sdk.info().historical_orders("0x...")          # Order history
sdk.info().user_fills("0x...")                 # Trade history
sdk.info().user_fills_by_time("0x...", start)  # Fills by time range
sdk.info().user_funding("0x...")               # Funding payments
sdk.info().user_fees("0x...")                  # Fee structure
sdk.info().user_rate_limit("0x...")            # Rate limit status
sdk.info().user_role("0x...")                  # Account type
sdk.info().portfolio("0x...")                  # Portfolio history
sdk.info().sub_accounts("0x...")               # Sub-accounts
sdk.info().extra_agents("0x...")               # API keys/agents

# TWAP
sdk.info().user_twap_slice_fills("0x...")      # TWAP slice fills

# Batch queries
sdk.info().batch_clearinghouse_states(["0x...", "0x..."])

# Vaults
sdk.info().vault_summaries()                   # All vault summaries
sdk.info().vault_details("0x...")              # Specific vault
sdk.info().user_vault_equities("0x...")        # User's vault equities
sdk.info().leading_vaults("0x...")             # Vaults user leads

# Delegation/Staking
sdk.info().delegations("0x...")                # Active delegations
sdk.info().delegator_summary("0x...")          # Delegation summary
sdk.info().delegator_history("0x...")          # Delegation history
sdk.info().delegator_rewards("0x...")          # Delegation rewards

# Tokens
sdk.info().token_details("token_id")           # Token details
sdk.info().spot_deploy_state("0x...")          # Spot deployment state

# Other
sdk.info().referral("0x...")                   # Referral info
sdk.info().max_builder_fee("0x...", "0x...")   # Builder fee limits
sdk.info().approved_builders("0x...")          # Approved builders
sdk.info().liquidatable()                      # Liquidatable positions
```

### HyperCore API

Block data, trading operations, and real-time data via JSON-RPC.

```python
sdk = HyperliquidSDK(endpoint)

# Block data
sdk.core().latest_block_number()                 # Latest block
sdk.core().get_block(12345)                      # Get specific block
sdk.core().get_batch_blocks(100, 110)            # Get block range
sdk.core().latest_blocks(count=10)               # Latest blocks

# Recent data
sdk.core().latest_trades(count=10)               # Recent trades (all coins)
sdk.core().latest_trades(count=10, coin="BTC")   # Recent BTC trades
sdk.core().latest_orders(count=10)               # Recent order events
sdk.core().latest_book_updates(count=10)         # Recent book updates

# Discovery
sdk.core().list_dexes()                          # All DEXes
sdk.core().list_markets()                        # All markets
sdk.core().list_markets(dex="hyperliquidity")    # Markets by DEX

# Order queries
sdk.core().open_orders("0x...")                  # User's open orders
sdk.core().order_status("0x...", oid)            # Specific order status
sdk.core().preflight(...)                        # Validate order before signing
```

### EVM (Ethereum JSON-RPC)

50+ Ethereum JSON-RPC methods for Hyperliquid's EVM chain (chain ID 999 mainnet, 998 testnet).

```python
sdk = HyperliquidSDK(endpoint)

# Chain info
sdk.evm().block_number()                       # Latest block
sdk.evm().chain_id()                           # 999 mainnet, 998 testnet
sdk.evm().gas_price()                          # Current gas price
sdk.evm().max_priority_fee_per_gas()           # Priority fee
sdk.evm().net_version()                        # Network version
sdk.evm().syncing()                            # Sync status

# Accounts
sdk.evm().get_balance("0x...")                 # Account balance
sdk.evm().get_transaction_count("0x...")       # Nonce
sdk.evm().get_code("0x...")                    # Contract code
sdk.evm().get_storage_at("0x...", position)    # Storage value

# Transactions
sdk.evm().call({"to": "0x...", "data": "0x..."})
sdk.evm().estimate_gas(tx)
sdk.evm().send_raw_transaction(signed_tx)
sdk.evm().get_transaction_by_hash("0x...")
sdk.evm().get_transaction_receipt("0x...")

# Blocks
sdk.evm().get_block_by_number(12345)
sdk.evm().get_block_by_hash("0x...")
sdk.evm().get_block_receipts(12345)
sdk.evm().get_block_transaction_count_by_number(12345)

# Logs
sdk.evm().get_logs({"address": "0x...", "topics": [...]})

# HyperEVM-specific
sdk.evm().big_block_gas_price()                # Big block gas price
sdk.evm().using_big_blocks()                   # Is using big blocks?
sdk.evm().get_system_txs_by_block_number(12345)

# Debug/Trace
sdk.evm().debug_trace_transaction("0x...", {"tracer": "callTracer"})
sdk.evm().debug_trace_block_by_number(12345)
sdk.evm().trace_transaction("0x...")
sdk.evm().trace_block(12345)
```

---

## Real-Time Streaming

### WebSocket Streaming

20+ subscription types for real-time data with automatic reconnection.

```python
sdk = HyperliquidSDK(endpoint)

# Subscribe to trades
sdk.stream().trades(["BTC", "ETH"], lambda t: print(f"Trade: {t}"))

# Subscribe to book updates
sdk.stream().book_updates(["BTC"], lambda b: print(f"Book: {b}"))

# Subscribe to orders (your orders)
sdk.stream().orders(["BTC"], lambda o: print(f"Order: {o}"), users=["0x..."])

# Run in background
sdk.stream().start()
# ... do other work ...
sdk.stream().stop()

# Or run blocking
sdk.stream().run()
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

Lower latency streaming via gRPC for high-frequency applications.

```python
sdk = HyperliquidSDK(endpoint)

# Subscribe to trades
sdk.grpc().trades(["BTC", "ETH"], lambda t: print(f"Trade: {t}"))

# Subscribe to L2 order book (aggregated by price level)
sdk.grpc().l2_book("BTC", lambda b: print(f"Book: {b}"), n_sig_figs=5)

# Subscribe to L4 order book (CRITICAL: individual orders with order IDs)
sdk.grpc().l4_book("BTC", lambda b: print(f"L4: {b}"))

# Subscribe to blocks
sdk.grpc().blocks(lambda b: print(f"Block: {b}"))

# Run in background
sdk.grpc().start()
# ... do other work ...
sdk.grpc().stop()

# Or run blocking
sdk.grpc().run()
```

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

sdk = HyperliquidSDK(endpoint)
sdk.grpc().l4_book("BTC", on_l4_book)
sdk.grpc().run()
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

### Leverage & Margin

```python
# Update leverage
sdk.update_leverage("BTC", leverage=10, is_cross=True)   # 10x cross
sdk.update_leverage("ETH", leverage=5, is_cross=False)   # 5x isolated

# Isolated margin management
sdk.update_isolated_margin("BTC", amount=100, is_buy=True)   # Add margin to long
sdk.update_isolated_margin("ETH", amount=-50, is_buy=False)  # Remove from short
sdk.top_up_isolated_only_margin("BTC", amount=100)           # Special maintenance mode
```

### Trigger Orders (Stop Loss / Take Profit)

```python
from hyperliquid_sdk import Side

# Stop loss (market order when triggered)
sdk.stop_loss("BTC", size=0.001, trigger_price=60000)

# Stop loss (limit order when triggered)
sdk.stop_loss("BTC", size=0.001, trigger_price=60000, limit_price=59500)

# Take profit
sdk.take_profit("BTC", size=0.001, trigger_price=70000)

# Buy-side (closing shorts)
sdk.stop_loss("BTC", size=0.001, trigger_price=70000, side=Side.BUY)
```

### TWAP Orders

```python
# Time-weighted average price order
result = sdk.twap_order(
    "BTC",
    size=0.01,
    is_buy=True,
    duration_minutes=60,
    randomize=True
)
twap_id = result["response"]["data"]["running"]["id"]

# Cancel TWAP
sdk.twap_cancel("BTC", twap_id)
```

### Transfers

```python
# Internal transfers
sdk.transfer_spot_to_perp(amount=100)
sdk.transfer_perp_to_spot(amount=100)

# External transfers
sdk.transfer_usd(destination="0x...", amount=100)
sdk.transfer_spot(destination="0x...", token="PURR", amount=100)
sdk.send_asset(destination="0x...", token="USDC", amount=100)

# Withdraw to L1 (Arbitrum)
sdk.withdraw(destination="0x...", amount=100)
```

### Vaults

```python
HLP_VAULT = "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303"
sdk.vault_deposit(vault_address=HLP_VAULT, amount=100)
sdk.vault_withdraw(vault_address=HLP_VAULT, amount=50)
```

### Staking

```python
# Stake/unstake HYPE
sdk.stake(amount=1000)
sdk.unstake(amount=500)  # 7-day queue

# Delegate to validators
sdk.delegate(validator="0x...", amount=500)
sdk.undelegate(validator="0x...", amount=250)
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

### HyperliquidSDK

```python
HyperliquidSDK(
    endpoint=None,         # Endpoint URL
    private_key=None,      # Falls back to PRIVATE_KEY env var
    auto_approve=True,     # Auto-approve builder fee (default: True)
    max_fee="1%",          # Max fee for auto-approval
    slippage=0.03,         # Default slippage for market orders (3%)
    timeout=30,            # Request timeout in seconds
)

# Access sub-clients
sdk.info()      # Info API (market data, user data, metadata)
sdk.core()      # HyperCore (blocks, trades, orders)
sdk.evm()       # EVM (Ethereum JSON-RPC)
sdk.stream()    # WebSocket streaming
sdk.grpc()      # gRPC streaming
sdk.evm_stream() # EVM WebSocket (eth_subscribe)
```

---

## Examples

See the `examples/` directory for complete, runnable examples:

**Trading:**
- [market_order.py](examples/market_order.py) — Place market orders
- [place_order.py](examples/place_order.py) — Place limit orders
- [modify_order.py](examples/modify_order.py) — Modify existing orders
- [cancel_order.py](examples/cancel_order.py) — Cancel orders
- [cancel_by_cloid.py](examples/cancel_by_cloid.py) — Cancel by client order ID
- [cancel_all.py](examples/cancel_all.py) — Cancel all orders
- [close_position.py](examples/close_position.py) — Close positions
- [fluent_builder.py](examples/fluent_builder.py) — Fluent order builder
- [roundtrip.py](examples/roundtrip.py) — Buy and sell round trip
- [hip3_order.py](examples/hip3_order.py) — HIP-3 DEX orders
- [schedule_cancel.py](examples/schedule_cancel.py) — Dead-man's switch / scheduled cancel

**Trigger Orders:**
- [trigger_orders.py](examples/trigger_orders.py) — Stop loss and take profit orders

**TWAP:**
- [twap.py](examples/twap.py) — Time-weighted average price orders

**Leverage & Margin:**
- [leverage.py](examples/leverage.py) — Update leverage
- [isolated_margin.py](examples/isolated_margin.py) — Isolated margin management

**Transfers & Withdrawals:**
- [transfers.py](examples/transfers.py) — USD and spot transfers
- [withdraw.py](examples/withdraw.py) — Withdraw to L1 (Arbitrum)

**Vaults:**
- [vaults.py](examples/vaults.py) — Vault deposits and withdrawals

**Staking:**
- [staking.py](examples/staking.py) — Stake, unstake, and delegate

**Approval:**
- [approve.py](examples/approve.py) — Builder fee approval
- [builder_fee.py](examples/builder_fee.py) — Check approval status

**Market Info:**
- [markets.py](examples/markets.py) — List markets and mid prices
- [open_orders.py](examples/open_orders.py) — Query open orders
- [preflight.py](examples/preflight.py) — Validate orders before sending

**Data APIs:**
- [info_market_data.py](examples/info_market_data.py) — Market data and order book
- [info_user_data.py](examples/info_user_data.py) — User positions and orders
- [info_candles.py](examples/info_candles.py) — Candlestick data
- [info_vaults.py](examples/info_vaults.py) — Vault information
- [info_batch_queries.py](examples/info_batch_queries.py) — Batch queries
- [hypercore_blocks.py](examples/hypercore_blocks.py) — Block and trade data
- [evm_basics.py](examples/evm_basics.py) — EVM chain interaction

**Streaming:**
- [stream_trades.py](examples/stream_trades.py) — WebSocket streaming basics
- [stream_grpc.py](examples/stream_grpc.py) — gRPC streaming basics
- [stream_l4_book.py](examples/stream_l4_book.py) — **L4 order book (individual orders) — CRITICAL**
- [stream_l2_book.py](examples/stream_l2_book.py) — L2 order book
- [stream_orderbook.py](examples/stream_orderbook.py) — L2 vs L4 comparison
- [stream_websocket_all.py](examples/stream_websocket_all.py) — Complete WebSocket reference (20+ types)

**Complete Demo:**
- [full_demo.py](examples/full_demo.py) — All features in one file

---

## Disclaimer

This is an **unofficial community SDK**. It is **not affiliated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing.

## License

MIT License - see [LICENSE](LICENSE) for details.
