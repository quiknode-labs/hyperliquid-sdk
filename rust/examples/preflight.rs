//! Preflight Validation Example
//!
//! Validate orders before sending to the exchange.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example preflight
//! ```

use hyperliquid_sdk::{HyperliquidSDK, Side};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example preflight");
        std::process::exit(1);
    }

    println!("Preflight Validation Example");
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

    // Get current price
    let mid = sdk.get_mid("BTC").await?;
    println!("\nBTC mid price: ${:.2}", mid);

    // Valid order
    println!("\n1. Valid Order Check:");
    let limit_price = mid * 0.97;
    match sdk.preflight("BTC", Side::Buy, limit_price, 0.001).await {
        Ok(result) => {
            let valid = result.get("valid").and_then(|v| v.as_bool()).unwrap_or(false);
            println!("   Valid: {}", valid);
            if let Some(errors) = result.get("errors").and_then(|v| v.as_array()) {
                if !errors.is_empty() {
                    println!("   Errors: {:?}", errors);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Invalid size (too small)
    println!("\n2. Invalid Size Check (too small):");
    match sdk.preflight("BTC", Side::Buy, limit_price, 0.0000001).await {
        Ok(result) => {
            let valid = result.get("valid").and_then(|v| v.as_bool()).unwrap_or(false);
            println!("   Valid: {}", valid);
            if let Some(errors) = result.get("errors").and_then(|v| v.as_array()) {
                if !errors.is_empty() {
                    println!("   Errors: {:?}", errors);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Invalid price (negative)
    println!("\n3. Invalid Price Check (negative):");
    match sdk.preflight("BTC", Side::Buy, -1000.0, 0.001).await {
        Ok(result) => {
            let valid = result.get("valid").and_then(|v| v.as_bool()).unwrap_or(false);
            println!("   Valid: {}", valid);
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "-".repeat(50));
    println!("Preflight Benefits:");
    println!("  - Catch errors before sending to exchange");
    println!("  - Validate size and price formatting");
    println!("  - Check margin requirements");
    println!("  - No gas cost for validation");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
