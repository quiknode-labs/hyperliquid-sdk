//! WebSocket All Channels Example
//!
//! Subscribe to multiple WebSocket channels simultaneously.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example stream_websocket_all
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint/TOKEN'");
        eprintln!("  cargo run --example stream_websocket_all");
        std::process::exit(1);
    }

    println!("WebSocket All Channels Example");
    println!("{}", "=".repeat(50));

    // Available channels
    println!("\n1. Available Channels:");
    println!("   - l2Book: L2 orderbook updates");
    println!("   - trades: Trade executions");
    println!("   - orderUpdates: Order updates (requires auth)");
    println!("   - fills: Fill notifications (requires auth)");
    println!("   - allMids: All mid prices");
    println!("   - candle: Candlestick data");

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create stream via SDK
    println!("\n2. Creating WebSocket stream...");

    let channel_counts: Arc<Mutex<HashMap<String, usize>>> = Arc::new(Mutex::new(HashMap::new()));
    let total_count = Arc::new(std::sync::atomic::AtomicUsize::new(0));

    let counts_l2 = channel_counts.clone();
    let total_l2 = total_count.clone();

    let counts_trades = channel_counts.clone();
    let total_trades = total_count.clone();

    let counts_mids = channel_counts.clone();
    let total_mids = total_count.clone();

    let mut stream = sdk.stream()
        .on_open(|| {
            println!("   [Connected]");
        })
        .on_error(|e| {
            eprintln!("   [Error] {}", e);
        });

    // Subscribe to multiple channels
    println!("\n3. Subscribing to channels...");

    // L2 Book
    let _h1 = stream.l2_book("BTC", move |_data| {
        let mut counts = counts_l2.lock().unwrap();
        *counts.entry("l2Book".to_string()).or_insert(0) += 1;
        let t = total_l2.fetch_add(1, std::sync::atomic::Ordering::SeqCst) + 1;
        if t <= 10 {
            println!("   [{}] l2Book", t);
        }
    });
    println!("   Subscribed: l2Book BTC");

    // Trades
    let _h2 = stream.trades(&["BTC"], move |_data| {
        let mut counts = counts_trades.lock().unwrap();
        *counts.entry("trades".to_string()).or_insert(0) += 1;
        let t = total_trades.fetch_add(1, std::sync::atomic::Ordering::SeqCst) + 1;
        if t <= 10 {
            println!("   [{}] trades", t);
        }
    });
    println!("   Subscribed: trades BTC");

    // All mids
    let _h3 = stream.all_mids(move |_data| {
        let mut counts = counts_mids.lock().unwrap();
        *counts.entry("allMids".to_string()).or_insert(0) += 1;
        let t = total_mids.fetch_add(1, std::sync::atomic::Ordering::SeqCst) + 1;
        if t <= 10 {
            println!("   [{}] allMids", t);
        }
    });
    println!("   Subscribed: allMids");

    // Start streaming
    println!("\n4. Receiving messages (30 seconds):");

    stream.start()?;

    // Run for 30 seconds
    tokio::time::sleep(Duration::from_secs(30)).await;

    stream.stop();

    // Summary
    let total = total_count.load(std::sync::atomic::Ordering::SeqCst);
    let counts = channel_counts.lock().unwrap();

    println!("\n5. Message Summary:");
    for (channel, count) in counts.iter() {
        println!("   {}: {} messages", channel, count);
    }
    println!("   Total: {} messages", total);

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
