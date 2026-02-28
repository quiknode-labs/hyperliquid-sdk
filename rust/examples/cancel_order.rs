//! Cancel Order Example
//!
//! Cancel a specific order by its OID.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example cancel_order
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
        eprintln!("  cargo run --example cancel_order");
        std::process::exit(1);
    }

    println!("Cancel Order Example");
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

    // Get open orders
    println!("\n1. Current Open Orders:");
    let orders = sdk.open_orders().await?;

    if let Some(arr) = orders.as_array() {
        println!("   {} open orders", arr.len());

        for order in arr.iter().take(5) {
            let coin = order.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
            let oid = order.get("oid").and_then(|v| v.as_u64()).unwrap_or(0);
            let side = order.get("side").and_then(|v| v.as_str()).unwrap_or("?");
            let sz = order.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
            let px = order.get("limitPx").and_then(|v| v.as_str()).unwrap_or("?");
            println!("   OID {} - {} {} {} @ {}", oid, coin, side, sz, px);
        }

        // Cancel first order if exists
        if let Some(first_order) = arr.first() {
            let oid = first_order.get("oid").and_then(|v| v.as_u64()).unwrap_or(0);
            let coin = first_order.get("coin").and_then(|v| v.as_str()).unwrap_or("BTC");

            if oid > 0 {
                println!("\n2. Cancelling Order {}:", oid);
                match sdk.cancel(oid, coin).await {
                    Ok(result) => println!("   Result: {:?}", result),
                    Err(e) => println!("   Error: {}", e),
                }
            }
        } else {
            println!("\n   No orders to cancel");
        }
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
