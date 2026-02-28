package hyperliquid

import (
	"fmt"
	"strings"
)

// ErrorCode represents semantic error codes.
type ErrorCode string

const (
	ErrorCodeUnknown           ErrorCode = "UNKNOWN_ERROR"
	ErrorCodeTimeout           ErrorCode = "TIMEOUT"
	ErrorCodeConnectionError   ErrorCode = "CONNECTION_ERROR"
	ErrorCodeParseError        ErrorCode = "PARSE_ERROR"
	ErrorCodeInvalidJSON       ErrorCode = "INVALID_JSON"
	ErrorCodeMissingField      ErrorCode = "MISSING_FIELD"
	ErrorCodeInvalidParams     ErrorCode = "INVALID_PARAMS"
	ErrorCodeInvalidPriceTick  ErrorCode = "INVALID_PRICE_TICK"
	ErrorCodeInvalidSize       ErrorCode = "INVALID_SIZE"
	ErrorCodeSignatureInvalid  ErrorCode = "SIGNATURE_INVALID"
	ErrorCodeNoPosition        ErrorCode = "NO_POSITION"
	ErrorCodeOrderNotFound     ErrorCode = "ORDER_NOT_FOUND"
	ErrorCodeGeoBlocked        ErrorCode = "GEO_BLOCKED"
	ErrorCodeNotApproved       ErrorCode = "NOT_APPROVED"
	ErrorCodeFeeExceedsApproved ErrorCode = "FEE_EXCEEDS_APPROVED"
	ErrorCodeInsufficientMargin ErrorCode = "INSUFFICIENT_MARGIN"
	ErrorCodeLeverageConflict  ErrorCode = "LEVERAGE_CONFLICT"
	ErrorCodeRateLimited       ErrorCode = "RATE_LIMITED"
	ErrorCodeMaxOrdersExceeded ErrorCode = "MAX_ORDERS_EXCEEDED"
	ErrorCodeReduceOnlyViolation ErrorCode = "REDUCE_ONLY_VIOLATION"
	ErrorCodeDuplicateOrder    ErrorCode = "DUPLICATE_ORDER"
	ErrorCodeUserNotFound      ErrorCode = "USER_NOT_FOUND"
	ErrorCodeMustDepositFirst  ErrorCode = "MUST_DEPOSIT_FIRST"
	ErrorCodeInvalidNonce      ErrorCode = "INVALID_NONCE"
	ErrorCodeGRPCError         ErrorCode = "GRPC_ERROR"
	ErrorCodeHTTPError         ErrorCode = "HTTP_ERROR"
	ErrorCodeBuildError        ErrorCode = "BUILD_ERROR"
	ErrorCodeSendError         ErrorCode = "SEND_ERROR"
)

// Error represents a Hyperliquid SDK error with structured information.
type Error struct {
	Code     ErrorCode      `json:"code,omitempty"`
	Message  string         `json:"message"`
	Guidance string         `json:"guidance,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	var parts []string
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}
	parts = append(parts, e.Message)
	if e.Guidance != "" {
		parts = append(parts, fmt.Sprintf("\n  Hint: %s", e.Guidance))
	}
	return strings.Join(parts, " ")
}

// Unwrap returns nil (Error is the root error type).
func (e *Error) Unwrap() error {
	return nil
}

// NewError creates a new Error.
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WithGuidance adds guidance to the error.
func (e *Error) WithGuidance(guidance string) *Error {
	e.Guidance = guidance
	return e
}

// WithRaw adds raw response data to the error.
func (e *Error) WithRaw(raw map[string]any) *Error {
	e.Raw = raw
	return e
}

// BuildError creates a build phase error.
func BuildError(message string) *Error {
	return NewError(ErrorCodeBuildError, message)
}

// SendError creates a send phase error.
func SendError(message string) *Error {
	return NewError(ErrorCodeSendError, message)
}

// ApprovalError creates an approval-related error.
func ApprovalError(message string, approvalData map[string]any) *Error {
	err := NewError(ErrorCodeNotApproved, message)
	err.Guidance = "Builder fee not approved. Call sdk.ApproveBuilderFee(\"1%\") or use WithAutoApprove(true)."
	err.Raw = approvalData
	return err
}

// ValidationError creates a validation error.
func ValidationError(message string) *Error {
	return NewError(ErrorCodeInvalidParams, message)
}

// NoPositionError creates a no position error.
func NoPositionError(asset string) *Error {
	return NewError(ErrorCodeNoPosition, fmt.Sprintf("No open position for %s", asset)).
		WithGuidance(fmt.Sprintf("You don't have a %s position to close. Check your positions via sdk.Info().ClearinghouseState() or the Hyperliquid app.", asset))
}

// OrderNotFoundError creates an order not found error.
func OrderNotFoundError(oid int64) *Error {
	return NewError(ErrorCodeOrderNotFound, fmt.Sprintf("Order %d not found", oid)).
		WithGuidance("The order may have already been filled or cancelled.")
}

// GeoBlockedError creates a geo-blocked error.
func GeoBlockedError(message string, data map[string]any) *Error {
	return NewError(ErrorCodeGeoBlocked, message).
		WithGuidance("Your IP is blocked by Hyperliquid's geo-restrictions.").
		WithRaw(data)
}

// InsufficientMarginError creates an insufficient margin error.
func InsufficientMarginError(rawError string) *Error {
	return NewError(ErrorCodeInsufficientMargin, "Insufficient margin for this order").
		WithGuidance("Deposit more collateral or reduce order size. Check your available margin at https://app.hyperliquid.xyz").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// LeverageError creates a leverage conflict error.
func LeverageError(rawError string) *Error {
	return NewError(ErrorCodeLeverageConflict, "Leverage configuration incompatible with this order").
		WithGuidance("Close your existing position or update leverage first. You may need to switch between cross/isolated margin modes.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// RateLimitError creates a rate limit error.
func RateLimitError(rawError string) *Error {
	return NewError(ErrorCodeRateLimited, "Rate limit exceeded").
		WithGuidance("Wait a moment and retry. Consider batching multiple orders.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// MaxOrdersError creates a max orders exceeded error.
func MaxOrdersError(rawError string) *Error {
	return NewError(ErrorCodeMaxOrdersExceeded, "Maximum open orders exceeded").
		WithGuidance("Cancel some existing orders before placing new ones. Use sdk.OpenOrders() to see your orders and sdk.CancelAll() to clear them.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// ReduceOnlyError creates a reduce-only violation error.
func ReduceOnlyError(rawError string) *Error {
	return NewError(ErrorCodeReduceOnlyViolation, "Reduce-only order would increase position").
		WithGuidance("Remove reduce_only flag or ensure the order direction actually reduces your current position.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// DuplicateOrderError creates a duplicate order error.
func DuplicateOrderError(rawError string) *Error {
	return NewError(ErrorCodeDuplicateOrder, "Duplicate order (client order ID already exists)").
		WithGuidance("Use a unique cloid for each order, or omit it to auto-generate.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// UserNotFoundError creates a user not found error.
func UserNotFoundError(rawError string) *Error {
	return NewError(ErrorCodeUserNotFound, "Wallet not recognized by Hyperliquid").
		WithGuidance("Deposit USDC to your Hyperliquid account first at https://app.hyperliquid.xyz — deposits go through the Arbitrum bridge.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// MustDepositError creates a must deposit first error.
func MustDepositError(rawError string) *Error {
	return NewError(ErrorCodeMustDepositFirst, "Your account needs a deposit before trading").
		WithGuidance("Go to https://app.hyperliquid.xyz and deposit USDC, then try again.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// InvalidNonceError creates an invalid nonce error.
func InvalidNonceError(rawError string) *Error {
	return NewError(ErrorCodeInvalidNonce, "Nonce invalid or expired").
		WithGuidance("The SDK handles nonces automatically. If you see this error, your request may have timed out. Retry.").
		WithRaw(map[string]any{"rawHlError": rawError})
}

// SignatureError creates a signature error.
func SignatureError(message string) *Error {
	return NewError(ErrorCodeSignatureInvalid, message).
		WithGuidance("Signature verification failed. This is usually an SDK bug — please report it.")
}

// ParseAPIError parses an API error response into an appropriate Error.
func ParseAPIError(data map[string]any, statusCode int) *Error {
	errVal := data["error"]
	message := ""
	guidance := ""
	rawHlError := ""

	if msg, ok := data["message"].(string); ok {
		message = msg
	} else if errVal != nil {
		message = fmt.Sprintf("%v", errVal)
	} else {
		message = "Unknown error"
	}

	if g, ok := data["guidance"].(string); ok {
		guidance = g
	}

	if rhe, ok := data["rawHlError"].(string); ok {
		rawHlError = rhe
	}

	// Handle nested error object
	if errMap, ok := errVal.(map[string]any); ok {
		if errMap["code"] == float64(403) || strings.Contains(strings.ToLower(fmt.Sprintf("%v", errMap["message"])), "restricted jurisdiction") {
			return GeoBlockedError(message, data)
		}
		if msg, ok := errMap["message"].(string); ok {
			message = msg
		}
	}

	// Normalize error code
	var errorCode ErrorCode
	if ec, ok := data["errorCode"].(string); ok {
		errorCode = ErrorCode(strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(ec, "-", "_"), " ", "_")))
	} else if errVal != nil {
		errorCode = ErrorCode(strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%v", errVal), "-", "_"), " ", "_")))
	}

	// Map error codes to specific error types
	switch errorCode {
	case ErrorCodeNotApproved, "BUILDER_APPROVAL_REQUIRED":
		return ApprovalError(message, data["approvalRequired"].(map[string]any))

	case ErrorCodeFeeExceedsApproved:
		err := NewError(ErrorCodeFeeExceedsApproved, message)
		err.Guidance = "Your approved max fee is too low. Re-approve with a higher rate: sdk.ApproveBuilderFee(\"1%\")"
		err.Raw = data
		return err

	case ErrorCodeInvalidJSON, ErrorCodeMissingField, ErrorCodeInvalidParams:
		return ValidationError(message).WithRaw(data)

	case ErrorCodeInvalidPriceTick, ErrorCodeInvalidSize:
		return ValidationError(message).
			WithGuidance("Use sdk.Preflight() to validate orders before placing them.").
			WithRaw(data)

	case ErrorCodeSignatureInvalid:
		return SignatureError(message).WithRaw(data)

	case ErrorCodeNoPosition:
		asset := ""
		if a, ok := data["asset"].(string); ok {
			asset = a
		}
		return NoPositionError(asset)

	case ErrorCodeInsufficientMargin:
		return InsufficientMarginError(rawHlError)

	case ErrorCodeLeverageConflict:
		return LeverageError(rawHlError)

	case ErrorCodeRateLimited:
		return RateLimitError(rawHlError)

	case ErrorCodeMaxOrdersExceeded:
		return MaxOrdersError(rawHlError)

	case ErrorCodeReduceOnlyViolation:
		return ReduceOnlyError(rawHlError)

	case ErrorCodeDuplicateOrder:
		return DuplicateOrderError(rawHlError)

	case ErrorCodeUserNotFound:
		return UserNotFoundError(rawHlError)

	case ErrorCodeMustDepositFirst:
		return MustDepositError(rawHlError)

	case ErrorCodeInvalidNonce:
		return InvalidNonceError(rawHlError)
	}

	// Check for geo-blocking
	if statusCode == 403 || strings.Contains(strings.ToLower(message), "restricted") || strings.Contains(strings.ToLower(message), "jurisdiction") {
		return GeoBlockedError(message, data)
	}

	// Check phase for categorization
	if phase, ok := data["phase"].(string); ok && strings.Contains(strings.ToLower(phase), "build") {
		return BuildError(message).WithGuidance(guidance).WithRaw(data)
	}

	// Default to SendError
	return SendError(message).WithGuidance(guidance).WithRaw(data)
}

// IsErrorCode checks if an error has a specific code.
func IsErrorCode(err error, code ErrorCode) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}

// IsApprovalError checks if the error is an approval-related error.
func IsApprovalError(err error) bool {
	return IsErrorCode(err, ErrorCodeNotApproved) || IsErrorCode(err, ErrorCodeFeeExceedsApproved)
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	return IsErrorCode(err, ErrorCodeInvalidParams) ||
		IsErrorCode(err, ErrorCodeInvalidPriceTick) ||
		IsErrorCode(err, ErrorCodeInvalidSize) ||
		IsErrorCode(err, ErrorCodeMissingField) ||
		IsErrorCode(err, ErrorCodeInvalidJSON)
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	if e, ok := err.(*Error); ok {
		switch e.Code {
		case ErrorCodeTimeout, ErrorCodeConnectionError, ErrorCodeRateLimited, ErrorCodeInvalidNonce:
			return true
		}
	}
	return false
}
