package hyperliquid

import (
	"context"
	"testing"
)

// testEndpoint is a syntactically valid endpoint that is never contacted —
// no test in this file performs any network call.
const testEndpoint = "https://example.quiknode.pro/TEST"

// testPrivateKey is a throwaway, well-known test key (the first Hardhat/Anvil
// account). It is used only to exercise the in-process wallet path and never
// holds real funds.
const testPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

func TestWithSignerSetsSignerAndNewSucceedsWithoutKey(t *testing.T) {
	// Ensure no ambient private key leaks into construction.
	t.Setenv("PRIVATE_KEY", "")

	called := false
	signer := func(_ context.Context, hashHex string) (*Signature, error) {
		called = true
		return &Signature{R: "0x1", S: "0x2", V: 27}, nil
	}

	sdk, err := New(testEndpoint, WithSigner(signer))
	if err != nil {
		t.Fatalf("New with only WithSigner returned error: %v", err)
	}
	if sdk == nil {
		t.Fatal("New returned nil SDK")
	}

	if sdk.signer == nil {
		t.Fatal("WithSigner did not set the SDK signer")
	}
	if sdk.wallet != nil {
		t.Fatal("a wallet was created despite no private key being supplied")
	}
	if sdk.config.PrivateKey != "" {
		t.Fatalf("private key was populated under the signer path: %q", sdk.config.PrivateKey)
	}

	// The configured callback must be the one we passed.
	if _, err := sdk.signer(context.Background(), "0x"+"00"); err != nil {
		t.Fatalf("signer callback returned error: %v", err)
	}
	if !called {
		t.Fatal("the SDK signer is not the callback supplied to WithSigner")
	}
}

func TestWithSignerAddressBackfillsAddress(t *testing.T) {
	t.Setenv("PRIVATE_KEY", "")

	const addr = "0x1234567890123456789012345678901234567890"
	sdk, err := New(testEndpoint,
		WithSigner(func(context.Context, string) (*Signature, error) { return &Signature{}, nil }),
		WithSignerAddress(addr),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if got := sdk.Address(); got != addr {
		t.Fatalf("Address() = %q, want %q", got, addr)
	}
}

func TestRequireWalletWithSignerDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("requireWallet panicked with a signer set: %v", r)
		}
	}()

	s := &SDK{signer: func(context.Context, string) (*Signature, error) { return &Signature{}, nil }}
	s.requireWallet()
}

func TestRequireWalletPanicsWhenNeitherSet(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("requireWallet did not panic when neither wallet nor signer is set")
		}
	}()

	s := &SDK{}
	s.requireWallet()
}

func TestRequireWalletWithWalletDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("requireWallet panicked with a wallet set: %v", r)
		}
	}()

	wallet, err := NewWallet(testPrivateKey)
	if err != nil {
		t.Fatalf("NewWallet returned error: %v", err)
	}
	s := &SDK{wallet: wallet}
	s.requireWallet()
}

func TestWithPrivateKeyStillConstructs(t *testing.T) {
	t.Setenv("PRIVATE_KEY", "")

	// AutoApprove is disabled so construction performs no network call.
	sdk, err := New(testEndpoint,
		WithPrivateKey(testPrivateKey),
		WithAutoApprove(false),
	)
	if err != nil {
		t.Fatalf("New with WithPrivateKey returned error: %v", err)
	}
	if sdk.wallet == nil {
		t.Fatal("WithPrivateKey did not create an in-process wallet")
	}
	if sdk.signer != nil {
		t.Fatal("a signer was set despite using WithPrivateKey")
	}
	if sdk.Address() == "" {
		t.Fatal("Address() is empty for an in-process wallet")
	}
}
