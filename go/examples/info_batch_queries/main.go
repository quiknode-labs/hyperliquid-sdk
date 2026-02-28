// Info Batch Queries Example — Query multiple users' account states.
//
// This example matches the Python info_batch_queries.py exactly.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Info Batch Queries Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Info Batch Queries Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}
	info := sdk.Info()

	// Example addresses to query
	addresses := []string{
		"0x0000000000000000000000000000000000000000",
		"0xB228634b61636ADF82501eD196Bec979B6aF4732",
	}

	for _, addr := range addresses {
		fmt.Println()
		fmt.Printf("User: %s\n", addr[:20]+"...")
		fmt.Println("------------------------------")

		// Query clearinghouse state (positions and margin)
		state, err := info.ClearinghouseState(addr)
		if err != nil {
			fmt.Printf("  Clearinghouse error: %v\n", err)
		} else {
			if marginSummary, ok := state["marginSummary"].(map[string]any); ok {
				if accountValue, ok := marginSummary["accountValue"].(string); ok {
					av, _ := strconv.ParseFloat(accountValue, 64)
					fmt.Printf("  Account Value: $%.2f\n", av)
				}
			}
		}

		// Query open orders
		orders, err := info.OpenOrders(addr)
		if err != nil {
			fmt.Printf("  Open orders error: %v\n", err)
		} else {
			fmt.Printf("  Open Orders: %d\n", len(orders))
		}

		// Query user fees
		fees, err := info.UserFees(addr)
		if err != nil {
			fmt.Printf("  Fees error: %v\n", err)
		} else {
			if activeReferralDiscount, ok := fees["activeReferralDiscount"].(string); ok {
				fmt.Printf("  Referral Discount: %s\n", activeReferralDiscount)
			}
		}
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
