//! Cancel All Orders Example
//!
//! Cancel all open orders, optionally filtered by asset.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example cancel_all
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
        eprintln!("  cargo run --example cancel_all");
        std::process::exit(1);
    }

    println!("Cancel All Orders Example");
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

    // Check open orders first
    println!("\n1. Current Open Orders:");
    match sdk.open_orders().await {
        Ok(orders) => {
            if let Some(arr) = orders.as_array() {
                println!("   {} open orders", arr.len());
                for (i, order) in arr.iter().take(5).enumerate() {
                    let coin = order.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                    let side = order.get("side").and_then(|v| v.as_str()).unwrap_or("?");
                    let sz = order.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                    let px = order.get("limitPx").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {} {} {} @ {}", i + 1, coin, side, sz, px);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Cancel all orders
    println!("\n2. Cancelling All Orders:");
    match sdk.cancel_all(None).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    // Cancel all BTC orders only
    // println!("\n3. Cancelling All BTC Orders:");
    // match sdk.cancel_all(Some("BTC")).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
