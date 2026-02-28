// Transfers Example — Transfer funds between accounts and wallets.
//
// This example demonstrates:
// - Transferring USDC between spot and perp accounts
// - Internal transfers between Hyperliquid addresses
// - Checking balances before/after transfers
//
// This example matches the Python SDK patterns exactly.
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
		fmt.Println("Transfers Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Transfers Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Check current balances
	fmt.Println("Current Balances:")
	fmt.Println("------------------------------")
	state, err := sdk.Info().ClearinghouseState(sdk.Address())
	if err != nil {
		fmt.Printf("Error fetching state: %v\n", err)
	} else {
		// Extract balance info
		if marginSummary, ok := state["marginSummary"].(map[string]any); ok {
			if balance, ok := marginSummary["accountValue"].(string); ok {
				fmt.Printf("  Perp Account Value: $%s\n", balance)
			}
		}
		if crossMarginSummary, ok := state["crossMarginSummary"].(map[string]any); ok {
			if balance, ok := crossMarginSummary["accountValue"].(string); ok {
				fmt.Printf("  Cross Margin Value: $%s\n", balance)
			}
		}
	}
	fmt.Println()

	// Transfer from Spot to Perp (example - commented out)
	fmt.Println("Transfer Spot to Perp:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.SpotToPerp(100.0)  // Transfer $100 USDC")
	fmt.Println()

	// Uncomment to transfer:
	// result, err := sdk.SpotToPerp(100.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Transfer result: %v\n", result)
	// }

	// Transfer from Perp to Spot (example - commented out)
	fmt.Println("Transfer Perp to Spot:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.PerpToSpot(100.0)  // Transfer $100 USDC")
	fmt.Println()

	// Uncomment to transfer:
	// result, err := sdk.PerpToSpot(100.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Transfer result: %v\n", result)
	// }

	// Internal transfer to another address (example - commented out)
	fmt.Println("Internal Transfer:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.InternalTransfer(\"0xRecipient...\", 50.0)  // Send $50 USDC")
	fmt.Println()

	// Uncomment to transfer:
	// recipient := "0x1234567890abcdef1234567890abcdef12345678"
	// result, err := sdk.InternalTransfer(recipient, 50.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Transfer result: %v\n", result)
	// }

	// Transfer between sub-accounts (example - commented out)
	fmt.Println("Sub-Account Transfer:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.SubAccountTransfer(\"subaccount_name\", true, 25.0)")
	fmt.Println("  // true = to sub-account, false = from sub-account")
	fmt.Println()

	// Uncomment to transfer:
	// result, err := sdk.SubAccountTransfer("trading_bot", true, 25.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Transfer result: %v\n", result)
	// }

	fmt.Println("Notes:")
	fmt.Println("  - All transfers are atomic and instant")
	fmt.Println("  - Internal transfers require destination to be on Hyperliquid")
	fmt.Println("  - For external withdrawals, use the Withdraw example")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
