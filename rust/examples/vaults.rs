//! Vaults Example
//!
//! Deposit and withdraw from Hyperliquid vaults.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example vaults
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
        eprintln!("  cargo run --example vaults");
        std::process::exit(1);
    }

    println!("Vaults Example");
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

    // List vaults
    println!("\n1. Available Vaults:");
    let info = sdk.info();
    match info.vault_summaries().await {
        Ok(vaults) => {
            if let Some(arr) = vaults.as_array() {
                println!("   Total: {} vaults", arr.len());
                for (i, vault) in arr.iter().take(5).enumerate() {
                    let name = vault.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    let tvl = vault.get("tvl").and_then(|v| v.as_str()).unwrap_or("?");
                    let addr = vault.get("vaultAddress").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {} (TVL: ${})", i + 1, name, tvl);
                    println!("       {}", addr);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Vault deposit
    println!("\n2. Vault Deposit:");
    println!("   sdk.vault_deposit(vault_address, amount)");
    // let vault_addr = "0xVaultAddress";
    // match sdk.vault_deposit(vault_addr, 100.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Vault withdraw
    println!("\n3. Vault Withdraw:");
    println!("   sdk.vault_withdraw(vault_address, shares)");
    // match sdk.vault_withdraw(vault_addr, 10.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // User vault positions
    println!("\n4. Your Vault Positions:");
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();
    match info.user_vault_equities(&address_str).await {
        Ok(positions) => {
            if let Some(arr) = positions.as_array() {
                if arr.is_empty() {
                    println!("   No vault positions");
                } else {
                    for pos in arr {
                        println!("   {:?}", pos);
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "-".repeat(50));
    println!("Vault Notes:");
    println!("  - Vaults are automated trading strategies");
    println!("  - Deposit USDC, receive vault shares");
    println!("  - APY varies based on strategy performance");
    println!("  - Check vault documentation before depositing");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
