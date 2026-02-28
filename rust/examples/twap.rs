//! TWAP Order Example
//!
//! Time-Weighted Average Price orders for large executions.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example twap
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
        eprintln!("  cargo run --example twap");
        std::process::exit(1);
    }

    println!("TWAP Order Example");
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

    // TWAP Buy Order
    println!("\n1. TWAP Buy Order:");
    println!("   Size: 0.1 BTC");
    println!("   Duration: 60 minutes");
    println!("   Randomize: true");

    match sdk.twap_order("BTC", 0.1, true, 60, true, false).await {
        Ok(result) => {
            println!("   Result: {:?}", result);
            if let Some(twap_id) = result.get("twapId").and_then(|v| v.as_i64()) {
                println!("   TWAP ID: {}", twap_id);
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // TWAP Sell Order
    println!("\n2. TWAP Sell Order:");
    println!("   Size: 0.1 BTC");
    println!("   Duration: 120 minutes");

    // match sdk.twap_order("BTC", false, 0.1, 120, true, false).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Cancel TWAP
    println!("\n3. Cancel TWAP:");
    println!("   sdk.twap_cancel(asset, twap_id)");
    // match sdk.twap_cancel("BTC", 123456).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "-".repeat(50));
    println!("TWAP Benefits:");
    println!("  - Minimize market impact for large orders");
    println!("  - Achieve better average price");
    println!("  - Avoid slippage on illiquid markets");
    println!("  - Automated execution over time");
    println!("  - Randomization prevents front-running");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
