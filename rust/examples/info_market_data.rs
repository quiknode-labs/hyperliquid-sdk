//! Info API Market Data Example
//!
//! Fetch comprehensive market data: prices, orderbooks, funding rates.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example info_market_data
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example info_market_data");
        std::process::exit(1);
    }

    println!("Info API Market Data Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;
    let info = sdk.info();

    // All mid prices
    println!("\n1. Mid Prices:");
    match info.all_mids(None).await {
        Ok(mids) => {
            let count = mids.as_object().map(|o| o.len()).unwrap_or(0);
            println!("   Total assets: {}", count);

            // Show a few
            let assets = ["BTC", "ETH", "SOL", "DOGE"];
            for asset in &assets {
                if let Some(mid) = mids.get(*asset).and_then(|v| v.as_str()) {
                    println!("   {}: ${}", asset, mid);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Single mid price
    println!("\n2. Single Asset Price:");
    match info.get_mid("BTC").await {
        Ok(mid) => println!("   BTC mid: ${:.2}", mid),
        Err(e) => println!("   Error: {}", e),
    }

    // L2 Orderbook
    println!("\n3. L2 Orderbook (BTC):");
    match info.l2_book("BTC", None, None).await {
        Ok(book) => {
            if let Some(levels) = book.get("levels").and_then(|v| v.as_array()) {
                if levels.len() >= 2 {
                    println!("   Bids:");
                    if let Some(bids) = levels[0].as_array() {
                        for bid in bids.iter().take(3) {
                            let px = bid.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let sz = bid.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("      {} @ ${}", sz, px);
                        }
                    }
                    println!("   Asks:");
                    if let Some(asks) = levels[1].as_array() {
                        for ask in asks.iter().take(3) {
                            let px = ask.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let sz = ask.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("      {} @ ${}", sz, px);
                        }
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Predicted funding rates
    println!("\n4. Predicted Funding Rates:");
    match info.predicted_fundings().await {
        Ok(fundings) => {
            if let Some(arr) = fundings.as_array() {
                let btc_funding = arr.iter().find(|f| {
                    f.get("asset").and_then(|v| v.as_str()) == Some("BTC")
                });
                if let Some(btc) = btc_funding {
                    let rate = btc.get("predictedFunding").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   BTC: {} (hourly)", rate);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Recent trades
    println!("\n5. Recent Trades (BTC):");
    match info.recent_trades("BTC").await {
        Ok(trades) => {
            if let Some(arr) = trades.as_array() {
                println!("   Last {} trades:", arr.len().min(5));
                for trade in arr.iter().take(5) {
                    let side = trade.get("side").and_then(|v| v.as_str()).unwrap_or("?");
                    let side_str = if side == "B" { "BUY" } else { "SELL" };
                    let sz = trade.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                    let px = trade.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("      {} {} @ ${}", side_str, sz, px);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
