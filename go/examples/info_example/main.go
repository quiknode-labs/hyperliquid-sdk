// Info API Example
//
// Query market data, account info, and more via the Info API.
// This example matches the Python info_example.py exactly.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("QUICKNODE_ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Hyperliquid Info API Example")
		fmt.Println("=" + string(make([]byte, 49)))
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Hyperliquid Info API Example")
	fmt.Println("==================================================")
	fmt.Printf("Endpoint: %s...\n", truncate(endpoint, 50))
	fmt.Println()

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	info := sdk.Info()

	// ==========================================================================
	// Market Data
	// ==========================================================================
	fmt.Println("Market Data")
	fmt.Println("------------------------------")

	// Get all mid prices
	mids, err := info.AllMids()
	if err != nil {
		log.Printf("Failed to get all mids: %v", err)
	} else {
		btcMid := parseFloat(mids["BTC"])
		ethMid := parseFloat(mids["ETH"])
		fmt.Printf("BTC mid: $%.2f\n", btcMid)
		fmt.Printf("ETH mid: $%.2f\n", ethMid)
		fmt.Printf("Total assets: %d\n", len(mids))
	}
	fmt.Println()

	// Get L2 order book
	book, err := info.L2Book("BTC")
	if err != nil {
		log.Printf("Failed to get L2 book: %v", err)
	} else {
		levels, _ := book["levels"].([]any)
		if len(levels) >= 2 {
			bids, _ := levels[0].([]any)
			asks, _ := levels[1].([]any)

			if len(bids) > 0 && len(asks) > 0 {
				bestBid := bids[0].(map[string]any)
				bestAsk := asks[0].(map[string]any)

				bidPx := parseFloat(bestBid["px"])
				askPx := parseFloat(bestAsk["px"])
				spread := askPx - bidPx

				fmt.Println("BTC Book:")
				fmt.Printf("  Best Bid: %s @ $%.2f\n", bestBid["sz"], bidPx)
				fmt.Printf("  Best Ask: %s @ $%.2f\n", bestAsk["sz"], askPx)
				fmt.Printf("  Spread: $%.2f\n", spread)
			}
		}
	}
	fmt.Println()

	// Get recent trades
	trades, err := info.RecentTrades("ETH")
	if err != nil {
		log.Printf("Failed to get recent trades: %v", err)
	} else {
		fmt.Printf("Recent ETH trades: %d\n", len(trades))
		if len(trades) > 0 {
			lastTrade := trades[0].(map[string]any)
			px := parseFloat(lastTrade["px"])
			fmt.Printf("  Last: %s @ $%.2f\n", lastTrade["sz"], px)
		}
	}
	fmt.Println()

	// ==========================================================================
	// Exchange Metadata
	// ==========================================================================
	fmt.Println("Exchange Metadata")
	fmt.Println("------------------------------")

	meta, err := info.Meta()
	if err != nil {
		log.Printf("Failed to get meta: %v", err)
	} else {
		universe, _ := meta["universe"].([]any)
		fmt.Printf("Total perp markets: %d\n", len(universe))

		// Show a few markets
		for i, asset := range universe {
			if i >= 5 {
				break
			}
			a := asset.(map[string]any)
			fmt.Printf("  %s: %v size decimals\n", a["name"], a["szDecimals"])
		}
	}
	fmt.Println()

	// ==========================================================================
	// User Account (requires a valid address)
	// ==========================================================================
	fmt.Println("User Account")
	fmt.Println("------------------------------")

	// Example address - replace with your address
	userAddress := "0x0000000000000000000000000000000000000000"

	// Get clearinghouse state (positions, margin)
	state, err := info.ClearinghouseState(userAddress)
	if err != nil {
		fmt.Printf("  Could not fetch user data: %v\n", err)
	} else {
		marginSummary, _ := state["marginSummary"].(map[string]any)
		equity := parseFloat(marginSummary["accountValue"])
		fmt.Printf("Account equity: $%.2f\n", equity)

		positions, _ := state["assetPositions"].([]any)
		if len(positions) > 0 {
			fmt.Printf("Open positions: %d\n", len(positions))
			for i, pos := range positions {
				if i >= 3 {
					break
				}
				p := pos.(map[string]any)
				position := p["position"].(map[string]any)
				coin := position["coin"]
				size := position["szi"]
				entry := position["entryPx"]
				pnl := parseFloat(position["unrealizedPnl"])
				fmt.Printf("  %s: %s @ %s (PnL: $%.2f)\n", coin, size, entry, pnl)
			}
		} else {
			fmt.Println("  No open positions")
		}
	}
	fmt.Println()

	// ==========================================================================
	// Funding Rates
	// ==========================================================================
	fmt.Println("Funding Rates")
	fmt.Println("------------------------------")

	fundings, err := info.PredictedFundings()
	if err != nil {
		fmt.Printf("  Could not fetch funding: %v\n", err)
	} else {
		// API returns [[coin, [[venue, fundingInfo], ...]], ...]
		fmt.Printf("Predicted funding rates for %d assets:\n", len(fundings))
		count := 0
		for _, f := range fundings {
			if count >= 5 {
				break
			}
			arr, ok := f.([]any)
			if !ok || len(arr) < 2 {
				continue
			}
			coin, _ := arr[0].(string)
			venues, _ := arr[1].([]any)
			if len(venues) == 0 {
				continue
			}
			// Get first venue's funding rate
			for _, v := range venues {
				venueArr, ok := v.([]any)
				if !ok || len(venueArr) < 2 {
					continue
				}
				fundingInfo, ok := venueArr[1].(map[string]any)
				if !ok {
					continue
				}
				rate := parseFloat(fundingInfo["fundingRate"]) * 100
				fmt.Printf("  %s: %.4f%%\n", coin, rate)
				count++
				break
			}
		}
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func parseFloat(v any) float64 {
	switch val := v.(type) {
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case float64:
		return val
	default:
		return 0
	}
}
