// Fluent Builder Example — Create orders using the fluent builder API.
//
// This example matches the Python fluent_builder.py exactly.
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
		fmt.Println("Fluent Builder Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Fluent Builder Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Example 1: Limit buy with size
	fmt.Println("Example 1: Limit Buy with Size")
	fmt.Println("------------------------------")
	order1 := hyperliquid.Order().
		Buy("BTC").
		Size(0.001).  // 0.001 BTC (5 decimal precision)
		Price(60000). // Limit price
		GTC()         // Good Till Cancelled

	fmt.Printf("Order: %+v\n", order1)
	fmt.Println()

	// Example 2: Limit sell with notional
	fmt.Println("Example 2: Limit Sell with Notional")
	fmt.Println("------------------------------")
	order2 := hyperliquid.Order().
		Sell("ETH").
		Notional(100). // $100 worth
		Price(4000).   // Limit price
		IOC()          // Immediate or Cancel

	fmt.Printf("Order: %+v\n", order2)
	fmt.Println()

	// Example 3: Market buy
	fmt.Println("Example 3: Market Buy")
	fmt.Println("------------------------------")
	order3 := hyperliquid.Order().
		Buy("SOL").
		Notional(50). // $50 worth
		Market()      // Market order

	fmt.Printf("Order: %+v\n", order3)
	fmt.Println()

	// Example 4: Post-only order (ALO)
	fmt.Println("Example 4: Post-Only Order (ALO)")
	fmt.Println("------------------------------")
	order4 := hyperliquid.Order().
		Buy("BTC").
		Size(0.001).
		Price(60000).
		ALO() // Add Liquidity Only

	fmt.Printf("Order: %+v\n", order4)
	fmt.Println()

	// Example 5: Reduce-only order
	fmt.Println("Example 5: Reduce-Only Order")
	fmt.Println("------------------------------")
	order5 := hyperliquid.Order().
		Sell("BTC").
		Size(0.001).
		Price(70000).
		GTC().
		ReduceOnly() // Only closes existing position

	fmt.Printf("Order: %+v\n", order5)
	fmt.Println()

	// Place an order (uncomment to execute)
	// result, err := sdk.PlaceOrder(order1)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Order placed: OID=%d, Status=%s\n", result.OID, result.Status)
	// }

	fmt.Println("Note: Minimum order size is $10")
	fmt.Println("      BTC allows 5 decimal precision for size")
	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
