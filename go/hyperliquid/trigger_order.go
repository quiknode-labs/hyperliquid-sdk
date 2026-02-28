package hyperliquid

import (
	"fmt"
)

// TriggerOrderBuilder is a fluent builder for stop-loss and take-profit orders.
//
// Examples:
//
//	// Stop loss: sell when price drops to 60000 (market)
//	TriggerOrder().StopLoss("BTC").Size(0.001).TriggerPrice(60000).Market()
//
//	// Stop loss: sell at limit 59900 when price drops to 60000
//	TriggerOrder().StopLoss("BTC").Size(0.001).TriggerPrice(60000).Limit(59900)
//
//	// Take profit: sell when price rises to 80000
//	TriggerOrder().TakeProfit("BTC").Size(0.001).TriggerPrice(80000).Market()
type TriggerOrderBuilder struct {
	asset      string
	tpsl       TpSl
	side       Side
	size       string
	triggerPx  string
	limitPx    string
	isMarket   bool
	reduceOnly bool
	cloid      string
}

// TriggerOrder creates a new trigger order builder.
func TriggerOrder() *TriggerOrderBuilder {
	return &TriggerOrderBuilder{
		isMarket: true, // Default to market execution
	}
}

// StopLoss creates a stop-loss trigger order.
//
// Stop loss triggers when price moves against your position:
//   - For longs: triggers when price FALLS to trigger_price (sell to exit)
//   - For shorts: triggers when price RISES to trigger_price (buy to exit)
func (t *TriggerOrderBuilder) StopLoss(asset string) *TriggerOrderBuilder {
	t.asset = asset
	t.tpsl = TpSlSL
	t.side = SideSell // Default side for closing longs
	return t
}

// TakeProfit creates a take-profit trigger order.
//
// Take profit triggers when price moves in favor of your position:
//   - For longs: triggers when price RISES to trigger_price (sell to take profits)
//   - For shorts: triggers when price FALLS to trigger_price (buy to take profits)
func (t *TriggerOrderBuilder) TakeProfit(asset string) *TriggerOrderBuilder {
	t.asset = asset
	t.tpsl = TpSlTP
	t.side = SideSell // Default side for closing longs
	return t
}

// SL is an alias for StopLoss.
func (t *TriggerOrderBuilder) SL(asset string) *TriggerOrderBuilder {
	return t.StopLoss(asset)
}

// TP is an alias for TakeProfit.
func (t *TriggerOrderBuilder) TP(asset string) *TriggerOrderBuilder {
	return t.TakeProfit(asset)
}

// Side sets the order side when triggered (default: SELL for closing longs).
func (t *TriggerOrderBuilder) Side(side Side) *TriggerOrderBuilder {
	t.side = side
	return t
}

// Size sets the order size in asset units.
func (t *TriggerOrderBuilder) Size(size any) *TriggerOrderBuilder {
	t.size = NewDecimal(size).String()
	return t
}

// TriggerPrice sets the price at which the order activates.
func (t *TriggerOrderBuilder) TriggerPrice(price any) *TriggerOrderBuilder {
	t.triggerPx = NewDecimal(price).String()
	return t
}

// Trigger is an alias for TriggerPrice.
func (t *TriggerOrderBuilder) Trigger(price any) *TriggerOrderBuilder {
	return t.TriggerPrice(price)
}

// Market sets the order to execute as a market order when triggered.
func (t *TriggerOrderBuilder) Market() *TriggerOrderBuilder {
	t.isMarket = true
	t.limitPx = ""
	return t
}

// Limit sets the order to execute as a limit order when triggered.
func (t *TriggerOrderBuilder) Limit(price any) *TriggerOrderBuilder {
	t.isMarket = false
	t.limitPx = NewDecimal(price).String()
	return t
}

// ReduceOnly marks the order as reduce-only (close position only).
func (t *TriggerOrderBuilder) ReduceOnly() *TriggerOrderBuilder {
	t.reduceOnly = true
	return t
}

// CLOID sets the client order ID for tracking.
func (t *TriggerOrderBuilder) CLOID(cloid string) *TriggerOrderBuilder {
	t.cloid = cloid
	return t
}

// Asset returns the order's asset.
func (t *TriggerOrderBuilder) Asset() string {
	return t.asset
}

// GetSide returns the order's side.
func (t *TriggerOrderBuilder) GetSide() Side {
	return t.side
}

// GetSize returns the order's size.
func (t *TriggerOrderBuilder) GetSize() string {
	return t.size
}

// GetTriggerPrice returns the order's trigger price.
func (t *TriggerOrderBuilder) GetTriggerPrice() string {
	return t.triggerPx
}

// GetLimitPrice returns the order's limit price.
func (t *TriggerOrderBuilder) GetLimitPrice() string {
	return t.limitPx
}

// Validate validates the trigger order before sending.
func (t *TriggerOrderBuilder) Validate() error {
	if t.asset == "" {
		return ValidationError("asset is required")
	}

	if t.size == "" {
		return ValidationError("size is required for trigger orders").
			WithGuidance("Use .Size(0.001) to set the order size")
	}

	sizeVal := NewDecimal(t.size).Float64()
	if sizeVal <= 0 {
		return ValidationError(fmt.Sprintf("size must be positive, got %s", t.size))
	}

	if t.triggerPx == "" {
		return ValidationError("trigger price is required").
			WithGuidance("Use .TriggerPrice(60000) to set when the order activates")
	}

	triggerVal := NewDecimal(t.triggerPx).Float64()
	if triggerVal <= 0 {
		return ValidationError(fmt.Sprintf("trigger price must be positive, got %s", t.triggerPx))
	}

	if !t.isMarket {
		if t.limitPx == "" {
			return ValidationError("limit price is required for limit trigger orders").
				WithGuidance("Use .Limit(59900) or .Market() for market execution")
		}
		limitVal := NewDecimal(t.limitPx).Float64()
		if limitVal <= 0 {
			return ValidationError(fmt.Sprintf("limit price must be positive, got %s", t.limitPx))
		}
	}

	return nil
}

// ToAction converts the trigger order to an API action format.
func (t *TriggerOrderBuilder) ToAction(grouping OrderGrouping) map[string]any {
	// For trigger orders, limit_px is always required by the API
	// For market orders, we use trigger_px as a placeholder
	limitPx := t.limitPx
	if t.isMarket {
		limitPx = t.triggerPx
	}

	orderSpec := map[string]any{
		"a": t.asset,                // Asset
		"b": t.side == SideBuy,      // is_buy
		"p": limitPx,                // limit_px
		"s": t.size,                 // size
		"r": t.reduceOnly,           // reduce_only
		"t": map[string]any{
			"trigger": map[string]any{
				"isMarket":  t.isMarket,
				"triggerPx": t.triggerPx,
				"tpsl":      string(t.tpsl),
			},
		},
	}

	if t.cloid != "" {
		orderSpec["c"] = t.cloid
	}

	return map[string]any{
		"type":     "order",
		"orders":   []any{orderSpec},
		"grouping": string(grouping),
	}
}

// String returns a string representation of the trigger order.
func (t *TriggerOrderBuilder) String() string {
	name := "stop_loss"
	if t.tpsl == TpSlTP {
		name = "take_profit"
	}
	s := fmt.Sprintf("TriggerOrder.%s(\"%s\")", name, t.asset)
	if t.size != "" {
		s += fmt.Sprintf(".Size(%s)", t.size)
	}
	if t.triggerPx != "" {
		s += fmt.Sprintf(".TriggerPrice(%s)", t.triggerPx)
	}
	if t.isMarket {
		s += ".Market()"
	} else if t.limitPx != "" {
		s += fmt.Sprintf(".Limit(%s)", t.limitPx)
	}
	if t.reduceOnly {
		s += ".ReduceOnly()"
	}
	return s
}
