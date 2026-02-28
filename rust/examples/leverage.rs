//! Leverage Example
//!
//! Set and manage leverage for trading.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example leverage
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example leverage");
        std::process::exit(1);
    }

    println!("Leverage Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    if let Some(pk) = &private_key {
        builder = builder.private_key(pk);
    }
    let sdk = builder.build().await?;

    if let Some(addr) = sdk.address() {
        println!("Address: {}", addr);
    }

    // Get max leverage for BTC
    println!("\n1. BTC Market Info:");
    let info = sdk.info();
    match info.meta().await {
        Ok(meta) => {
            if let Some(universe) = meta.get("universe").and_then(|v| v.as_array()) {
                for asset in universe {
                    if asset.get("name").and_then(|v| v.as_str()) == Some("BTC") {
                        let max_leverage = asset.get("maxLeverage").and_then(|v| v.as_u64()).unwrap_or(0);
                        println!("   Max Leverage: {}x", max_leverage);
                        break;
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Set leverage (cross margin)
    println!("\n2. Set BTC Leverage to 10x (Cross):");
    match sdk.update_leverage("BTC", 10, true).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    // Set leverage (isolated margin)
    println!("\n3. Set BTC Leverage to 5x (Isolated):");
    match sdk.update_leverage("BTC", 5, false).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "-".repeat(50));
    println!("Leverage Tips:");
    println!("  - Higher leverage = higher risk");
    println!("  - Cross margin: leverage shared across positions");
    println!("  - Isolated margin: leverage per position");
    println!("  - Check max leverage per asset");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
