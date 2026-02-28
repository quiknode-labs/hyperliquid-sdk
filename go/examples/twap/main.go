// TWAP Example — Time-Weighted Average Price order execution.
//
// This example demonstrates:
// - Placing TWAP orders for gradual execution
// - Configuring TWAP parameters (duration, slices)
// - Monitoring TWAP progress
// - Cancelling active TWAP orders
//
// This example matches the Python SDK patterns exactly.
package main

import (
	"fmt"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	privateKey := os.Getenv("PRIVATE_KEY")
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if privateKey == "" {
		fmt.Println("TWAP Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("TWAP Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Basic TWAP Order
	fmt.Println("1. Basic TWAP Order:")
	fmt.Println("------------------------------")
	fmt.Println("  Execute $10,000 BTC buy over 1 hour")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TWAPOrder().")
	fmt.Println("      Buy(\"BTC\").")
	fmt.Println("      Notional(10000).")
	fmt.Println("      Duration(60)  // 60 minutes")
	fmt.Println()
	fmt.Println("  result, err := sdk.PlaceTWAP(order)")
	fmt.Println()

	// Uncomment to place TWAP:
	// order := hyperliquid.TWAPOrder().Buy("BTC").Notional(10000).Duration(60)
	// result, err := sdk.PlaceTWAP(order)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("TWAP placed: %v\n", result)
	// }

	// TWAP with size instead of notional
	fmt.Println("2. TWAP with Size:")
	fmt.Println("------------------------------")
	fmt.Println("  Buy 1 BTC over 2 hours")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TWAPOrder().")
	fmt.Println("      Buy(\"BTC\").")
	fmt.Println("      Size(1.0).")
	fmt.Println("      Duration(120)")
	fmt.Println()

	// TWAP Sell Order
	fmt.Println("3. TWAP Sell Order:")
	fmt.Println("------------------------------")
	fmt.Println("  Sell 0.5 ETH over 30 minutes")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TWAPOrder().")
	fmt.Println("      Sell(\"ETH\").")
	fmt.Println("      Size(0.5).")
	fmt.Println("      Duration(30)")
	fmt.Println()

	// TWAP with randomization
	fmt.Println("4. TWAP with Randomization:")
	fmt.Println("------------------------------")
	fmt.Println("  Add randomization to slice timing")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TWAPOrder().")
	fmt.Println("      Buy(\"BTC\").")
	fmt.Println("      Notional(5000).")
	fmt.Println("      Duration(60).")
	fmt.Println("      Randomize(true)")
	fmt.Println()

	// Check active TWAP orders - would use WebSocket streaming
	fmt.Println("Checking Active TWAP Orders:")
	fmt.Println("------------------------------")
	fmt.Println("  Use WebSocket stream.TWAPStates(user) to monitor active TWAPs")
	fmt.Println("  TWAP progress is reported in real-time via the stream")
	fmt.Println()

	// Cancel TWAP
	fmt.Println("5. Cancel TWAP:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.CancelTWAP(twapId)")
	fmt.Println()

	// Uncomment to cancel:
	// twapId := 12345
	// result, err := sdk.CancelTWAP(twapId)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("TWAP cancelled: %v\n", result)
	// }

	fmt.Println("Notes:")
	fmt.Println("  - TWAP splits orders into slices over the duration")
	fmt.Println("  - Each slice executes as a market order")
	fmt.Println("  - Slippage protection is built-in")
	fmt.Println("  - Can be cancelled at any time (unfilled portion)")
	fmt.Println("  - Progress can be monitored via ActiveTWAPs()")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
