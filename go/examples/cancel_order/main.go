// Cancel Order Example — Place an order and then cancel it.
//
// This example matches the Python cancel_order.py exactly.
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
		fmt.Println("Cancel Order Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Cancel Order Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Get current mid price
	fmt.Println("Getting BTC mid price...")
	mid, err := sdk.GetMid("BTC")
	if err != nil {
		fmt.Printf("Error getting mid: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("BTC mid: $%.2f\n", mid)

	// Calculate a price 3% below market (so it won't fill)
	price := int(mid * 0.97)
	fmt.Printf("Placing order at $%d (3%% below market)\n", price)
	fmt.Println()

	// Place a limit buy order
	fmt.Println("Placing limit buy order...")
	order, err := sdk.Buy("BTC",
		hyperliquid.WithNotional(11),
		hyperliquid.WithPrice(float64(price)),
		hyperliquid.WithTIF(hyperliquid.TIFGTC),
	)
	if err != nil {
		fmt.Printf("Error placing order: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Order placed: OID=%d, Status=%s\n", order.OID, order.Status)
	fmt.Println()

	// Cancel the order using the order object
	fmt.Println("Canceling order...")
	_, err = order.Cancel()
	if err != nil {
		fmt.Printf("Error canceling: %v\n", err)
	} else {
		fmt.Println("Order canceled successfully!")
	}
	fmt.Println()

	// Alternative: Cancel by OID directly
	// result, err := sdk.Cancel(order.OID, "BTC")
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Cancel result: %v\n", result)
	// }

	fmt.Println("==================================================")
	fmt.Println("Done!")
}
