//! HyperEVM API Example — Interact with Hyperliquid's EVM chain.
//!
//! This example shows how to query the Hyperliquid EVM chain (chain ID 999 mainnet, 998 testnet).
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example evm_example
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").unwrap_or_else(|_| {
        eprintln!("Error: Set ENDPOINT environment variable");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        std::process::exit(1);
    });

    println!("Hyperliquid EVM API Example");
    println!("{}", "=".repeat(50));
    println!("Endpoint: {}...", &endpoint[..endpoint.len().min(50)]);
    println!();

    // Create SDK and get EVM client
    let sdk = HyperliquidSDK::new().endpoint(&endpoint).build().await?;
    let evm = sdk.evm();

    // ══════════════════════════════════════════════════════════════════════════
    // Chain Info
    // ══════════════════════════════════════════════════════════════════════════
    println!("Chain Info");
    println!("{}", "-".repeat(30));

    // Get chain ID
    let chain_id = evm.chain_id().await?;
    let network = match chain_id {
        999 => "Mainnet",
        998 => "Testnet",
        _ => "Unknown",
    };
    println!("Chain ID: {}", chain_id);
    println!("Network: {}", network);

    // Get latest block number
    let block_num = evm.block_number().await?;
    println!("Latest block: {}", block_num);

    // Get gas price
    let gas_price = evm.gas_price().await?;
    let gas_gwei = gas_price as f64 / 1e9;
    println!("Gas price: {:.2} Gwei", gas_gwei);
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Account Balance
    // ══════════════════════════════════════════════════════════════════════════
    println!("Account Balance");
    println!("{}", "-".repeat(30));

    // Example address - replace with your address
    let address = "0x0000000000000000000000000000000000000000";

    let balance_hex = evm.get_balance(address, None).await?;
    // Parse hex balance to u128
    let balance_wei = u128::from_str_radix(balance_hex.trim_start_matches("0x"), 16).unwrap_or(0);
    let balance_eth = balance_wei as f64 / 1e18;
    println!("Address: {}", address);
    println!("Balance: {:.6} HYPE", balance_eth);
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Block Data
    // ══════════════════════════════════════════════════════════════════════════
    println!("Block Data");
    println!("{}", "-".repeat(30));

    // Get latest block
    let block_hex = format!("0x{:x}", block_num);
    let block = evm.get_block_by_number(&block_hex, false).await?;
    if !block.is_null() {
        println!("Block {}:", block_num);
        if let Some(hash) = block.get("hash").and_then(|h| h.as_str()) {
            println!("  Hash: {}...", &hash[..hash.len().min(20)]);
        }
        if let Some(parent) = block.get("parentHash").and_then(|p| p.as_str()) {
            println!("  Parent: {}...", &parent[..parent.len().min(20)]);
        }
        if let Some(ts) = block.get("timestamp").and_then(|t| t.as_str()) {
            println!("  Timestamp: {}", ts);
        }
        if let Some(gas_used) = block.get("gasUsed").and_then(|g| g.as_str()) {
            let gas = u64::from_str_radix(gas_used.trim_start_matches("0x"), 16).unwrap_or(0);
            println!("  Gas Used: {}", gas);
        }
        if let Some(txs) = block.get("transactions").and_then(|t| t.as_array()) {
            println!("  Transactions: {}", txs.len());
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Transaction Count
    // ══════════════════════════════════════════════════════════════════════════
    println!("Transaction Count");
    println!("{}", "-".repeat(30));

    let tx_count = evm.get_transaction_count(address, None).await?;
    println!("Nonce for {}...: {}", &address[..10], tx_count);
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Smart Contract Call (Example: ERC20 balanceOf)
    // ══════════════════════════════════════════════════════════════════════════
    println!("Smart Contract Call");
    println!("{}", "-".repeat(30));

    // Example: Read a contract (this is just a demonstration)
    // In real usage, you'd use actual contract addresses and proper ABI encoding
    println!("  (Contract call example would go here)");
    println!("  Use evm.call() with proper contract address and data");
    println!();

    println!("{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
