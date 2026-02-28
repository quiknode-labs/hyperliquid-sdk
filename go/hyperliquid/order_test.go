package hyperliquid

import (
	"testing"
)

// Test OrderBuilder
func TestOrderBuilder(t *testing.T) {
	// Test basic buy order
	order := Order().Buy("BTC").Size(0.001).Price(67000).GTC()

	if order.Asset() != "BTC" {
		t.Errorf("Asset() = %q, want %q", order.Asset(), "BTC")
	}
	if order.GetSide() != SideBuy {
		t.Errorf("GetSide() = %q, want %q", order.GetSide(), SideBuy)
	}
	if order.GetSize() != "0.001" {
		t.Errorf("GetSize() = %q, want %q", order.GetSize(), "0.001")
	}
	if order.GetPrice() != "67000" {
		t.Errorf("GetPrice() = %q, want %q", order.GetPrice(), "67000")
	}
	if order.GetTIF() != TIFGTC {
		t.Errorf("GetTIF() = %q, want %q", order.GetTIF(), TIFGTC)
	}
}

// Test OrderBuilder sell
func TestOrderBuilderSell(t *testing.T) {
	order := Order().Sell("ETH").Size(1.5).Price(4000).IOC()

	if order.Asset() != "ETH" {
		t.Errorf("Asset() = %q, want %q", order.Asset(), "ETH")
	}
	if order.GetSide() != SideSell {
		t.Errorf("GetSide() = %q, want %q", order.GetSide(), SideSell)
	}
	if order.GetTIF() != TIFIOC {
		t.Errorf("GetTIF() = %q, want %q", order.GetTIF(), TIFIOC)
	}
}

// Test OrderBuilder market
func TestOrderBuilderMarket(t *testing.T) {
	order := Order().Buy("BTC").Size(0.01).Market()

	if order.GetTIF() != TIFMarket {
		t.Errorf("GetTIF() = %q, want %q", order.GetTIF(), TIFMarket)
	}
	if order.GetPrice() != "" {
		t.Errorf("GetPrice() = %q, want empty for market order", order.GetPrice())
	}
}

// Test OrderBuilder notional
func TestOrderBuilderNotional(t *testing.T) {
	order := Order().Buy("BTC").Notional(100)

	if order.GetNotional() != 100 {
		t.Errorf("GetNotional() = %f, want %f", order.GetNotional(), 100.0)
	}
}

// Test OrderBuilder reduce only
func TestOrderBuilderReduceOnly(t *testing.T) {
	order := Order().Sell("BTC").Size(0.01).Price(60000).ReduceOnly()

	if !order.IsReduceOnly() {
		t.Error("IsReduceOnly() = false, want true")
	}
}

// Test OrderBuilder validation - missing asset
func TestOrderBuilderValidationMissingAsset(t *testing.T) {
	order := Order().Size(0.01).Price(67000)

	err := order.Validate()
	if err == nil {
		t.Error("Expected validation error for missing asset")
	}
}

// Test OrderBuilder validation - missing size and notional
func TestOrderBuilderValidationMissingSizeAndNotional(t *testing.T) {
	order := Order().Buy("BTC").Price(67000)

	err := order.Validate()
	if err == nil {
		t.Error("Expected validation error for missing size/notional")
	}
}

// Test OrderBuilder validation - missing price for limit order
func TestOrderBuilderValidationMissingPrice(t *testing.T) {
	order := Order().Buy("BTC").Size(0.01).GTC()

	err := order.Validate()
	if err == nil {
		t.Error("Expected validation error for missing price on limit order")
	}
}

// Test OrderBuilder validation - valid order
func TestOrderBuilderValidationSuccess(t *testing.T) {
	order := Order().Buy("BTC").Size(0.01).Price(67000).GTC()

	err := order.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

// Test OrderBuilder ToAction
func TestOrderBuilderToAction(t *testing.T) {
	order := Order().Buy("BTC").Size(0.01).Price(67000).GTC()

	action := order.ToAction()

	if action["type"] != "order" {
		t.Errorf("action type = %q, want %q", action["type"], "order")
	}

	orders, ok := action["orders"].([]any)
	if !ok || len(orders) != 1 {
		t.Error("Expected 1 order in action")
	}
}

// Test TriggerOrderBuilder
func TestTriggerOrderBuilder(t *testing.T) {
	order := TriggerOrder().StopLoss("BTC").Size(0.01).TriggerPrice(60000).Limit(59900).ReduceOnly()

	if order.Asset() != "BTC" {
		t.Errorf("Asset() = %q, want %q", order.Asset(), "BTC")
	}
	if order.GetSize() != "0.01" {
		t.Errorf("GetSize() = %q, want %q", order.GetSize(), "0.01")
	}
	if order.GetTriggerPrice() != "60000" {
		t.Errorf("GetTriggerPrice() = %q, want %q", order.GetTriggerPrice(), "60000")
	}
	if order.GetLimitPrice() != "59900" {
		t.Errorf("GetLimitPrice() = %q, want %q", order.GetLimitPrice(), "59900")
	}
}

// Test TriggerOrderBuilder take profit
func TestTriggerOrderBuilderTakeProfit(t *testing.T) {
	order := TriggerOrder().TakeProfit("BTC").Size(0.01).TriggerPrice(80000).Market()

	if order.Asset() != "BTC" {
		t.Errorf("Asset() = %q, want %q", order.Asset(), "BTC")
	}
}

// Test TriggerOrderBuilder validation
func TestTriggerOrderBuilderValidation(t *testing.T) {
	// Missing asset
	order1 := TriggerOrder().Size(0.01).TriggerPrice(60000)
	if err := order1.Validate(); err == nil {
		t.Error("Expected validation error for missing asset")
	}

	// Missing size
	order2 := TriggerOrder().StopLoss("BTC").TriggerPrice(60000)
	if err := order2.Validate(); err == nil {
		t.Error("Expected validation error for missing size")
	}

	// Missing trigger price
	order3 := TriggerOrder().StopLoss("BTC").Size(0.01)
	if err := order3.Validate(); err == nil {
		t.Error("Expected validation error for missing trigger price")
	}

	// Valid order
	order4 := TriggerOrder().StopLoss("BTC").Size(0.01).TriggerPrice(60000).Market()
	if err := order4.Validate(); err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}
