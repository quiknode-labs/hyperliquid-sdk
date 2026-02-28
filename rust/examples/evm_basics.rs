//! HyperEVM Basics Example
//!
//! Shows how to use standard Ethereum JSON-RPC calls on Hyperliquid's EVM chain.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example evm_basics
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example evm_basics");
        std::process::exit(1);
    }

    println!("HyperEVM Basics Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;
    let evm = sdk.evm();

    // Chain ID
    println!("\n1. Chain Info:");
    match evm.chain_id().await {
        Ok(chain_id) => println!("   Chain ID: {}", chain_id),
        Err(e) => println!("   Error: {}", e),
    }

    // Block number
    match evm.block_number().await {
        Ok(block) => println!("   Block Number: {}", block),
        Err(e) => println!("   Error: {}", e),
    }

    // Gas price
    match evm.gas_price().await {
        Ok(gas) => println!("   Gas Price: {} wei ({:.2} gwei)", gas, gas as f64 / 1e9),
        Err(e) => println!("   Error: {}", e),
    }

    // Latest block
    println!("\n2. Latest Block:");
    match evm.get_block_by_number("latest", false).await {
        Ok(block) => {
            let hash = block.get("hash").and_then(|v| v.as_str()).unwrap_or("?");
            let number = block.get("number").and_then(|v| v.as_str()).unwrap_or("?");
            let txs = block.get("transactions").and_then(|v| v.as_array()).map(|a| a.len()).unwrap_or(0);
            if hash.len() > 20 {
                println!("   Hash: {}...", &hash[..20]);
            }
            println!("   Number: {}", number);
            println!("   Txs: {}", txs);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Balance check
    println!("\n3. Balance Check:");
    let addr = "0x0000000000000000000000000000000000000000";
    match evm.get_balance(addr, Some("latest")).await {
        Ok(balance) => {
            // Parse hex string to u128
            let balance_num = u128::from_str_radix(balance.trim_start_matches("0x"), 16).unwrap_or(0);
            let eth = balance_num as f64 / 1e18;
            println!("   {}...: {:.6} ETH", &addr[..12], eth);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Transaction count
    println!("\n4. Transaction Count:");
    match evm.get_transaction_count(addr, Some("latest")).await {
        Ok(count) => println!("   Nonce: {}", count),
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
