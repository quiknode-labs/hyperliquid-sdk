// Cancel By CLOID Example — Cancel orders using client-provided order IDs.
//
// This example matches the Python cancel_by_cloid.py exactly.
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
		fmt.Println("Cancel By CLOID Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Cancel By CLOID Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// CLOIDs are hex strings you provide when placing orders
	// Example: Place an order with a custom CLOID
	fmt.Println("To place an order with a CLOID:")
	fmt.Println("  order := hyperliquid.Order().Buy(\"BTC\").Size(0.001).Price(60000).GTC().CLOID(\"0x1234...\")")
	fmt.Println("  result, err := sdk.PlaceOrder(order)")
	fmt.Println()

	// Cancel by CLOID
	fmt.Println("To cancel by CLOID:")
	fmt.Println("  result, err := sdk.CancelByCLOID(\"0x1234...\", \"BTC\")")
	fmt.Println()

	// Example cancellation (uncomment with your actual CLOID)
	// cloid := "0x1234567890abcdef1234567890abcdef"
	// result, err := sdk.CancelByCLOID(cloid, "BTC")
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Cancel result: %v\n", result)
	// }

	fmt.Println("CLOIDs are useful for:")
	fmt.Println("  - Tracking orders by your own custom identifiers")
	fmt.Println("  - Canceling without needing to store exchange OIDs")
	fmt.Println("  - Idempotent order operations")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
