package hyperliquid

import (
	"strings"
	"testing"
)

// Test HyperliquidError
func TestHyperliquidError(t *testing.T) {
	err := NewError(ErrorCodeBuildError, "test error")

	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("Error() = %q, should contain %q", err.Error(), "test error")
	}

	if err.Code != ErrorCodeBuildError {
		t.Errorf("Code = %q, want %q", err.Code, ErrorCodeBuildError)
	}
}

// Test HyperliquidError with raw data
func TestHyperliquidErrorWithRaw(t *testing.T) {
	rawData := map[string]any{"foo": "bar"}
	err := NewError(ErrorCodeBuildError, "test error").WithRaw(rawData)

	if err.Raw == nil {
		t.Error("Raw should not be nil")
	}

	if err.Raw["foo"] != "bar" {
		t.Errorf("Raw[foo] = %v, want %q", err.Raw["foo"], "bar")
	}
}

// Test HyperliquidError with guidance
func TestHyperliquidErrorWithGuidance(t *testing.T) {
	err := NewError(ErrorCodeInvalidParams, "test error").WithGuidance("try doing X instead")

	if !strings.Contains(err.Error(), "try doing X instead") {
		t.Errorf("Error() = %q, should contain guidance", err.Error())
	}
}

// Test specific error constructors
func TestSpecificErrorConstructors(t *testing.T) {
	buildErr := BuildError("build failed")
	if buildErr.Code != ErrorCodeBuildError {
		t.Errorf("BuildError code = %q, want %q", buildErr.Code, ErrorCodeBuildError)
	}

	sendErr := SendError("send failed")
	if sendErr.Code != ErrorCodeSendError {
		t.Errorf("SendError code = %q, want %q", sendErr.Code, ErrorCodeSendError)
	}

	validationErr := ValidationError("validation failed")
	if validationErr.Code != ErrorCodeInvalidParams {
		t.Errorf("ValidationError code = %q, want %q", validationErr.Code, ErrorCodeInvalidParams)
	}

	signatureErr := SignatureError("signature failed")
	if signatureErr.Code != ErrorCodeSignatureInvalid {
		t.Errorf("SignatureError code = %q, want %q", signatureErr.Code, ErrorCodeSignatureInvalid)
	}
}

// Test error code constants
func TestErrorCodeConstants(t *testing.T) {
	codes := []ErrorCode{
		ErrorCodeBuildError,
		ErrorCodeSendError,
		ErrorCodeNotApproved,
		ErrorCodeInvalidParams,
		ErrorCodeSignatureInvalid,
		ErrorCodeNoPosition,
		ErrorCodeOrderNotFound,
		ErrorCodeGeoBlocked,
		ErrorCodeInsufficientMargin,
		ErrorCodeLeverageConflict,
		ErrorCodeRateLimited,
		ErrorCodeMaxOrdersExceeded,
		ErrorCodeReduceOnlyViolation,
		ErrorCodeDuplicateOrder,
		ErrorCodeUserNotFound,
		ErrorCodeMustDepositFirst,
		ErrorCodeInvalidNonce,
		ErrorCodeConnectionError,
		ErrorCodeTimeout,
		ErrorCodeParseError,
		ErrorCodeInvalidJSON,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
	}
}

// Test ParseAPIError
func TestParseAPIError(t *testing.T) {
	// Test GEO_BLOCKED error
	geoBlockedResponse := map[string]any{
		"error": "GEO_BLOCKED",
	}
	err := ParseAPIError(geoBlockedResponse, 403)
	if err.Code != ErrorCodeGeoBlocked {
		t.Errorf("Expected GeoBlocked error, got %q", err.Code)
	}

	// Test rate limit error
	rateLimitResponse := map[string]any{
		"error": "RATE_LIMITED",
	}
	err = ParseAPIError(rateLimitResponse, 429)
	if err.Code != ErrorCodeRateLimited {
		t.Errorf("Expected RateLimit error, got %q", err.Code)
	}

	// Test insufficient margin error
	marginResponse := map[string]any{
		"error": "INSUFFICIENT_MARGIN",
	}
	err = ParseAPIError(marginResponse, 400)
	if err.Code != ErrorCodeInsufficientMargin {
		t.Errorf("Expected InsufficientMargin error, got %q", err.Code)
	}
}

// Test IsErrorCode helper
func TestIsErrorCode(t *testing.T) {
	err := BuildError("test")
	if !IsErrorCode(err, ErrorCodeBuildError) {
		t.Error("IsErrorCode should return true for matching code")
	}
	if IsErrorCode(err, ErrorCodeSendError) {
		t.Error("IsErrorCode should return false for non-matching code")
	}
}

// Test IsValidationError helper
func TestIsValidationError(t *testing.T) {
	err := ValidationError("test")
	if !IsValidationError(err) {
		t.Error("IsValidationError should return true for validation error")
	}

	buildErr := BuildError("test")
	if IsValidationError(buildErr) {
		t.Error("IsValidationError should return false for build error")
	}
}

// Test IsRetryable helper
func TestIsRetryable(t *testing.T) {
	timeoutErr := NewError(ErrorCodeTimeout, "test")
	if !IsRetryable(timeoutErr) {
		t.Error("IsRetryable should return true for timeout error")
	}

	rateLimitErr := NewError(ErrorCodeRateLimited, "test")
	if !IsRetryable(rateLimitErr) {
		t.Error("IsRetryable should return true for rate limit error")
	}

	buildErr := BuildError("test")
	if IsRetryable(buildErr) {
		t.Error("IsRetryable should return false for build error")
	}
}
