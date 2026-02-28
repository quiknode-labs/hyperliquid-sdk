//! Staking Example
//!
//! Stake and unstake HYPE tokens.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example staking
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
        eprintln!("  cargo run --example staking");
        std::process::exit(1);
    }

    println!("Staking Example");
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

    // Check delegations
    println!("\n1. Current Delegations:");
    let info = sdk.info();
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();

    match info.delegations(&address_str).await {
        Ok(delegations) => {
            if let Some(arr) = delegations.as_array() {
                if arr.is_empty() {
                    println!("   No delegations");
                } else {
                    for del in arr {
                        println!("   {:?}", del);
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Stake HYPE
    println!("\n2. Stake HYPE:");
    println!("   sdk.stake(amount_tokens)");
    // match sdk.stake(100.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Unstake HYPE
    println!("\n3. Unstake HYPE:");
    println!("   sdk.unstake(amount_tokens)");
    // match sdk.unstake(50.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Delegate to validator
    println!("\n4. Delegate to Validator:");
    println!("   sdk.delegate(validator_address, amount_tokens)");
    // match sdk.delegate("0xValidatorAddress", 100.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Undelegate from validator
    println!("\n5. Undelegate from Validator:");
    println!("   sdk.undelegate(validator_address, amount_tokens)");
    // match sdk.undelegate("0xValidatorAddress", 50.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "-".repeat(50));
    println!("Staking Notes:");
    println!("  - Staking earns HYPE rewards");
    println!("  - Unstaking has a cooldown period");
    println!("  - Choose validators carefully");
    println!("  - Higher stake = more network security");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
