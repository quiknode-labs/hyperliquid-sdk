//! Info API Candles Example
//!
//! Fetch historical candlestick (OHLCV) data.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example info_candles
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::time::{SystemTime, UNIX_EPOCH};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example info_candles");
        std::process::exit(1);
    }

    println!("Info API Candles Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;
    let info = sdk.info();

    // Time range: last 24 hours
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as u64;
    let start_time = now - (24 * 60 * 60 * 1000);

    // Fetch 1-hour candles
    println!("\n1. BTC 1-Hour Candles (last 24h):");
    match info.candles("BTC", "1h", start_time, Some(now)).await {
        Ok(candles) => {
            if let Some(arr) = candles.as_array() {
                println!("   Received {} candles", arr.len());
                for (i, candle) in arr.iter().take(5).enumerate() {
                    let t = candle.get("t").and_then(|v| v.as_i64()).unwrap_or(0);
                    let o = candle.get("o").and_then(|v| v.as_str()).unwrap_or("?");
                    let h = candle.get("h").and_then(|v| v.as_str()).unwrap_or("?");
                    let l = candle.get("l").and_then(|v| v.as_str()).unwrap_or("?");
                    let c = candle.get("c").and_then(|v| v.as_str()).unwrap_or("?");
                    let v = candle.get("v").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] t={} O={} H={} L={} C={} V={}", i + 1, t, o, h, l, c, v);
                }
                if arr.len() > 5 {
                    println!("   ... and {} more", arr.len() - 5);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Fetch 15-minute candles (last 4 hours)
    println!("\n2. ETH 15-Minute Candles (last 4h):");
    let start_4h = now - (4 * 60 * 60 * 1000);
    match info.candles("ETH", "15m", start_4h, Some(now)).await {
        Ok(candles) => {
            if let Some(arr) = candles.as_array() {
                println!("   Received {} candles", arr.len());
                for (i, candle) in arr.iter().take(5).enumerate() {
                    let c = candle.get("c").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] close={}", i + 1, c);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Available intervals
    println!("\n3. Available Intervals:");
    let intervals = ["1m", "5m", "15m", "30m", "1h", "4h", "1d"];
    for interval in &intervals {
        println!("   - {}", interval);
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
