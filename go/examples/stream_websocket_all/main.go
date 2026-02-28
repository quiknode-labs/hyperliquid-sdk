// Stream WebSocket All Example — Subscribe to all available WebSocket channels.
//
// This example demonstrates subscribing to multiple WebSocket channels
// simultaneously: trades, book updates, and user events.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")

	if endpoint == "" {
		fmt.Println("Stream WebSocket All Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Stream WebSocket All Example")
	fmt.Println("==================================================")
	displayEndpoint := endpoint
	if len(displayEndpoint) > 60 {
		displayEndpoint = displayEndpoint[:60] + "..."
	}
	fmt.Printf("Endpoint: %s\n", displayEndpoint)
	fmt.Println()

	// Create SDK
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	// Track statistics
	stats := struct {
		trades int
		books  int
		start  time.Time
	}{start: time.Now()}

	// Create stream via SDK
	stream := sdk.NewStream(&hyperliquid.StreamConfig{
		Reconnect:    true,
		PingInterval: 30,
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
		OnClose: func() {
			fmt.Println("[CLOSED] Stream stopped")
		},
		OnStateChange: func(state hyperliquid.ConnectionState) {
			fmt.Printf("[STATE] %s\n", state)
		},
		OnReconnect: func(attempt int) {
			fmt.Printf("[RECONNECT] Attempt %d\n", attempt)
		},
	})

	// Subscribe to trades for multiple coins
	coins := []string{"BTC", "ETH", "SOL", "DOGE", "AVAX"}
	stream.Trades(coins, func(data map[string]any) {
		stats.trades++
		trades, ok := data["data"].([]any)
		if ok && len(trades) > 0 {
			for _, trade := range trades {
				t := trade.(map[string]any)
				coin, _ := t["coin"].(string)
				px, _ := t["px"].(string)
				sz, _ := t["sz"].(string)
				side, _ := t["side"].(string)
				pxFloat, _ := strconv.ParseFloat(px, 64)

				sideStr := "SELL"
				if side == "B" {
					sideStr = "BUY"
				}
				fmt.Printf("[TRADE] %s: %s %s @ $%.2f\n", coin, sideStr, sz, pxFloat)
			}
		}
	})
	fmt.Printf("Subscribed to trades: %v\n", coins)

	// Subscribe to book updates
	stream.BookUpdates([]string{"BTC", "ETH"}, func(data map[string]any) {
		stats.books++
		bookData, ok := data["data"].(map[string]any)
		if !ok {
			return
		}

		coin, _ := bookData["coin"].(string)
		levels, ok := bookData["levels"].([]any)
		if !ok || len(levels) < 2 {
			return
		}

		bids, _ := levels[0].([]any)
		asks, _ := levels[1].([]any)

		if len(bids) > 0 && len(asks) > 0 {
			bestBid := bids[0].(map[string]any)
			bestAsk := asks[0].(map[string]any)

			bidPx, _ := bestBid["px"].(string)
			askPx, _ := bestAsk["px"].(string)
			bidPxFloat, _ := strconv.ParseFloat(bidPx, 64)
			askPxFloat, _ := strconv.ParseFloat(askPx, 64)
			spread := askPxFloat - bidPxFloat

			fmt.Printf("[BOOK] %s: Bid $%.2f | Ask $%.2f | Spread $%.2f\n",
				coin, bidPxFloat, askPxFloat, spread)
		}
	})
	fmt.Println("Subscribed to book updates: BTC, ETH")

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		duration := time.Since(stats.start)
		fmt.Println("\nShutting down gracefully...")
		fmt.Printf("Duration: %v\n", duration.Round(time.Second))
		fmt.Printf("Total trades: %d\n", stats.trades)
		fmt.Printf("Total book updates: %d\n", stats.books)
		if duration.Seconds() > 0 {
			fmt.Printf("Trades/sec: %.2f\n", float64(stats.trades)/duration.Seconds())
		}
		stream.Stop()
		os.Exit(0)
	}()

	fmt.Println()
	fmt.Println("Streaming all channels... Press Ctrl+C to stop")
	fmt.Println("--------------------------------------------------")

	// Start the stream
	if err := stream.Start(); err != nil {
		fmt.Printf("Failed to start stream: %v\n", err)
		return
	}

	// Keep running until signal
	select {}
}
