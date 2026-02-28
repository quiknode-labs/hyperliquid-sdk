// Info Candles Example — Fetch historical candlestick data and funding rates.
//
// This example matches the Python info_candles.py exactly.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Info Candles Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Info Candles Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}
	info := sdk.Info()

	// Get candles for the last 24 hours
	now := time.Now().UnixMilli()
	oneDayAgo := now - (24 * 60 * 60 * 1000)

	fmt.Println()
	fmt.Println("BTC 1-Hour Candles (Last 24h)")
	fmt.Println("------------------------------")
	candles, err := info.Candles("BTC", "1h", oneDayAgo, now)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Note: candleSnapshot may not be available on all QuickNode endpoints")
	} else {
		fmt.Printf("Received %d candles\n", len(candles))
		for i, candle := range candles {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(candles)-3)
				break
			}
			c := candle.(map[string]any)
			open, _ := c["o"].(string)
			high, _ := c["h"].(string)
			low, _ := c["l"].(string)
			close, _ := c["c"].(string)
			openF, _ := strconv.ParseFloat(open, 64)
			highF, _ := strconv.ParseFloat(high, 64)
			lowF, _ := strconv.ParseFloat(low, 64)
			closeF, _ := strconv.ParseFloat(close, 64)
			fmt.Printf("  O: $%.2f H: $%.2f L: $%.2f C: $%.2f\n", openF, highF, lowF, closeF)
		}
	}

	// Get predicted funding rates
	fmt.Println()
	fmt.Println("Predicted Funding Rates")
	fmt.Println("------------------------------")
	fundings, err := info.PredictedFundings()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Total assets: %d\n", len(fundings))
		count := 0
		for _, f := range fundings {
			if count >= 5 {
				fmt.Printf("  ... and %d more\n", len(fundings)-5)
				break
			}
			// Format: [[coin, [[source, {fundingRate, ...}], ...]], ...]
			arr, ok := f.([]any)
			if !ok || len(arr) < 2 {
				continue
			}
			coin, _ := arr[0].(string)
			venues, _ := arr[1].([]any)
			if len(venues) == 0 {
				continue
			}
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
				fmt.Printf("  %s: %.4f%% (8h)\n", coin, rate*100)
				count++
				break
			}
		}
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
