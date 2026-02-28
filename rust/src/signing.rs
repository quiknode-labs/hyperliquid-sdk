//! Signing utilities for Hyperliquid transactions.
//!
//! Implements EIP-712 signing and MessagePack hashing for order authentication.

use alloy::primitives::{keccak256, Address, B256};
use alloy::signers::local::PrivateKeySigner;
use alloy::signers::Signer;
use alloy::sol;
use alloy::sol_types::SolStruct;
use serde::Serialize;

use crate::types::{Chain, Signature, CORE_MAINNET_EIP712_DOMAIN};

// EIP-712 Agent struct for signing
sol! {
    struct Agent {
        string source;
        bytes32 connectionId;
    }
}

/// Compute the EIP-712 signing hash for an Agent struct wrapping a connection ID.
///
/// This is the final hash that users sign for MessagePack-based actions.
#[inline]
pub fn agent_signing_hash(chain: Chain, connection_id: B256) -> B256 {
    let agent = Agent {
        source: if chain.is_mainnet() { "a" } else { "b" }.to_string(),
        connectionId: connection_id,
    };
    agent.eip712_signing_hash(&CORE_MAINNET_EIP712_DOMAIN)
}

/// Compute the MessagePack hash of a value for signing.
///
/// Serializes to named MessagePack, appends nonce, optional vault address
/// (1-byte tag + 20 bytes) and optional expiry (1-byte tag + 8 bytes),
/// then returns keccak256 of the concatenation.
pub fn rmp_hash<T: Serialize>(
    value: &T,
    nonce: u64,
    vault_address: Option<Address>,
    expires_after: Option<u64>,
) -> Result<B256, rmp_serde::encode::Error> {
    let mut bytes = rmp_serde::to_vec_named(value)?;
    bytes.extend(nonce.to_be_bytes());

    if let Some(vault_address) = vault_address {
        bytes.push(1);
        bytes.extend(vault_address.as_slice());
    } else {
        bytes.push(0);
    }

    if let Some(expires_after) = expires_after {
        bytes.push(0);
        bytes.extend(expires_after.to_be_bytes());
    }

    Ok(keccak256(bytes))
}

/// Sign a hash with a private key.
pub async fn sign_hash(signer: &PrivateKeySigner, hash: B256) -> crate::Result<Signature> {
    let sig = signer
        .sign_hash(&hash)
        .await
        .map_err(|e| crate::Error::SigningError(e.to_string()))?;
    Ok(sig.into())
}

/// Sign an action for the Hyperliquid exchange.
pub async fn sign_action<T: Serialize>(
    signer: &PrivateKeySigner,
    chain: Chain,
    action: &T,
    nonce: u64,
    vault_address: Option<Address>,
    expires_after: Option<u64>,
) -> crate::Result<Signature> {
    // Step 1: Compute MessagePack hash
    let connection_id = rmp_hash(action, nonce, vault_address, expires_after)
        .map_err(|e| crate::Error::SigningError(format!("MessagePack serialization failed: {}", e)))?;

    // Step 2: Compute EIP-712 Agent signing hash
    let signing_hash = agent_signing_hash(chain, connection_id);

    // Step 3: Sign the hash
    sign_hash(signer, signing_hash).await
}

/// Recover signer address from a signature.
pub fn recover_signer(hash: B256, sig: &Signature) -> crate::Result<Address> {
    let alloy_sig = alloy::signers::Signature::new(
        alloy::primitives::U256::from(sig.r),
        alloy::primitives::U256::from(sig.s),
        sig.v == 28,
    );

    alloy_sig
        .recover_address_from_prehash(&hash)
        .map_err(|e| crate::Error::SigningError(format!("Failed to recover signer: {}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy::primitives::B256;

    #[test]
    fn test_agent_signing_hash_mainnet() {
        let connection_id = B256::ZERO;
        let hash = agent_signing_hash(Chain::Mainnet, connection_id);
        // Hash should be deterministic
        assert!(!hash.is_zero());
    }

    #[test]
    fn test_agent_signing_hash_testnet() {
        let connection_id = B256::ZERO;
        let hash_mainnet = agent_signing_hash(Chain::Mainnet, connection_id);
        let hash_testnet = agent_signing_hash(Chain::Testnet, connection_id);
        // Different chains should produce different hashes
        assert_ne!(hash_mainnet, hash_testnet);
    }

    #[test]
    fn test_rmp_hash_deterministic() {
        #[derive(Serialize)]
        struct TestAction {
            value: u64,
        }

        let action = TestAction { value: 42 };
        let hash1 = rmp_hash(&action, 1000, None, None).unwrap();
        let hash2 = rmp_hash(&action, 1000, None, None).unwrap();
        assert_eq!(hash1, hash2);
    }

    #[test]
    fn test_rmp_hash_with_vault() {
        #[derive(Serialize)]
        struct TestAction {
            value: u64,
        }

        let action = TestAction { value: 42 };
        let vault = Address::ZERO;
        let hash_no_vault = rmp_hash(&action, 1000, None, None).unwrap();
        let hash_with_vault = rmp_hash(&action, 1000, Some(vault), None).unwrap();
        // Vault should change the hash
        assert_ne!(hash_no_vault, hash_with_vault);
    }
}
