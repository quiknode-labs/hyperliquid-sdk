//! L4 Orderbook Stream Example
//!
//! Stream full L4 orderbook with individual orders via gRPC.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example stream_l4_book
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint/TOKEN'");
        eprintln!("  cargo run --example stream_l4_book");
        std::process::exit(1);
    }

    println!("L4 Orderbook Stream Example");
    println!("{}", "=".repeat(50));

    println!("\n1. L4 Book Details:");
    println!("   - Shows individual orders at each price level");
    println!("   - Includes order IDs and sizes");
    println!("   - Higher bandwidth than L2");
    println!("   - Use for market making or detailed analysis");

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create gRPC stream via SDK for L4 (more efficient for full book)
    println!("\n2. Creating gRPC stream...");

    let update_count = Arc::new(AtomicUsize::new(0));
    let update_count_cb = update_count.clone();

    let mut grpc = sdk.grpc()
        .on_connect(|| {
            println!("   [Connected]");
        })
        .on_error(|e| {
            eprintln!("   [Error] {}", e);
        });

    // Subscribe to L4 book
    let _sub = grpc.l4_book("BTC", move |data| {
        let count = update_count_cb.fetch_add(1, Ordering::SeqCst) + 1;
        if count <= 20 {
            if let Some(orders) = data.get("orders").and_then(|v| v.as_array()) {
                println!("   [{}] {} orders in update", count, orders.len());
                for order in orders.iter().take(3) {
                    let px = order.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                    let sz = order.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                    let oid = order.get("oid").and_then(|v| v.as_u64()).unwrap_or(0);
                    println!("       {} @ {} (oid: {})", sz, px, oid);
                }
            }
        }
    });

    // Start streaming
    println!("3. Receiving order updates (10 seconds):");

    grpc.start()?;

    // Run for 10 seconds
    tokio::time::sleep(Duration::from_secs(10)).await;

    grpc.stop();

    let total = update_count.load(Ordering::SeqCst);
    println!("\n   Total updates: {}", total);

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
