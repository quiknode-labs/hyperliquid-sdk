// Full Demo — Comprehensive example of all SDK capabilities.
//
// This example demonstrates all major SDK features:
// - Info API (market data, user info)
// - HyperCore API (blocks, trades, orders)
// - EVM API (chain data, balances)
// - WebSocket streaming
// - gRPC streaming
// - Trading (orders, positions)
package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func separator(title string) {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Printf("  %s\n", title)
	fmt.Println("============================================================")
}

func subsection(title string) {
	fmt.Println()
	fmt.Printf("--- %s ---\n", title)
}

func demoInfoAPI(sdk *hyperliquid.SDK) {
	separator("INFO API")

	info := sdk.Info()

	subsection("Market Prices")
	mids, err := info.AllMids()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Total markets: %d\n", len(mids))
		for _, coin := range []string{"BTC", "ETH", "SOL", "DOGE"} {
			if price, ok := mids[coin].(string); ok {
				pxFloat, _ := strconv.ParseFloat(price, 64)
				fmt.Printf("  %s: $%.2f\n", coin, pxFloat)
			}
		}
	}

	subsection("Order Book")
	book, err := info.L2Book("BTC")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		levels, _ := book["levels"].([]any)
		if len(levels) >= 2 {
			bids, _ := levels[0].([]any)
			asks, _ := levels[1].([]any)
			if len(bids) > 0 && len(asks) > 0 {
				bestBid := bids[0].(map[string]any)
				bestAsk := asks[0].(map[string]any)
				bidPx, _ := strconv.ParseFloat(bestBid["px"].(string), 64)
				askPx, _ := strconv.ParseFloat(bestAsk["px"].(string), 64)
				spread := askPx - bidPx
				fmt.Printf("  Best Bid: %s @ $%.2f\n", bestBid["sz"], bidPx)
				fmt.Printf("  Best Ask: %s @ $%.2f\n", bestAsk["sz"], askPx)
				fmt.Printf("  Spread: $%.2f\n", spread)
			}
		}
	}

	subsection("Recent Trades")
	trades, err := info.RecentTrades("ETH")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Last 3 ETH trades:")
		for i, trade := range trades {
			if i >= 3 {
				break
			}
			t := trade.(map[string]any)
			px, _ := strconv.ParseFloat(t["px"].(string), 64)
			fmt.Printf("  %s @ $%.2f (%s)\n", t["sz"], px, t["side"])
		}
	}

	subsection("Exchange Metadata")
	meta, err := info.Meta()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		universe, _ := meta["universe"].([]any)
		fmt.Printf("Total perp markets: %d\n", len(universe))
	}

	subsection("Predicted Funding")
	fundings, err := info.PredictedFundings()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		// Extract funding rates - API returns [[coin, [[venue, fundingInfo], ...]], ...]
		type fundingEntry struct {
			coin string
			rate float64
		}
		var entries []fundingEntry

		for _, f := range fundings {
			arr, ok := f.([]any)
			if !ok || len(arr) < 2 {
				continue
			}
			coin, _ := arr[0].(string)
			venues, _ := arr[1].([]any)
			if len(venues) == 0 {
				continue
			}
			// Use the first venue's funding rate (HlPerp if available)
			for _, v := range venues {
				venueArr, ok := v.([]any)
				if !ok || len(venueArr) < 2 {
					continue
				}
				fundingInfo, ok := venueArr[1].(map[string]any)
				if !ok {
					continue
				}
				rateStr, _ := fundingInfo["fundingRate"].(string)
				rate, _ := strconv.ParseFloat(rateStr, 64)
				entries = append(entries, fundingEntry{coin: coin, rate: rate})
				break
			}
		}

		// Sort by absolute funding rate
		sort.Slice(entries, func(i, j int) bool {
			ri := entries[i].rate
			rj := entries[j].rate
			if ri < 0 {
				ri = -ri
			}
			if rj < 0 {
				rj = -rj
			}
			return ri > rj
		})

		fmt.Println("Top 3 funding rates:")
		for i, e := range entries {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s: %+.4f%% (8h)\n", e.coin, e.rate*100)
		}
	}
}

func demoHyperCoreAPI(sdk *hyperliquid.SDK) {
	separator("HYPERCORE API")

	hc := sdk.Core()

	subsection("Latest Block")
	blockNum, err := hc.LatestBlockNumber()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Latest block: %d\n", blockNum)

		block, err := hc.GetBlock(blockNum)
		if err != nil {
			log.Printf("Error: %v", err)
		} else if block != nil {
			events, _ := block["events"].([]any)
			fmt.Printf("Block %d: %d events\n", blockNum, len(events))
		}
	}

	subsection("Recent Trades")
	trades, err := hc.LatestTrades(5, "")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Last 5 trades across all markets:")
		for _, trade := range trades {
			coin, _ := trade["coin"].(string)
			sz, _ := trade["sz"].(string)
			px, _ := trade["px"].(string)
			pxFloat, _ := strconv.ParseFloat(px, 64)
			fmt.Printf("  %s: %s @ $%.2f\n", coin, sz, pxFloat)
		}
	}

	subsection("Recent Orders")
	orders, err := hc.LatestOrders(5, "")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Last 5 orders:")
		for _, order := range orders {
			coin, _ := order["coin"].(string)
			side, _ := order["side"].(string)
			px, _ := order["limitPx"].(string)
			status, _ := order["status"].(string)
			pxFloat, _ := strconv.ParseFloat(px, 64)
			fmt.Printf("  %s: %s @ $%.2f - %s\n", coin, side, pxFloat, status)
		}
	}
}

func demoEVMAPI(sdk *hyperliquid.SDK) {
	separator("EVM API")

	evm := sdk.EVM()

	subsection("Chain Info")
	chainID, _ := evm.ChainID()
	blockNum, _ := evm.BlockNumber()
	gasPrice, _ := evm.GasPrice()

	network := "Unknown"
	if chainID == 999 {
		network = "Mainnet"
	} else if chainID == 998 {
		network = "Testnet"
	}
	fmt.Printf("Chain ID: %d (%s)\n", chainID, network)
	fmt.Printf("Block: %d\n", blockNum)
	fmt.Printf("Gas: %.2f Gwei\n", float64(gasPrice)/1e9)

	subsection("Latest Block")
	block, err := evm.GetBlockByNumber(fmt.Sprintf("0x%x", blockNum), false)
	if err != nil {
		log.Printf("Error: %v", err)
	} else if block != nil {
		hash, _ := block["hash"].(string)
		if len(hash) > 30 {
			hash = hash[:30]
		}
		gasUsed, _ := block["gasUsed"].(string)
		gasUsedInt := parseHex(gasUsed)
		fmt.Printf("Block %d:\n", blockNum)
		fmt.Printf("  Hash: %s...\n", hash)
		fmt.Printf("  Gas Used: %d\n", gasUsedInt)
	}
}

func demoWebSocket(sdk *hyperliquid.SDK, duration int) {
	separator("WEBSOCKET STREAMING")

	tradeCount := 0
	bookCount := 0

	stream := sdk.NewStream(&hyperliquid.StreamConfig{
		Reconnect: false,
		OnError: func(err error) {
			fmt.Printf("  [ERROR] %v\n", err)
		},
		OnOpen: func() {
			fmt.Println("  [CONNECTED] WebSocket stream ready")
		},
	})

	stream.Trades([]string{"BTC", "ETH"}, func(data map[string]any) {
		tradeCount++
		if tradeCount <= 3 {
			d, _ := data["data"].(map[string]any)
			if d != nil {
				fmt.Printf("  [TRADE] %s: %s @ %s\n", d["coin"], d["sz"], d["px"])
			}
		}
	})

	stream.BookUpdates([]string{"BTC"}, func(data map[string]any) {
		bookCount++
		if bookCount <= 2 {
			d, _ := data["data"].(map[string]any)
			if d != nil {
				fmt.Printf("  [BOOK] %s update\n", d["coin"])
			}
		}
	})

	fmt.Printf("Streaming for %d seconds...\n", duration)

	// Run in background
	if err := stream.Start(); err != nil {
		log.Printf("Failed to start stream: %v", err)
		return
	}
	time.Sleep(time.Duration(duration) * time.Second)
	stream.Stop()

	fmt.Println()
	fmt.Printf("Received: %d trades, %d book updates\n", tradeCount, bookCount)
}

func demoGRPC(sdk *hyperliquid.SDK, duration int) {
	separator("GRPC STREAMING")

	tradeCount := 0

	stream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Reconnect: false,
		OnError: func(err error) {
			fmt.Printf("  [ERROR] %v\n", err)
		},
		OnConnect: func() {
			fmt.Println("  [CONNECTED] gRPC stream ready")
		},
	})

	stream.Trades([]string{"BTC", "ETH"}, func(data map[string]any) {
		tradeCount++
		if tradeCount <= 3 {
			fmt.Printf("  [TRADE] %v\n", data)
		}
	})

	fmt.Printf("Streaming for %d seconds...\n", duration)

	// Run in background
	if err := stream.Start(); err != nil {
		log.Printf("Failed to start stream: %v", err)
		return
	}
	time.Sleep(time.Duration(duration) * time.Second)
	stream.Stop()

	fmt.Println()
	fmt.Printf("Received: %d trades\n", tradeCount)
}

func demoTrading(sdk *hyperliquid.SDK) {
	separator("TRADING")

	fmt.Printf("Address: %s\n", sdk.Address())

	subsection("Account Check")
	fmt.Println("  Trading SDK initialized successfully")
	fmt.Println("  Ready to place orders (not executing in demo)")

	subsection("Order Building (Example)")
	fmt.Println("  Market buy: sdk.MarketBuy(\"BTC\", hyperliquid.WithNotional(100))")
	fmt.Println("  Limit sell: sdk.PlaceOrder(hyperliquid.Order().Sell(\"ETH\").Size(1.0).Price(4000).GTC())")
	fmt.Println("  Close pos:  sdk.ClosePosition(\"BTC\")")
}

func main() {
	fmt.Println()
	fmt.Println("************************************************************")
	fmt.Println("  HYPERLIQUID SDK - FULL DEMO")
	fmt.Println("************************************************************")

	endpoint := os.Getenv("ENDPOINT")
	privateKey := os.Getenv("PRIVATE_KEY")

	if endpoint == "" {
		fmt.Println()
		fmt.Println("Error: ENDPOINT not set")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://your-endpoint/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	// Create SDK once
	var opts []hyperliquid.Option
	if privateKey != "" {
		opts = append(opts, hyperliquid.WithPrivateKey(privateKey))
	}
	sdk, err := hyperliquid.New(endpoint, opts...)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	displayEndpoint := endpoint
	if len(displayEndpoint) > 50 {
		displayEndpoint = displayEndpoint[:50]
	}
	fmt.Println()
	fmt.Printf("Endpoint: %s...\n", displayEndpoint)

	// Run all demos using the same SDK instance
	demoInfoAPI(sdk)
	demoHyperCoreAPI(sdk)
	demoEVMAPI(sdk)
	demoWebSocket(sdk, 5)
	demoGRPC(sdk, 5)

	if privateKey != "" {
		demoTrading(sdk)
	} else {
		fmt.Println()
		fmt.Println("--- TRADING (skipped - no PRIVATE_KEY) ---")
	}

	separator("DONE")
	fmt.Println("All demos completed successfully!")
	fmt.Println()
}

func parseHex(s string) int64 {
	if len(s) < 2 {
		return 0
	}
	if s[:2] == "0x" {
		s = s[2:]
	}
	val, _ := strconv.ParseInt(s, 16, 64)
	return val
}
