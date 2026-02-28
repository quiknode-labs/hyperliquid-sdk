//! Schedule Cancel Example
//!
//! Dead-man's switch: auto-cancel orders if not refreshed.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example schedule_cancel
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::time::{SystemTime, UNIX_EPOCH};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example schedule_cancel");
        std::process::exit(1);
    }

    println!("Schedule Cancel Example");
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

    // Calculate cancel time (60 seconds from now)
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as u64;
    let cancel_time = now + 60_000;  // 60 seconds

    // Schedule cancel
    println!("\n1. Schedule Cancel in 60 seconds:");
    println!("   Cancel time: {}", cancel_time);

    match sdk.schedule_cancel(Some(cancel_time)).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    // Refresh the schedule (heartbeat pattern)
    println!("\n2. Refresh Schedule (heartbeat):");
    let new_cancel_time = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as u64 + 120_000;  // 120 seconds

    match sdk.schedule_cancel(Some(new_cancel_time)).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    // Cancel the schedule (disable dead-man's switch)
    println!("\n3. Cancel Schedule (disable):");
    match sdk.schedule_cancel(None).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "-".repeat(50));
    println!("Use Cases:");
    println!("  - Bot safety: Cancel orders if bot stops");
    println!("  - Network issues: Cancel if connection drops");
    println!("  - Risk management: Time-limited order validity");
    println!("  - Automated strategies: Clean up stale orders");

    println!("\nHeartbeat Pattern:");
    println!("  1. Set initial schedule (e.g., 60s)");
    println!("  2. Periodically refresh (e.g., every 30s)");
    println!("  3. If refresh stops, orders auto-cancel");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
