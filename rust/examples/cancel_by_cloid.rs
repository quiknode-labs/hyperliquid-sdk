//! Cancel by Client Order ID Example
//!
//! Cancel an order using its client order ID (cloid).
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example cancel_by_cloid
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
        eprintln!("  cargo run --example cancel_by_cloid");
        std::process::exit(1);
    }

    println!("Cancel by Client Order ID Example");
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

    // Check open orders with cloid
    println!("\n1. Open Orders (checking for cloid):");
    match sdk.open_orders().await {
        Ok(orders) => {
            if let Some(arr) = orders.as_array() {
                println!("   {} open orders", arr.len());
                for order in arr.iter().take(5) {
                    let coin = order.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                    let oid = order.get("oid").and_then(|v| v.as_u64()).unwrap_or(0);
                    let cloid = order.get("cloid").and_then(|v| v.as_str()).unwrap_or("none");
                    println!("   {} OID={} CLOID={}", coin, oid, cloid);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Cancel by cloid example (using a specific cloid)
    // Replace with an actual cloid from your orders
    let example_cloid = "0x0123456789abcdef0123456789abcdef";
    println!("\n2. Cancel by CLOID:");
    println!("   Attempting to cancel cloid: {}", example_cloid);

    match sdk.cancel_by_cloid(example_cloid, "BTC").await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error (expected if no matching order): {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
