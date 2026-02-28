//! Place Order Example
//!
//! Place limit orders using various methods.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example place_order
//! ```

use hyperliquid_sdk::{HyperliquidSDK, Order, TIF};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example place_order");
        std::process::exit(1);
    }

    println!("Place Order Example");
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

    // Method 1: Simple buy/sell
    println!("\n1. Simple Limit Buy (3% below mid):");
    let buy_price = mid * 0.97;
    match sdk.buy("BTC", 0.001, buy_price, TIF::Gtc).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            // Cancel the order
            if let Some(oid) = order.oid {
                let _ = sdk.cancel(oid, "BTC").await;
                println!("   (Cancelled)");
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Method 2: Fluent builder
    println!("\n2. Fluent Builder Order:");
    let order = Order::buy("BTC")
        .size(0.001)
        .price(buy_price)
        .gtc();

    match sdk.order(order).await {
        Ok(result) => {
            println!("   Status: {}", result.status);
            println!("   OID: {:?}", result.oid);
            // Cancel
            if let Some(oid) = result.oid {
                let _ = sdk.cancel(oid, "BTC").await;
                println!("   (Cancelled)");
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Method 3: IOC order
    println!("\n3. IOC Order (Immediate or Cancel):");
    let order = Order::buy("BTC")
        .size(0.001)
        .price(buy_price)
        .ioc();

    match sdk.order(order).await {
        Ok(result) => {
            println!("   Status: {}", result.status);
            println!("   Filled: {:?}", result.filled_size);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Method 4: Post-only (ALO)
    println!("\n4. Post-Only Order (Add Liquidity Only):");
    let order = Order::buy("BTC")
        .size(0.001)
        .price(buy_price)
        .alo();

    match sdk.order(order).await {
        Ok(result) => {
            println!("   Status: {}", result.status);
            println!("   OID: {:?}", result.oid);
            // Cancel
            if let Some(oid) = result.oid {
                let _ = sdk.cancel(oid, "BTC").await;
                println!("   (Cancelled)");
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
