package hyperliquid

import (
	"testing"
)

// Test 1: Endpoint parsing - Test buildBaseURL handles various formats
func TestBuildBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
		},
		{
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/",
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
		},
		{
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/info",
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
		},
		{
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/hypercore",
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
		},
		{
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/evm",
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
		},
		{
			"https://x.quiknode.pro/TOKEN/nanoreth",
			"https://x.quiknode.pro/TOKEN",
		},
	}

	for _, tt := range tests {
		result := buildBaseURL(tt.input)
		if result != tt.expected {
			t.Errorf("buildBaseURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestHypeToWei(t *testing.T) {
	tests := []struct {
		amount float64
		want   int64
	}{
		{0.001, 100000},
		{0.58, 58000000},
		{0.00000001, 1},
		{1.234567891, 123456789},
		{1, 100000000},
	}

	for _, tt := range tests {
		wei, err := hypeToWei(tt.amount)
		if err != nil {
			t.Fatalf("hypeToWei(%v) returned error: %v", tt.amount, err)
		}
		if wei != tt.want {
			t.Fatalf("hypeToWei(%v) = %d, want %d", tt.amount, wei, tt.want)
		}
	}

	if _, err := hypeToWei(0.000000001); err == nil {
		t.Fatal("hypeToWei(0.000000001) returned nil error, want too-small error")
	}
}

// Test buildInfoURL
func TestBuildInfoURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://x.quiknode.pro/TOKEN",
			"https://x.quiknode.pro/TOKEN/info",
		},
		{
			"https://x.quiknode.pro/TOKEN/info",
			"https://x.quiknode.pro/TOKEN/info",
		},
		{
			"https://x.quiknode.pro/TOKEN/evm",
			"https://x.quiknode.pro/TOKEN/info",
		},
		{
			"https://x.quiknode.pro/TOKEN/hypercore",
			"https://x.quiknode.pro/TOKEN/info",
		},
	}

	for _, tt := range tests {
		result := buildInfoURL(tt.input)
		if result != tt.expected {
			t.Errorf("buildInfoURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test buildHyperCoreURL
func TestBuildHyperCoreURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://x.quiknode.pro/TOKEN",
			"https://x.quiknode.pro/TOKEN/hypercore",
		},
		{
			"https://x.quiknode.pro/TOKEN/info",
			"https://x.quiknode.pro/TOKEN/hypercore",
		},
	}

	for _, tt := range tests {
		result := buildHyperCoreURL(tt.input)
		if result != tt.expected {
			t.Errorf("buildHyperCoreURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test buildEVMURL
func TestBuildEVMURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://x.quiknode.pro/TOKEN",
			"https://x.quiknode.pro/TOKEN/evm",
		},
		{
			"https://x.quiknode.pro/TOKEN/info",
			"https://x.quiknode.pro/TOKEN/evm",
		},
	}

	for _, tt := range tests {
		result := buildEVMURL(tt.input)
		if result != tt.expected {
			t.Errorf("buildEVMURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test buildWebSocketURL
func TestBuildWebSocketURL(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		isQuickNode bool
	}{
		{
			"https://x.quiknode.pro/TOKEN",
			"wss://x.quiknode.pro/TOKEN/hypercore/ws",
			true,
		},
		{
			"https://x.quiknode.pro/TOKEN/info",
			"wss://x.quiknode.pro/TOKEN/hypercore/ws",
			true,
		},
		{
			"https://api.hyperliquid.xyz",
			"wss://api.hyperliquid.xyz/ws",
			false,
		},
		{
			"wss://api.hyperliquid.xyz/ws",
			"wss://api.hyperliquid.xyz/ws",
			false,
		},
	}

	for _, tt := range tests {
		result, isQN := buildWebSocketURL(tt.input)
		if result != tt.expected {
			t.Errorf("buildWebSocketURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
		if isQN != tt.isQuickNode {
			t.Errorf("buildWebSocketURL(%q) isQuickNode = %v, want %v", tt.input, isQN, tt.isQuickNode)
		}
	}
}

// Test extractToken
func TestExtractToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://x.quiknode.pro/TOKEN",
			"TOKEN",
		},
		{
			"https://x.quiknode.pro/TOKEN/info",
			"TOKEN",
		},
		{
			"https://x.quiknode.pro/TOKEN/evm",
			"TOKEN",
		},
		{
			"https://spring-billowing-film.hype-mainnet.quiknode.pro/454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8/evm",
			"454a21b53b2ca93a2fe51ffd0708a6ffe4bc97c8",
		},
	}

	for _, tt := range tests {
		result := extractToken(tt.input)
		if result != tt.expected {
			t.Errorf("extractToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test extractHost
func TestExtractHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://x.quiknode.pro/TOKEN",
			"x.quiknode.pro",
		},
		{
			"https://api.hyperliquid.xyz:8080/ws",
			"api.hyperliquid.xyz",
		},
	}

	for _, tt := range tests {
		result := extractHost(tt.input)
		if result != tt.expected {
			t.Errorf("extractHost(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
