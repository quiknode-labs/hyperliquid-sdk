// Isolated Margin Example — Managing margin in isolated positions.
//
// This example matches the Python isolated_margin.py exactly.
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
		fmt.Println("Isolated Margin Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Isolated Margin Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Add margin to isolated long position
	fmt.Println("Add Margin to Long Position:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.UpdateIsolatedMargin(\"BTC\", 100, true)  // Add $100 to long")
	fmt.Println()

	// Remove margin from isolated short position
	fmt.Println("Remove Margin from Short Position:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.UpdateIsolatedMargin(\"ETH\", -50, false)  // Remove $50 from short")
	fmt.Println()

	// Top up isolated-only margin
	fmt.Println("Top Up Isolated-Only Margin:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.TopUpIsolatedOnlyMargin(\"BTC\", 100)")
	fmt.Println()

	// Uncomment to execute (requires an isolated position):
	// result, err := sdk.UpdateIsolatedMargin("BTC", 100, true)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Result: %v\n", result)
	// }

	fmt.Println("Note: These operations require an existing isolated position.")
	fmt.Println("      Use sdk.UpdateLeverage() first to set isolated margin mode.")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
