// Modify Order Example — Place a limit order and modify its price.
//
// This example matches the Python modify_order.py exactly.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	privateKey := os.Getenv("PRIVATE_KEY")
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if privateKey == "" {
		fmt.Println("Modify Order Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Modify Order Example")
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

	// Calculate prices for order
	initialPrice := int(mid * 0.95)  // 5% below market
	modifiedPrice := int(mid * 0.96) // 4% below market
	fmt.Printf("Initial price: $%d (5%% below)\n", initialPrice)
	fmt.Printf("Modified price: $%d (4%% below)\n", modifiedPrice)
	fmt.Println()

	// Place initial order
	fmt.Println("Placing initial order...")
	order, err := sdk.Buy("BTC",
		hyperliquid.WithNotional(11),
		hyperliquid.WithPrice(float64(initialPrice)),
		hyperliquid.WithTIF(hyperliquid.TIFGTC),
	)
	if err != nil {
		fmt.Printf("Error placing order: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Order placed: OID=%d, Status=%s\n", order.OID, order.Status)
	fmt.Println()

	// Modify the order price
	fmt.Println("Modifying order price...")
	_, err = order.Modify(strconv.Itoa(modifiedPrice), "")
	if err != nil {
		fmt.Printf("Error modifying: %v\n", err)
	} else {
		fmt.Printf("Order modified to price $%d\n", modifiedPrice)
	}
	fmt.Println()

	// Cancel the order
	fmt.Println("Canceling order...")
	_, err = order.Cancel()
	if err != nil {
		fmt.Printf("Error canceling: %v\n", err)
	} else {
		fmt.Println("Order canceled!")
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
