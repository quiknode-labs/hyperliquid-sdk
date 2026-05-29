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
