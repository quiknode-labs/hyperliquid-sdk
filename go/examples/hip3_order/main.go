// HIP-3 Order Example — Trading on HIP-3 markets (community perpetuals).
//
// This example matches the Python hip3_order.py exactly.
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
		fmt.Println("HIP-3 Order Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("HIP-3 Order Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// List available HIP-3 DEXes
	fmt.Println("Available HIP-3 DEXes:")
	fmt.Println("------------------------------")
	dexesResp, err := sdk.Dexes()
	if err != nil {
		fmt.Printf("Error fetching DEXes: %v\n", err)
	} else {
		dexes, _ := dexesResp["dexes"].([]any)
		count := 0
		for _, dex := range dexes {
			if count >= 5 {
				fmt.Printf("  ... and %d more\n", len(dexes)-5)
				break
			}
			if d, ok := dex.(map[string]any); ok {
				name, _ := d["name"].(string)
				fmt.Printf("  %s\n", name)
			}
			count++
		}
	}
	fmt.Println()

	// HIP-3 markets use "dex:SYMBOL" format
	fmt.Println("Trading on HIP-3 Markets:")
	fmt.Println("------------------------------")
	fmt.Println("HIP-3 markets use 'dex:SYMBOL' format, e.g.:")
	fmt.Println("  - xyz:SILVER (Hypersea Silver)")
	fmt.Println("  - xyz:GOLD (Hypersea Gold)")
	fmt.Println()

	// Example: Buy on a HIP-3 market (uncomment to execute)
	// order, err := sdk.Buy("xyz:SILVER",
	//     hyperliquid.WithNotional(11),
	//     hyperliquid.WithTIF(hyperliquid.TIFIOC),
	// )
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Order placed: OID=%d, Status=%s\n", order.OID, order.Status)
	// }

	fmt.Println("To place a HIP-3 order:")
	fmt.Println("  order, err := sdk.Buy(\"xyz:SILVER\", hyperliquid.WithNotional(11))")
	fmt.Println()
	fmt.Println("The API for HIP-3 markets is the same as regular markets.")
	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
