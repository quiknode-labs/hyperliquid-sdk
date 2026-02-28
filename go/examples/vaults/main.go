// Vaults Example — Hyperliquid vault operations.
//
// This example demonstrates:
// - Listing available vaults
// - Getting vault details and performance
// - Depositing into vaults
// - Withdrawing from vaults
// - Checking your vault positions
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
		fmt.Println("Vaults Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Vaults Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// List all vaults
	fmt.Println("1. List Vaults:")
	fmt.Println("------------------------------")
	vaults, err := sdk.Info().VaultSummaries()
	if err != nil {
		fmt.Printf("Error fetching vaults: %v\n", err)
	} else {
		fmt.Printf("Total vaults: %d\n", len(vaults))
		for i, vault := range vaults {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(vaults)-5)
				break
			}
			if v, ok := vault.(map[string]any); ok {
				name, _ := v["name"].(string)
				address, _ := v["vaultAddress"].(string)
				tvl, _ := v["tvl"].(string)
				if len(address) > 20 {
					fmt.Printf("  %s: $%s TVL (%s...)\n", name, tvl, address[:20])
				} else {
					fmt.Printf("  %s: $%s TVL\n", name, tvl)
				}
			}
		}
	}
	fmt.Println()

	// Get vault details
	fmt.Println("2. Get Vault Details:")
	fmt.Println("------------------------------")
	fmt.Println("  details, err := sdk.Info().VaultDetails(vaultAddress)")
	fmt.Println()

	// Example vault address (replace with actual)
	// vaultAddress := "0x1234..."
	// details, err := sdk.Info().VaultDetails(vaultAddress)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Vault details: %v\n", details)
	// }

	// Get user's vault positions
	fmt.Println("3. Your Vault Positions:")
	fmt.Println("------------------------------")
	equities, err := sdk.Info().UserVaultEquities(sdk.Address())
	if err != nil {
		fmt.Printf("Error fetching positions: %v\n", err)
	} else {
		if len(equities) == 0 {
			fmt.Println("  No vault positions")
		} else {
			for vaultAddr, equity := range equities {
				if e, ok := equity.(map[string]any); ok {
					value, _ := e["equity"].(string)
					if len(vaultAddr) > 20 {
						fmt.Printf("  Vault %s...: $%s\n", vaultAddr[:20], value)
					} else {
						fmt.Printf("  Vault %s: $%s\n", vaultAddr, value)
					}
				}
			}
		}
	}
	fmt.Println()

	// Deposit into vault
	fmt.Println("4. Deposit into Vault:")
	fmt.Println("------------------------------")
	fmt.Println("  result, err := sdk.VaultDeposit(vaultAddress, 100.0)")
	fmt.Println()

	// Uncomment to deposit:
	// vaultAddress := "0x1234..."
	// result, err := sdk.VaultDeposit(vaultAddress, 100.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Deposit result: %v\n", result)
	// }

	// Withdraw from vault
	fmt.Println("5. Withdraw from Vault:")
	fmt.Println("------------------------------")
	fmt.Println("  result, err := sdk.VaultWithdraw(vaultAddress, 50.0)")
	fmt.Println()

	// Uncomment to withdraw:
	// result, err := sdk.VaultWithdraw(vaultAddress, 50.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Withdraw result: %v\n", result)
	// }

	// Create a vault
	fmt.Println("6. Create a Vault:")
	fmt.Println("------------------------------")
	fmt.Println("  config := hyperliquid.VaultConfig{")
	fmt.Println("      Name: \"My Trading Vault\",")
	fmt.Println("      Description: \"Automated trading strategy\",")
	fmt.Println("      PerformanceFee: 0.2,  // 20%")
	fmt.Println("  }")
	fmt.Println("  result, err := sdk.CreateVault(config)")
	fmt.Println()

	fmt.Println("Notes:")
	fmt.Println("  - Vaults are managed trading strategies")
	fmt.Println("  - Depositors share in vault performance")
	fmt.Println("  - Vault managers earn performance fees")
	fmt.Println("  - Withdrawals may have a lock-up period")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
