package hyperliquid

import (
	"testing"

	pb "github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid/proto"
)

// Test that the new StreamType enum values match the proto contract.
func TestGRPCStreamTypeEnumValues(t *testing.T) {
	if pb.StreamType_MEMPOOL_TXS != 8 {
		t.Errorf("StreamType_MEMPOOL_TXS = %d, want 8", pb.StreamType_MEMPOOL_TXS)
	}
	if pb.StreamType_ORDER_PRIORITY != 9 {
		t.Errorf("StreamType_ORDER_PRIORITY = %d, want 9", pb.StreamType_ORDER_PRIORITY)
	}
	if pb.StreamType_GOSSIP_PRIORITY != 10 {
		t.Errorf("StreamType_GOSSIP_PRIORITY = %d, want 10", pb.StreamType_GOSSIP_PRIORITY)
	}
}

// Test the stream type name -> proto enum map, including the new types.
func TestGRPCStreamTypeMap(t *testing.T) {
	want := map[string]pb.StreamType{
		"TRADES":          pb.StreamType_TRADES,
		"ORDERS":          pb.StreamType_ORDERS,
		"BOOK_UPDATES":    pb.StreamType_BOOK_UPDATES,
		"TWAP":            pb.StreamType_TWAP,
		"EVENTS":          pb.StreamType_EVENTS,
		"BLOCKS":          pb.StreamType_BLOCKS,
		"WRITER_ACTIONS":  pb.StreamType_WRITER_ACTIONS,
		"MEMPOOL_TXS":     pb.StreamType_MEMPOOL_TXS,
		"ORDER_PRIORITY":  pb.StreamType_ORDER_PRIORITY,
		"GOSSIP_PRIORITY": pb.StreamType_GOSSIP_PRIORITY,
	}
	if len(grpcStreamTypeMap) != len(want) {
		t.Errorf("grpcStreamTypeMap has %d entries, want %d", len(grpcStreamTypeMap), len(want))
	}
	for name, streamType := range want {
		if got, ok := grpcStreamTypeMap[name]; !ok || got != streamType {
			t.Errorf("grpcStreamTypeMap[%q] = %v (present=%v), want %v", name, got, ok, streamType)
		}
	}
}

func noopCallback(map[string]any) {}

// Test that the new generic stream helpers register the expected subscriptions.
func TestGRPCNewGenericSubscriptions(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)

	s.MempoolTxs([]string{"BTC", "ETH"}, noopCallback)
	s.RawMempoolTxs(nil, noopCallback)
	s.OrderPriority(noopCallback)
	s.RawOrderPriority(noopCallback)
	s.GossipPriority(noopCallback)
	s.RawGossipPriority(noopCallback)

	subs := s.subscriptions
	if len(subs) != 6 {
		t.Fatalf("got %d subscriptions, want 6", len(subs))
	}

	if subs[0].streamType != "MEMPOOL_TXS" || subs[0].raw {
		t.Errorf("MempoolTxs registered %q raw=%v, want MEMPOOL_TXS raw=false", subs[0].streamType, subs[0].raw)
	}
	if len(subs[0].coins) != 2 || subs[0].coins[0] != "BTC" {
		t.Errorf("MempoolTxs coins = %v, want [BTC ETH]", subs[0].coins)
	}
	if subs[1].streamType != "MEMPOOL_TXS" || !subs[1].raw || len(subs[1].coins) != 0 {
		t.Errorf("RawMempoolTxs registered %q raw=%v coins=%v", subs[1].streamType, subs[1].raw, subs[1].coins)
	}
	if subs[2].streamType != "ORDER_PRIORITY" || subs[2].raw {
		t.Errorf("OrderPriority registered %q raw=%v", subs[2].streamType, subs[2].raw)
	}
	if subs[3].streamType != "ORDER_PRIORITY" || !subs[3].raw {
		t.Errorf("RawOrderPriority registered %q raw=%v", subs[3].streamType, subs[3].raw)
	}
	if subs[4].streamType != "GOSSIP_PRIORITY" || subs[4].raw {
		t.Errorf("GossipPriority registered %q raw=%v", subs[4].streamType, subs[4].raw)
	}
	if subs[5].streamType != "GOSSIP_PRIORITY" || !subs[5].raw {
		t.Errorf("RawGossipPriority registered %q raw=%v", subs[5].streamType, subs[5].raw)
	}
}

// Test that StreamWithStartBlock is recorded and plumbed into StreamSubscribe.
func TestGRPCStartBlockOption(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)
	s.Trades([]string{"BTC"}, noopCallback, StreamWithStartBlock(12345))
	s.MempoolTxs(nil, noopCallback, StreamWithStartBlock(999))

	if s.subscriptions[0].startBlock != 12345 {
		t.Errorf("Trades startBlock = %d, want 12345", s.subscriptions[0].startBlock)
	}
	if s.subscriptions[1].startBlock != 999 {
		t.Errorf("MempoolTxs startBlock = %d, want 999", s.subscriptions[1].startBlock)
	}

	req := buildSubscribeRequest(s.subscriptions[0], false, 0)
	sub := req.GetSubscribe()
	if sub == nil {
		t.Fatal("expected subscribe request")
	}
	if sub.StreamType != pb.StreamType_TRADES {
		t.Errorf("StreamType = %v, want TRADES", sub.StreamType)
	}
	if sub.StartBlock != 12345 {
		t.Errorf("StartBlock = %d, want 12345", sub.StartBlock)
	}
	if coins := sub.Filters["coin"]; coins == nil || len(coins.Values) != 1 || coins.Values[0] != "BTC" {
		t.Errorf("Filters[coin] = %v, want [BTC]", coins)
	}
}

// Test that OrdersWithOptions/RawOrdersWithOptions accept StreamOptions and
// plumb both user filters and startBlock into the subscribe request.
func TestGRPCOrdersWithOptionsStartBlock(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)
	s.OrdersWithOptions([]string{"BTC"}, []string{"0xabc"}, noopCallback, StreamWithStartBlock(4242))
	s.RawOrdersWithOptions(nil, nil, noopCallback, StreamWithStartBlock(777))

	req := buildSubscribeRequest(s.subscriptions[0], false, 0)
	sub := req.GetSubscribe()
	if sub == nil {
		t.Fatal("expected subscribe request")
	}
	if sub.StreamType != pb.StreamType_ORDERS {
		t.Errorf("StreamType = %v, want ORDERS", sub.StreamType)
	}
	if sub.StartBlock != 4242 {
		t.Errorf("StartBlock = %d, want 4242", sub.StartBlock)
	}
	if users := sub.Filters["user"]; users == nil || len(users.Values) != 1 || users.Values[0] != "0xabc" {
		t.Errorf("Filters[user] = %v, want [0xabc]", users)
	}
	if !s.subscriptions[1].raw {
		t.Error("RawOrdersWithOptions subscription not marked raw")
	}
	if s.subscriptions[1].startBlock != 777 {
		t.Errorf("RawOrders startBlock = %d, want 777", s.subscriptions[1].startBlock)
	}

	// The legacy variadic signatures still work and stay option-free.
	s.Orders([]string{"ETH"}, noopCallback, "0xdef")
	if s.subscriptions[2].startBlock != 0 {
		t.Errorf("legacy Orders startBlock = %d, want 0", s.subscriptions[2].startBlock)
	}
}

// Test reconnect start_block semantics: the first connect sends the user's
// original startBlock, reconnects resume past the highest block already
// delivered, and an unset startBlock stays unset (tip-following).
func TestGRPCSubscribeRequestReconnectStartBlock(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)
	s.Trades([]string{"BTC"}, noopCallback, StreamWithStartBlock(1000))
	s.Trades([]string{"ETH"}, noopCallback) // no startBlock

	withStart := s.subscriptions[0]
	unset := s.subscriptions[1]

	// First connect uses the original startBlock.
	if got := buildSubscribeRequest(withStart, false, 0).GetSubscribe().StartBlock; got != 1000 {
		t.Errorf("first connect StartBlock = %d, want 1000", got)
	}

	// Reconnect after delivering blocks resumes past the last seen block.
	if got := buildSubscribeRequest(withStart, true, 5000).GetSubscribe().StartBlock; got != 5001 {
		t.Errorf("reconnect StartBlock = %d, want 5001", got)
	}

	// Reconnect before any block was delivered keeps the original.
	if got := buildSubscribeRequest(withStart, true, 0).GetSubscribe().StartBlock; got != 1000 {
		t.Errorf("reconnect (no data seen) StartBlock = %d, want 1000", got)
	}

	// Reconnect where last seen is still below the original keeps the original.
	if got := buildSubscribeRequest(withStart, true, 500).GetSubscribe().StartBlock; got != 1000 {
		t.Errorf("reconnect (behind original) StartBlock = %d, want 1000", got)
	}

	// Unset startBlock stays unset on reconnect — no cursor is introduced.
	if got := buildSubscribeRequest(unset, true, 5000).GetSubscribe().StartBlock; got != 0 {
		t.Errorf("reconnect (unset) StartBlock = %d, want 0", got)
	}
}

// Test that skip_initial_snapshot only applies to the first L2BookDiff
// connect: reconnect requests always ask for the snapshot to resync.
func TestGRPCL2BookDiffRequestReconnectSnapshot(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)
	s.L2BookDiff([]string{"BTC"}, noopCallback, L2BookNSigFigs(5), L2BookMantissa(2), L2BookSkipInitialSnapshot())
	s.L2BookDiff([]string{"ETH"}, noopCallback)

	skip := s.subscriptions[0]
	noSkip := s.subscriptions[1]

	// First connect honors the user's skipInitialSnapshot.
	if !buildL2BookDiffRequest(skip, false).SkipInitialSnapshot {
		t.Error("first connect SkipInitialSnapshot = false, want true")
	}
	if buildL2BookDiffRequest(noSkip, false).SkipInitialSnapshot {
		t.Error("first connect (no option) SkipInitialSnapshot = true, want false")
	}

	// Reconnects always request the snapshot.
	if buildL2BookDiffRequest(skip, true).SkipInitialSnapshot {
		t.Error("reconnect SkipInitialSnapshot = true, want false")
	}
	if buildL2BookDiffRequest(noSkip, true).SkipInitialSnapshot {
		t.Error("reconnect (no option) SkipInitialSnapshot = true, want false")
	}

	// Other request fields are still plumbed on reconnect.
	req := buildL2BookDiffRequest(skip, true)
	if len(req.Coins) != 1 || req.Coins[0] != "BTC" || req.NLevels != 20 {
		t.Errorf("reconnect request coins=%v nLevels=%d, want [BTC] 20", req.Coins, req.NLevels)
	}
	if req.NSigFigs == nil || *req.NSigFigs != 5 {
		t.Errorf("reconnect request NSigFigs = %v, want 5", req.NSigFigs)
	}
	if req.Mantissa == nil || *req.Mantissa != 2 {
		t.Errorf("reconnect request Mantissa = %v, want 2", req.Mantissa)
	}
}

// Test that the mempool coin filter uses the generic filters map.
func TestGRPCMempoolCoinFilter(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)
	s.MempoolTxs([]string{"BTC", "ETH"}, noopCallback)
	s.MempoolTxs(nil, noopCallback)

	req := buildSubscribeRequest(s.subscriptions[0], false, 0)
	sub := req.GetSubscribe()
	if sub.StreamType != pb.StreamType_MEMPOOL_TXS {
		t.Errorf("StreamType = %v, want MEMPOOL_TXS", sub.StreamType)
	}
	coins := sub.Filters["coin"]
	if coins == nil || len(coins.Values) != 2 || coins.Values[0] != "BTC" || coins.Values[1] != "ETH" {
		t.Errorf("Filters[coin] = %v, want [BTC ETH]", coins)
	}

	// No coins = unfiltered
	req = buildSubscribeRequest(s.subscriptions[1], false, 0)
	if _, ok := req.GetSubscribe().Filters["coin"]; ok {
		t.Error("expected no coin filter for unfiltered mempool subscription")
	}
}

// Test that the new orderbook helpers register the expected subscriptions.
func TestGRPCNewOrderbookSubscriptions(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)

	s.BboBook([]string{"BTC"}, noopCallback)
	s.L2BookDiff([]string{"BTC", "ETH"}, noopCallback, L2BookNLevels(50), L2BookNSigFigs(5), L2BookMantissa(2), L2BookSkipInitialSnapshot())
	s.L4BookUpdates(nil, noopCallback)
	s.TpslUpdates([]string{"SOL"}, noopCallback)
	s.L2BookPacked("BTC", noopCallback, L2BookNLevels(10))
	s.BboBookPacked(nil, noopCallback)
	s.L4BookBytes("ETH", noopCallback)

	subs := s.subscriptions
	if len(subs) != 7 {
		t.Fatalf("got %d subscriptions, want 7", len(subs))
	}

	if subs[0].streamType != "BBO_BOOK" || len(subs[0].coins) != 1 {
		t.Errorf("BboBook registered %q coins=%v", subs[0].streamType, subs[0].coins)
	}

	diff := subs[1]
	if diff.streamType != "L2_BOOK_DIFF" {
		t.Errorf("L2BookDiff streamType = %q", diff.streamType)
	}
	if diff.nLevels != 50 {
		t.Errorf("L2BookDiff nLevels = %d, want 50", diff.nLevels)
	}
	if diff.nSigFigs == nil || *diff.nSigFigs != 5 {
		t.Errorf("L2BookDiff nSigFigs = %v, want 5", diff.nSigFigs)
	}
	if diff.mantissa == nil || *diff.mantissa != 2 {
		t.Errorf("L2BookDiff mantissa = %v, want 2", diff.mantissa)
	}
	if !diff.skipInitialSnapshot {
		t.Error("L2BookDiff skipInitialSnapshot = false, want true")
	}

	if subs[2].streamType != "L4_BOOK_UPDATES" || len(subs[2].coins) != 0 {
		t.Errorf("L4BookUpdates registered %q coins=%v", subs[2].streamType, subs[2].coins)
	}
	if subs[3].streamType != "TPSL_UPDATES" || len(subs[3].coins) != 1 {
		t.Errorf("TpslUpdates registered %q coins=%v", subs[3].streamType, subs[3].coins)
	}
	if subs[4].streamType != "L2_BOOK_PACKED" || subs[4].coin != "BTC" || subs[4].nLevels != 10 {
		t.Errorf("L2BookPacked registered %q coin=%q nLevels=%d", subs[4].streamType, subs[4].coin, subs[4].nLevels)
	}
	if subs[5].streamType != "BBO_BOOK_PACKED" {
		t.Errorf("BboBookPacked registered %q", subs[5].streamType)
	}
	if subs[6].streamType != "L4_BOOK_BYTES" || subs[6].coin != "ETH" {
		t.Errorf("L4BookBytes registered %q coin=%q", subs[6].streamType, subs[6].coin)
	}
}

// Test the StreamBytes low-level subscription.
func TestGRPCStreamBytesSubscription(t *testing.T) {
	s := NewGRPCStream("https://x.quiknode.pro/token", nil)
	s.StreamBytes("MEMPOOL_TXS", func(blockNumber, timestamp uint64, data []byte) {},
		StreamWithCoins("BTC"), StreamWithUsers("0xabc"), StreamWithStartBlock(7))

	if len(s.subscriptions) != 1 {
		t.Fatalf("got %d subscriptions, want 1", len(s.subscriptions))
	}
	sub := s.subscriptions[0]
	if sub.bytesCallback == nil {
		t.Fatal("bytesCallback not set")
	}
	if sub.streamType != "MEMPOOL_TXS" {
		t.Errorf("streamType = %q, want MEMPOOL_TXS", sub.streamType)
	}

	req := buildSubscribeRequest(sub, false, 0)
	pbSub := req.GetSubscribe()
	if pbSub.StreamType != pb.StreamType_MEMPOOL_TXS {
		t.Errorf("StreamType = %v, want MEMPOOL_TXS", pbSub.StreamType)
	}
	if pbSub.StartBlock != 7 {
		t.Errorf("StartBlock = %d, want 7", pbSub.StartBlock)
	}
	if coins := pbSub.Filters["coin"]; coins == nil || coins.Values[0] != "BTC" {
		t.Errorf("Filters[coin] = %v, want [BTC]", coins)
	}
	if users := pbSub.Filters["user"]; users == nil || users.Values[0] != "0xabc" {
		t.Errorf("Filters[user] = %v, want [0xabc]", users)
	}
}

// Test BBO update conversion, including absent bid/ask.
func TestGRPCBboBookUpdateToMap(t *testing.T) {
	update := &pb.BboBookUpdate{
		Coin:        "BTC",
		Time:        1000,
		BlockNumber: 42,
		Bid:         &pb.L2Level{Px: "65000", Sz: "1.5", N: 3},
	}

	m := bboBookUpdateToMap(update)
	if m["coin"] != "BTC" || m["time"] != uint64(1000) || m["block_number"] != uint64(42) {
		t.Errorf("unexpected header fields: %v", m)
	}
	bid, ok := m["bid"].([]any)
	if !ok || bid[0] != "65000" || bid[1] != "1.5" || bid[2] != uint32(3) {
		t.Errorf("bid = %v, want [65000 1.5 3]", m["bid"])
	}
	if m["ask"] != nil {
		t.Errorf("ask = %v, want nil", m["ask"])
	}
}

// Test L2 diff conversion.
func TestGRPCL2BookDiffUpdateToMap(t *testing.T) {
	update := &pb.L2BookDiffUpdate{
		Time:     2000,
		Height:   100,
		Snapshot: true,
		Diffs: []*pb.L2CoinDiff{
			{
				Coin:     "ETH",
				Seq:      5,
				PrevSeq:  4,
				Bids:     []*pb.L2Level{{Px: "4000", Sz: "0", N: 0}},
				Asks:     []*pb.L2Level{{Px: "4001", Sz: "2", N: 1}},
				Snapshot: false,
			},
		},
	}

	m := l2BookDiffUpdateToMap(update)
	if m["time"] != uint64(2000) || m["height"] != uint64(100) || m["snapshot"] != true {
		t.Errorf("unexpected header fields: %v", m)
	}
	diffs := m["diffs"].([]map[string]any)
	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	d := diffs[0]
	if d["coin"] != "ETH" || d["seq"] != uint64(5) || d["prev_seq"] != uint64(4) {
		t.Errorf("unexpected diff fields: %v", d)
	}
	bids := d["bids"].([][]any)
	if bids[0][1] != "0" { // sz=0 level = removed
		t.Errorf("bids = %v, want removed level with sz=0", bids)
	}
}

// Test typed L4 updates conversion.
func TestGRPCL4BookUpdatesUpdateToMap(t *testing.T) {
	update := &pb.L4BookUpdatesUpdate{
		Time:     3000,
		Height:   200,
		Snapshot: true,
		Diffs: []*pb.L4OrderDiff{
			{
				DiffType: pb.L4OrderDiffType_L4_ORDER_DIFF_TYPE_NEW,
				Coin:     "BTC",
				Oid:      777,
				User:     "0xuser",
				Side:     "B",
				Px:       "65000",
				Sz:       "0.5",
			},
		},
	}

	m := l4BookUpdatesUpdateToMap(update)
	if m["snapshot"] != true {
		t.Error("snapshot = false, want true")
	}
	diffs := m["diffs"].([]map[string]any)
	d := diffs[0]
	if d["diff_type"] != "L4_ORDER_DIFF_TYPE_NEW" || d["oid"] != uint64(777) || d["side"] != "B" {
		t.Errorf("unexpected diff fields: %v", d)
	}
}

// Test TP/SL updates conversion.
func TestGRPCTpslUpdatesUpdateToMap(t *testing.T) {
	update := &pb.TpslUpdatesUpdate{
		Time:   4000,
		Height: 300,
		Diffs: []*pb.TpslOrderDiff{
			{
				DiffType:  pb.TpslDiffType_TPSL_DIFF_TYPE_REMOVE,
				Oid:       888,
				Coin:      "ETH",
				User:      "0xuser",
				Side:      "A",
				TriggerPx: "3900",
				Reason:    "filled",
			},
		},
	}

	m := tpslUpdatesUpdateToMap(update)
	diffs := m["diffs"].([]map[string]any)
	d := diffs[0]
	if d["diff_type"] != "TPSL_DIFF_TYPE_REMOVE" || d["oid"] != uint64(888) || d["reason"] != "filled" {
		t.Errorf("unexpected diff fields: %v", d)
	}
}

// Test packed conversions preserve uint64 fixed-point values (scaled by 1e8).
func TestGRPCPackedUpdateToMap(t *testing.T) {
	update := &pb.L2BookPackedUpdate{
		Coin:        "BTC",
		Time:        5000,
		BlockNumber: 400,
		Bids:        []*pb.L2LevelPacked{{Px: 6500000000000, Sz: 150000000, N: 2}},
	}

	m := l2BookPackedUpdateToMap(update)
	bids := m["bids"].([][]any)
	if bids[0][0] != uint64(6500000000000) || bids[0][1] != uint64(150000000) || bids[0][2] != uint32(2) {
		t.Errorf("bids = %v, want fixed-point uint64 values", bids)
	}

	bbo := bboBookPackedUpdateToMap(&pb.BboBookPackedUpdate{
		Coin: "BTC",
		Ask:  &pb.L2LevelPacked{Px: 6500100000000, Sz: 200000000, N: 1},
	})
	if bbo["bid"] != nil {
		t.Errorf("bid = %v, want nil", bbo["bid"])
	}
	ask := bbo["ask"].([]any)
	if ask[0] != uint64(6500100000000) {
		t.Errorf("ask = %v, want fixed-point px", ask)
	}
}

// Test L4 bytes conversion: snapshot mirrors L4Book, diff keeps raw JSON bytes.
func TestGRPCL4BookBytesUpdateToMap(t *testing.T) {
	snapshot := &pb.L4BookBytesUpdate{
		Update: &pb.L4BookBytesUpdate_Snapshot{
			Snapshot: &pb.L4BookSnapshot{
				Coin:   "BTC",
				Time:   6000,
				Height: 500,
				Bids:   []*pb.L4Order{{User: "0xuser", Coin: "BTC", Side: "B", LimitPx: "65000", Sz: "1", Oid: 1}},
			},
		},
	}

	m := l4BookBytesUpdateToMap(snapshot)
	if m["type"] != "snapshot" || m["coin"] != "BTC" {
		t.Errorf("unexpected snapshot map: %v", m)
	}
	bids := m["bids"].([]map[string]any)
	if bids[0]["oid"] != uint64(1) {
		t.Errorf("bids = %v", bids)
	}

	raw := []byte(`{"order_statuses":[],"book_diffs":[]}`)
	diff := &pb.L4BookBytesUpdate{
		Update: &pb.L4BookBytesUpdate_Diff{
			Diff: &pb.L4BookBytesDiff{Time: 6001, Height: 501, Data: raw},
		},
	}

	m = l4BookBytesUpdateToMap(diff)
	if m["type"] != "diff" || m["height"] != uint64(501) {
		t.Errorf("unexpected diff map: %v", m)
	}
	data, ok := m["data"].([]byte)
	if !ok || string(data) != string(raw) {
		t.Errorf("data = %v, want undecoded JSON bytes", m["data"])
	}
}
