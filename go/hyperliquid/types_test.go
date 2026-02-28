package hyperliquid

import (
	"testing"
)

// Test Side enum
func TestSideEnum(t *testing.T) {
	if SideBuy != "buy" {
		t.Errorf("SideBuy = %q, want %q", SideBuy, "buy")
	}
	if SideSell != "sell" {
		t.Errorf("SideSell = %q, want %q", SideSell, "sell")
	}
}

// Test TIF enum
func TestTIFEnum(t *testing.T) {
	if TIFGTC != "gtc" {
		t.Errorf("TIFGTC = %q, want %q", TIFGTC, "gtc")
	}
	if TIFIOC != "ioc" {
		t.Errorf("TIFIOC = %q, want %q", TIFIOC, "ioc")
	}
	if TIFALO != "alo" {
		t.Errorf("TIFALO = %q, want %q", TIFALO, "alo")
	}
	if TIFMarket != "market" {
		t.Errorf("TIFMarket = %q, want %q", TIFMarket, "market")
	}
}

// Test TpSl enum
func TestTpSlEnum(t *testing.T) {
	if TpSlSL != "sl" {
		t.Errorf("TpSlSL = %q, want %q", TpSlSL, "sl")
	}
	if TpSlTP != "tp" {
		t.Errorf("TpSlTP = %q, want %q", TpSlTP, "tp")
	}
}

// Test OrderGrouping enum
func TestOrderGroupingEnum(t *testing.T) {
	if OrderGroupingNA != "na" {
		t.Errorf("OrderGroupingNA = %q, want %q", OrderGroupingNA, "na")
	}
	if OrderGroupingPositionTPSL != "positionTpsl" {
		t.Errorf("OrderGroupingPositionTPSL = %q, want %q", OrderGroupingPositionTPSL, "positionTpsl")
	}
}

// Test ConnectionState enum
func TestConnectionStateEnum(t *testing.T) {
	if ConnectionStateDisconnected != "disconnected" {
		t.Errorf("ConnectionStateDisconnected = %q, want %q", ConnectionStateDisconnected, "disconnected")
	}
	if ConnectionStateConnecting != "connecting" {
		t.Errorf("ConnectionStateConnecting = %q, want %q", ConnectionStateConnecting, "connecting")
	}
	if ConnectionStateConnected != "connected" {
		t.Errorf("ConnectionStateConnected = %q, want %q", ConnectionStateConnected, "connected")
	}
	if ConnectionStateReconnecting != "reconnecting" {
		t.Errorf("ConnectionStateReconnecting = %q, want %q", ConnectionStateReconnecting, "reconnecting")
	}
}

// Test Decimal
func TestDecimal(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{100, "100"},
		{0.001, "0.001"},
		{"123.456", "123.456"},
		{100.0, "100"},
		{1.23456789, "1.23456789"},
	}

	for _, tt := range tests {
		d := NewDecimal(tt.input)
		if d.String() != tt.expected {
			t.Errorf("NewDecimal(%v).String() = %q, want %q", tt.input, d.String(), tt.expected)
		}
	}
}

// Test Decimal Float64
func TestDecimalFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
	}{
		{100, 100.0},
		{"123.456", 123.456},
		{0.001, 0.001},
	}

	for _, tt := range tests {
		d := NewDecimal(tt.input)
		if d.Float64() != tt.expected {
			t.Errorf("NewDecimal(%v).Float64() = %f, want %f", tt.input, d.Float64(), tt.expected)
		}
	}
}
