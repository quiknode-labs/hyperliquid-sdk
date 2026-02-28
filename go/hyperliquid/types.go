// Package hyperliquid provides a Go SDK for the Hyperliquid exchange.
//
// The SDK provides unified access to all Hyperliquid APIs through a single client:
//   - Trading: Place orders, cancel, modify, close positions
//   - Info: Market data, positions, orders, vaults
//   - HyperCore: Block data, trades, orders from the L1
//   - EVM: Ethereum JSON-RPC for HyperEVM
//   - Streaming: Real-time data via WebSocket and gRPC
//
// Example:
//
//	sdk, err := hyperliquid.New("https://your-endpoint.quiknode.pro/TOKEN",
//		hyperliquid.WithPrivateKey("0x..."),
//		hyperliquid.WithAutoApprove(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer sdk.Close()
//
//	// Place a market buy
//	order, err := sdk.MarketBuy("BTC", hyperliquid.WithSize(0.001))
package hyperliquid

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Side represents the order side (buy or sell).
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"

	// Aliases for perp traders
	SideLong  Side = "buy"
	SideShort Side = "sell"
)

// TIF represents time in force for orders.
type TIF string

const (
	TIFIOC    TIF = "ioc"    // Immediate or cancel
	TIFGTC    TIF = "gtc"    // Good till cancelled
	TIFALO    TIF = "alo"    // Add liquidity only (post-only)
	TIFMarket TIF = "market" // Market order (auto-price)
)

// TpSl represents trigger order type (take profit or stop loss).
type TpSl string

const (
	TpSlTP TpSl = "tp" // Take profit
	TpSlSL TpSl = "sl" // Stop loss
)

// OrderGrouping represents order grouping for TP/SL attachment.
type OrderGrouping string

const (
	OrderGroupingNA         OrderGrouping = "na"           // No grouping (standalone order)
	OrderGroupingNormalTPSL OrderGrouping = "normalTpsl"   // Attach TP/SL to the fill of this order
	OrderGroupingPositionTPSL OrderGrouping = "positionTpsl" // Attach TP/SL to the entire position
)

// StreamType represents available WebSocket stream types.
type StreamType string

const (
	// QuickNode-supported streams
	StreamTypeTrades       StreamType = "trades"
	StreamTypeOrders       StreamType = "orders"
	StreamTypeBookUpdates  StreamType = "book_updates"
	StreamTypeTWAP         StreamType = "twap"
	StreamTypeEvents       StreamType = "events"
	StreamTypeWriterActions StreamType = "writer_actions"

	// Public Hyperliquid API streams
	StreamTypeL2Book                StreamType = "l2Book"
	StreamTypeAllMids               StreamType = "allMids"
	StreamTypeCandle                StreamType = "candle"
	StreamTypeBBO                   StreamType = "bbo"
	StreamTypeOpenOrders            StreamType = "openOrders"
	StreamTypeOrderUpdates          StreamType = "orderUpdates"
	StreamTypeUserEvents            StreamType = "userEvents"
	StreamTypeUserFills             StreamType = "userFills"
	StreamTypeUserFundings          StreamType = "userFundings"
	StreamTypeUserNonFundingLedger  StreamType = "userNonFundingLedgerUpdates"
	StreamTypeClearinghouseState    StreamType = "clearinghouseState"
	StreamTypeActiveAssetCtx        StreamType = "activeAssetCtx"
	StreamTypeActiveAssetData       StreamType = "activeAssetData"
	StreamTypeTWAPStates            StreamType = "twapStates"
	StreamTypeUserTWAPSliceFills    StreamType = "userTwapSliceFills"
	StreamTypeUserTWAPHistory       StreamType = "userTwapHistory"
	StreamTypeNotification          StreamType = "notification"
	StreamTypeWebData3              StreamType = "webData3"
)

// GRPCStreamType represents available gRPC stream types.
type GRPCStreamType string

const (
	GRPCStreamTypeTrades       GRPCStreamType = "TRADES"
	GRPCStreamTypeOrders       GRPCStreamType = "ORDERS"
	GRPCStreamTypeBookUpdates  GRPCStreamType = "BOOK_UPDATES"
	GRPCStreamTypeTWAP         GRPCStreamType = "TWAP"
	GRPCStreamTypeEvents       GRPCStreamType = "EVENTS"
	GRPCStreamTypeBlocks       GRPCStreamType = "BLOCKS"
	GRPCStreamTypeWriterActions GRPCStreamType = "WRITER_ACTIONS"
)

// ConnectionState represents WebSocket/gRPC connection states.
type ConnectionState string

const (
	ConnectionStateDisconnected ConnectionState = "disconnected"
	ConnectionStateConnecting   ConnectionState = "connecting"
	ConnectionStateConnected    ConnectionState = "connected"
	ConnectionStateReconnecting ConnectionState = "reconnecting"
)

// PlacedOrder represents a successfully placed order with full context.
type PlacedOrder struct {
	OID        int64   `json:"oid,omitempty"`
	Status     string  `json:"status"`
	Asset      string  `json:"asset"`
	Side       string  `json:"side"`
	Size       string  `json:"size"`
	Price      string  `json:"price,omitempty"`
	FilledSize string  `json:"filled_size,omitempty"`
	AvgPrice   string  `json:"avg_price,omitempty"`
	RawResponse map[string]any `json:"raw_response,omitempty"`

	// Internal reference to SDK for cancel/modify operations
	sdk *SDK
}

// Cancel cancels this order.
func (o *PlacedOrder) Cancel() (map[string]any, error) {
	if o.sdk == nil {
		return nil, fmt.Errorf("order not linked to SDK")
	}
	if o.OID == 0 {
		return nil, fmt.Errorf("no OID to cancel")
	}
	return o.sdk.Cancel(o.OID, o.Asset)
}

// Modify modifies this order's price and/or size.
func (o *PlacedOrder) Modify(price, size string) (*PlacedOrder, error) {
	if o.sdk == nil {
		return nil, fmt.Errorf("order not linked to SDK")
	}
	if o.OID == 0 {
		return nil, fmt.Errorf("no OID to modify")
	}
	if price == "" {
		price = o.Price
	}
	if size == "" {
		size = o.Size
	}
	return o.sdk.Modify(o.OID, o.Asset, o.Side, price, size)
}

// IsResting returns true if the order is resting on the order book.
func (o *PlacedOrder) IsResting() bool {
	return o.Status == "resting"
}

// IsFilled returns true if the order is completely filled.
func (o *PlacedOrder) IsFilled() bool {
	return o.Status == "filled"
}

// IsError returns true if the order encountered an error.
func (o *PlacedOrder) IsError() bool {
	return strings.HasPrefix(o.Status, "error")
}

// ParsePlacedOrder parses an exchange response into a PlacedOrder.
func ParsePlacedOrder(response map[string]any, asset string, side Side, size, price string, sdk *SDK) *PlacedOrder {
	order := &PlacedOrder{
		Asset:       asset,
		Side:        string(side),
		Size:        size,
		Price:       price,
		Status:      "unknown",
		RawResponse: response,
		sdk:         sdk,
	}

	// Navigate to statuses
	if resp, ok := response["response"].(map[string]any); ok {
		if data, ok := resp["data"].(map[string]any); ok {
			if statuses, ok := data["statuses"].([]any); ok && len(statuses) > 0 {
				s := statuses[0]
				switch v := s.(type) {
				case map[string]any:
					if resting, ok := v["resting"].(map[string]any); ok {
						if oid, ok := resting["oid"].(float64); ok {
							order.OID = int64(oid)
						}
						order.Status = "resting"
					} else if filled, ok := v["filled"].(map[string]any); ok {
						if oid, ok := filled["oid"].(float64); ok {
							order.OID = int64(oid)
						}
						order.Status = "filled"
						if totalSz, ok := filled["totalSz"].(string); ok {
							order.FilledSize = totalSz
						}
						if avgPx, ok := filled["avgPx"].(string); ok {
							order.AvgPrice = avgPx
						}
					} else if errMsg, ok := v["error"].(string); ok {
						order.Status = "error: " + errMsg
					}
				case string:
					if v == "success" {
						order.Status = "success"
					}
				}
			}
		}
	}

	return order
}

// Market represents market metadata.
type Market struct {
	Name       string `json:"name"`
	Index      int    `json:"index"`
	SzDecimals int    `json:"szDecimals"`
	IsSpot     bool   `json:"isSpot,omitempty"`
	Dex        string `json:"dex,omitempty"`
}

// Markets represents all available markets.
type Markets struct {
	Perps []Market            `json:"perps"`
	Spot  []Market            `json:"spot"`
	HIP3  map[string][]Market `json:"hip3"`
}

// Signature represents an ECDSA signature.
type Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

// BuildResponse represents the response from a build request.
type BuildResponse struct {
	Hash   string         `json:"hash"`
	Nonce  int64          `json:"nonce"`
	Action map[string]any `json:"action"`
}

// ExchangeResponse represents the response from an exchange request.
type ExchangeResponse struct {
	Success          bool           `json:"success,omitempty"`
	User             string         `json:"user,omitempty"`
	ExchangeResponse map[string]any `json:"exchangeResponse,omitempty"`
}

// L2BookLevel represents a level in the L2 order book.
type L2BookLevel struct {
	Price string `json:"px"`
	Size  string `json:"sz"`
	Count int    `json:"n"`
}

// L2Book represents the L2 order book.
type L2Book struct {
	Coin        string        `json:"coin"`
	Time        int64         `json:"time"`
	BlockNumber int64         `json:"block_number"`
	Bids        []L2BookLevel `json:"bids"`
	Asks        []L2BookLevel `json:"asks"`
}

// L4Order represents an individual order in the L4 order book.
type L4Order struct {
	User             string  `json:"user"`
	Coin             string  `json:"coin"`
	Side             string  `json:"side"`
	LimitPx          string  `json:"limit_px"`
	Size             string  `json:"sz"`
	OID              int64   `json:"oid"`
	Timestamp        int64   `json:"timestamp"`
	TriggerCondition string  `json:"trigger_condition,omitempty"`
	IsTrigger        bool    `json:"is_trigger,omitempty"`
	TriggerPx        string  `json:"trigger_px,omitempty"`
	IsPositionTPSL   bool    `json:"is_position_tpsl,omitempty"`
	ReduceOnly       bool    `json:"reduce_only,omitempty"`
	OrderType        string  `json:"order_type,omitempty"`
	TIF              string  `json:"tif,omitempty"`
	CLOID            string  `json:"cloid,omitempty"`
}

// Decimal is a helper for precise decimal string handling.
type Decimal string

// NewDecimal creates a Decimal from various types.
func NewDecimal(v any) Decimal {
	switch val := v.(type) {
	case string:
		return Decimal(val)
	case float64:
		return Decimal(strconv.FormatFloat(val, 'f', -1, 64))
	case float32:
		return Decimal(strconv.FormatFloat(float64(val), 'f', -1, 32))
	case int:
		return Decimal(strconv.Itoa(val))
	case int64:
		return Decimal(strconv.FormatInt(val, 10))
	case Decimal:
		return val
	default:
		return Decimal(fmt.Sprintf("%v", v))
	}
}

// String returns the decimal as a string.
func (d Decimal) String() string {
	return string(d)
}

// Float64 returns the decimal as a float64.
func (d Decimal) Float64() float64 {
	f, _ := strconv.ParseFloat(string(d), 64)
	return f
}

// MarshalJSON implements json.Marshaler.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(d))
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try as number
		var f float64
		if err := json.Unmarshal(data, &f); err != nil {
			return err
		}
		*d = NewDecimal(f)
		return nil
	}
	*d = Decimal(s)
	return nil
}
