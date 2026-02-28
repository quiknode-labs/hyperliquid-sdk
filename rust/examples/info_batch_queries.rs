//! Info API Batch Queries Example
//!
//! Efficiently fetch multiple pieces of data.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example info_batch_queries
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example info_batch_queries");
        std::process::exit(1);
    }

    println!("Info API Batch Queries Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;
    let info = sdk.info();

    // Get all mid prices
    println!("\n1. All Mid Prices:");
    match info.all_mids(None).await {
        Ok(mids) => {
            let assets = ["BTC", "ETH", "SOL", "DOGE", "ARB"];
            for asset in &assets {
                if let Some(mid) = mids.get(*asset).and_then(|v| v.as_str()) {
                    println!("   {}: ${}", asset, mid);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Get exchange metadata
    println!("\n2. Exchange Metadata:");
    match info.meta().await {
        Ok(meta) => {
            if let Some(universe) = meta.get("universe").and_then(|v| v.as_array()) {
                println!("   Perp markets: {}", universe.len());
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    match info.spot_meta().await {
        Ok(spot) => {
            if let Some(tokens) = spot.get("tokens").and_then(|v| v.as_array()) {
                println!("   Spot tokens: {}", tokens.len());
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Get L2 orderbook for multiple assets
    println!("\n3. Orderbook Spreads:");
    for asset in &["BTC", "ETH", "SOL"] {
        match info.l2_book(asset, None, None).await {
            Ok(book) => {
                if let Some(levels) = book.get("levels").and_then(|v| v.as_array()) {
                    if levels.len() >= 2 {
                        let bids = levels[0].as_array();
                        let asks = levels[1].as_array();
                        if let (Some(bids), Some(asks)) = (bids, asks) {
                            if let (Some(bid), Some(ask)) = (bids.first(), asks.first()) {
                                let bid_px = bid.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                                let ask_px = ask.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                                println!("   {}: bid={} ask={}", asset, bid_px, ask_px);
                            }
                        }
                    }
                }
            }
            Err(e) => println!("   {}: Error - {}", asset, e),
        }
    }

    // Get predicted funding rates
    println!("\n4. Predicted Funding Rates:");
    match info.predicted_fundings().await {
        Ok(fundings) => {
            if let Some(arr) = fundings.as_array() {
                for (i, funding) in arr.iter().take(5).enumerate() {
                    let asset = funding.get("asset").and_then(|v| v.as_str()).unwrap_or("?");
                    let rate = funding.get("predictedFunding").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {}: {}", i + 1, asset, rate);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
