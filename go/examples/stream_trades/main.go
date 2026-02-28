// WebSocket Streaming Example — Real-Time HyperCore Data
//
// Stream trades, orders, book updates, events, and TWAP via WebSocket.
//
// Available WebSocket streams:
// - trades: Executed trades with price, size, direction
// - orders: Order lifecycle events (open, filled, cancelled)
// - book_updates: Order book changes (incremental deltas)
// - events: Balance changes, transfers, deposits, withdrawals
// - twap: TWAP execution data
// - writer_actions: HyperCore <-> HyperEVM asset transfers
//
// Note: L2/L4 order book snapshots are available via gRPC (see stream_orderbook example).
//
// Usage:
//
//	export ENDPOINT="https://your-endpoint/YOUR_TOKEN"
//	go run main.go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		fmt.Println("WebSocket Streaming Example")
		fmt.Println(string(repeat('=', 60)))
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println(string(repeat('=', 60)))
	fmt.Println("WebSocket Trade Streaming")
	fmt.Println(string(repeat('=', 60)))

	// Create SDK
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	tradeCount := 0

	stream := sdk.NewStream(&hyperliquid.StreamConfig{
		Reconnect: false,
		OnOpen: func() {
			fmt.Println("[CONNECTED]")
		},
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
	})

	stream.Trades([]string{"BTC", "ETH"}, func(data map[string]any) {
		block, ok := data["block"].(map[string]any)
		if !ok {
			return
		}
		events, ok := block["events"].([]any)
		if !ok {
			return
		}

		for _, event := range events {
			eventArr, ok := event.([]any)
			if !ok || len(eventArr) < 2 {
				continue
			}
			t, ok := eventArr[1].(map[string]any)
			if !ok {
				continue
			}

			tradeCount++
			coin := t["coin"]
			px := t["px"]
			sz := t["sz"]
			side := "BUY "
			if t["side"] == "A" {
				side = "SELL"
			}
			fmt.Printf("[%s] %s %v %v @ $%v\n", timestamp(), side, sz, coin, px)

			if tradeCount >= 10 {
				fmt.Printf("\nReceived %d trades.\n", tradeCount)
				return
			}
		}
	})

	fmt.Println("\nSubscribing to BTC and ETH trades...")
	fmt.Println(string(repeat('-', 60)))

	if err := stream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start := time.Now()
	for tradeCount < 10 && time.Since(start) < 30*time.Second {
		time.Sleep(100 * time.Millisecond)
	}

	stream.Stop()

	fmt.Println("\n" + string(repeat('=', 60)))
	fmt.Println("Done!")
}

func timestamp() string {
	return time.Now().Format("15:04:05.000")
}

func repeat(b byte, n int) []byte {
	result := make([]byte, n)
	for i := range result {
		result[i] = b
	}
	return result
}
