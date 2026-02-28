package hyperliquid

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Wallet wraps an Ethereum private key for signing operations.
type Wallet struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// NewWallet creates a wallet from a hex-encoded private key.
// The private key can optionally include the "0x" prefix.
func NewWallet(privateKeyHex string) (*Wallet, error) {
	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &Wallet{
		privateKey: privateKey,
		address:    address,
	}, nil
}

// Address returns the wallet's Ethereum address.
func (w *Wallet) Address() common.Address {
	return w.address
}

// AddressString returns the wallet's Ethereum address as a hex string.
func (w *Wallet) AddressString() string {
	return w.address.Hex()
}

// SignHash signs a 32-byte hash and returns the signature components.
// This is an "unsafe" sign that doesn't prefix the message - used for
// signing pre-computed hashes from the Hyperliquid build endpoint.
func (w *Wallet) SignHash(hashHex string) (*Signature, error) {
	// Remove 0x prefix if present
	hashHex = strings.TrimPrefix(hashHex, "0x")

	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hash hex: %w", err)
	}

	if len(hashBytes) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes, got %d", len(hashBytes))
	}

	// Sign the hash directly (no Ethereum message prefix)
	sig, err := crypto.Sign(hashBytes, w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	if len(sig) != 65 {
		return nil, fmt.Errorf("invalid signature length: %d", len(sig))
	}

	// Extract r, s, v from signature
	// Ethereum signatures are [R || S || V] where V is 0 or 1
	// Hyperliquid expects V as 27 or 28
	r := sig[:32]
	s := sig[32:64]
	v := int(sig[64])

	// Adjust v to be 27 or 28
	if v < 27 {
		v += 27
	}

	return &Signature{
		R: "0x" + hex.EncodeToString(r),
		S: "0x" + hex.EncodeToString(s),
		V: v,
	}, nil
}

// ValidateAddress checks if a string is a valid Ethereum address.
func ValidateAddress(addr string) bool {
	if !strings.HasPrefix(addr, "0x") {
		return false
	}
	if len(addr) != 42 {
		return false
	}
	_, err := hex.DecodeString(addr[2:])
	return err == nil
}

// NormalizeAddress normalizes an Ethereum address to checksummed format.
func NormalizeAddress(addr string) string {
	if !ValidateAddress(addr) {
		return addr
	}
	return common.HexToAddress(addr).Hex()
}
