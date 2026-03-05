# Hyperliquid SDK for Go

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

```go
import "github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"

sdk, _ := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
order, _ := sdk.MarketBuy("BTC", hyperliquid.WithNotional(100))  // Buy $100 of BTC
```

That's it. No build-sign-send ceremony. No manual hash signing. No nonce tracking. Just trading.

> **Community SDK** — Not affiliated with Hyperliquid Foundation.

## Installation

```bash
go get github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid
```

That's it. Everything is included: trading, Info API, WebSocket streaming, gRPC streaming, HyperCore, and EVM.

## Quick Start

### Endpoint Flexibility

The SDK automatically handles any endpoint format you provide:

```go
// All of these work - the SDK extracts the token and routes correctly
endpoint := "https://x.quiknode.pro/TOKEN"
endpoint := "https://x.quiknode.pro/TOKEN/"
endpoint := "https://x.quiknode.pro/TOKEN/info"
endpoint := "https://x.quiknode.pro/TOKEN/hypercore"
endpoint := "https://x.quiknode.pro/TOKEN/evm"
```

Just pass your endpoint - the SDK handles the rest.

You can create a Hyperliquid endpoint on [Quicknode](https://www.quicknode.com/) to get access to the data APIs.

### 1. Set your private key

```bash
export PRIVATE_KEY="0xYOUR_PRIVATE_KEY"
```

### 2. Start trading

```go
import "github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"

sdk, _ := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(os.Getenv("PRIVATE_KEY")))

// Market orders
order, _ := sdk.MarketBuy("BTC", hyperliquid.WithSize(0.001))
order, _ := sdk.MarketSell("ETH", hyperliquid.WithNotional(100))  // $100 worth

// Limit orders
order, _ := sdk.Buy("BTC", hyperliquid.WithSize(0.001), hyperliquid.WithPrice(65000), hyperliquid.WithTIF(hyperliquid.TIFGtc))

// Check your order
fmt.Println(order.Status)  // "filled" or "resting"
fmt.Println(order.OID)     // Order ID
```

---

## Data APIs

Query Hyperliquid data with clean, simple interfaces.

### Info API

60+ methods for account state, positions, market data, and metadata.

```go
info := sdk.Info()

// Market data
info.AllMids()                              // All mid prices
info.L2Book("BTC")                          // Order book
info.RecentTrades("BTC")                    // Recent trades
info.Candles("BTC", "1h", start, end)       // OHLCV candles
info.FundingHistory("BTC", start, end)      // Funding history
info.PredictedFundings()                    // Predicted funding rates

// Metadata
info.Meta()                                 // Exchange metadata
info.SpotMeta()                             // Spot metadata
info.ExchangeStatus()                       // Exchange status
info.PerpDexs()                             // Perpetual DEX info
info.MaxMarketOrderNtls()                   // Max market order notionals

// User data
info.ClearinghouseState("0x...")            // Positions & margin
info.SpotClearinghouseState("0x...")        // Spot balances
info.OpenOrders("0x...")                    // Open orders
info.FrontendOpenOrders("0x...")            // Enhanced open orders
info.OrderStatus("0x...", oid)              // Specific order status
info.HistoricalOrders("0x...")              // Order history
info.UserFills("0x...", false)              // Trade history
info.UserFillsByTime("0x...", start, end)   // Fills by time range
info.UserFunding("0x...", start, end)       // Funding payments
info.UserFees("0x...")                      // Fee structure
info.UserRateLimit("0x...")                 // Rate limit status
info.UserRole("0x...")                      // Account type
info.Portfolio("0x...")                     // Portfolio history
info.SubAccounts("0x...")                   // Sub-accounts
info.ExtraAgents("0x...")                   // API keys/agents

// TWAP
info.UserTWAPSliceFills("0x...", 500)       // TWAP slice fills

// Batch queries
info.BatchClearinghouseStates([]string{"0x...", "0x..."})

// Vaults
info.VaultSummaries()                       // All vault summaries
info.VaultDetails("0x...", "")              // Specific vault
info.UserVaultEquities("0x...")             // User's vault equities
info.LeadingVaults("0x...")                 // Vaults user leads

// Delegation/Staking
info.Delegations("0x...")                   // Active delegations
info.DelegatorSummary("0x...")              // Delegation summary
info.DelegatorHistory("0x...")              // Delegation history
info.DelegatorRewards("0x...")              // Delegation rewards

// Tokens
info.TokenDetails("token_id")               // Token details
info.SpotDeployState("0x...")               // Spot deployment state

// Other
info.Referral("0x...")                      // Referral info
info.MaxBuilderFee("0x...", "0x...")        // Builder fee limits
info.ApprovedBuilders("0x...")              // Approved builders
info.Liquidatable()                         // Liquidatable positions
```

### HyperCore API

Block data, trading operations, and real-time data via JSON-RPC.

```go
hc := sdk.Core()

// Block data
hc.LatestBlockNumber()                      // Latest block
hc.GetBlock(12345)                          // Get specific block
hc.GetBatchBlocks(100, 110)                 // Get block range
hc.LatestBlocks("trades", 10)               // Latest blocks

// Recent data
hc.LatestTrades(10, "")                     // Recent trades (all coins)
hc.LatestTrades(10, "BTC")                  // Recent BTC trades
hc.LatestOrders(10, "")                     // Recent order events
hc.LatestBookUpdates(10, "")                // Recent book updates

// Discovery
hc.ListDexes()                              // All DEXes
hc.ListMarkets("")                          // All markets
hc.ListMarkets("hyperliquidity")            // Markets by DEX

// Order queries
hc.OpenOrders("0x...")                      // User's open orders
hc.OrderStatus("0x...", oid)                // Specific order status
hc.Preflight(...)                           // Validate order before signing

// Order building (for manual signing)
hc.BuildOrder(coin, isBuy, limitPx, sz, user, reduceOnly, orderType, cloid)
hc.BuildCancel(coin, oid, user)
hc.BuildModify(coin, oid, user, limitPx, sz, isBuy)
hc.BuildApproveBuilderFee(user, builder, rate, nonce)
hc.BuildRevokeBuilderFee(user, builder, nonce)

// Send signed actions
hc.SendOrder(action, signature, nonce)
hc.SendCancel(action, signature, nonce)
hc.SendModify(action, signature, nonce)

// Builder fees
hc.GetMaxBuilderFee("0x...", "0x...")

// Subscriptions
hc.Subscribe(map[string]any{"type": "trades", "coin": "BTC"})
hc.Unsubscribe(map[string]any{"type": "trades", "coin": "BTC"})
```

### EVM (Ethereum JSON-RPC)

50+ Ethereum JSON-RPC methods for Hyperliquid's EVM chain (chain ID 999 mainnet, 998 testnet).

```go
evm := sdk.EVM()

// Chain info
evm.BlockNumber()                           // Latest block
evm.ChainID()                               // 999 mainnet, 998 testnet
evm.GasPrice()                              // Current gas price
evm.MaxPriorityFeePerGas()                  // Priority fee
evm.NetVersion()                            // Network version
evm.Syncing()                               // Sync status

// Accounts
evm.GetBalance("0x...", "latest")           // Account balance
evm.GetNonce("0x...", "latest")             // Nonce
evm.GetCode("0x...", "latest")              // Contract code
evm.GetStorageAt("0x...", position, "latest")  // Storage value

// Transactions
evm.Call(to, data, "latest")
evm.EstimateGas(to, data)
evm.SendRawTransaction(signedTx)
evm.GetTransactionByHash("0x...")
evm.GetTransactionReceipt("0x...")

// Blocks
evm.GetBlockByNumber("0x...", false)
evm.GetBlockByHash("0x...", false)
evm.GetBlockReceipts("0x...")
evm.GetBlockTransactionCountByNumber("0x...")
evm.GetBlockTransactionCountByHash("0x...")
evm.GetTransactionByBlockNumberAndIndex("0x...", 0)
evm.GetTransactionByBlockHashAndIndex("0x...", 0)

// Logs
evm.GetLogs(map[string]any{"address": "0x...", "topics": [...]})

// Fees
evm.FeeHistory(10, "latest", []float64{25, 50, 75})

// HyperEVM-specific
evm.BigBlockGasPrice()                      // Big block gas price
evm.UsingBigBlocks()                        // Is using big blocks?
evm.GetSystemTxsByBlockNumber("0x...")

// Debug/Trace
evm.DebugTraceTransaction("0x...", map[string]any{"tracer": "callTracer"})
evm.DebugTraceBlockByNumber("0x...", nil)
evm.DebugStorageRangeAt(blockHash, txIdx, addr, key, max)
evm.TraceTransaction("0x...")
evm.TraceBlock("0x...")
evm.TraceCall(tx, []string{"trace", "vmTrace"}, "latest")
evm.TraceFilter(map[string]any{"fromBlock": "0x1", "toBlock": "0x10"})
evm.TraceReplayTransaction("0x...", []string{"trace"})
```

---

## Real-Time Streaming

### WebSocket Streaming

20+ subscription types for real-time data with automatic reconnection.

```go
stream := hyperliquid.NewStream(endpoint, &hyperliquid.StreamConfig{
    Reconnect: true,
    OnError:   func(err error) { log.Printf("Error: %v", err) },
    OnOpen:    func() { log.Println("Connected") },
})

// Subscribe to trades
stream.Trades([]string{"BTC", "ETH"}, func(t map[string]any) {
    fmt.Printf("Trade: %v\n", t)
})

// Subscribe to book updates
stream.BookUpdates([]string{"BTC"}, func(b map[string]any) {
    fmt.Printf("Book: %v\n", b)
})

// Subscribe to orders (your orders)
stream.Orders([]string{"BTC"}, func(o map[string]any) {
    fmt.Printf("Order: %v\n", o)
}, "0x...")

// Run in background
if err := stream.Start(); err != nil {
    log.Fatal(err)
}
// ... do other work ...
stream.Stop()

// Or run blocking
stream.Run()
```

**Available WebSocket Streams:**

**Market Data:**
- `Trades(coins, callback)` — Executed trades
- `BookUpdates(coins, callback)` — Order book changes
- `L2Book(coin, callback)` — L2 order book snapshots
- `AllMids(callback)` — All mid price updates
- `Candle(coin, interval, callback)` — Candlestick data
- `BBO(coin, callback)` — Best bid/offer updates
- `ActiveAssetCtx(coin, callback)` — Asset context (pricing, volume)

**User Data:**
- `Orders(coins, callback, users...)` — Order lifecycle events
- `OpenOrders(user, callback)` — User's open orders
- `OrderUpdates(user, callback)` — Order status changes
- `UserEvents(user, callback)` — All user events
- `UserFills(user, callback)` — Trade fills
- `UserFundings(user, callback)` — Funding payments
- `UserNonFundingLedger(user, callback)` — Ledger changes
- `ClearinghouseState(user, callback)` — Position updates
- `ActiveAssetData(user, coin, callback)` — Trading parameters

**TWAP:**
- `TWAP(coins, callback)` — TWAP execution
- `TWAPStates(user, callback)` — TWAP algorithm states
- `UserTWAPSliceFills(user, callback)` — TWAP slice fills
- `UserTWAPHistory(user, callback)` — TWAP history

**System:**
- `Events(callback)` — System events (funding, liquidations)
- `Notification(user, callback)` — User notifications
- `WebData3(user, callback)` — Aggregate user info
- `WriterActions(user, callback)` — Writer actions

### gRPC Streaming (High Performance)

Lower latency streaming via gRPC for high-frequency applications.

```go
stream := hyperliquid.NewGRPCStream(endpoint, &hyperliquid.GRPCStreamConfig{
    Reconnect:  true,
    OnError:    func(err error) { log.Printf("Error: %v", err) },
    OnConnect:  func() { log.Println("Connected") },
})

// Subscribe to trades
stream.Trades([]string{"BTC", "ETH"}, func(t map[string]any) {
    fmt.Printf("Trade: %v\n", t)
})

// Subscribe to L2 order book (aggregated by price level)
stream.L2Book("BTC", func(b map[string]any) {
    fmt.Printf("Book: %v\n", b)
}, hyperliquid.WithNSigFigs(5))

// Subscribe to L4 order book (CRITICAL: individual orders with order IDs)
stream.L4Book("BTC", func(b map[string]any) {
    fmt.Printf("L4: %v\n", b)
})

// Subscribe to blocks
stream.Blocks(func(b map[string]any) {
    fmt.Printf("Block: %v\n", b)
})

// Run in background
if err := stream.Start(); err != nil {
    log.Fatal(err)
}
// ... do other work ...
stream.Stop()

// Or run blocking
stream.Run()
```

**Available gRPC Streams:**

| Method | Parameters | Description |
|--------|-----------|-------------|
| `Trades(coins, callback)` | coins: `[]string` | Executed trades with price, size, direction |
| `Orders(coins, callback, users...)` | coins: `[]string`, users: `...string` | Order lifecycle events |
| `BookUpdates(coins, callback)` | coins: `[]string` | Order book changes (deltas) |
| `L2Book(coin, callback, opts...)` | coin: `string`, opts: `L2BookOption` | L2 order book (aggregated by price) |
| `L4Book(coin, callback)` | coin: `string` | **L4 order book (individual orders)** |
| `Blocks(callback)` | - | Block data |
| `TWAP(coins, callback)` | coins: `[]string` | TWAP execution updates |
| `Events(callback)` | - | System events (funding, liquidations) |
| `WriterActions(callback)` | - | Writer actions |

### L4 Order Book (Critical for Trading)

L4 order book shows **every individual order** with its order ID. This is essential for:

- **Market Making**: Know your exact queue position
- **Order Flow Analysis**: Detect large orders and icebergs
- **Optimal Execution**: See exactly what you're crossing
- **HFT**: Lower latency than WebSocket

```go
stream := hyperliquid.NewGRPCStream(endpoint, nil)

stream.L4Book("BTC", func(data map[string]any) {
    // L4 book data structure:
    // {
    //     "coin": "BTC",
    //     "bids": [[price, size, order_id], ...],
    //     "asks": [[price, size, order_id], ...]
    // }
    bids, _ := data["bids"].([]any)
    for i, bid := range bids {
        if i >= 3 { break }
        arr := bid.([]any)
        fmt.Printf("Bid: $%v x %v (order: %v)\n", arr[0], arr[1], arr[2])
    }
})

stream.Run()
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

```go
// Market orders
sdk.MarketBuy("BTC", hyperliquid.WithSize(0.001))
sdk.MarketSell("ETH", hyperliquid.WithNotional(100))

// Limit orders
sdk.Buy("BTC", hyperliquid.WithSize(0.001), hyperliquid.WithPrice(65000))
sdk.Sell("ETH", hyperliquid.WithSize(0.5), hyperliquid.WithPrice(4000), hyperliquid.WithTIF(hyperliquid.TIFGtc))

// Perp trader aliases
sdk.Long("BTC", hyperliquid.WithSize(0.001), hyperliquid.WithPrice(65000))
sdk.Short("ETH", hyperliquid.WithNotional(500), hyperliquid.WithTIF(hyperliquid.TIFIoc))
```

### Fluent Order Builder

```go
order := hyperliquid.Order().
    Buy("BTC").
    Size(0.001).
    Price(65000).
    GTC().
    ReduceOnly().
    RandomCloid()

result, _ := sdk.PlaceOrder(order)
```

### Order Management

```go
// Place, modify, cancel
order, _ := sdk.Buy("BTC", hyperliquid.WithSize(0.001), hyperliquid.WithPrice(60000), hyperliquid.WithTIF(hyperliquid.TIFGtc))
order.Modify(61000, 0, "")
order.Cancel()

// Cancel all
sdk.CancelAll("")           // All orders
sdk.CancelAll("BTC")        // Just BTC orders

// Dead-man's switch
sdk.ScheduleCancel(timestampMs)
```

### Position Management

```go
sdk.ClosePosition("BTC")  // Close entire position
```

### Querying Open Orders by Trading Pair

```go
// Get all open orders
result, _ := sdk.OpenOrders("")
orders := result["orders"].([]any)
fmt.Printf("Total open orders: %d\n", len(orders))

// Order fields: coin, limitPx (price), sz (size), side, oid, timestamp,
//               orderType, tif, cloid, reduceOnly
for _, o := range orders {
	order := o.(map[string]any)
	fmt.Printf("%s %s %s@%s\n", order["coin"], order["side"], order["sz"], order["limitPx"])
}

// Filter by trading pair
for _, o := range orders {
	order := o.(map[string]any)
	if order["coin"].(string) == "BTC" {
		fmt.Printf("  %s %s @ %s | type=%s tif=%s oid=%v\n",
			order["side"], order["sz"], order["limitPx"], order["orderType"], order["tif"], order["oid"])
	}
}

// For enhanced data (triggers, children), use FrontendOpenOrders()
enhanced, _ := sdk.Info().FrontendOpenOrders(sdk.Address())
```

### Partial Position Close by Percentage

`ClosePosition()` closes the entire position. To close a percentage, read the current size and place a reduce-only market order for the desired amount:

```go
func closePercentage(sdk *hyperliquid.SDK, coin string, percent float64) error {
	state, err := sdk.Info().ClearinghouseState(sdk.Address())
	if err != nil {
		return err
	}

	var szi float64
	positions := state["assetPositions"].([]any)
	for _, ap := range positions {
		pos := ap.(map[string]any)["position"].(map[string]any)
		if pos["coin"].(string) == coin {
			szi, _ = strconv.ParseFloat(pos["szi"].(string), 64)
			break
		}
	}
	if szi == 0 {
		return fmt.Errorf("no open position for %s", coin)
	}

	// szi is signed: positive = long, negative = short
	closeSize := math.Abs(szi) * (percent / 100)

	if szi > 0 {
		// Long position: sell to close
		_, err = sdk.MarketSell(coin, hyperliquid.WithSize(closeSize), hyperliquid.WithReduceOnly())
	} else {
		// Short position: buy to close
		_, err = sdk.MarketBuy(coin, hyperliquid.WithSize(closeSize), hyperliquid.WithReduceOnly())
	}
	return err
}

// Close 50% of BTC position
closePercentage(sdk, "BTC", 50)
```

### Batch Cancel with Partial Failure Handling

```go
// Get open orders
result, _ := sdk.OpenOrders("")
orders := result["orders"].([]any)

// Cancel all orders for a specific asset
sdk.CancelAll("BTC")

// Cancel specific orders with per-order error handling
type cancelFailure struct {
	OID   int64
	Error string
}
var failures []cancelFailure

for _, o := range orders {
	order := o.(map[string]any)
	if order["coin"].(string) == "BTC" {
		limitPx, _ := strconv.ParseFloat(order["limitPx"].(string), 64)
		if limitPx < 50000 {
			oid := int64(order["oid"].(float64))
			if _, err := sdk.Cancel(oid, "BTC"); err != nil {
				failures = append(failures, cancelFailure{OID: oid, Error: err.Error()})
			}
		}
	}
}

if len(failures) > 0 {
	fmt.Printf("Failed to cancel %d orders: %v\n", len(failures), failures)
}

// Cancel by client order ID (for CLOID-tracked orders)
sdk.CancelByCloid("0xmycloid...", "BTC")
```

### Resilient Order Placement

Use client order IDs (CLOIDs) for idempotent orders and categorize errors for retry logic:

```go
// Set a CLOID for idempotency — the exchange rejects duplicates
order := hyperliquid.Order().
	Buy("BTC").
	Size(0.001).
	Price(65000).
	GTC().
	CLOID("0x" + generateCLOID())

result, _ := sdk.PlaceOrder(order)

// Error categories:
//   Transient (retry):   ErrorCodeRateLimited, ErrorCodeInvalidNonce
//   Permanent (fail):    ErrorCodeGeoBlocked, ErrorCodeInsufficientMargin,
//                        ErrorCodeInvalidParams, ErrorCodeSignatureInvalid, ErrorCodeMaxOrdersExceeded
//   Already done:        ErrorCodeDuplicateOrder (order already placed)

func placeWithRetry(sdk *hyperliquid.SDK, builder *hyperliquid.OrderBuilder, maxRetries int) error {
	builder = builder.CLOID("0x" + generateCLOID())

	for attempt := 0; attempt < maxRetries; attempt++ {

		_, err := sdk.PlaceOrder(builder)
		if err == nil {
			return nil
		}

		var hlErr *hyperliquid.Error
		if errors.As(err, &hlErr) {
			switch hlErr.Code {
			case hyperliquid.ErrorCodeDuplicateOrder:
				return nil // Order already went through
			case hyperliquid.ErrorCodeRateLimited, hyperliquid.ErrorCodeInvalidNonce:
				if attempt == maxRetries-1 {
					return err
				}
				wait := time.Duration((1<<attempt)*1000+rand.Intn(1000)) * time.Millisecond
				time.Sleep(wait)
				continue
			}
		}
		return err
	}
	return nil
}

// Timeout configuration on SDK constructor
sdk, _ := hyperliquid.New(endpoint,
	hyperliquid.WithPrivateKey(key),
	hyperliquid.WithTimeout(30*time.Second),
)
```

### Leverage & Margin

```go
// Update leverage
sdk.UpdateLeverage("BTC", 10, true)   // 10x cross
sdk.UpdateLeverage("ETH", 5, false)   // 5x isolated

// Isolated margin management
sdk.UpdateIsolatedMargin("BTC", true, 100)    // Add margin to long
sdk.UpdateIsolatedMargin("ETH", false, -50)   // Remove from short
sdk.TopUpIsolatedOnlyMargin("BTC", 100)       // Special maintenance mode
```

### Trigger Orders (Stop Loss / Take Profit)

```go
// Stop loss (market order when triggered)
sdk.StopLoss("BTC", 0.001, 60000)

// Stop loss (limit order when triggered)
sdk.StopLoss("BTC", 0.001, 60000, hyperliquid.WithLimitPrice(59500))

// Take profit
sdk.TakeProfit("BTC", 0.001, 70000)

// Using fluent builder
trigger := hyperliquid.TriggerOrder().
    StopLoss("BTC").
    Size(0.001).
    TriggerPrice(60000).
    Market()

sdk.PlaceTriggerOrder(trigger)

// Buy-side (closing shorts)
trigger := hyperliquid.TriggerOrder().
    StopLoss("BTC").
    Size(0.001).
    TriggerPrice(70000).
    Side(hyperliquid.SideBuy).
    Market()
```

### TWAP Orders

```go
// Time-weighted average price order
result, _ := sdk.TWAPOrder(
    "BTC",
    0.01,          // size
    true,          // isBuy
    60,            // durationMinutes
    true,          // randomize
    false,         // reduceOnly
)
twapID := result["response"].(map[string]any)["data"].(map[string]any)["running"].(map[string]any)["id"].(float64)

// Cancel TWAP
sdk.TWAPCancel("BTC", int64(twapID))
```

### Transfers

```go
// Internal transfers
sdk.TransferSpotToPerp(100)
sdk.TransferPerpToSpot(100)

// External transfers
sdk.TransferUSD("0x...", 100)
sdk.TransferSpot("0x...", "PURR", 100)
sdk.SendAsset("0x...", "USDC", 100)

// Withdraw to L1 (Arbitrum)
sdk.Withdraw("0x...", 100)
```

### Vaults

```go
hlpVault := "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303"
sdk.VaultDeposit(hlpVault, 100)
sdk.VaultWithdraw(hlpVault, 50)
```

### Staking

```go
// Stake/unstake HYPE
sdk.Stake(1000)
sdk.Unstake(500)  // 7-day queue

// Delegate to validators
sdk.Delegate("0x...", 500)
sdk.Undelegate("0x...", 250)
```

### Builder Fee

```go
// Check approval
status, _ := sdk.ApprovalStatus()

// Approve builder fee
sdk.ApproveBuilderFee("1%")

// Revoke
sdk.RevokeBuilderFee()
```

### Agent/API Key Management

```go
// Approve an agent to trade on your behalf
sdk.ApproveAgent("0xAgent...", "my-bot")
```

### Account Abstraction

```go
// Set abstraction mode
sdk.SetAbstraction("unifiedAccount", "")
sdk.SetAbstraction("portfolioMargin", "")
sdk.SetAbstraction("disabled", "")

// As an agent
sdk.AgentSetAbstraction("unifiedAccount")
```

### Advanced Transfers

```go
// Send asset between DEXs
sdk.SendAsset("0xDest...", "USDC:0x...", 100)

// Send to EVM with data
sdk.SendToEVMWithData(
    "PURR:0x...", 100, "0xContract...", "0xcalldata...",
    "", 999, 100000,
)
```

### Additional Operations

```go
// Cancel by client order ID
sdk.CancelByCloid("0xmycloid...", "BTC")

// Reserve rate limit capacity
sdk.ReserveRequestWeight(1000)

// No-op (consume nonce)
sdk.Noop()

// Preflight validation
sdk.Preflight("BTC", hyperliquid.SideBuy, 67000, 0.001)

// Refresh markets cache
sdk.RefreshMarkets()
```

---

## Error Handling

All errors include a code and message.

```go
order, err := sdk.Buy("BTC", hyperliquid.WithSize(0.001), hyperliquid.WithPrice(65000))
if err != nil {
    if hlErr, ok := err.(*hyperliquid.Error); ok {
        switch hlErr.Code {
        case hyperliquid.ErrorCodeApproval:
            fmt.Printf("Need approval: %s\n", hlErr.Guidance)
        case hyperliquid.ErrorCodeInsufficientMargin:
            fmt.Printf("Not enough margin: %s\n", hlErr.Guidance)
        case hyperliquid.ErrorCodeGeoBlocked:
            fmt.Printf("Geo-blocked: %s\n", hlErr.Message)
        default:
            fmt.Printf("Error [%s]: %s\n", hlErr.Code, hlErr.Message)
        }
    }
}
```

**Available Error Codes:**
- `ErrorCodeBuild` — Order building failed
- `ErrorCodeSend` — Transaction send failed
- `ErrorCodeApproval` — Builder fee approval needed
- `ErrorCodeValidation` — Invalid parameters
- `ErrorCodeSignature` — Signature verification failed
- `ErrorCodeNoPosition` — No position to close
- `ErrorCodeOrderNotFound` — Order not found
- `ErrorCodeGeoBlocked` — Region blocked
- `ErrorCodeInsufficientMargin` — Not enough margin
- `ErrorCodeLeverage` — Invalid leverage
- `ErrorCodeRateLimit` — Rate limited
- `ErrorCodeMaxOrders` — Too many orders
- `ErrorCodeReduceOnly` — Reduce-only constraint
- `ErrorCodeDuplicateOrder` — Duplicate order
- `ErrorCodeUserNotFound` — User not found
- `ErrorCodeMustDeposit` — Deposit required
- `ErrorCodeInvalidNonce` — Invalid nonce

---

## API Reference

### HyperliquidSDK (Trading)

```go
hyperliquid.New(
    endpoint,                                  // Endpoint URL
    hyperliquid.WithPrivateKey(key),           // Private key for signing
    hyperliquid.WithAutoApprove(true),         // Auto-approve builder fee (default: true)
    hyperliquid.WithMaxFee("1%"),              // Max fee for auto-approval
    hyperliquid.WithSlippage(0.03),            // Default slippage for market orders (3%)
    hyperliquid.WithTimeout(30*time.Second),   // Request timeout
)
```

### Info (Account & Metadata)

```go
info := sdk.Info()
// Or standalone:
info := hyperliquid.NewInfoClient(endpoint, httpClient)
```

### HyperCore (Blocks & Trades)

```go
hc := sdk.Core()
// Or standalone:
hc := hyperliquid.NewHyperCoreClient(endpoint, httpClient)
```

### EVM (Ethereum JSON-RPC)

```go
evm := sdk.EVM()
// Or standalone:
evm := hyperliquid.NewEVMClient(endpoint, httpClient)
```

### Stream (WebSocket)

```go
hyperliquid.NewStream(endpoint, &hyperliquid.StreamConfig{
    Reconnect:    true,               // Auto-reconnect
    PingInterval: 30 * time.Second,   // Ping interval
    OnError:      func(err error) {}, // Error callback
    OnClose:      func() {},          // Close callback
    OnOpen:       func() {},          // Open callback
    OnReconnect:  func(n int) {},     // Reconnect callback
})
```

### GRPCStream (gRPC)

```go
hyperliquid.NewGRPCStream(endpoint, &hyperliquid.GRPCStreamConfig{
    Reconnect:  true,                 // Auto-reconnect
    OnError:    func(err error) {},   // Error callback
    OnClose:    func() {},            // Close callback
    OnConnect:  func() {},            // Connect callback
})
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PRIVATE_KEY` | Your Ethereum private key (hex, with or without 0x) |
| `ENDPOINT` | Endpoint URL |

---

## Examples

See the [hyperliquid-examples](https://github.com/quiknode-labs/hyperliquid-examples) repository for 44 complete, runnable examples covering trading, streaming, market data, and more.

Learn more at [Hyperliquidapi.com](https://hyperliquidapi.com/).

---

## Disclaimer

This is an **unofficial community SDK**. It is **not affiliated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing.

## License

MIT License - see [LICENSE](LICENSE) for details.
