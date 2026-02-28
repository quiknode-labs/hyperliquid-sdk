// Info User Data Example — Query user positions, orders, fees, and balances.
//
// This example matches the Python info_user_data.py exactly.
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
	userAddress := os.Getenv("USER_ADDRESS")
	if userAddress == "" {
		userAddress = "0xB228634b61636ADF82501eD196Bec979B6aF4732"
	}

	if endpoint == "" {
		fmt.Println("Info User Data Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  export USER_ADDRESS='0x...'  # Optional")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Info User Data Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}
	info := sdk.Info()

	fmt.Printf("Querying data for: %s\n", userAddress)

	// Clearinghouse state (positions and margin)
	fmt.Println()
	fmt.Println("Clearinghouse State")
	fmt.Println("------------------------------")
	state, err := info.ClearinghouseState(userAddress)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		if marginSummary, ok := state["marginSummary"].(map[string]any); ok {
			if accountValue, ok := marginSummary["accountValue"].(string); ok {
				av, _ := strconv.ParseFloat(accountValue, 64)
				fmt.Printf("Account Value: $%.2f\n", av)
			}
			if totalMarginUsed, ok := marginSummary["totalMarginUsed"].(string); ok {
				tm, _ := strconv.ParseFloat(totalMarginUsed, 64)
				fmt.Printf("Margin Used: $%.2f\n", tm)
			}
		}

		// Positions
		if positions, ok := state["assetPositions"].([]any); ok {
			fmt.Printf("Positions: %d\n", len(positions))
			for i, pos := range positions {
				if i >= 3 {
					fmt.Printf("  ... and %d more\n", len(positions)-3)
					break
				}
				p := pos.(map[string]any)
				position, _ := p["position"].(map[string]any)
				coin, _ := position["coin"].(string)
				szi, _ := position["szi"].(string)
				entryPx, _ := position["entryPx"].(string)
				szFloat, _ := strconv.ParseFloat(szi, 64)
				pxFloat, _ := strconv.ParseFloat(entryPx, 64)
				fmt.Printf("  %s: %.4f @ $%.2f\n", coin, szFloat, pxFloat)
			}
		}
	}

	// Open orders
	fmt.Println()
	fmt.Println("Open Orders")
	fmt.Println("------------------------------")
	orders, err := info.OpenOrders(userAddress)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Open orders: %d\n", len(orders))
		for i, order := range orders {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(orders)-3)
				break
			}
			o := order.(map[string]any)
			coin, _ := o["coin"].(string)
			side, _ := o["side"].(string)
			sz, _ := o["sz"].(string)
			limitPx, _ := o["limitPx"].(string)
			sideStr := "SELL"
			if side == "B" {
				sideStr = "BUY"
			}
			fmt.Printf("  %s %s %s @ %s\n", coin, sideStr, sz, limitPx)
		}
	}

	// User fees
	fmt.Println()
	fmt.Println("User Fees")
	fmt.Println("------------------------------")
	fees, err := info.UserFees(userAddress)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Fee structure: %v\n", fees)
	}

	// Spot clearinghouse state
	fmt.Println()
	fmt.Println("Spot Balances")
	fmt.Println("------------------------------")
	spotState, err := info.SpotClearinghouseState(userAddress)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		if balances, ok := spotState["balances"].([]any); ok {
			fmt.Printf("Spot balances: %d\n", len(balances))
			for i, bal := range balances {
				if i >= 3 {
					break
				}
				b := bal.(map[string]any)
				coin, _ := b["coin"].(string)
				total, _ := b["total"].(string)
				fmt.Printf("  %s: %s\n", coin, total)
			}
		}
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
