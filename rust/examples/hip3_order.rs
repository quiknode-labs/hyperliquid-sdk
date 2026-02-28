//! HIP-3 Market Order Example
//!
//! Trade on HIP-3 markets (community perps).
//! Uses "dex:SYMBOL" format.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example hip3_order
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
        eprintln!("  cargo run --example hip3_order");
        std::process::exit(1);
    }

    println!("HIP-3 Market Order Example");
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

    // List HIP-3 DEXes
    println!("\n1. Available HIP-3 DEXes:");
    match sdk.dexes().await {
        Ok(dexes) => {
            if let Some(arr) = dexes.get("perpDexs").and_then(|v| v.as_array()) {
                for (i, dex) in arr.iter().take(5).enumerate() {
                    let name = dex.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {}", i + 1, name);
                }
                if arr.len() > 5 {
                    println!("   ... and {} more", arr.len() - 5);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // HIP-3 market format
    println!("\n2. HIP-3 Market Format:");
    println!("   Use 'dex:SYMBOL' format");
    println!("   Example: 'xyz:SILVER', 'abc:GOLD'");

    // Trade on HIP-3 market
    println!("\n3. Trade on HIP-3 Market:");
    println!("   sdk.market_buy(\"xyz:SILVER\").notional(11.0)");

    // Uncomment to execute
    // match sdk.market_buy("xyz:SILVER").await.notional(11.0).await {
    //     Ok(order) => {
    //         println!("   Status: {}", order.status);
    //         println!("   OID: {:?}", order.oid);
    //     }
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "-".repeat(50));
    println!("HIP-3 Notes:");
    println!("  - Community-created perp markets");
    println!("  - Same API as regular markets");
    println!("  - Use 'dex:SYMBOL' format");
    println!("  - Check liquidity before trading");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
