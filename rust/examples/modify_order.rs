//! Modify Order Example
//!
//! Modify an existing order's price or size.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example modify_order
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
        eprintln!("  cargo run --example modify_order");
        std::process::exit(1);
    }

    println!("Modify Order Example");
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

    // Place an order first
    println!("\n1. Placing initial order...");
    let order = Order::buy("BTC")
        .size(0.001)
        .price(mid * 0.95)  // 5% below mid
        .gtc();

    let placed = sdk.order(order).await?;
    println!("   Status: {}", placed.status);
    println!("   OID: {:?}", placed.oid);

    if let Some(oid) = placed.oid {
        // Modify the order - improve price
        println!("\n2. Modifying order (better price)...");
        match sdk.modify(
            oid,
            "BTC",
            true,           // is_buy
            0.001,          // size
            mid * 0.96,     // new price (closer to mid)
            TIF::Gtc,
            false,          // reduce_only
            None,           // cloid
        ).await {
            Ok(modified) => {
                println!("   Status: {}", modified.status);
                println!("   New OID: {:?}", modified.oid);
            }
            Err(e) => println!("   Error: {}", e),
        }

        // Clean up
        println!("\n3. Cancelling order...");
        match sdk.cancel(oid, "BTC").await {
            Ok(_) => println!("   Cancelled"),
            Err(e) => println!("   Error: {}", e),
        }
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
