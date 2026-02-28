// Info Market Data Example — Query market metadata, prices, order book, and trades.
//
// This example matches the Python info_market_data.py exactly.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Info Market Data Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Info Market Data Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}
	info := sdk.Info()

	// Exchange metadata
	fmt.Println()
	fmt.Println("Exchange Metadata")
	fmt.Println("------------------------------")
	meta, err := info.Meta()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		universe, _ := meta["universe"].([]any)
		fmt.Printf("Perp markets: %d\n", len(universe))
	}

	// Spot metadata
	spotMeta, err := info.SpotMeta()
	if err != nil {
		fmt.Printf("Spot meta error: %v\n", err)
	} else {
		tokens, _ := spotMeta["tokens"].([]any)
		fmt.Printf("Spot tokens: %d\n", len(tokens))
	}

	// Exchange status
	status, err := info.ExchangeStatus()
	if err != nil {
		fmt.Printf("Status error: %v\n", err)
	} else {
		fmt.Printf("Exchange status: %v\n", status)
	}

	// All mid prices
	fmt.Println()
	fmt.Println("Mid Prices")
	fmt.Println("------------------------------")
	mids, err := info.AllMids()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Total assets: %d\n", len(mids))
		for _, coin := range []string{"BTC", "ETH", "SOL"} {
			if price, ok := mids[coin].(string); ok {
				pxFloat, _ := strconv.ParseFloat(price, 64)
				fmt.Printf("  %s: $%.2f\n", coin, pxFloat)
			}
		}
	}

	// Order book
	fmt.Println()
	fmt.Println("BTC Order Book")
	fmt.Println("------------------------------")
	book, err := info.L2Book("BTC")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
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

	// Recent trades
	fmt.Println()
	fmt.Println("Recent BTC Trades")
	fmt.Println("------------------------------")
	trades, err := info.RecentTrades("BTC")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Last %d trades:\n", min(3, len(trades)))
		for i, trade := range trades {
			if i >= 3 {
				break
			}
			t := trade.(map[string]any)
			px, _ := strconv.ParseFloat(t["px"].(string), 64)
			side := "SELL"
			if t["side"] == "B" {
				side = "BUY"
			}
			fmt.Printf("  %s %s @ $%.2f\n", side, t["sz"], px)
		}
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
