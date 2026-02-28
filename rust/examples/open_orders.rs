//! Open Orders Example
//!
//! Query and display open orders.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example open_orders
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
        eprintln!("  cargo run --example open_orders");
        std::process::exit(1);
    }

    println!("Open Orders Example");
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
    println!("\nOpen Orders:");
    println!("{}", "-".repeat(50));

    match sdk.open_orders().await {
        Ok(orders) => {
            if let Some(arr) = orders.as_array() {
                if arr.is_empty() {
                    println!("No open orders");
                } else {
                    println!("Total: {} orders\n", arr.len());

                    for order in arr {
                        let oid = order.get("oid").and_then(|v| v.as_u64()).unwrap_or(0);
                        let coin = order.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                        let side = order.get("side").and_then(|v| v.as_str()).unwrap_or("?");
                        let side_str = if side == "B" { "BUY" } else { "SELL" };
                        let sz = order.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                        let px = order.get("limitPx").and_then(|v| v.as_str()).unwrap_or("?");
                        let order_type = order.get("orderType").and_then(|v| v.as_str()).unwrap_or("?");
                        let cloid = order.get("cloid").and_then(|v| v.as_str()).unwrap_or("");

                        println!("[OID {}] {} {} {} @ {} ({})",
                            oid, coin, side_str, sz, px, order_type);
                        if !cloid.is_empty() {
                            println!("         cloid: {}", cloid);
                        }
                    }
                }
            }
        }
        Err(e) => println!("Error: {}", e),
    }

    // Get order status for specific order (if any)
    println!("\n{}", "-".repeat(50));
    println!("Order Status Query:");

    // Example: query a specific OID
    let example_oid: u64 = 12345;  // Replace with real OID
    match sdk.order_status(example_oid, None).await {
        Ok(status) => println!("   Order {}: {:?}", example_oid, status),
        Err(e) => println!("   Order {} not found: {}", example_oid, e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
