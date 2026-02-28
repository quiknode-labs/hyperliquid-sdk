package hyperliquid

import (
	"fmt"
)

// Order is a fluent order builder.
//
// Examples:
//
//	// Simple limit buy
//	Order().Buy("BTC").Size(0.001).Price(67000).GTC()
//
//	// Market sell by notional
//	Order().Sell("ETH").Notional(500).Market()
//
//	// Post-only with reduce_only
//	Order().Buy("BTC").Size(0.01).Price(65000).ALO().ReduceOnly()
type OrderBuilder struct {
	asset      string
	side       Side
	size       string
	price      string
	tif        TIF
	reduceOnly bool
	notional   float64
	cloid      string
}

// Order creates a new order builder.
func Order() *OrderBuilder {
	return &OrderBuilder{
		tif: TIFIOC, // Default to IOC
	}
}

// Buy creates a buy order for the given asset.
func (o *OrderBuilder) Buy(asset string) *OrderBuilder {
	o.asset = asset
	o.side = SideBuy
	return o
}

// Sell creates a sell order for the given asset.
func (o *OrderBuilder) Sell(asset string) *OrderBuilder {
	o.asset = asset
	o.side = SideSell
	return o
}

// Long is an alias for Buy (perps terminology).
func (o *OrderBuilder) Long(asset string) *OrderBuilder {
	return o.Buy(asset)
}

// Short is an alias for Sell (perps terminology).
func (o *OrderBuilder) Short(asset string) *OrderBuilder {
	return o.Sell(asset)
}

// Size sets the order size in asset units.
func (o *OrderBuilder) Size(size any) *OrderBuilder {
	o.size = NewDecimal(size).String()
	return o
}

// Notional sets the order size by USD notional value.
// SDK will calculate size based on current price.
func (o *OrderBuilder) Notional(usd float64) *OrderBuilder {
	o.notional = usd
	return o
}

// Price sets the limit price.
func (o *OrderBuilder) Price(price any) *OrderBuilder {
	o.price = NewDecimal(price).String()
	return o
}

// Limit is an alias for Price.
func (o *OrderBuilder) Limit(price any) *OrderBuilder {
	return o.Price(price)
}

// TIF sets the time in force.
func (o *OrderBuilder) TIF(tif TIF) *OrderBuilder {
	o.tif = tif
	return o
}

// IOC sets time in force to Immediate or Cancel.
func (o *OrderBuilder) IOC() *OrderBuilder {
	o.tif = TIFIOC
	return o
}

// GTC sets time in force to Good Till Cancelled.
func (o *OrderBuilder) GTC() *OrderBuilder {
	o.tif = TIFGTC
	return o
}

// ALO sets time in force to Add Liquidity Only (post-only, maker only).
func (o *OrderBuilder) ALO() *OrderBuilder {
	o.tif = TIFALO
	return o
}

// Market sets the order as a market order (price computed automatically).
func (o *OrderBuilder) Market() *OrderBuilder {
	o.tif = TIFMarket
	o.price = ""
	return o
}

// ReduceOnly marks the order as reduce-only.
func (o *OrderBuilder) ReduceOnly() *OrderBuilder {
	o.reduceOnly = true
	return o
}

// CLOID sets the client order ID for tracking.
func (o *OrderBuilder) CLOID(cloid string) *OrderBuilder {
	o.cloid = cloid
	return o
}

// Asset returns the order's asset.
func (o *OrderBuilder) Asset() string {
	return o.asset
}

// GetSide returns the order's side.
func (o *OrderBuilder) GetSide() Side {
	return o.side
}

// GetSize returns the order's size.
func (o *OrderBuilder) GetSize() string {
	return o.size
}

// GetPrice returns the order's price.
func (o *OrderBuilder) GetPrice() string {
	return o.price
}

// GetTIF returns the order's time in force.
func (o *OrderBuilder) GetTIF() TIF {
	return o.tif
}

// GetNotional returns the order's notional value.
func (o *OrderBuilder) GetNotional() float64 {
	return o.notional
}

// IsReduceOnly returns whether the order is reduce-only.
func (o *OrderBuilder) IsReduceOnly() bool {
	return o.reduceOnly
}

// SetSize sets the computed size (used internally for notional orders).
func (o *OrderBuilder) SetSize(size string) {
	o.size = size
}

// Validate validates the order before sending.
func (o *OrderBuilder) Validate() error {
	if o.asset == "" {
		return ValidationError("asset is required")
	}

	if o.size == "" && o.notional == 0 {
		return ValidationError("either size or notional is required").
			WithGuidance("Use .Size(0.001) or .Notional(100)")
	}

	if o.size != "" {
		sizeVal := NewDecimal(o.size).Float64()
		if sizeVal <= 0 {
			return ValidationError(fmt.Sprintf("size must be positive, got %s", o.size))
		}
	}

	if o.notional != 0 && o.notional <= 0 {
		return ValidationError(fmt.Sprintf("notional must be positive, got %f", o.notional))
	}

	if o.tif != TIFMarket && o.price == "" && o.notional == 0 {
		return ValidationError("price is required for limit orders").
			WithGuidance("Use .Price(67000) or .Market() for market orders")
	}

	if o.price != "" {
		priceVal := NewDecimal(o.price).Float64()
		if priceVal <= 0 {
			return ValidationError(fmt.Sprintf("price must be positive, got %s", o.price))
		}
	}

	return nil
}

// ToAction converts the order to an API action format.
func (o *OrderBuilder) ToAction() map[string]any {
	orderSpec := map[string]any{
		"asset": o.asset,
		"side":  string(o.side),
		"size":  o.size,
	}

	if o.tif == TIFMarket {
		orderSpec["tif"] = "market"
	} else {
		if o.price != "" {
			orderSpec["price"] = o.price
		}
		orderSpec["tif"] = string(o.tif)
	}

	if o.reduceOnly {
		orderSpec["reduceOnly"] = true
	}

	if o.cloid != "" {
		orderSpec["cloid"] = o.cloid
	}

	return map[string]any{
		"type":   "order",
		"orders": []any{orderSpec},
	}
}

// String returns a string representation of the order.
func (o *OrderBuilder) String() string {
	s := fmt.Sprintf("Order.%s(\"%s\")", o.side, o.asset)
	if o.size != "" {
		s += fmt.Sprintf(".Size(%s)", o.size)
	}
	if o.notional > 0 {
		s += fmt.Sprintf(".Notional(%f)", o.notional)
	}
	if o.price != "" {
		s += fmt.Sprintf(".Price(%s)", o.price)
	}
	if o.tif != TIFIOC {
		s += fmt.Sprintf(".%s()", o.tif)
	}
	if o.reduceOnly {
		s += ".ReduceOnly()"
	}
	return s
}
