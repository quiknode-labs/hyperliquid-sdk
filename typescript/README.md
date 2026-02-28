# Hyperliquid SDK for TypeScript

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```typescript
import { HyperliquidSDK } from 'hyperliquid-sdk';

const sdk = new HyperliquidSDK();
const order = await sdk.marketBuy("BTC", { notional: 100 });  // Buy $100 of BTC
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

> **Community SDK** — Not affiliated with Hyperliquid Foundation.

## Installation

```bash
npm install hyperliquid-sdk
# or
yarn add hyperliquid-sdk
```

Everything is included: trading, Info API, WebSocket streaming, gRPC streaming, HyperCore, and EVM.

## Quick Start

### Endpoint Flexibility

The SDK automatically handles any endpoint format you provide:

```typescript
// All of these work - the SDK extracts the token and routes correctly
const endpoint = "https://x.quiknode.pro/TOKEN";
const endpoint = "https://x.quiknode.pro/TOKEN/";
const endpoint = "https://x.quiknode.pro/TOKEN/info";
const endpoint = "https://x.quiknode.pro/TOKEN/hypercore";
```

Just pass your endpoint - the SDK handles the rest.

### 1. Set your private key

```bash
export PRIVATE_KEY="0xYOUR_PRIVATE_KEY"
```

### 2. Start trading

```typescript
import { HyperliquidSDK } from 'hyperliquid-sdk';

const sdk = new HyperliquidSDK();

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

Query Hyperliquid data with clean, simple interfaces.

### Info API

50+ methods for account state, positions, market data, and metadata.

```typescript
import { Info } from 'hyperliquid-sdk';

const info = new Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");

// Market data
await info.allMids();                          // All mid prices
await info.l2Book("BTC");                      // Order book
await info.recentTrades("BTC");                // Recent trades
await info.candles("BTC", "1h", start, end);   // OHLCV candles
await info.fundingHistory("BTC", start, end);  // Funding history
await info.predictedFundings();                // Predicted funding rates

// Metadata
await info.meta();                             // Exchange metadata
await info.spotMeta();                         // Spot metadata
await info.exchangeStatus();                   // Exchange status
await info.perpDexs();                         // Perpetual DEX info
await info.maxMarketOrderNtls();               // Max market order notionals

// User data
await info.clearinghouseState("0x...");        // Positions & margin
await info.spotClearinghouseState("0x...");    // Spot balances
await info.openOrders("0x...");                // Open orders
await info.frontendOpenOrders("0x...");        // Enhanced open orders
await info.orderStatus("0x...", oid);          // Specific order status
await info.historicalOrders("0x...");          // Order history
await info.userFills("0x...");                 // Trade history
await info.userFillsByTime("0x...", start);    // Fills by time range
await info.userFunding("0x...");               // Funding payments
await info.userFees("0x...");                  // Fee structure
await info.userRateLimit("0x...");             // Rate limit status
await info.userRole("0x...");                  // Account type
await info.portfolio("0x...");                 // Portfolio history
await info.subAccounts("0x...");               // Sub-accounts
await info.extraAgents("0x...");               // API keys/agents

// TWAP
await info.userTwapSliceFills("0x...");        // TWAP slice fills

// Batch queries
await info.batchClearinghouseStates(["0x...", "0x..."]);

// Vaults
await info.vaultSummaries();                   // All vault summaries
await info.vaultDetails("0x...");              // Specific vault
await info.userVaultEquities("0x...");         // User's vault equities
await info.leadingVaults("0x...");             // Vaults user leads

// Delegation/Staking
await info.delegations("0x...");               // Active delegations
await info.delegatorSummary("0x...");          // Delegation summary
await info.delegatorHistory("0x...");          // Delegation history
await info.delegatorRewards("0x...");          // Delegation rewards

// Tokens
await info.tokenDetails(tokenId);              // Token details
await info.spotDeployState("0x...");           // Spot deployment state

// Other
await info.referral("0x...");                  // Referral info
await info.maxBuilderFee("0x...", "0x...");    // Builder fee limits
await info.approvedBuilders("0x...");          // Approved builders
await info.liquidatable();                     // Liquidatable positions
```

### HyperCore API

Block data, trading operations, and real-time data via JSON-RPC.

```typescript
import { HyperCore } from 'hyperliquid-sdk';

const hc = new HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");

// Block data
await hc.latestBlockNumber();                  // Latest block
await hc.getBlock(12345);                      // Get specific block
await hc.getBatchBlocks(100, 110);             // Get block range
await hc.latestBlocks({ count: 10 });          // Latest blocks

// Recent data
await hc.latestTrades({ count: 10 });          // Recent trades (all coins)
await hc.latestTrades({ count: 10, coin: "BTC" }); // Recent BTC trades
await hc.latestOrders({ count: 10 });          // Recent order events
await hc.latestBookUpdates({ count: 10 });     // Recent book updates

// Discovery
await hc.listDexes();                          // All DEXes
await hc.listMarkets();                        // All markets
await hc.listMarkets({ dex: "hyperliquidity" }); // Markets by DEX

// Order queries
await hc.openOrders("0x...");                  // User's open orders
await hc.orderStatus("0x...", oid);            // Specific order status
await hc.preflight(...);                       // Validate order before signing

// Order building (for manual signing)
await hc.buildOrder({ coin, isBuy, limitPx, sz, user });
await hc.buildCancel({ coin, oid, user });
await hc.buildModify({ coin, oid, user, limitPx, sz });
await hc.buildApproveBuilderFee({ user, builder, rate, nonce });
await hc.buildRevokeBuilderFee({ user, builder, nonce });

// Send signed actions
await hc.sendOrder({ action, signature, nonce });
await hc.sendCancel({ action, signature, nonce });
await hc.sendModify({ action, signature, nonce });
await hc.sendApproval({ action, signature });
await hc.sendRevocation({ action, signature });

// Builder fees
await hc.getMaxBuilderFee("0x...", "0x...");

// Subscriptions
await hc.subscribe({ type: "trades", coin: "BTC" });
await hc.unsubscribe({ type: "trades", coin: "BTC" });
```

### EVM (Ethereum JSON-RPC)

50+ Ethereum JSON-RPC methods for Hyperliquid's EVM chain (chain ID 999 mainnet, 998 testnet).

```typescript
import { EVM } from 'hyperliquid-sdk';

const evm = new EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");

// Chain info
await evm.blockNumber();                       // Latest block
await evm.chainId();                           // 999 mainnet, 998 testnet
await evm.gasPrice();                          // Current gas price
await evm.maxPriorityFeePerGas();              // Priority fee
await evm.netVersion();                        // Network version
await evm.syncing();                           // Sync status

// Accounts
await evm.getBalance("0x...");                 // Account balance
await evm.getTransactionCount("0x...");        // Nonce
await evm.getCode("0x...");                    // Contract code
await evm.getStorageAt("0x...", position);     // Storage value

// Transactions
await evm.call({ to: "0x...", data: "0x..." });
await evm.estimateGas(tx);
await evm.sendRawTransaction(signedTx);
await evm.getTransactionByHash("0x...");
await evm.getTransactionReceipt("0x...");

// Blocks
await evm.getBlockByNumber(12345);
await evm.getBlockByHash("0x...");
await evm.getBlockReceipts(12345);
await evm.getBlockTransactionCountByNumber(12345);

// Logs
await evm.getLogs({ address: "0x...", topics: [...] });

// HyperEVM-specific
await evm.bigBlockGasPrice();                  // Big block gas price
await evm.usingBigBlocks();                    // Is using big blocks?
await evm.getSystemTxsByBlockNumber(12345);

// Debug/Trace (use new EVM(endpoint, { debug: true }))
const debugEvm = new EVM(endpoint, { debug: true });
await debugEvm.debugTraceTransaction("0x...", { tracer: "callTracer" });
await debugEvm.debugTraceBlockByNumber(12345);
await debugEvm.traceTransaction("0x...");
await debugEvm.traceBlock(12345);
await debugEvm.traceCall(tx, ["trace", "vmTrace"]);
await debugEvm.traceFilter({ fromBlock: "0x1", toBlock: "0x10" });
await debugEvm.traceReplayTransaction("0x...", ["trace"]);
```

---

## Real-Time Streaming

### WebSocket Streaming

20+ subscription types for real-time data with automatic reconnection.

```typescript
import { Stream } from 'hyperliquid-sdk';

const stream = new Stream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");

// Subscribe to trades
stream.trades(["BTC", "ETH"], (t) => console.log(`Trade: ${JSON.stringify(t)}`));

// Subscribe to book updates
stream.bookUpdates(["BTC"], (b) => console.log(`Book: ${JSON.stringify(b)}`));

// Subscribe to orders (your orders)
stream.orders(["BTC"], (o) => console.log(`Order: ${JSON.stringify(o)}`), { users: ["0x..."] });

// Start streaming
stream.start();

// ... do other work ...

// Stop streaming
stream.stop();
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

Lower latency streaming via gRPC for high-frequency applications. gRPC is included with all QuickNode Hyperliquid endpoints - no add-on needed.

```typescript
import { GRPCStream } from 'hyperliquid-sdk';

const stream = new GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");

// Subscribe to trades
stream.trades(["BTC", "ETH"], (t) => console.log(`Trade: ${JSON.stringify(t)}`));

// Subscribe to L2 order book (aggregated by price level)
stream.l2Book("BTC", (b) => console.log(`Book: ${JSON.stringify(b)}`), { nSigFigs: 5 });

// Subscribe to L4 order book (CRITICAL: individual orders with order IDs)
stream.l4Book("BTC", (b) => console.log(`L4: ${JSON.stringify(b)}`));

// Subscribe to blocks
stream.blocks((b) => console.log(`Block: ${JSON.stringify(b)}`));

// Start streaming
await stream.start();

// ... do other work ...

stream.stop();
```

The SDK automatically connects to port 10000 with your token.

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
import { GRPCStream } from 'hyperliquid-sdk';

const stream = new GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");

stream.l4Book("BTC", (data) => {
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

await stream.start();
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
import { Side } from 'hyperliquid-sdk';

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
import { Order } from 'hyperliquid-sdk';

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
} from 'hyperliquid-sdk';

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

### HyperliquidSDK (Trading)

```typescript
new HyperliquidSDK(
  endpoint?: string,           // QuickNode endpoint URL
  options?: {
    privateKey?: string,       // Falls back to PRIVATE_KEY env var
    autoApprove?: boolean,     // Auto-approve builder fee (default: true)
    maxFee?: string,           // Max fee for auto-approval (default: "1%")
    slippage?: number,         // Default slippage for market orders (default: 0.03)
    timeout?: number,          // Request timeout in ms (default: 30000)
    testnet?: boolean,         // Use testnet (default: false)
  }
)
```

### Info (Account & Metadata)

```typescript
new Info(
  endpoint: string,            // Endpoint URL
  options?: {
    timeout?: number,          // Request timeout in ms
  }
)
```

### HyperCore (Blocks & Trades)

```typescript
new HyperCore(
  endpoint: string,            // Endpoint URL
  options?: {
    timeout?: number,          // Request timeout in ms
  }
)
```

### EVM (Ethereum JSON-RPC)

```typescript
new EVM(
  endpoint: string,            // Endpoint URL
  options?: {
    timeout?: number,          // Request timeout in ms
    debug?: boolean,           // Enable debug/trace APIs
  }
)
```

### Stream (WebSocket)

```typescript
new Stream(
  endpoint: string,            // Endpoint URL
  options?: {
    onError?: (error: Error) => void,
    onClose?: () => void,
    onOpen?: () => void,
    onStateChange?: (state: ConnectionState) => void,
    onReconnect?: (attempt: number) => void,
    reconnect?: boolean,       // Auto-reconnect (default: true)
    pingInterval?: number,     // Heartbeat interval in ms
  }
)
```

### GRPCStream (gRPC)

```typescript
new GRPCStream(
  endpoint: string,            // Endpoint URL (token extracted)
  options?: {
    onError?: (error: Error) => void,
    onClose?: () => void,
    onConnect?: () => void,
    onStateChange?: (state: ConnectionState) => void,
    onReconnect?: (attempt: number) => void,
    secure?: boolean,          // Use TLS (default: true)
    reconnect?: boolean,       // Auto-reconnect (default: true)
  }
)
```

---

## Examples

See the [hyperliquid-examples](https://github.com/quiknode-labs/hyperliquid-examples) repository for complete, runnable examples:

**Trading:**
- [market_order.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/market_order.ts) — Place market orders
- [place_order.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/place_order.ts) — Place limit orders
- [modify_order.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/modify_order.ts) — Modify existing orders
- [cancel_order.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/cancel_order.ts) — Cancel orders
- [cancel_by_cloid.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/cancel_by_cloid.ts) — Cancel by client order ID
- [cancel_all.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/cancel_all.ts) — Cancel all orders
- [close_position.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/close_position.ts) — Close positions
- [fluent_builder.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/fluent_builder.ts) — Fluent order builder
- [roundtrip.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/roundtrip.ts) — Buy and sell round trip
- [hip3_order.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/hip3_order.ts) — HIP-3 DEX orders

**Trigger Orders:**
- [trigger_orders.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/trigger_orders.ts) — Stop loss and take profit orders

**TWAP:**
- [twap.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/twap.ts) — Time-weighted average price orders

**Leverage & Margin:**
- [leverage.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/leverage.ts) — Update leverage
- [isolated_margin.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/isolated_margin.ts) — Isolated margin management

**Transfers & Withdrawals:**
- [transfers.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/transfers.ts) — USD and spot transfers
- [withdraw.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/withdraw.ts) — Withdraw to L1 (Arbitrum)

**Vaults:**
- [vaults.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/vaults.ts) — Vault deposits and withdrawals

**Staking:**
- [staking.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/staking.ts) — Stake, unstake, and delegate

**Approval:**
- [approve.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/approve.ts) — Builder fee approval
- [builder_fee.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/builder_fee.ts) — Check approval status

**Market Info:**
- [markets.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/markets.ts) — List markets and mid prices
- [open_orders.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/open_orders.ts) — Query open orders
- [preflight.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/preflight.ts) — Validate orders before sending

**Data APIs:**
- [info_market_data.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/info_market_data.ts) — Market data and order book
- [info_user_data.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/info_user_data.ts) — User positions and orders
- [info_candles.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/info_candles.ts) — Candlestick data
- [info_vaults.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/info_vaults.ts) — Vault information
- [info_batch_queries.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/info_batch_queries.ts) — Batch queries
- [hypercore_blocks.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/hypercore_blocks.ts) — Block and trade data
- [evm_basics.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/evm_basics.ts) — EVM chain interaction

**Streaming:**
- [stream_trades.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/stream_trades.ts) — WebSocket streaming basics
- [stream_grpc.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/stream_grpc.ts) — gRPC streaming basics
- [stream_l4_book.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/stream_l4_book.ts) — **L4 order book (individual orders) — CRITICAL**
- [stream_l2_book.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/stream_l2_book.ts) — L2 order book (gRPC vs WebSocket)
- [stream_orderbook.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/stream_orderbook.ts) — L2 vs L4 comparison
- [stream_websocket_all.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/stream_websocket_all.ts) — Complete WebSocket reference (20+ types)

**Complete Demo:**
- [full_demo.ts](https://github.com/quiknode-labs/hyperliquid-examples/blob/main/typescript/full_demo.ts) — All features in one file

**Learn More**
- Learn more about [Hyperliquid API](https://hyperliquidapi.com) here

---

## Architecture Notes (For SDK Implementers)

This section documents the routing logic for implementing SDKs in other languages.

### URL Routing

The SDK routes requests to different endpoints based on the operation:

| Endpoint | Routes To | Notes |
|----------|-----------|-------|
| `/exchange` | Worker | ALL trading operations (orders, cancels, etc.) |
| `/info` (supported methods) | QuickNode | Methods in `QN_SUPPORTED_INFO_METHODS` |
| `/info` (unsupported methods) | Worker | allMids, l2Book, recentTrades, candleSnapshot, predictedFundings |
| `/approval`, `/markets`, `/dexes`, `/preflight` | Worker | Always route to public worker |

### QuickNode Supported Info Methods

QuickNode nodes with `--serve-info-endpoint` support these methods:

```
meta, spotMeta, clearinghouseState, spotClearinghouseState,
openOrders, exchangeStatus, frontendOpenOrders, liquidatable,
activeAssetData, maxMarketOrderNtls, vaultSummaries, userVaultEquities,
leadingVaults, extraAgents, subAccounts, userFees, userRateLimit,
spotDeployState, perpDeployAuctionStatus, delegations, delegatorSummary,
maxBuilderFee, userToMultiSigSigners, userRole, perpsAtOpenInterestCap,
validatorL1Votes, marginTable, perpDexs, webData2
```

Methods NOT in this list (e.g., `allMids`, `l2Book`, `recentTrades`, `candleSnapshot`, `predictedFundings`) must route through the worker.

### Endpoint Parsing

The SDK extracts the token from any endpoint format:

```
https://x.quiknode.pro/TOKEN → base = https://x.quiknode.pro/TOKEN
https://x.quiknode.pro/TOKEN/info → base = https://x.quiknode.pro/TOKEN
https://x.quiknode.pro/TOKEN/evm → base = https://x.quiknode.pro/TOKEN
https://x.quiknode.pro/TOKEN/hypercore → base = https://x.quiknode.pro/TOKEN
```

Known path suffixes to strip: `info`, `hypercore`, `evm`, `nanoreth`, `ws`, `send`

### Worker URL

Public worker: `https://send.hyperliquidapi.com`

The worker handles:
- `/exchange` - ALL trading operations (orders, cancels, positions, etc.)
- `/info` - Info API fallback for unsupported methods
- `/approval` - Builder fee approval status
- `/markets` - Market metadata
- `/dexes` - DEX info
- `/preflight` - Order preflight validation

### Signature Chain IDs

```typescript
const MAINNET_CHAIN_ID = "0xa4b1";  // Arbitrum
const TESTNET_CHAIN_ID = "0x66eee"; // Arbitrum Sepolia
```

---

## API Parity with Python SDK

This TypeScript SDK implements all features of the Python SDK:

| Feature | Python | TypeScript |
|---------|--------|------------|
| Order Placement | ✅ | ✅ |
| Fluent Builders | ✅ | ✅ |
| Trigger Orders | ✅ | ✅ |
| TWAP Orders | ✅ | ✅ |
| Order Management | ✅ | ✅ |
| Info API (50+ methods) | ✅ | ✅ |
| HyperCore API | ✅ | ✅ |
| EVM JSON-RPC | ✅ | ✅ |
| WebSocket Streaming (20+ types) | ✅ | ✅ |
| gRPC Streaming (L2/L4 book) | ✅ | ✅ |
| Transfers | ✅ | ✅ |
| Vaults | ✅ | ✅ |
| Staking | ✅ | ✅ |
| Builder Fee | ✅ | ✅ |
| Agent Management | ✅ | ✅ |
| Account Abstraction | ✅ | ✅ |
| Advanced Transfers | ✅ | ✅ |
| Auto-reconnect | ✅ | ✅ |
| Error Translation | ✅ | ✅ |

---

## Disclaimer

This is an **unofficial community SDK**. It is **not affiliated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing.

## License

MIT License - see [LICENSE](LICENSE) for details.
