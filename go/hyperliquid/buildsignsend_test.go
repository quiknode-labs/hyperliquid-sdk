package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newFakeExchange returns an httptest server that mimics the build→send worker:
// a request without a "signature" field is treated as the build step (returns a
// hash + nonce); one with a "signature" is the send step (returns ok). onSign,
// if non-nil, is invoked with the decoded send payload.
func newFakeExchange(t *testing.T, hash string, nonce int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")
		if _, isSend := req["signature"]; isSend {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"hash": hash, "nonce": nonce})
	}))
}

func newTestSDK(exchangeURL string, signer Signer) *SDK {
	return &SDK{
		config:      &Config{Timeout: 30 * time.Second},
		http:        NewHTTPClient(30 * time.Second),
		exchangeURL: exchangeURL,
		signer:      signer,
	}
}

// newRecordingExchange is like newFakeExchange but records every decoded
// request body in order (build first, then send), so tests can assert the
// exact wire payloads produced by buildSignSend.
func newRecordingExchange(t *testing.T, requests *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		*requests = append(*requests, req)

		w.Header().Set("Content-Type", "application/json")
		if _, isSend := req["signature"]; isSend {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"hash": "0xabc123", "nonce": 42})
	}))
}

func okSigner(context.Context, string) (*Signature, error) {
	return &Signature{R: "0x1", S: "0x2", V: 27}, nil
}

func TestBuildSignSend_SignerReceivesDeadlineContext(t *testing.T) {
	srv := newFakeExchange(t, "0xabc123", 42)
	defer srv.Close()

	var gotDeadline bool
	var gotHash string
	signer := func(ctx context.Context, hashHex string) (*Signature, error) {
		_, gotDeadline = ctx.Deadline()
		gotHash = hashHex
		return &Signature{R: "0x1", S: "0x2", V: 27}, nil
	}

	s := newTestSDK(srv.URL, signer)
	if _, err := s.buildSignSend(map[string]any{"type": "order"}, nil); err != nil {
		t.Fatalf("buildSignSend returned error: %v", err)
	}
	if !gotDeadline {
		t.Fatal("signer received a context without a deadline; expected one bounded by Timeout")
	}
	if gotHash != "0xabc123" {
		t.Fatalf("signer received hash %q, want %q", gotHash, "0xabc123")
	}
}

func TestBuildSignSend_SignerErrorIsSignerError(t *testing.T) {
	srv := newFakeExchange(t, "0xabc123", 42)
	defer srv.Close()

	sentinel := errors.New("kms unavailable")
	signer := func(context.Context, string) (*Signature, error) {
		return nil, sentinel
	}

	s := newTestSDK(srv.URL, signer)
	_, err := s.buildSignSend(map[string]any{"type": "order"}, nil)
	if err == nil {
		t.Fatal("expected an error when the signer fails")
	}
	if !IsSignerError(err) {
		t.Fatalf("IsSignerError = false, want true (err: %v)", err)
	}
	if !IsErrorCode(err, ErrorCodeSignerFailed) {
		t.Fatalf("error code = %v, want %v", err, ErrorCodeSignerFailed)
	}
	if IsErrorCode(err, ErrorCodeSignatureInvalid) {
		t.Fatal("signer failure must not be reported as SIGNATURE_INVALID (venue rejection)")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is did not find the wrapped cause; err: %v", err)
	}
}

func TestBuildSignSend_NilSignatureIsSignerError(t *testing.T) {
	srv := newFakeExchange(t, "0xabc123", 42)
	defer srv.Close()

	// A signer that returns (nil, nil) — no error, but no signature either.
	signer := func(context.Context, string) (*Signature, error) {
		return nil, nil
	}

	s := newTestSDK(srv.URL, signer)
	_, err := s.buildSignSend(map[string]any{"type": "order"}, nil)
	if err == nil {
		t.Fatal("expected an error when the signer returns a nil signature")
	}
	if !IsSignerError(err) {
		t.Fatalf("IsSignerError = false, want true (err: %v)", err)
	}
}

func TestBuildSignSend_MalformedSignatureIsSignerError(t *testing.T) {
	srv := newFakeExchange(t, "0xabc123", 42)
	defer srv.Close()

	cases := map[string]*Signature{
		"empty r/s":   {R: "", S: "", V: 27},
		"bad v (0/1)": {R: "0x1", S: "0x2", V: 0},
	}
	for name, sig := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestSDK(srv.URL, func(context.Context, string) (*Signature, error) {
				return sig, nil
			})
			_, err := s.buildSignSend(map[string]any{"type": "order"}, nil)
			if !IsSignerError(err) {
				t.Fatalf("IsSignerError = false, want true (err: %v)", err)
			}
		})
	}
}

func TestValidateSignerSignature(t *testing.T) {
	if err := validateSignerSignature(&Signature{R: "0x1", S: "0x2", V: 27}); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := validateSignerSignature(&Signature{R: "0x1", S: "0x2", V: 28}); err != nil {
		t.Fatalf("valid signature (v=28) rejected: %v", err)
	}
	if validateSignerSignature(nil) == nil {
		t.Fatal("nil signature accepted")
	}
	if validateSignerSignature(&Signature{R: "", S: "0x2", V: 27}) == nil {
		t.Fatal("empty r accepted")
	}
	if validateSignerSignature(&Signature{R: "0x1", S: "0x2", V: 1}) == nil {
		t.Fatal("invalid v accepted")
	}
}

func TestBuildSignSend_VaultAddressAndExpiresAfterInBuildAndSend(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)
	vault := "0x1234567890123456789012345678901234567890"
	var expires int64 = 1750000000000
	_, err := s.buildSignSendParams(map[string]any{"type": "order"}, &exchangeParams{
		vaultAddress: vault,
		expiresAfter: &expires,
	})
	if err != nil {
		t.Fatalf("buildSignSendParams returned error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests (build, send), got %d", len(requests))
	}

	for i, name := range []string{"build", "send"} {
		req := requests[i]
		if got, _ := req["vaultAddress"].(string); got != vault {
			t.Fatalf("%s payload vaultAddress = %v, want %q", name, req["vaultAddress"], vault)
		}
		if got, _ := req["expiresAfter"].(float64); int64(got) != expires {
			t.Fatalf("%s payload expiresAfter = %v, want %d", name, req["expiresAfter"], expires)
		}
	}
}

func TestBuildSignSend_OptionalFieldsAbsentWhenUnset(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)
	if _, err := s.buildSignSend(map[string]any{"type": "order"}, nil); err != nil {
		t.Fatalf("buildSignSend returned error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests (build, send), got %d", len(requests))
	}

	for i, name := range []string{"build", "send"} {
		for _, key := range []string{"vaultAddress", "expiresAfter"} {
			if _, present := requests[i][key]; present {
				t.Fatalf("%s payload contains %q when unset; must never be emitted", name, key)
			}
		}
	}
}

func TestCancel_FastFlagEmittedOnlyWhenSet(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)

	// Without the fast option: no "f" key in the cancel action.
	if _, err := s.Cancel(7, ""); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	action, _ := requests[0]["action"].(map[string]any)
	if action == nil {
		t.Fatal("build payload missing action")
	}
	if _, present := action["f"]; present {
		t.Fatal(`cancel action contains "f" without CancelWithFast; must never be emitted`)
	}

	// With the fast option: "f": true alongside cancels.
	requests = nil
	if _, err := s.Cancel(7, "", CancelWithFast()); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	action, _ = requests[0]["action"].(map[string]any)
	if action == nil {
		t.Fatal("build payload missing action")
	}
	if fast, _ := action["f"].(bool); !fast {
		t.Fatalf(`cancel action "f" = %v, want true`, action["f"])
	}
	if _, ok := action["cancels"].([]any); !ok {
		t.Fatalf("cancel action lost its cancels entries: %v", action)
	}
}

func TestCancelByCloid_FastAndVaultOptions(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)
	vault := "0x1234567890123456789012345678901234567890"

	// Numeric asset avoids the markets lookup in resolveAssetIndex.
	_, err := s.CancelByCloid("0x00000000000000000000000000000001", "0",
		CancelWithFast(), CancelWithVaultAddress(vault), CancelWithExpiresAfter(1750000000000))
	if err != nil {
		t.Fatalf("CancelByCloid returned error: %v", err)
	}

	action, _ := requests[0]["action"].(map[string]any)
	if fast, _ := action["f"].(bool); !fast {
		t.Fatalf(`cancelByCloid action "f" = %v, want true`, action["f"])
	}
	for i, name := range []string{"build", "send"} {
		if got, _ := requests[i]["vaultAddress"].(string); got != vault {
			t.Fatalf("%s payload vaultAddress = %v, want %q", name, requests[i]["vaultAddress"], vault)
		}
		if got, _ := requests[i]["expiresAfter"].(float64); int64(got) != 1750000000000 {
			t.Fatalf("%s payload expiresAfter = %v, want 1750000000000", name, requests[i]["expiresAfter"])
		}
	}
}

func TestModify_VaultAddressAndExpiresAfter(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)
	vault := "0x1234567890123456789012345678901234567890"

	_, err := s.Modify(9, "BTC", "buy", "67000", "0.001",
		ModifyWithVaultAddress(vault), ModifyWithExpiresAfter(1750000000000))
	if err != nil {
		t.Fatalf("Modify returned error: %v", err)
	}
	for i, name := range []string{"build", "send"} {
		if got, _ := requests[i]["vaultAddress"].(string); got != vault {
			t.Fatalf("%s payload vaultAddress = %v, want %q", name, requests[i]["vaultAddress"], vault)
		}
		if got, _ := requests[i]["expiresAfter"].(float64); int64(got) != 1750000000000 {
			t.Fatalf("%s payload expiresAfter = %v, want 1750000000000", name, requests[i]["expiresAfter"])
		}
	}
}

func TestPlaceOrder_BuilderVaultAddressAndExpiresAfter(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)
	vault := "0x1234567890123456789012345678901234567890"

	order := Order().Buy("BTC").Size(0.001).Price(67000).GTC().
		VaultAddress(vault).ExpiresAfter(1750000000000)
	if _, err := s.PlaceOrder(order); err != nil {
		t.Fatalf("PlaceOrder returned error: %v", err)
	}
	for i, name := range []string{"build", "send"} {
		if got, _ := requests[i]["vaultAddress"].(string); got != vault {
			t.Fatalf("%s payload vaultAddress = %v, want %q", name, requests[i]["vaultAddress"], vault)
		}
		if got, _ := requests[i]["expiresAfter"].(float64); int64(got) != 1750000000000 {
			t.Fatalf("%s payload expiresAfter = %v, want 1750000000000", name, requests[i]["expiresAfter"])
		}
	}
}

func TestClosePosition_VaultAddressSetsUser(t *testing.T) {
	var requests []map[string]any
	srv := newRecordingExchange(t, &requests)
	defer srv.Close()

	s := newTestSDK(srv.URL, okSigner)
	vault := "0x1234567890123456789012345678901234567890"

	if _, err := s.ClosePosition("BTC", CloseWithVaultAddress(vault)); err != nil {
		t.Fatalf("ClosePosition returned error: %v", err)
	}

	buildReq := requests[0]
	if got, _ := buildReq["vaultAddress"].(string); got != vault {
		t.Fatalf("build payload vaultAddress = %v, want %q", buildReq["vaultAddress"], vault)
	}
	action, _ := buildReq["action"].(map[string]any)
	if got, _ := action["user"].(string); got != vault {
		t.Fatalf("closePosition action.user = %v, want vault %q (backend queries this position)", action["user"], vault)
	}
}

func TestSignerErrorUnwrap(t *testing.T) {
	sentinel := errors.New("boom")
	err := SignerError(sentinel)

	if !errors.Is(err, sentinel) {
		t.Fatal("SignerError should unwrap to its cause")
	}
	if err.Code != ErrorCodeSignerFailed {
		t.Fatalf("code = %q, want %q", err.Code, ErrorCodeSignerFailed)
	}

	// An error constructed without a cause must still unwrap to nil
	// (backward-compatible root-error behaviour).
	plain := NewError(ErrorCodeBuildError, "no cause")
	if plain.Unwrap() != nil {
		t.Fatalf("Unwrap() on a causeless Error = %v, want nil", plain.Unwrap())
	}
}
