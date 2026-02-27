# Hyperliquid SDK

> **Community SDK by [QuickNode](https://quicknode.com)** — Not affiliated with Hyperliquid Foundation.

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK()
order = sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

## Installation

```bash
pip install hyperliquid-sdk
```

## Quick Start

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

## Features

### One-Line Orders

```python
# Market orders - execute immediately
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
# Place a resting order
order = sdk.buy("BTC", size=0.001, price=60000, tif="gtc")

# Modify it
order.modify(price=61000)

# Cancel it
order.cancel()

# Cancel all orders
sdk.cancel_all()
sdk.cancel_all("BTC")  # Just BTC orders

# Cancel by client order ID
sdk.cancel_by_cloid("0x1234...", "BTC")

# Dead-man's switch: cancel all orders in 60 seconds
import time
sdk.schedule_cancel(int(time.time() * 1000) + 60000)

# Cancel the scheduled cancel
sdk.schedule_cancel(None)
```

### Order Validation (Preflight)

```python
# Validate order before signing (catches tick/lot size errors)
result = sdk.preflight("BTC", "buy", price=67000.123456, size=0.001)
if not result["valid"]:
    print(f"Invalid: {result['error']}")
    print(f"Suggestion: {result['suggestion']}")
```

### Position Management

```python
# Close a position completely (SDK figures out size and direction)
sdk.close_position("BTC")
```

### Fluent Order Builder

For power users who want maximum control:

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

### HIP-3 Markets

Trade HIP-3 markets (like Hypersea) with the same API:

```python
sdk.buy("xyz:SILVER", notional=100, tif="ioc")
```

### Market Data

```python
# Get mid price
mid = sdk.get_mid("BTC")

# List all markets
markets = sdk.markets()

# View open orders
orders = sdk.open_orders()
```

## Error Handling

Clear, actionable errors with guidance. Every error has:
- `message` - What went wrong
- `code` - Error code (e.g., "INSUFFICIENT_MARGIN")
- `guidance` - How to fix it

```python
from hyperliquid_sdk import (
    HyperliquidSDK,
    HyperliquidError,
    ApprovalError,
    ValidationError,
    GeoBlockedError,
    InsufficientMarginError,
)

try:
    order = sdk.buy("BTC", size=0.001, price=65000)
except ApprovalError as e:
    print(f"Need approval: {e.guidance}")
    sdk.approve_builder_fee("1%")
except InsufficientMarginError as e:
    print(f"Not enough margin: {e.guidance}")
except ValidationError as e:
    print(f"Invalid order: {e.message}")
except GeoBlockedError as e:
    print(f"Access denied: {e.guidance}")
except HyperliquidError as e:
    print(f"Error [{e.code}]: {e.message}")
    print(f"Hint: {e.guidance}")
```

### Error Classes

| Error | When |
|-------|------|
| `ApprovalError` | Builder fee not approved or fee exceeds approved amount |
| `ValidationError` | Invalid order params (price tick, size decimals) |
| `GeoBlockedError` | Access from restricted jurisdiction |
| `InsufficientMarginError` | Not enough margin for order |
| `LeverageError` | Leverage configuration conflict |
| `NoPositionError` | No position to close |
| `MaxOrdersError` | Too many open orders |
| `ReduceOnlyError` | Reduce-only order would increase position |
| `RateLimitError` | Rate limit exceeded |
| `UserNotFoundError` | Wallet not recognized (need to deposit first) |
| `MustDepositError` | Account needs initial deposit |

## Auto-Approval

Skip the approval step entirely:

```python
sdk = HyperliquidSDK(auto_approve=True)
# Now just trade - approval happens automatically
```

## API Reference

### HyperliquidSDK

```python
HyperliquidSDK(
    private_key=None,      # Falls back to PRIVATE_KEY env var
    auto_approve=False,    # Auto-approve builder fee
    max_fee="1%",          # Max fee for auto-approval
)
```

### Order Methods

| Method | Description |
|--------|-------------|
| `buy(asset, size=, price=, tif=)` | Place a buy order |
| `sell(asset, size=, price=, tif=)` | Place a sell order |
| `market_buy(asset, size=)` | Market buy |
| `market_sell(asset, size=)` | Market sell |
| `long(asset, ...)` | Alias for buy |
| `short(asset, ...)` | Alias for sell |

### Order Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `asset` | str | Asset name ("BTC", "ETH", "xyz:SILVER") |
| `size` | float | Size in asset units |
| `notional` | float | Size in USD (alternative to size) |
| `price` | float | Limit price |
| `tif` | str | Time in force: "ioc", "gtc", "alo", "market" |
| `reduce_only` | bool | Close position only |

### Management Methods

| Method | Description |
|--------|-------------|
| `cancel(oid)` | Cancel order by ID |
| `cancel_by_cloid(cloid, asset)` | Cancel order by client order ID |
| `cancel_all(asset=None)` | Cancel all orders |
| `schedule_cancel(time)` | Schedule cancel all (dead-man's switch) |
| `close_position(asset)` | Close a position |
| `modify(oid, price=, size=)` | Modify an order |

### Query Methods

| Method | Description |
|--------|-------------|
| `get_mid(asset)` | Get mid price |
| `markets()` | List all markets |
| `dexes()` | List all HIP-3 DEXes |
| `open_orders()` | Get open orders |
| `order_status(oid)` | Get order status |
| `approval_status()` | Check builder fee approval |
| `preflight(asset, side, price, size)` | Validate order before signing |

### PlacedOrder Object

```python
order = sdk.buy("BTC", size=0.001, price=65000, tif="gtc")

order.oid          # Order ID
order.status       # "filled", "resting", "error: ..."
order.asset        # "BTC"
order.side         # "buy" or "sell"
order.size         # "0.001"
order.price        # "65000"
order.filled_size  # Filled amount (if filled)
order.is_filled    # True if filled
order.is_resting   # True if resting

order.cancel()     # Cancel this order
order.modify(price=66000)  # Modify this order
```

## Examples

See the [examples repository](https://github.com/quiknode-labs/hyperliquid-examples) for complete working examples:

- `market_order.py` - Market order
- `place_order.py` - Limit order
- `cancel_order.py` - Cancel by OID
- `cancel_by_cloid.py` - Cancel by client order ID
- `cancel_all.py` - Cancel all orders
- `schedule_cancel.py` - Dead-man's switch
- `modify_order.py` - Modify order
- `close_position.py` - Close position
- `roundtrip.py` - Buy then sell
- `preflight.py` - Validate before signing
- `markets.py` - List markets and DEXes
- `fluent_builder.py` - Power user patterns
- `full_demo.py` - All features

## Links

- **Examples**: https://github.com/quiknode-labs/hyperliquid-examples
- **Documentation**: https://hyperliquidapi.com/docs
- **Approval Page**: https://hyperliquidapi.com/approve
- **GitHub**: https://github.com/quiknode-labs/hyperliquid-sdk

## Disclaimer

This is an **unofficial community SDK** developed and maintained by [QuickNode](https://quicknode.com). It is **not affiliated with, endorsed by, or associated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing. QuickNode is not responsible for any losses incurred through use of this SDK.

## License

MIT License - see [LICENSE](LICENSE) for details.
