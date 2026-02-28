// Info Vaults Example — Query vault information and user delegations.
//
// This example matches the Python info_vaults.py exactly.
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

	if endpoint == "" {
		fmt.Println("Info Vaults Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  export USER_ADDRESS='0x...'  # Optional, for delegation info")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Info Vaults Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}
	info := sdk.Info()

	// Get vault summaries
	fmt.Println()
	fmt.Println("Vault Summaries")
	fmt.Println("------------------------------")
	vaults, err := info.VaultSummaries()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Total vaults: %d\n", len(vaults))
		for i, vault := range vaults {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(vaults)-5)
				break
			}
			v := vault.(map[string]any)
			name, _ := v["name"].(string)
			tvl := "?"
			if summary, ok := v["summary"].(map[string]any); ok {
				if tvlStr, ok := summary["tvl"].(string); ok {
					tvlFloat, _ := strconv.ParseFloat(tvlStr, 64)
					tvl = fmt.Sprintf("$%.2f", tvlFloat)
				}
			}
			fmt.Printf("  %s - TVL: %s\n", name, tvl)
		}
	}

	// Get user delegations (if user address provided)
	if userAddress != "" {
		fmt.Println()
		fmt.Println("User Delegations")
		fmt.Println("------------------------------")
		fmt.Printf("User: %s\n", userAddress)

		delegations, err := info.Delegations(userAddress)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Delegations: %d\n", len(delegations))
			for i, del := range delegations {
				if i >= 3 {
					fmt.Printf("  ... and %d more\n", len(delegations)-3)
					break
				}
				d := del.(map[string]any)
				validator, _ := d["validator"].(string)
				amount, _ := d["amount"].(string)
				fmt.Printf("  Validator: %s, Amount: %s\n", validator[:20]+"...", amount)
			}
		}
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
