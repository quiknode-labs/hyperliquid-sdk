package hyperliquid

import (
	"os"
	"testing"
	"time"
)

// Integration tests that require a real endpoint.
// Run with: go test -v -tags=integration ./hyperliquid/...
// Or with: ENDPOINT=https://... go test -v ./hyperliquid/...

func getTestEndpoint() string {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		ep = os.Getenv("QUICKNODE_ENDPOINT")
	}
	return ep
}

func skipIfNoEndpoint(t *testing.T) string {
	ep := getTestEndpoint()
	if ep == "" {
		t.Skip("ENDPOINT or QUICKNODE_ENDPOINT not set, skipping integration test")
	}
	return ep
}

// Test Info API - AllMids
func TestInfoAllMids(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	info := sdk.Info()
	mids, err := info.AllMids()
	if err != nil {
		// Check if geo-blocked
		if IsErrorCode(err, ErrorCodeGeoBlocked) {
			t.Skip("Geo-blocked, skipping test")
		}
		t.Fatalf("AllMids failed: %v", err)
	}

	if len(mids) == 0 {
		t.Error("Expected non-empty mids map")
	}

	if _, ok := mids["BTC"]; !ok {
		t.Error("Expected BTC in mids")
	}

	if _, ok := mids["ETH"]; !ok {
		t.Error("Expected ETH in mids")
	}

	t.Logf("Got %d markets", len(mids))
}

// Test Info API - Meta
func TestInfoMeta(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	info := sdk.Info()
	meta, err := info.Meta()
	if err != nil {
		t.Fatalf("Meta failed: %v", err)
	}

	universe, ok := meta["universe"].([]any)
	if !ok {
		t.Error("Expected universe in meta")
	}

	if len(universe) < 100 {
		t.Errorf("Expected at least 100 markets, got %d", len(universe))
	}

	t.Logf("Got %d perp markets", len(universe))
}

// Test Info API - L2Book
func TestInfoL2Book(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	info := sdk.Info()
	book, err := info.L2Book("BTC")
	if err != nil {
		if IsErrorCode(err, ErrorCodeGeoBlocked) {
			t.Skip("Geo-blocked, skipping test")
		}
		t.Fatalf("L2Book failed: %v", err)
	}

	levels, ok := book["levels"].([]any)
	if !ok || len(levels) < 2 {
		t.Error("Expected levels with bids and asks")
	}

	bids := levels[0].([]any)
	asks := levels[1].([]any)

	if len(bids) == 0 {
		t.Error("Expected non-empty bids")
	}
	if len(asks) == 0 {
		t.Error("Expected non-empty asks")
	}

	t.Logf("Got %d bids, %d asks", len(bids), len(asks))
}

// Test Info API - RecentTrades
func TestInfoRecentTrades(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	info := sdk.Info()
	trades, err := info.RecentTrades("BTC")
	if err != nil {
		if IsErrorCode(err, ErrorCodeGeoBlocked) {
			t.Skip("Geo-blocked, skipping test")
		}
		t.Fatalf("RecentTrades failed: %v", err)
	}

	if len(trades) == 0 {
		t.Error("Expected non-empty trades")
	}

	trade := trades[0].(map[string]any)
	if _, ok := trade["px"]; !ok {
		t.Error("Expected px in trade")
	}
	if _, ok := trade["sz"]; !ok {
		t.Error("Expected sz in trade")
	}

	t.Logf("Got %d trades", len(trades))
}

// Test HyperCore API
func TestHyperCore(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	core := sdk.Core()

	// Get latest block number
	blockNum, err := core.LatestBlockNumber()
	if err != nil {
		t.Fatalf("LatestBlockNumber failed: %v", err)
	}

	if blockNum == 0 {
		t.Error("Expected non-zero block number")
	}

	t.Logf("Latest block: %d", blockNum)

	// Get block details
	block, err := core.GetBlock(blockNum)
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}

	if block == nil {
		t.Error("Expected non-nil block")
	}

	t.Logf("Block %d retrieved", blockNum)
}

// Test HyperCore API - LatestTrades
func TestHyperCoreLatestTrades(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	core := sdk.Core()
	trades, err := core.LatestTrades(5, "BTC")
	if err != nil {
		t.Fatalf("LatestTrades failed: %v", err)
	}

	if len(trades) == 0 {
		t.Skip("No trades returned")
	}

	t.Logf("Got %d trades", len(trades))
}

// Test EVM API
func TestEVM(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	evm := sdk.EVM()

	// Get chain ID
	chainID, err := evm.ChainID()
	if err != nil {
		t.Fatalf("ChainID failed: %v", err)
	}

	// Mainnet = 999, Testnet = 998
	if chainID != 999 && chainID != 998 {
		t.Errorf("Expected chain ID 999 or 998, got %d", chainID)
	}

	t.Logf("Chain ID: %d", chainID)

	// Get block number
	blockNum, err := evm.BlockNumber()
	if err != nil {
		t.Fatalf("BlockNumber failed: %v", err)
	}

	if blockNum == 0 {
		t.Error("Expected non-zero block number")
	}

	t.Logf("EVM block: %d", blockNum)

	// Get gas price
	gasPrice, err := evm.GasPrice()
	if err != nil {
		t.Fatalf("GasPrice failed: %v", err)
	}

	t.Logf("Gas price: %d wei", gasPrice)
}

// Test WebSocket streaming
func TestWebSocketStreaming(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	tradesReceived := 0
	var states []ConnectionState
	var streamErr error

	stream := NewStream(endpoint, &StreamConfig{
		Reconnect: false,
		OnStateChange: func(state ConnectionState) {
			states = append(states, state)
		},
		OnError: func(err error) {
			streamErr = err
		},
	})

	stream.Trades([]string{"BTC"}, func(data map[string]any) {
		tradesReceived++
	})

	if err := stream.Start(); err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}

	// Wait up to 10 seconds for trades or error
	start := time.Now()
	for time.Since(start) < 10*time.Second {
		if tradesReceived > 0 || streamErr != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	stream.Stop()

	// Should have at least tried to connect
	hasConnecting := false
	for _, s := range states {
		if s == ConnectionStateConnecting {
			hasConnecting = true
			break
		}
	}

	if !hasConnecting {
		t.Error("Expected CONNECTING state")
	}

	if tradesReceived > 0 {
		t.Logf("Received %d trades via WebSocket", tradesReceived)
	} else if streamErr != nil {
		t.Logf("WebSocket error (may be geo-blocked): %v", streamErr)
	} else {
		t.Log("No trades received within timeout")
	}
}

// Test gRPC streaming initialization
func TestGRPCStreamInitialization(t *testing.T) {
	endpoint := "https://test.quiknode.pro/TOKEN"

	stream := NewGRPCStream(endpoint, &GRPCStreamConfig{
		Reconnect: false,
	})

	// Chain subscriptions
	stream.Trades([]string{"BTC"}, func(data map[string]any) {})
	stream.Orders([]string{"ETH"}, func(data map[string]any) {})
	stream.Blocks(func(data map[string]any) {})
	stream.L2Book("BTC", func(data map[string]any) {})

	// Verify subscriptions were added
	if len(stream.subscriptions) != 4 {
		t.Errorf("Expected 4 subscriptions, got %d", len(stream.subscriptions))
	}
}

// Test SDK GetMid
func TestSDKGetMid(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	mid, err := sdk.GetMid("BTC")
	if err != nil {
		if IsErrorCode(err, ErrorCodeGeoBlocked) {
			t.Skip("Geo-blocked, skipping test")
		}
		t.Fatalf("GetMid failed: %v", err)
	}

	if mid <= 0 {
		t.Error("Expected positive mid price")
	}

	t.Logf("BTC mid: $%.2f", mid)
}

// Test SDK Markets
func TestSDKMarkets(t *testing.T) {
	endpoint := skipIfNoEndpoint(t)

	sdk, err := New(endpoint)
	if err != nil {
		t.Fatalf("Failed to create SDK: %v", err)
	}

	markets, err := sdk.Markets()
	if err != nil {
		t.Fatalf("Markets failed: %v", err)
	}

	if len(markets.Perps) == 0 {
		t.Error("Expected non-empty perps")
	}

	t.Logf("Got %d perps, %d spot markets", len(markets.Perps), len(markets.Spot))
}
