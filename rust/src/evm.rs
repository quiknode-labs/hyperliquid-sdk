//! EVM (HyperEVM) client for Hyperliquid.
//!
//! Provides Ethereum JSON-RPC compatibility for the Hyperliquid EVM.

use serde_json::{json, Value};
use std::sync::Arc;

use crate::client::HyperliquidSDKInner;
use crate::error::Result;

/// EVM API client (Ethereum JSON-RPC)
pub struct EVM {
    inner: Arc<HyperliquidSDKInner>,
    debug: bool,
}

impl EVM {
    pub(crate) fn new(inner: Arc<HyperliquidSDKInner>) -> Self {
        Self {
            inner,
            debug: false,
        }
    }

    /// Enable debug/trace methods (mainnet only)
    pub fn with_debug(mut self, debug: bool) -> Self {
        self.debug = debug;
        self
    }

    /// Get the EVM endpoint URL
    fn evm_url(&self) -> String {
        self.inner.evm_url(self.debug)
    }

    /// Make a JSON-RPC request
    async fn rpc(&self, method: &str, params: Value) -> Result<Value> {
        let url = self.evm_url();

        let body = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1,
        });

        let response = self
            .inner
            .http_client
            .post(&url)
            .json(&body)
            .send()
            .await?;

        let status = response.status();
        let text = response.text().await?;

        if !status.is_success() {
            return Err(crate::Error::NetworkError(format!(
                "EVM request failed {}: {}",
                status, text
            )));
        }

        let result: Value = serde_json::from_str(&text)?;

        // Check for JSON-RPC error
        if let Some(error) = result.get("error") {
            let message = error
                .get("message")
                .and_then(|m| m.as_str())
                .unwrap_or("Unknown error");
            return Err(crate::Error::ApiError {
                code: crate::error::ErrorCode::Unknown,
                message: message.to_string(),
                guidance: "Check your EVM request parameters.".to_string(),
                raw: Some(error.to_string()),
            });
        }

        Ok(result.get("result").cloned().unwrap_or(Value::Null))
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Chain Info
    // ──────────────────────────────────────────────────────────────────────────

    /// Get the current block number
    pub async fn block_number(&self) -> Result<u64> {
        let result = self.rpc("eth_blockNumber", json!([])).await?;
        parse_hex_u64(&result)
    }

    /// Get the chain ID
    pub async fn chain_id(&self) -> Result<u64> {
        let result = self.rpc("eth_chainId", json!([])).await?;
        parse_hex_u64(&result)
    }

    /// Check if the node is syncing
    pub async fn syncing(&self) -> Result<Value> {
        self.rpc("eth_syncing", json!([])).await
    }

    /// Get current gas price
    pub async fn gas_price(&self) -> Result<u64> {
        let result = self.rpc("eth_gasPrice", json!([])).await?;
        parse_hex_u64(&result)
    }

    /// Get network version
    pub async fn net_version(&self) -> Result<String> {
        let result = self.rpc("net_version", json!([])).await?;
        Ok(result.as_str().unwrap_or("").to_string())
    }

    /// Get client version
    pub async fn web3_client_version(&self) -> Result<String> {
        let result = self.rpc("web3_clientVersion", json!([])).await?;
        Ok(result.as_str().unwrap_or("").to_string())
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Account
    // ──────────────────────────────────────────────────────────────────────────

    /// Get balance of an address
    pub async fn get_balance(&self, address: &str, block: Option<&str>) -> Result<String> {
        let block = block.unwrap_or("latest");
        let result = self.rpc("eth_getBalance", json!([address, block])).await?;
        Ok(result.as_str().unwrap_or("0x0").to_string())
    }

    /// Get transaction count (nonce) of an address
    pub async fn get_transaction_count(&self, address: &str, block: Option<&str>) -> Result<u64> {
        let block = block.unwrap_or("latest");
        let result = self
            .rpc("eth_getTransactionCount", json!([address, block]))
            .await?;
        parse_hex_u64(&result)
    }

    /// Get code at an address
    pub async fn get_code(&self, address: &str, block: Option<&str>) -> Result<String> {
        let block = block.unwrap_or("latest");
        let result = self.rpc("eth_getCode", json!([address, block])).await?;
        Ok(result.as_str().unwrap_or("0x").to_string())
    }

    /// Get storage at a position
    pub async fn get_storage_at(
        &self,
        address: &str,
        position: &str,
        block: Option<&str>,
    ) -> Result<String> {
        let block = block.unwrap_or("latest");
        let result = self
            .rpc("eth_getStorageAt", json!([address, position, block]))
            .await?;
        Ok(result.as_str().unwrap_or("0x0").to_string())
    }

    /// Get list of accounts (usually empty for remote nodes)
    pub async fn accounts(&self) -> Result<Vec<String>> {
        let result = self.rpc("eth_accounts", json!([])).await?;
        Ok(result
            .as_array()
            .map(|arr| {
                arr.iter()
                    .filter_map(|v| v.as_str().map(|s| s.to_string()))
                    .collect()
            })
            .unwrap_or_default())
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Transactions
    // ──────────────────────────────────────────────────────────────────────────

    /// Call a contract (read-only)
    pub async fn call(&self, tx: &Value, block: Option<&str>) -> Result<String> {
        let block = block.unwrap_or("latest");
        let result = self.rpc("eth_call", json!([tx, block])).await?;
        Ok(result.as_str().unwrap_or("0x").to_string())
    }

    /// Estimate gas for a transaction
    pub async fn estimate_gas(&self, tx: &Value) -> Result<u64> {
        let result = self.rpc("eth_estimateGas", json!([tx])).await?;
        parse_hex_u64(&result)
    }

    /// Send a raw (signed) transaction
    pub async fn send_raw_transaction(&self, signed_tx: &str) -> Result<String> {
        let result = self.rpc("eth_sendRawTransaction", json!([signed_tx])).await?;
        Ok(result.as_str().unwrap_or("").to_string())
    }

    /// Get transaction by hash
    pub async fn get_transaction_by_hash(&self, tx_hash: &str) -> Result<Value> {
        self.rpc("eth_getTransactionByHash", json!([tx_hash])).await
    }

    /// Get transaction receipt
    pub async fn get_transaction_receipt(&self, tx_hash: &str) -> Result<Value> {
        self.rpc("eth_getTransactionReceipt", json!([tx_hash])).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Blocks
    // ──────────────────────────────────────────────────────────────────────────

    /// Get block by number
    pub async fn get_block_by_number(
        &self,
        block_number: &str,
        full_transactions: bool,
    ) -> Result<Value> {
        self.rpc(
            "eth_getBlockByNumber",
            json!([block_number, full_transactions]),
        )
        .await
    }

    /// Get block by hash
    pub async fn get_block_by_hash(&self, block_hash: &str, full_transactions: bool) -> Result<Value> {
        self.rpc(
            "eth_getBlockByHash",
            json!([block_hash, full_transactions]),
        )
        .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Logs
    // ──────────────────────────────────────────────────────────────────────────

    /// Get logs matching a filter
    pub async fn get_logs(&self, filter: &Value) -> Result<Value> {
        self.rpc("eth_getLogs", json!([filter])).await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Fees
    // ──────────────────────────────────────────────────────────────────────────

    /// Get fee history
    pub async fn fee_history(
        &self,
        block_count: u64,
        newest_block: &str,
        reward_percentiles: Option<&[f64]>,
    ) -> Result<Value> {
        let percentiles = reward_percentiles.unwrap_or(&[]);
        self.rpc(
            "eth_feeHistory",
            json!([format!("0x{:x}", block_count), newest_block, percentiles]),
        )
        .await
    }

    /// Get max priority fee per gas
    pub async fn max_priority_fee_per_gas(&self) -> Result<u64> {
        let result = self.rpc("eth_maxPriorityFeePerGas", json!([])).await?;
        parse_hex_u64(&result)
    }

    /// Get block receipts
    pub async fn get_block_receipts(&self, block_number: &str) -> Result<Value> {
        self.rpc("eth_getBlockReceipts", json!([block_number])).await
    }

    /// Get block transaction count by hash
    pub async fn get_block_transaction_count_by_hash(&self, block_hash: &str) -> Result<u64> {
        let result = self
            .rpc("eth_getBlockTransactionCountByHash", json!([block_hash]))
            .await?;
        parse_hex_u64(&result)
    }

    /// Get block transaction count by number
    pub async fn get_block_transaction_count_by_number(&self, block_number: &str) -> Result<u64> {
        let result = self
            .rpc("eth_getBlockTransactionCountByNumber", json!([block_number]))
            .await?;
        parse_hex_u64(&result)
    }

    /// Get transaction by block hash and index
    pub async fn get_transaction_by_block_hash_and_index(
        &self,
        block_hash: &str,
        index: u64,
    ) -> Result<Value> {
        self.rpc(
            "eth_getTransactionByBlockHashAndIndex",
            json!([block_hash, format!("0x{:x}", index)]),
        )
        .await
    }

    /// Get transaction by block number and index
    pub async fn get_transaction_by_block_number_and_index(
        &self,
        block_number: &str,
        index: u64,
    ) -> Result<Value> {
        self.rpc(
            "eth_getTransactionByBlockNumberAndIndex",
            json!([block_number, format!("0x{:x}", index)]),
        )
        .await
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Debug/Trace (requires debug=true, mainnet only)
    // ──────────────────────────────────────────────────────────────────────────

    /// Debug trace a transaction
    pub async fn debug_trace_transaction(
        &self,
        tx_hash: &str,
        tracer_config: Option<&Value>,
    ) -> Result<Value> {
        if !self.debug {
            return Err(crate::Error::ConfigError(
                "Debug mode not enabled. Use .with_debug(true)".to_string(),
            ));
        }
        let config = tracer_config.cloned().unwrap_or(json!({}));
        self.rpc("debug_traceTransaction", json!([tx_hash, config]))
            .await
    }

    /// Trace a transaction
    pub async fn trace_transaction(&self, tx_hash: &str) -> Result<Value> {
        if !self.debug {
            return Err(crate::Error::ConfigError(
                "Debug mode not enabled. Use .with_debug(true)".to_string(),
            ));
        }
        self.rpc("trace_transaction", json!([tx_hash])).await
    }

    /// Trace a block
    pub async fn trace_block(&self, block_number: &str) -> Result<Value> {
        if !self.debug {
            return Err(crate::Error::ConfigError(
                "Debug mode not enabled. Use .with_debug(true)".to_string(),
            ));
        }
        self.rpc("trace_block", json!([block_number])).await
    }
}

/// Parse a hex string to u64
fn parse_hex_u64(value: &Value) -> Result<u64> {
    let s = value.as_str().unwrap_or("0x0");
    let s = s.trim_start_matches("0x");
    u64::from_str_radix(s, 16)
        .map_err(|e| crate::Error::JsonError(format!("Invalid hex number: {}", e)))
}
