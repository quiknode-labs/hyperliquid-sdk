//! Markets Example
//!
//! List all available markets and their metadata.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example markets
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example markets");
        std::process::exit(1);
    }

    println!("Markets Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;

    // Get all markets
    println!("\n1. Perpetual Markets:");
    let info = sdk.info();
    match info.meta().await {
        Ok(meta) => {
            if let Some(universe) = meta.get("universe").and_then(|v| v.as_array()) {
                println!("   Total: {} markets", universe.len());
                for (i, asset) in universe.iter().take(10).enumerate() {
                    let name = asset.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    let sz_decimals = asset.get("szDecimals").and_then(|v| v.as_u64()).unwrap_or(0);
                    let max_leverage = asset.get("maxLeverage").and_then(|v| v.as_u64()).unwrap_or(0);
                    println!("   [{}] {}: sz_decimals={}, max_leverage={}x",
                        i + 1, name, sz_decimals, max_leverage);
                }
                if universe.len() > 10 {
                    println!("   ... and {} more", universe.len() - 10);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Get spot markets
    println!("\n2. Spot Markets:");
    match info.spot_meta().await {
        Ok(spot) => {
            if let Some(universe) = spot.get("universe").and_then(|v| v.as_array()) {
                println!("   Total: {} markets", universe.len());
                for (i, market) in universe.iter().take(5).enumerate() {
                    let name = market.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {}", i + 1, name);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Get DEXes (HIP-3)
    println!("\n3. HIP-3 DEXes:");
    match sdk.dexes().await {
        Ok(dexes) => {
            if let Some(arr) = dexes.get("perpDexs").and_then(|v| v.as_array()) {
                println!("   Total: {} DEXes", arr.len());
                for (i, dex) in arr.iter().take(5).enumerate() {
                    let name = dex.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {}", i + 1, name);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
