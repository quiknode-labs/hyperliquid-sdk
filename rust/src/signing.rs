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

/// External signing abstraction.
///
/// Implement this to sign Hyperliquid action hashes via a remote KMS/HSM or
/// signing service, without giving the SDK a raw private key. Supply it through
/// `HyperliquidSDKBuilder::signer`. The in-process key path uses [`LocalSigner`].
///
/// `sign_hash` is awaited inside the build→sign→send pipeline, so it is cancelled
/// when the caller drops the future; an optional per-sign deadline can be set via
/// `HyperliquidSDKBuilder::signer_deadline`. Returned errors surface to callers as
/// [`crate::Error::SignerError`] (code `SIGNER_FAILED`), distinct from a venue
/// rejection.
#[async_trait::async_trait]
pub trait HyperliquidSigner: Send + Sync {
    /// The acting agent address this signer signs as.
    fn address(&self) -> Address;

    /// Sign the 32-byte Hyperliquid build hash, returning r/s/v.
    async fn sign_hash(&self, hash: B256) -> crate::Result<Signature>;
}

/// In-process signer backed by a local private key (the default path).
pub struct LocalSigner(pub PrivateKeySigner);

#[async_trait::async_trait]
impl HyperliquidSigner for LocalSigner {
    fn address(&self) -> Address {
        self.0.address()
    }

    async fn sign_hash(&self, hash: B256) -> crate::Result<Signature> {
        sign_hash(&self.0, hash).await
    }
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

    #[tokio::test]
    async fn test_local_signer_address_matches_key() {
        let key = PrivateKeySigner::random();
        let expected = key.address();
        let local = LocalSigner(key);
        assert_eq!(HyperliquidSigner::address(&local), expected);
    }

    #[tokio::test]
    async fn test_local_signer_sign_hash_recovers_to_address() {
        let key = PrivateKeySigner::random();
        let address = key.address();
        let local = LocalSigner(key);

        let hash = B256::repeat_byte(0x42);
        let sig = local.sign_hash(hash).await.expect("local signer should sign");

        let recovered = recover_signer(hash, &sig).expect("should recover");
        assert_eq!(recovered, address);
    }

    /// A trait object can be backed by an arbitrary external signer.
    #[tokio::test]
    async fn test_external_signer_trait_object() {
        struct FailingSigner;

        #[async_trait::async_trait]
        impl HyperliquidSigner for FailingSigner {
            fn address(&self) -> Address {
                Address::ZERO
            }
            async fn sign_hash(&self, _hash: B256) -> crate::Result<Signature> {
                Err(crate::Error::SignerError("kms unavailable".to_string()))
            }
        }

        let signer: std::sync::Arc<dyn HyperliquidSigner> = std::sync::Arc::new(FailingSigner);
        assert_eq!(signer.address(), Address::ZERO);

        let err = signer.sign_hash(B256::ZERO).await.unwrap_err();
        assert_eq!(err.code(), crate::ErrorCode::SignerFailed);
    }
}
