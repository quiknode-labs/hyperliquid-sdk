//! Info API Vaults Example
//!
//! Query vault information and user vault positions.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example info_vaults
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  cargo run --example info_vaults");
        std::process::exit(1);
    }

    println!("Info API Vaults Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    let sdk = builder.build().await?;
    let info = sdk.info();

    // List all vaults
    println!("\n1. Vault Summaries:");
    match info.vault_summaries().await {
        Ok(vaults) => {
            if let Some(arr) = vaults.as_array() {
                println!("   Total vaults: {}", arr.len());
                for (i, vault) in arr.iter().take(5).enumerate() {
                    let name = vault.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    let tvl = vault.get("tvl").and_then(|v| v.as_str()).unwrap_or("?");
                    let addr = vault.get("vaultAddress").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {} (TVL: ${})", i + 1, name, tvl);
                    if addr.len() > 20 {
                        println!("       {}...", &addr[..20]);
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Leading vaults (requires a user address)
    println!("\n2. Leading Vaults:");
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_else(|| "0x0000000000000000000000000000000000000000".to_string());
    match info.leading_vaults(&address_str).await {
        Ok(vaults) => {
            if let Some(arr) = vaults.as_array() {
                for (i, vault) in arr.iter().take(3).enumerate() {
                    let name = vault.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                    let pnl = vault.get("pnl").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [{}] {}: PnL ${}", i + 1, name, pnl);
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Vault details (if we have a vault address)
    println!("\n3. Vault Details:");
    match info.vault_summaries().await {
        Ok(vaults) => {
            if let Some(arr) = vaults.as_array() {
                if let Some(first) = arr.first() {
                    let addr = first.get("vaultAddress").and_then(|v| v.as_str()).unwrap_or("");
                    if !addr.is_empty() {
                        match info.vault_details(addr, None).await {
                            Ok(details) => {
                                let name = details.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                                let tvl = details.get("tvl").and_then(|v| v.as_str()).unwrap_or("?");
                                println!("   Name: {}", name);
                                println!("   TVL: ${}", tvl);
                            }
                            Err(e) => println!("   Error: {}", e),
                        }
                    }
                }
            }
        }
        Err(_) => {}
    }

    // User vault positions (if private key provided)
    if !address_str.is_empty() && address_str != "0x0000000000000000000000000000000000000000" {
        println!("\n4. Your Vault Positions:");
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
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
