// HyperEVM API Example — Interact with Hyperliquid's EVM chain.
//
// This example shows how to query the Hyperliquid EVM chain (chain ID 999 mainnet, 998 testnet).
// This example matches the Python evm_example.py exactly.
package main

import (
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("QUICKNODE_ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Error: Set QUICKNODE_ENDPOINT environment variable")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
		os.Exit(1)
	}

	fmt.Println("Hyperliquid EVM API Example")
	fmt.Println("==================================================")
	fmt.Printf("Endpoint: %s...\n", truncate(endpoint, 50))
	fmt.Println()

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	// Create EVM client
	evm := sdk.EVM()

	// ==========================================================================
	// Chain Info
	// ==========================================================================
	fmt.Println("Chain Info")
	fmt.Println("------------------------------")

	// Get chain ID
	chainID, err := evm.ChainID()
	if err != nil {
		log.Printf("Failed to get chain ID: %v", err)
	} else {
		fmt.Printf("Chain ID: %d\n", chainID)
		network := "Unknown"
		if chainID == 999 {
			network = "Mainnet"
		} else if chainID == 998 {
			network = "Testnet"
		}
		fmt.Printf("Network: %s\n", network)
	}

	// Get latest block number
	blockNum, err := evm.BlockNumber()
	if err != nil {
		log.Printf("Failed to get block number: %v", err)
	} else {
		fmt.Printf("Latest block: %d\n", blockNum)
	}

	// Get gas price
	gasPrice, err := evm.GasPrice()
	if err != nil {
		log.Printf("Failed to get gas price: %v", err)
	} else {
		gasGwei := float64(gasPrice) / 1e9
		fmt.Printf("Gas price: %.2f Gwei\n", gasGwei)
	}
	fmt.Println()

	// ==========================================================================
	// Account Balance
	// ==========================================================================
	fmt.Println("Account Balance")
	fmt.Println("------------------------------")

	// Example address - replace with your address
	address := "0x0000000000000000000000000000000000000000"

	balanceStr, err := evm.GetBalance(address, "latest")
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
	} else {
		// Parse hex balance to big.Int
		balanceWei := new(big.Int)
		balanceWei.SetString(balanceStr[2:], 16) // Remove "0x" prefix
		balanceEth := new(big.Float).Quo(new(big.Float).SetInt(balanceWei), big.NewFloat(1e18))
		fmt.Printf("Address: %s\n", address)
		fmt.Printf("Balance: %s HYPE\n", balanceEth.Text('f', 6))
	}
	fmt.Println()

	// ==========================================================================
	// Block Data
	// ==========================================================================
	fmt.Println("Block Data")
	fmt.Println("------------------------------")

	// Get latest block
	block, err := evm.GetBlockByNumber(fmt.Sprintf("0x%x", blockNum), false)
	if err != nil {
		log.Printf("Failed to get block: %v", err)
	} else if block != nil {
		fmt.Printf("Block %d:\n", blockNum)
		hash, _ := block["hash"].(string)
		if len(hash) > 20 {
			hash = hash[:20]
		}
		fmt.Printf("  Hash: %s...\n", hash)

		parentHash, _ := block["parentHash"].(string)
		if len(parentHash) > 20 {
			parentHash = parentHash[:20]
		}
		fmt.Printf("  Parent: %s...\n", parentHash)
		fmt.Printf("  Timestamp: %v\n", block["timestamp"])

		gasUsed, _ := block["gasUsed"].(string)
		gasUsedInt := parseHex(gasUsed)
		fmt.Printf("  Gas Used: %d\n", gasUsedInt)

		txs, _ := block["transactions"].([]any)
		fmt.Printf("  Transactions: %d\n", len(txs))
	}
	fmt.Println()

	// ==========================================================================
	// Transaction Count
	// ==========================================================================
	fmt.Println("Transaction Count")
	fmt.Println("------------------------------")

	txCount, err := evm.GetNonce(address, "latest")
	if err != nil {
		log.Printf("Failed to get nonce: %v", err)
	} else {
		fmt.Printf("Nonce for %s...: %d\n", address[:10], txCount)
	}
	fmt.Println()

	// ==========================================================================
	// Smart Contract Call (Example: ERC20 balanceOf)
	// ==========================================================================
	fmt.Println("Smart Contract Call")
	fmt.Println("------------------------------")

	// Example: Read a contract (this is just a demonstration)
	// In real usage, you'd use actual contract addresses and proper ABI encoding
	fmt.Println("  (Contract call example would go here)")
	fmt.Println("  Use evm.Call() with proper contract address and data")
	fmt.Println()

	fmt.Println("==================================================")
	fmt.Println("Done!")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func parseHex(s string) int64 {
	if len(s) < 2 {
		return 0
	}
	if s[:2] == "0x" {
		s = s[2:]
	}
	val, _ := new(big.Int).SetString(s, 16)
	if val == nil {
		return 0
	}
	return val.Int64()
}
