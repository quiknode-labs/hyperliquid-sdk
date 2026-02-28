// WebSocket Streaming Example — Real-Time HyperCore Data
//
// Stream trades, orders, book updates, events, and TWAP via WebSocket.
// These are the data streams available on QuickNode endpoints.
//
// Available QuickNode WebSocket streams:
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
//	export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
//	go run main.go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/quiknode-labs/raptor/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		fmt.Println("WebSocket Streaming Example")
		fmt.Println(string(repeat('=', 60)))
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println(string(repeat('=', 60)))
	fmt.Println("WebSocket Trade Streaming")
	fmt.Println(string(repeat('=', 60)))

	tradeCount := 0

	stream := hyperliquid.NewStream(endpoint, &hyperliquid.StreamConfig{
		Reconnect: false,
		OnOpen: func() {
			fmt.Println("[CONNECTED]")
		},
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
	})

	stream.Trades([]string{"BTC", "ETH"}, func(data map[string]any) {
		// QuickNode format: {"type": "data", "stream": "hl.trades", "block": {"events": [...]}}
		// Events are [[user, trade_data], ...]
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
