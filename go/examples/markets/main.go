// Markets Example — List all available markets and DEXes.
//
// This example matches the Python markets.py exactly.
package main

import (
	"fmt"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	// This is a read-only example, doesn't require endpoint
	fmt.Println("Markets Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	// Get all markets
	fmt.Println()
	fmt.Println("Available Markets")
	fmt.Println("------------------------------")
	markets, err := sdk.Markets()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Perp markets: %d\n", len(markets.Perps))
		fmt.Printf("Spot markets: %d\n", len(markets.Spot))

		fmt.Println()
		fmt.Println("Sample Perp Markets:")
		for i, m := range markets.Perps {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(markets.Perps)-5)
				break
			}
			fmt.Printf("  %s (size decimals: %d)\n", m.Name, m.SzDecimals)
		}

		fmt.Println()
		fmt.Println("Sample Spot Markets:")
		for i, m := range markets.Spot {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(markets.Spot)-5)
				break
			}
			fmt.Printf("  %s\n", m.Name)
		}
	}

	// Get HIP-3 DEXes
	fmt.Println()
	fmt.Println("HIP-3 DEXes")
	fmt.Println("------------------------------")
	dexesResp, err := sdk.Dexes()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		dexes, _ := dexesResp["dexes"].([]any)
		fmt.Printf("Total DEXes: %d\n", len(dexes))
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
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
