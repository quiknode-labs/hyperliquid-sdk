// HyperEVM Example
//
// Shows how to use standard Ethereum JSON-RPC calls on Hyperliquid's EVM chain.
//
// Usage:
//
//	export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
//	go run main.go
package main

import (
	"fmt"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		fmt.Println("Set ENDPOINT environment variable")
		os.Exit(1)
	}

	// Single SDK instance — access everything through sdk.Info, sdk.Core, sdk.EVM, etc.
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Error creating SDK: %v\n", err)
		os.Exit(1)
	}
	evm := sdk.EVM()

	fmt.Println(string(repeat('=', 50)))
	fmt.Println("HyperEVM (Ethereum JSON-RPC)")
	fmt.Println(string(repeat('=', 50)))

	// Chain info
	fmt.Println("\n1. Chain Info:")
	chainID, err := evm.ChainID()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Chain ID: %d\n", chainID)
	}

	blockNum, err := evm.BlockNumber()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Block: %d\n", blockNum)
	}

	gasPrice, err := evm.GasPrice()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Gas Price: %.2f gwei\n", float64(gasPrice)/1e9)
	}

	// Latest block
	fmt.Println("\n2. Latest Block:")
	block, err := evm.GetBlockByNumber("latest", false)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else if block != nil {
		hash, _ := block["hash"].(string)
		if len(hash) > 20 {
			hash = hash[:20]
		}
		fmt.Printf("   Hash: %s...\n", hash)
		txs, _ := block["transactions"].([]any)
		fmt.Printf("   Txs: %d\n", len(txs))
	}

	// Check balance
	fmt.Println("\n3. Balance Check:")
	addr := "0x0000000000000000000000000000000000000000"
	balance, err := evm.GetBalance(addr, "latest")
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   %s...: %s wei\n", addr[:12], balance)
	}

	fmt.Println("\n" + string(repeat('=', 50)))
	fmt.Println("Done!")
	fmt.Println("\nFor debug/trace APIs, use: NewEVM(endpoint, &EVMOptions{Debug: true})")
}

func repeat(b byte, n int) []byte {
	result := make([]byte, n)
	for i := range result {
		result[i] = b
	}
	return result
}
