# Hyperliquid SDK for TypeScript

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```typescript
import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const sdk = new HyperliquidSDK(endpoint);
const order = await sdk.marketBuy("BTC", { notional: 100 });  // Buy $100 of BTC
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

> **Community SDK** — Not affiliated with Hyperliquid Foundation.

## Installation

```bash
npm install @quicknode/hyperliquid-sdk
# or
yarn add @quicknode/hyperliquid-sdk
```

Everything is included: trading, market data, WebSocket streaming, gRPC streaming, HyperCore blocks, and EVM.

## Quick Start

### 1. Set your private key

```bash
export PRIVATE_KEY="0xYOUR_PRIVATE_KEY"
```

### 2. Start trading

```typescript
import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const sdk = new HyperliquidSDK(endpoint);

// Market orders
const order1 = await sdk.marketBuy("BTC", { size: 0.001 });
const order2 = await sdk.marketSell("ETH", { notional: 100 });  // $100 worth

// Limit orders
const order3 = await sdk.buy("BTC", { size: 0.001, price: 65000, tif: "gtc" });

// Check your order
console.log(order3.status);  // "filled" or "resting"
console.log(order3.oid);     // Order ID
```

---

## Data APIs

All data APIs are accessed through the SDK instance. You can create a Hyperliquid endpoint on [Quicknode](https://www.quicknode.com/) to get access to the data APIs.

### Info API

50+ methods for account state, positions, market data, and metadata.

```typescript
const sdk = new HyperliquidSDK(endpoint);

// Market data
await sdk.info.allMids();                          // All mid prices
await sdk.info.l2Book("BTC");                      // Order book
await sdk.info.recentTrades("BTC");                // Recent trades
await sdk.info.candles("BTC", "1h", start, end);   // OHLCV candles
await sdk.info.fundingHistory("BTC", start, end);  // Funding history
await sdk.info.predictedFundings();                // Predicted funding rates

// Metadata
await sdk.info.meta();                             // Exchange metadata
await sdk.info.spotMeta();                         // Spot metadata
await sdk.info.exchangeStatus();                   // Exchange status
await sdk.info.perpDexs();                         // Perpetual DEX info
await sdk.info.maxMarketOrderNtls();               // Max market order notionals

// User data
await sdk.info.clearinghouseState("0x...");        // Positions & margin
await sdk.info.spotClearinghouseState("0x...");    // Spot balances
await sdk.info.openOrders("0x...");                // Open orders
await sdk.info.frontendOpenOrders("0x...");        // Enhanced open orders
await sdk.info.orderStatus("0x...", oid);          // Specific order status
await sdk.info.historicalOrders("0x...");          // Order history
await sdk.info.userFills("0x...");                 // Trade history
await sdk.info.userFillsByTime("0x...", start);    // Fills by time range
await sdk.info.userFunding("0x...");               // Funding payments
await sdk.info.userFees("0x...");                  // Fee structure
await sdk.info.userRateLimit("0x...");             // Rate limit status
await sdk.info.userRole("0x...");                  // Account type
await sdk.info.portfolio("0x...");                 // Portfolio history
await sdk.info.subAccounts("0x...");               // Sub-accounts
await sdk.info.extraAgents("0x...");               // API keys/agents

// TWAP
await sdk.info.userTwapSliceFills("0x...");        // TWAP slice fills

// Batch queries
await sdk.info.batchClearinghouseStates(["0x...", "0x..."]);

// Vaults
await sdk.info.vaultSummaries();                   // All vault summaries
await sdk.info.vaultDetails("0x...");              // Specific vault
await sdk.info.userVaultEquities("0x...");         // User's vault equities
await sdk.info.leadingVaults("0x...");             // Vaults user leads

// Delegation/Staking
await sdk.info.delegations("0x...");               // Active delegations
await sdk.info.delegatorSummary("0x...");          // Delegation summary
await sdk.info.delegatorHistory("0x...");          // Delegation history
await sdk.info.delegatorRewards("0x...");          // Delegation rewards

// Tokens
await sdk.info.tokenDetails(tokenId);              // Token details
await sdk.info.spotDeployState("0x...");           // Spot deployment state

// Other
await sdk.info.referral("0x...");                  // Referral info
await sdk.info.maxBuilderFee("0x...", "0x...");    // Builder fee limits
await sdk.info.approvedBuilders("0x...");          // Approved builders
await sdk.info.liquidatable();                     // Liquidatable positions
```

### HyperCore API

Block data, trading operations, and real-time data via JSON-RPC.

```typescript
const sdk = new HyperliquidSDK(endpoint);

// Block data
await sdk.core.latestBlockNumber();                  // Latest block
await sdk.core.getBlock(12345);                      // Get specific block
await sdk.core.getBatchBlocks(100, 110);             // Get block range
await sdk.core.latestBlocks({ count: 10 });          // Latest blocks

// Recent data
await sdk.core.latestTrades({ count: 10 });          // Recent trades (all coins)
await sdk.core.latestTrades({ count: 10, coin: "BTC" }); // Recent BTC trades
await sdk.core.latestOrders({ count: 10 });          // Recent order events
await sdk.core.latestBookUpdates({ count: 10 });     // Recent book updates

// Discovery
await sdk.core.listDexes();                          // All DEXes
await sdk.core.listMarkets();                        // All markets
await sdk.core.listMarkets({ dex: "hyperliquidity" }); // Markets by DEX

// Order queries
await sdk.core.openOrders("0x...");                  // User's open orders
await sdk.core.orderStatus("0x...", oid);            // Specific order status
await sdk.core.preflight(...);                       // Validate order before signing
```

### EVM (Ethereum JSON-RPC)

50+ Ethereum JSON-RPC methods for Hyperliquid's EVM chain (chain ID 999 mainnet, 998 testnet).

```typescript
const sdk = new HyperliquidSDK(endpoint);

// Chain info
await sdk.evm.blockNumber();                       // Latest block
await sdk.evm.chainId();                           // 999 mainnet, 998 testnet
await sdk.evm.gasPrice();                          // Current gas price
await sdk.evm.maxPriorityFeePerGas();              // Priority fee
await sdk.evm.netVersion();                        // Network version
await sdk.evm.syncing();                           // Sync status

// Accounts
await sdk.evm.getBalance("0x...");                 // Account balance
await sdk.evm.getTransactionCount("0x...");        // Nonce
await sdk.evm.getCode("0x...");                    // Contract code
await sdk.evm.getStorageAt("0x...", position);     // Storage value

// Transactions
await sdk.evm.call({ to: "0x...", data: "0x..." });
await sdk.evm.estimateGas(tx);
await sdk.evm.sendRawTransaction(signedTx);
await sdk.evm.getTransactionByHash("0x...");
await sdk.evm.getTransactionReceipt("0x...");

// Blocks
await sdk.evm.getBlockByNumber(12345);
await sdk.evm.getBlockByHash("0x...");
await sdk.evm.getBlockReceipts(12345);
await sdk.evm.getBlockTransactionCountByNumber(12345);

// Logs
await sdk.evm.getLogs({ address: "0x...", topics: [...] });

// HyperEVM-specific
await sdk.evm.bigBlockGasPrice();                  // Big block gas price
await sdk.evm.usingBigBlocks();                    // Is using big blocks?
await sdk.evm.getSystemTxsByBlockNumber(12345);

// Debug/Trace
await sdk.evm.debugTraceTransaction("0x...", { tracer: "callTracer" });
await sdk.evm.debugTraceBlockByNumber(12345);
await sdk.evm.traceTransaction("0x...");
await sdk.evm.traceBlock(12345);
```

---

## Real-Time Streaming

### WebSocket Streaming

20+ subscription types for real-time data with automatic reconnection.

```typescript
const sdk = new HyperliquidSDK(endpoint);

// Subscribe to trades
sdk.stream.trades(["BTC", "ETH"], (t) => console.log(`Trade: ${JSON.stringify(t)}`));

// Subscribe to book updates
sdk.stream.bookUpdates(["BTC"], (b) => console.log(`Book: ${JSON.stringify(b)}`));

// Subscribe to orders (your orders)
sdk.stream.orders(["BTC"], (o) => console.log(`Order: ${JSON.stringify(o)}`), { users: ["0x..."] });

// Start streaming
sdk.stream.start();

// ... do other work ...

// Stop streaming
sdk.stream.stop();
```

Available streams:

**Market Data:**
- `trades(coins, callback)` — Executed trades
- `bookUpdates(coins, callback)` — Order book changes
- `l2Book(coin, callback)` — L2 order book snapshots
- `allMids(callback)` — All mid price updates
- `candle(coin, interval, callback)` — Candlestick data
- `bbo(coin, callback)` — Best bid/offer updates
- `activeAssetCtx(coin, callback)` — Asset context (pricing, volume)

**User Data:**
- `orders(coins, callback, options)` — Order lifecycle events
- `openOrders(user, callback)` — User's open orders
- `orderUpdates(user, callback)` — Order status changes
- `userEvents(user, callback)` — All user events
- `userFills(user, callback)` — Trade fills
- `userFundings(user, callback)` — Funding payments
- `userNonFundingLedger(user, callback)` — Ledger changes
- `clearinghouseState(user, callback)` — Position updates
- `activeAssetData(user, coin, callback)` — Trading parameters

**TWAP:**
- `twap(coins, callback)` — TWAP execution
- `twapStates(user, callback)` — TWAP algorithm states
- `userTwapSliceFills(user, callback)` — TWAP slice fills
- `userTwapHistory(user, callback)` — TWAP history

**System:**
- `events(callback)` — System events (funding, liquidations)
- `notification(user, callback)` — User notifications
- `webData3(user, callback)` — Aggregate user info
- `writerActions(user, callback)` — Writer actions

### gRPC Streaming (High Performance)

Lower latency streaming via gRPC for high-frequency applications.

```typescript
const sdk = new HyperliquidSDK(endpoint);

// Subscribe to trades
sdk.grpc.trades(["BTC", "ETH"], (t) => console.log(`Trade: ${JSON.stringify(t)}`));

// Subscribe to L2 order book (aggregated by price level)
sdk.grpc.l2Book("BTC", (b) => console.log(`Book: ${JSON.stringify(b)}`), { nSigFigs: 5 });

// Subscribe to L4 order book (CRITICAL: individual orders with order IDs)
sdk.grpc.l4Book("BTC", (b) => console.log(`L4: ${JSON.stringify(b)}`));

// Subscribe to blocks
sdk.grpc.blocks((b) => console.log(`Block: ${JSON.stringify(b)}`));

// Start streaming
await sdk.grpc.start();

// ... do other work ...

sdk.grpc.stop();
```

**Available gRPC Streams:**

| Method | Parameters | Description |
|--------|-----------|-------------|
| `trades(coins, callback)` | coins: `string[]` | Executed trades with price, size, direction |
| `orders(coins, callback, options)` | coins: `string[]`, users?: `string[]` | Order lifecycle events |
| `bookUpdates(coins, callback)` | coins: `string[]` | Order book changes (deltas) |
| `l2Book(coin, callback, options)` | coin: `string`, nSigFigs?: `number` | L2 order book (aggregated by price) |
| `l4Book(coin, callback)` | coin: `string` | **L4 order book (individual orders)** |
| `blocks(callback)` | - | Block data |
| `twap(coins, callback)` | coins: `string[]` | TWAP execution updates |
| `events(callback)` | - | System events (funding, liquidations) |
| `writerActions(callback)` | - | Writer actions |

### L4 Order Book (Critical for Trading)

L4 order book shows **every individual order** with its order ID. This is essential for:

- **Market Making**: Know your exact queue position
- **Order Flow Analysis**: Detect large orders and icebergs
- **Optimal Execution**: See exactly what you're crossing
- **HFT**: Lower latency than WebSocket

```typescript
const sdk = new HyperliquidSDK(endpoint);

sdk.grpc.l4Book("BTC", (data) => {
  // L4 book data structure:
  // {
  //   "coin": "BTC",
  //   "bids": [[price, size, order_id], ...],
  //   "asks": [[price, size, order_id], ...]
  // }
  const bids = data.bids || [];
  for (const bid of bids.slice(0, 3)) {
    const [px, sz, oid] = bid;
    console.log(`Bid: $${Number(px).toLocaleString()} x ${sz} (order: ${oid})`);
  }
});

await sdk.grpc.start();
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

```typescript
// Market orders
await sdk.marketBuy("BTC", { size: 0.001 });
await sdk.marketSell("ETH", { notional: 100 });

// Limit orders
await sdk.buy("BTC", { size: 0.001, price: 65000 });
await sdk.sell("ETH", { size: 0.5, price: 4000, tif: "gtc" });

// Perp trader aliases
await sdk.long("BTC", { size: 0.001, price: 65000 });
await sdk.short("ETH", { notional: 500, tif: "ioc" });
```

### Order Management

```typescript
// Place, modify, cancel
const order = await sdk.buy("BTC", { size: 0.001, price: 60000, tif: "gtc" });
await order.modify({ price: 61000 });
await order.cancel();

// Cancel all
await sdk.cancelAll();
await sdk.cancelAll("BTC");  // Just BTC orders

// Dead-man's switch
await sdk.scheduleCancel(Date.now() + 60000);
```

### Position Management

```typescript
await sdk.closePosition("BTC");  // Close entire position
```

### Querying Open Orders by Trading Pair

```typescript
// Get all open orders
const result = await sdk.openOrders() as any;
console.log(`Total open orders: ${result.orders.length}`);

// Order fields: coin, limitPx (price), sz (size), side, oid, timestamp,
//               orderType, tif, cloid, reduceOnly
for (const order of result.orders) {
  console.log(`${order.coin} ${order.side} ${order.sz}@${order.limitPx}`);
}

// Filter by trading pair
const btcOrders = result.orders.filter(o => o.coin === 'BTC');
for (const order of btcOrders) {
  console.log(`  ${order.side} ${order.sz} @ ${order.limitPx} | type=${order.orderType} tif=${order.tif} oid=${order.oid}`);
}

// For enhanced data (triggers, children), use frontendOpenOrders()
const enhanced = await sdk.info.frontendOpenOrders(sdk.address!);
```

### Partial Position Close by Percentage

`closePosition()` closes the entire position. To close a percentage, read the current size and place a reduce-only market order for the desired amount:

```typescript
async function closePercentage(sdk: HyperliquidSDK, coin: string, percent: number) {
  const state = await sdk.info.clearinghouseState(sdk.address!) as any;
  const position = state.assetPositions.find((p: any) => p.position.coin === coin);
  if (!position) throw new Error(`No open position for ${coin}`);

  // szi is signed: positive = long, negative = short
  const szi = parseFloat(position.position.szi);
  const closeSize = Math.abs(szi) * (percent / 100);

  if (szi > 0) {
    // Long position: sell to close
    await sdk.sell(coin, { size: closeSize, tif: 'market', reduceOnly: true });
  } else {
    // Short position: buy to close
    await sdk.buy(coin, { size: closeSize, tif: 'market', reduceOnly: true });
  }
}

// Close 50% of BTC position
await closePercentage(sdk, "BTC", 50);
```

### Batch Cancel with Partial Failure Handling

```typescript
import { HyperliquidError } from '@quicknode/hyperliquid-sdk';

// Get open orders
const result = await sdk.openOrders() as any;
const orders = result.orders;

// Cancel all orders for a specific asset
await sdk.cancelAll("BTC");

// Cancel specific orders with per-order error handling
const targetOrders = orders.filter(
  o => o.coin === 'BTC' && parseFloat(o.limitPx) < 50000
);
const failures: Array<{ oid: number; error: string }> = [];

for (const order of targetOrders) {
  try {
    await sdk.cancel(order.oid, order.coin);
  } catch (e) {
    if (e instanceof HyperliquidError) {
      failures.push({ oid: order.oid, error: e.message });
    }
  }
}

if (failures.length > 0) {
  console.log(`Failed to cancel ${failures.length} orders:`, failures);
}

// Cancel by client order ID (for CLOID-tracked orders)
await sdk.cancelByCloid("0xmycloid...", "BTC");
```

### Resilient Order Placement

Use client order IDs (CLOIDs) for idempotent orders and categorize errors for retry logic:

```typescript
import {
  Order, PlacedOrder, HyperliquidError, RateLimitError, InvalidNonceError,
  DuplicateOrderError, GeoBlockedError, InsufficientMarginError,
  ValidationError, SignatureError, MaxOrdersError,
} from '@quicknode/hyperliquid-sdk';

// Set a CLOID for idempotency — the exchange rejects duplicates
const cloid = `0x${crypto.randomUUID().replace(/-/g, '')}`;
const order = await sdk.order(Order.buy("BTC").size(0.001).price(65000).gtc().cloid(cloid));

// Error categories:
//   Transient (retry):   RateLimitError, InvalidNonceError
//   Permanent (fail):    GeoBlockedError, InsufficientMarginError, ValidationError,
//                        SignatureError, MaxOrdersError
//   Already done:        DuplicateOrderError (order already placed)

const TRANSIENT_ERRORS = [RateLimitError, InvalidNonceError];

async function placeWithRetry(
  sdk: HyperliquidSDK, orderBuilder: Order, maxRetries = 3
): Promise<PlacedOrder | null> {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      return await sdk.order(orderBuilder);
    } catch (e) {
      if (e instanceof DuplicateOrderError) {
        return null; // Order already went through
      }
      if (TRANSIENT_ERRORS.some(cls => e instanceof cls)) {
        if (attempt === maxRetries - 1) throw e;
        const wait = (2 ** attempt) + Math.random();
        await new Promise(r => setTimeout(r, wait * 1000));
        continue;
      }
      throw e;
    }
  }
  return null;
}

// Generate CLOID before the retry loop so the same ID is reused on retries
const retryCloid = `0x${crypto.randomUUID().replace(/-/g, '')}`;
const retryOrder = Order.buy("BTC").size(0.001).price(65000).gtc().cloid(retryCloid);
await placeWithRetry(sdk, retryOrder);
```

#### Timeout Configuration

```typescript
// Configure timeout on SDK constructor
const sdkWithTimeout = new HyperliquidSDK(endpoint, { privateKey: "0x...", timeout: 30000 });
```

### Leverage & Margin

```typescript
// Update leverage
await sdk.updateLeverage("BTC", { leverage: 10, isCross: true });   // 10x cross
await sdk.updateLeverage("ETH", { leverage: 5, isCross: false });   // 5x isolated

// Isolated margin management
await sdk.updateIsolatedMargin("BTC", { amount: 100, isBuy: true });   // Add margin to long
await sdk.updateIsolatedMargin("ETH", { amount: -50, isBuy: false });  // Remove from short
await sdk.topUpIsolatedOnlyMargin("BTC", 100);                         // Special maintenance mode
```

### Trigger Orders (Stop Loss / Take Profit)

```typescript
import { Side } from '@quicknode/hyperliquid-sdk';

// Stop loss (market order when triggered)
await sdk.stopLoss("BTC", { size: 0.001, triggerPrice: 60000 });

// Stop loss (limit order when triggered)
await sdk.stopLoss("BTC", { size: 0.001, triggerPrice: 60000, limitPrice: 59500 });

// Take profit
await sdk.takeProfit("BTC", { size: 0.001, triggerPrice: 70000 });

// Buy-side (closing shorts)
await sdk.stopLoss("BTC", { size: 0.001, triggerPrice: 70000, side: Side.BUY });
```

### TWAP Orders

```typescript
// Time-weighted average price order
const result = await sdk.twapOrder("BTC", {
  size: 0.01,
  isBuy: true,
  durationMinutes: 60,
  randomize: true
});
const twapId = result.response.data.running.id;

// Cancel TWAP
await sdk.twapCancel("BTC", twapId);
```

### Transfers

```typescript
// Internal transfers
await sdk.transferSpotToPerp(100);
await sdk.transferPerpToSpot(100);

// External transfers
await sdk.transferUsd("0x...", 100);
await sdk.transferSpot("0x...", "PURR", 100);
await sdk.sendAsset("0x...", "USDC", 100);

// Withdraw to L1 (Arbitrum)
await sdk.withdraw("0x...", 100);
```

### Vaults

```typescript
const HLP_VAULT = "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303";
await sdk.vaultDeposit(HLP_VAULT, 100);
await sdk.vaultWithdraw(HLP_VAULT, 50);
```

### Staking

```typescript
// Stake/unstake HYPE
await sdk.stake(1000);
await sdk.unstake(500);  // 7-day queue

// Delegate to validators
await sdk.delegate("0x...", 500);
await sdk.undelegate("0x...", 250);
```

### Fluent Order Builder

```typescript
import { Order } from '@quicknode/hyperliquid-sdk';

const order = await sdk.order(
  Order.buy("BTC")
    .size(0.001)
    .price(65000)
    .gtc()
    .reduceOnly()
);
```

---

## Error Handling

All errors inherit from `HyperliquidError` with a `code` and `message`.

```typescript
import {
  HyperliquidError,
  ApprovalError,
  InsufficientMarginError,
  GeoBlockedError,
} from '@quicknode/hyperliquid-sdk';

try {
  const order = await sdk.buy("BTC", { size: 0.001, price: 65000 });
} catch (e) {
  if (e instanceof ApprovalError) {
    console.log(`Need approval: ${e.guidance}`);
  } else if (e instanceof InsufficientMarginError) {
    console.log(`Not enough margin: ${e.guidance}`);
  } else if (e instanceof GeoBlockedError) {
    console.log(`Geo-blocked: ${e.message}`);
  } else if (e instanceof HyperliquidError) {
    console.log(`Error [${e.code}]: ${e.message}`);
  }
}
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

```typescript
new HyperliquidSDK(
  endpoint?: string,           // Endpoint URL
  options?: {
    privateKey?: string,       // Falls back to PRIVATE_KEY env var
    autoApprove?: boolean,     // Auto-approve builder fee (default: true)
    maxFee?: string,           // Max fee for auto-approval (default: "1%")
    slippage?: number,         // Default slippage for market orders (default: 0.03)
    timeout?: number,          // Request timeout in ms (default: 30000)
    testnet?: boolean,         // Use testnet (default: false)
  }
)

// Access sub-clients
sdk.info      // Info API (market data, user data, metadata)
sdk.core      // HyperCore (blocks, trades, orders)
sdk.evm       // EVM (Ethereum JSON-RPC)
sdk.stream    // WebSocket streaming
sdk.grpc      // gRPC streaming
```

---

## Examples

See the [hyperliquid-examples](https://github.com/quiknode-labs/hyperliquid-examples) repository for 44 complete, runnable examples covering trading, streaming, market data, and more.

---

## Disclaimer

This is an **unofficial community SDK**. It is **not affiliated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing.

## License

MIT License - see [LICENSE](LICENSE) for details.
