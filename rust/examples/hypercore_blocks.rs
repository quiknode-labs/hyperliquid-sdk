//! HyperCore Block Data Example
//!
//! Fetch block data from HyperCore using native API.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example hypercore_blocks
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example hypercore_blocks");
        std::process::exit(1);
    }

    println!("HyperCore Block Data Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;
    let core = sdk.core();

    // Get latest block number
    println!("\n1. Latest Block Number:");
    match core.latest_block_number(None).await {
        Ok(height) => {
            println!("   Height: {}", height);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Get latest blocks
    println!("\n2. Latest Blocks (last 5):");
    match core.latest_blocks(None, Some(5)).await {
        Ok(blocks) => {
            if let Some(arr) = blocks.as_array() {
                for (i, block) in arr.iter().enumerate() {
                    let height = block.get("height").and_then(|v| v.as_u64()).unwrap_or(0);
                    let hash = block.get("hash").and_then(|v| v.as_str()).unwrap_or("?");
                    let display_hash = if hash.len() > 16 {
                        format!("{}...", &hash[..16])
                    } else {
                        hash.to_string()
                    };
                    println!("   [{}] Block {}: {}", i + 1, height, display_hash);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Get block transactions
    println!("\n3. Block Transaction Info:");
    match core.latest_blocks(None, Some(1)).await {
        Ok(blocks) => {
            if let Some(arr) = blocks.as_array() {
                if let Some(block) = arr.first() {
                    let height = block.get("height").and_then(|v| v.as_u64()).unwrap_or(0);
                    let txs = block.get("transactions").and_then(|v| v.as_array()).map(|a| a.len()).unwrap_or(0);
                    let time = block.get("timestamp").and_then(|v| v.as_u64()).unwrap_or(0);
                    println!("   Block: {}", height);
                    println!("   Transactions: {}", txs);
                    println!("   Timestamp: {}", time);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
