// Place Order Example — Place a limit order that rests on the orderbook.
//
// This example matches the Python place_order.py exactly.
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
		fmt.Println("Place Order Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Place Order Example")
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
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("BTC mid: $%.2f\n", mid)

	// Calculate a price below market so it rests
	price := int(mid * 0.95) // 5% below market
	fmt.Printf("Order price: $%d (5%% below market)\n", price)
	fmt.Println()

	// Place limit buy order
	fmt.Println("Placing limit buy order...")
	order, err := sdk.Buy("BTC",
		hyperliquid.WithNotional(11), // $11 worth
		hyperliquid.WithPrice(float64(price)),
		hyperliquid.WithTIF(hyperliquid.TIFGTC), // Good Till Cancelled
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Order placed!\n")
	fmt.Printf("  OID: %d\n", order.OID)
	fmt.Printf("  Status: %s\n", order.Status)
	fmt.Printf("  Filled Size: %s\n", order.FilledSize)
	fmt.Println()

	// Cancel the order
	fmt.Println("Canceling order...")
	_, err = order.Cancel()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Order canceled!")
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
