//! Error types for the Hyperliquid SDK.
//!
//! Provides comprehensive error handling with actionable guidance.

use serde::{Deserialize, Serialize};
use std::fmt;
use thiserror::Error;

/// Result type for the SDK
pub type Result<T> = std::result::Result<T, Error>;

/// SDK error types
#[derive(Error, Debug)]
pub enum Error {
    /// Configuration error
    #[error("Configuration error: {0}")]
    ConfigError(String),

    /// Network/HTTP error
    #[error("Network error: {0}")]
    NetworkError(String),

    /// JSON serialization/deserialization error
    #[error("JSON error: {0}")]
    JsonError(String),

    /// Signing error
    #[error("Signing error: {0}")]
    SigningError(String),

    /// Validation error
    #[error("Validation error: {0}")]
    ValidationError(String),

    /// Order-related error
    #[error("Order error: {0}")]
    OrderError(String),

    /// API error from Hyperliquid
    #[error("{message}")]
    ApiError {
        code: ErrorCode,
        message: String,
        guidance: String,
        raw: Option<String>,
    },

    /// Builder fee approval required
    #[error("Builder fee approval required")]
    ApprovalRequired {
        user: String,
        builder: String,
        max_fee_rate: String,
        approval_hash: Option<String>,
    },

    /// No position found
    #[error("No position found for {asset}")]
    NoPosition { asset: String },

    /// Order not found
    #[error("Order {oid} not found")]
    OrderNotFound { oid: u64 },

    /// Rate limited
    #[error("Rate limited: {message}")]
    RateLimited { message: String },

    /// Geo-blocked
    #[error("Access denied from your region")]
    GeoBlocked,

    /// WebSocket error
    #[error("WebSocket error: {0}")]
    WebSocketError(String),

    /// gRPC error
    #[error("gRPC error: {0}")]
    GrpcError(String),
}

impl Error {
    /// Create an API error from a raw error string
    pub fn from_api_error(raw: &str) -> Self {
        let (code, message, guidance) = parse_hl_error(raw);
        Self::ApiError {
            code,
            message,
            guidance,
            raw: Some(raw.to_string()),
        }
    }

    /// Get the semantic error code
    pub fn code(&self) -> ErrorCode {
        match self {
            Error::ConfigError(_) => ErrorCode::ConfigError,
            Error::NetworkError(_) => ErrorCode::NetworkError,
            Error::JsonError(_) => ErrorCode::JsonError,
            Error::SigningError(_) => ErrorCode::SignatureInvalid,
            Error::ValidationError(_) => ErrorCode::InvalidParams,
            Error::OrderError(_) => ErrorCode::OrderError,
            Error::ApiError { code, .. } => *code,
            Error::ApprovalRequired { .. } => ErrorCode::NotApproved,
            Error::NoPosition { .. } => ErrorCode::NoPosition,
            Error::OrderNotFound { .. } => ErrorCode::OrderNotFound,
            Error::RateLimited { .. } => ErrorCode::RateLimited,
            Error::GeoBlocked => ErrorCode::GeoBlocked,
            Error::WebSocketError(_) => ErrorCode::WebSocketError,
            Error::GrpcError(_) => ErrorCode::GrpcError,
        }
    }

    /// Get actionable guidance for this error
    pub fn guidance(&self) -> &str {
        match self {
            Error::ConfigError(_) => {
                "Check your SDK configuration: endpoint URL, private key format, and chain selection."
            }
            Error::NetworkError(_) => {
                "Network request failed. Check your internet connection and try again."
            }
            Error::JsonError(_) => {
                "JSON parsing failed. This may indicate an API change or invalid response."
            }
            Error::SigningError(_) => {
                "Signature verification failed. Ensure you're using the correct private key."
            }
            Error::ValidationError(_) => {
                "Order validation failed. Check size, price, and asset parameters."
            }
            Error::OrderError(_) => {
                "Order operation failed. Check the order state and try again."
            }
            Error::ApiError { guidance, .. } => guidance,
            Error::ApprovalRequired { .. } => {
                "You need to approve the builder fee before trading. \
                 Call sdk.approve_builder_fee() or visit /approve in a browser."
            }
            Error::NoPosition { .. } => {
                "No open position found. Check your positions with sdk.info().clearinghouse_state()."
            }
            Error::OrderNotFound { .. } => {
                "Order not found. It may have been filled or cancelled."
            }
            Error::RateLimited { .. } => {
                "You've exceeded the rate limit. Wait a moment and try again."
            }
            Error::GeoBlocked => {
                "Access is restricted from your region."
            }
            Error::WebSocketError(_) => {
                "WebSocket connection failed. Check your endpoint and network connection."
            }
            Error::GrpcError(_) => {
                "gRPC connection failed. Ensure gRPC port 10000 is accessible."
            }
        }
    }
}

/// Semantic error codes
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ErrorCode {
    // SDK errors
    ConfigError,
    NetworkError,
    JsonError,
    SignatureInvalid,
    InvalidParams,
    OrderError,
    WebSocketError,
    GrpcError,

    // API errors
    NotApproved,
    FeeExceedsApproved,
    FeeExceedsMax,
    InsufficientMargin,
    LeverageConflict,
    InvalidPriceTick,
    InvalidSizeDecimals,
    MaxOrdersExceeded,
    ReduceOnlyViolation,
    DuplicateOrder,
    UserNotFound,
    MustDeposit,
    InvalidNonce,
    NoPosition,
    OrderNotFound,
    RateLimited,
    GeoBlocked,
    Unknown,
}

impl fmt::Display for ErrorCode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ErrorCode::ConfigError => write!(f, "CONFIG_ERROR"),
            ErrorCode::NetworkError => write!(f, "NETWORK_ERROR"),
            ErrorCode::JsonError => write!(f, "JSON_ERROR"),
            ErrorCode::SignatureInvalid => write!(f, "SIGNATURE_INVALID"),
            ErrorCode::InvalidParams => write!(f, "INVALID_PARAMS"),
            ErrorCode::OrderError => write!(f, "ORDER_ERROR"),
            ErrorCode::WebSocketError => write!(f, "WEBSOCKET_ERROR"),
            ErrorCode::GrpcError => write!(f, "GRPC_ERROR"),
            ErrorCode::NotApproved => write!(f, "NOT_APPROVED"),
            ErrorCode::FeeExceedsApproved => write!(f, "FEE_EXCEEDS_APPROVED"),
            ErrorCode::FeeExceedsMax => write!(f, "FEE_EXCEEDS_MAX"),
            ErrorCode::InsufficientMargin => write!(f, "INSUFFICIENT_MARGIN"),
            ErrorCode::LeverageConflict => write!(f, "LEVERAGE_CONFLICT"),
            ErrorCode::InvalidPriceTick => write!(f, "INVALID_PRICE_TICK"),
            ErrorCode::InvalidSizeDecimals => write!(f, "INVALID_SIZE_DECIMALS"),
            ErrorCode::MaxOrdersExceeded => write!(f, "MAX_ORDERS_EXCEEDED"),
            ErrorCode::ReduceOnlyViolation => write!(f, "REDUCE_ONLY_VIOLATION"),
            ErrorCode::DuplicateOrder => write!(f, "DUPLICATE_ORDER"),
            ErrorCode::UserNotFound => write!(f, "USER_NOT_FOUND"),
            ErrorCode::MustDeposit => write!(f, "MUST_DEPOSIT"),
            ErrorCode::InvalidNonce => write!(f, "INVALID_NONCE"),
            ErrorCode::NoPosition => write!(f, "NO_POSITION"),
            ErrorCode::OrderNotFound => write!(f, "ORDER_NOT_FOUND"),
            ErrorCode::RateLimited => write!(f, "RATE_LIMITED"),
            ErrorCode::GeoBlocked => write!(f, "GEO_BLOCKED"),
            ErrorCode::Unknown => write!(f, "UNKNOWN"),
        }
    }
}

/// Parse a raw Hyperliquid error into (code, message, guidance)
fn parse_hl_error(raw: &str) -> (ErrorCode, String, String) {
    let lower = raw.to_lowercase();

    // Pattern match on error messages
    if lower.contains("insufficient margin") || lower.contains("not enough margin") {
        (
            ErrorCode::InsufficientMargin,
            "Insufficient margin for this order".to_string(),
            "Reduce position size or add more margin to your account.".to_string(),
        )
    } else if lower.contains("leverage") && lower.contains("conflict") {
        (
            ErrorCode::LeverageConflict,
            "Leverage conflict with existing position".to_string(),
            "Update leverage before placing this order.".to_string(),
        )
    } else if lower.contains("price") && (lower.contains("tick") || lower.contains("decimal")) {
        (
            ErrorCode::InvalidPriceTick,
            "Invalid price tick size".to_string(),
            "Round your price to the valid tick size for this asset.".to_string(),
        )
    } else if lower.contains("size") && lower.contains("decimal") {
        (
            ErrorCode::InvalidSizeDecimals,
            "Invalid size decimals".to_string(),
            "Round your size to the valid decimal places for this asset.".to_string(),
        )
    } else if lower.contains("max") && lower.contains("order") {
        (
            ErrorCode::MaxOrdersExceeded,
            "Maximum orders exceeded".to_string(),
            "Cancel some existing orders before placing new ones.".to_string(),
        )
    } else if lower.contains("reduce only") {
        (
            ErrorCode::ReduceOnlyViolation,
            "Reduce-only order would increase position".to_string(),
            "Check your position direction and order side.".to_string(),
        )
    } else if lower.contains("duplicate") {
        (
            ErrorCode::DuplicateOrder,
            "Duplicate order".to_string(),
            "This exact order already exists. Use a different cloid if intentional.".to_string(),
        )
    } else if lower.contains("user not found") || lower.contains("unknown user") {
        (
            ErrorCode::UserNotFound,
            "User not found".to_string(),
            "Ensure the address is correct and has been used on Hyperliquid.".to_string(),
        )
    } else if lower.contains("must deposit") || lower.contains("no deposit") {
        (
            ErrorCode::MustDeposit,
            "Account must deposit first".to_string(),
            "Deposit USDC to your Hyperliquid account before trading.".to_string(),
        )
    } else if lower.contains("nonce") {
        (
            ErrorCode::InvalidNonce,
            "Invalid nonce".to_string(),
            "Retry the request - the SDK will generate a fresh nonce.".to_string(),
        )
    } else if lower.contains("rate limit") {
        (
            ErrorCode::RateLimited,
            "Rate limited".to_string(),
            "Wait a moment and try again. Consider using reserve_request_weight().".to_string(),
        )
    } else if lower.contains("geo") || lower.contains("blocked") || lower.contains("restricted") {
        (
            ErrorCode::GeoBlocked,
            "Access denied from your region".to_string(),
            "Trading is not available in your jurisdiction.".to_string(),
        )
    } else {
        (
            ErrorCode::Unknown,
            raw.to_string(),
            "An unexpected error occurred. Check the raw error message for details.".to_string(),
        )
    }
}

// Implement From traits for common error types

impl From<reqwest::Error> for Error {
    fn from(err: reqwest::Error) -> Self {
        Error::NetworkError(err.to_string())
    }
}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Self {
        Error::JsonError(err.to_string())
    }
}

impl From<url::ParseError> for Error {
    fn from(err: url::ParseError) -> Self {
        Error::ConfigError(format!("Invalid URL: {}", err))
    }
}

impl From<std::env::VarError> for Error {
    fn from(err: std::env::VarError) -> Self {
        Error::ConfigError(format!("Environment variable error: {}", err))
    }
}

impl From<alloy::signers::local::LocalSignerError> for Error {
    fn from(err: alloy::signers::local::LocalSignerError) -> Self {
        Error::SigningError(err.to_string())
    }
}
