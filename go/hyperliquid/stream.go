package hyperliquid

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// StreamConfig holds configuration for WebSocket streaming.
type StreamConfig struct {
	OnError       func(error)
	OnClose       func()
	OnOpen        func()
	OnReconnect   func(attempt int)
	OnStateChange func(ConnectionState)
	Reconnect     bool
	MaxReconnect  int
	PingInterval  time.Duration
}

// DefaultStreamConfig returns default stream configuration.
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		Reconnect:    true,
		MaxReconnect: 0, // Infinite
		PingInterval: 30 * time.Second,
	}
}

// Stream is a WebSocket client for real-time data streams.
type Stream struct {
	wsURL       string
	isQuickNode bool
	config      *StreamConfig

	conn          *websocket.Conn
	connMu        sync.RWMutex
	state         atomic.Int32
	running       atomic.Bool
	reconnectNum  atomic.Int32
	reconnectDelay time.Duration
	lastPong      atomic.Int64
	subID         atomic.Int32
	jsonrpcID     atomic.Int32

	subscriptions map[string]subscriptionInfo
	callbacks     map[string][]func(map[string]any)
	subMu         sync.RWMutex

	done     chan struct{}
	pingDone chan struct{}
}

type subscriptionInfo struct {
	params   map[string]any
	callback func(map[string]any)
}

const (
	stateDisconnected int32 = iota
	stateConnecting
	stateConnected
	stateReconnecting
)

const (
	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 60 * time.Second
	reconnectBackoff      = 2.0
	pingTimeout           = 10 * time.Second
)

// NewStream creates a new WebSocket stream client.
func NewStream(endpoint string, config *StreamConfig) *Stream {
	if config == nil {
		config = DefaultStreamConfig()
	} else {
		// Apply defaults for zero values
		if config.PingInterval == 0 {
			config.PingInterval = 30 * time.Second
		}
	}

	wsURL, isQN := buildWebSocketURL(endpoint)

	s := &Stream{
		wsURL:          wsURL,
		isQuickNode:    isQN,
		config:         config,
		subscriptions:  make(map[string]subscriptionInfo),
		callbacks:      make(map[string][]func(map[string]any)),
		reconnectDelay: initialReconnectDelay,
		done:           make(chan struct{}),
		pingDone:       make(chan struct{}),
	}

	return s
}

func (s *Stream) setState(state ConnectionState) {
	var stateInt int32
	switch state {
	case ConnectionStateDisconnected:
		stateInt = stateDisconnected
	case ConnectionStateConnecting:
		stateInt = stateConnecting
	case ConnectionStateConnected:
		stateInt = stateConnected
	case ConnectionStateReconnecting:
		stateInt = stateReconnecting
	}

	if s.state.Swap(stateInt) != stateInt && s.config.OnStateChange != nil {
		s.config.OnStateChange(state)
	}
}

func (s *Stream) getState() ConnectionState {
	switch s.state.Load() {
	case stateConnected:
		return ConnectionStateConnected
	case stateConnecting:
		return ConnectionStateConnecting
	case stateReconnecting:
		return ConnectionStateReconnecting
	default:
		return ConnectionStateDisconnected
	}
}

// Subscribe subscribes to a stream type.
func (s *Stream) Subscribe(streamType StreamType, callback func(map[string]any), opts ...SubscribeOption) string {
	params := &subscribeParams{}
	for _, opt := range opts {
		opt(params)
	}

	subID := fmt.Sprintf("sub_%d", s.subID.Add(1))

	subParams := map[string]any{"streamType": string(streamType)}
	if len(params.coins) > 0 {
		subParams["coin"] = params.coins
	}
	if len(params.users) > 0 {
		subParams["user"] = params.users
	}

	s.subMu.Lock()
	s.subscriptions[subID] = subscriptionInfo{params: subParams, callback: callback}
	streamKey := string(streamType)
	s.callbacks[streamKey] = append(s.callbacks[streamKey], callback)
	s.subMu.Unlock()

	// Send subscription if connected
	if s.getState() == ConnectionStateConnected {
		s.sendSubscribe(subParams)
	}

	return subID
}

// Trades subscribes to trade stream.
func (s *Stream) Trades(coins []string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeTrades, callback, SubscribeWithCoins(coins...))
}

// Orders subscribes to order stream.
func (s *Stream) Orders(coins []string, callback func(map[string]any), users ...string) string {
	opts := []SubscribeOption{SubscribeWithCoins(coins...)}
	if len(users) > 0 {
		opts = append(opts, SubscribeWithUsers(users...))
	}
	return s.Subscribe(StreamTypeOrders, callback, opts...)
}

// BookUpdates subscribes to order book updates.
func (s *Stream) BookUpdates(coins []string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeBookUpdates, callback, SubscribeWithCoins(coins...))
}

// TWAP subscribes to TWAP execution stream.
func (s *Stream) TWAP(coins []string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeTWAP, callback, SubscribeWithCoins(coins...))
}

// Events subscribes to system events.
func (s *Stream) Events(callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeEvents, callback)
}

// WriterActions subscribes to spot token transfers.
func (s *Stream) WriterActions(callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeWriterActions, callback)
}

// L2Book subscribes to L2 order book snapshots.
func (s *Stream) L2Book(coin string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeL2Book, callback, SubscribeWithCoins(coin))
}

// AllMids subscribes to all mid price updates.
func (s *Stream) AllMids(callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeAllMids, callback)
}

// Candle subscribes to candlestick data.
func (s *Stream) Candle(coin, interval string, callback func(map[string]any)) string {
	subID := fmt.Sprintf("sub_%d", s.subID.Add(1))
	subParams := map[string]any{
		"streamType": string(StreamTypeCandle),
		"coin":       []string{coin},
		"interval":   interval,
	}

	s.subMu.Lock()
	s.subscriptions[subID] = subscriptionInfo{params: subParams, callback: callback}
	streamKey := string(StreamTypeCandle)
	s.callbacks[streamKey] = append(s.callbacks[streamKey], callback)
	s.subMu.Unlock()

	if s.getState() == ConnectionStateConnected {
		s.sendSubscribe(subParams)
	}
	return subID
}

// BBO subscribes to best bid/offer updates.
func (s *Stream) BBO(coin string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeBBO, callback, SubscribeWithCoins(coin))
}

// OpenOrders subscribes to user's open orders.
func (s *Stream) OpenOrders(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeOpenOrders, callback, SubscribeWithUsers(user))
}

// OrderUpdates subscribes to user's order status changes.
func (s *Stream) OrderUpdates(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeOrderUpdates, callback, SubscribeWithUsers(user))
}

// UserEvents subscribes to comprehensive user events (fills, funding, liquidations).
func (s *Stream) UserEvents(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeUserEvents, callback, SubscribeWithUsers(user))
}

// UserFills subscribes to user's trade fills.
func (s *Stream) UserFills(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeUserFills, callback, SubscribeWithUsers(user))
}

// UserFundings subscribes to user's funding payment updates.
func (s *Stream) UserFundings(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeUserFundings, callback, SubscribeWithUsers(user))
}

// UserNonFundingLedger subscribes to user's ledger changes (deposits, withdrawals, transfers).
func (s *Stream) UserNonFundingLedger(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeUserNonFundingLedger, callback, SubscribeWithUsers(user))
}

// ClearinghouseState subscribes to user's clearinghouse state updates.
func (s *Stream) ClearinghouseState(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeClearinghouseState, callback, SubscribeWithUsers(user))
}

// ActiveAssetCtx subscribes to asset context data (pricing, volume, supply).
func (s *Stream) ActiveAssetCtx(coin string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeActiveAssetCtx, callback, SubscribeWithCoins(coin))
}

// ActiveAssetData subscribes to user's active asset trading parameters.
func (s *Stream) ActiveAssetData(user, coin string, callback func(map[string]any)) string {
	subID := fmt.Sprintf("sub_%d", s.subID.Add(1))
	subParams := map[string]any{
		"streamType": string(StreamTypeActiveAssetData),
		"user":       []string{user},
		"coin":       []string{coin},
	}

	s.subMu.Lock()
	s.subscriptions[subID] = subscriptionInfo{params: subParams, callback: callback}
	streamKey := string(StreamTypeActiveAssetData)
	s.callbacks[streamKey] = append(s.callbacks[streamKey], callback)
	s.subMu.Unlock()

	if s.getState() == ConnectionStateConnected {
		s.sendSubscribe(subParams)
	}
	return subID
}

// TWAPStates subscribes to TWAP algorithm states.
func (s *Stream) TWAPStates(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeTWAPStates, callback, SubscribeWithUsers(user))
}

// UserTWAPSliceFills subscribes to individual TWAP order slice fills.
func (s *Stream) UserTWAPSliceFills(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeUserTWAPSliceFills, callback, SubscribeWithUsers(user))
}

// UserTWAPHistory subscribes to TWAP execution history and status.
func (s *Stream) UserTWAPHistory(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeUserTWAPHistory, callback, SubscribeWithUsers(user))
}

// Notification subscribes to user notifications.
func (s *Stream) Notification(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeNotification, callback, SubscribeWithUsers(user))
}

// WebData3 subscribes to aggregate user information for frontend use.
func (s *Stream) WebData3(user string, callback func(map[string]any)) string {
	return s.Subscribe(StreamTypeWebData3, callback, SubscribeWithUsers(user))
}

// ReconnectAttempts returns the number of reconnection attempts since last successful connection.
func (s *Stream) ReconnectAttempts() int {
	return int(s.reconnectNum.Load())
}

// Unsubscribe unsubscribes from a stream.
func (s *Stream) Unsubscribe(subID string) {
	s.subMu.Lock()
	info, ok := s.subscriptions[subID]
	if ok {
		delete(s.subscriptions, subID)
		streamType := info.params["streamType"].(string)
		callbacks := s.callbacks[streamType]
		for i, cb := range callbacks {
			if &cb == &info.callback {
				s.callbacks[streamType] = append(callbacks[:i], callbacks[i+1:]...)
				break
			}
		}
	}
	s.subMu.Unlock()

	if ok && s.getState() == ConnectionStateConnected {
		s.sendUnsubscribe(info.params)
	}
}

func (s *Stream) sendSubscribe(params map[string]any) {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()

	if conn == nil {
		return
	}

	streamType := params["streamType"].(string)

	var msg []byte
	var err error

	if s.isQuickNode {
		// QuickNode JSON-RPC format
		qnParams := map[string]any{"streamType": streamType}
		filters := map[string]any{}
		if coins, ok := params["coin"]; ok {
			filters["coin"] = coins
		}
		if users, ok := params["user"]; ok {
			filters["user"] = users
		}
		if len(filters) > 0 {
			qnParams["filters"] = filters
		}

		msg, err = json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "hl_subscribe",
			"params":  qnParams,
			"id":      s.jsonrpcID.Add(1),
		})
	} else {
		// Public API format
		subscription := map[string]any{"type": streamType}
		if coins, ok := params["coin"]; ok {
			if coinsList, ok := coins.([]string); ok && len(coinsList) == 1 {
				subscription["coin"] = coinsList[0]
			}
		}
		if users, ok := params["user"]; ok {
			if usersList, ok := users.([]string); ok && len(usersList) == 1 {
				subscription["user"] = usersList[0]
			}
		}
		msg, err = json.Marshal(map[string]any{
			"method":       "subscribe",
			"subscription": subscription,
		})
	}

	if err != nil {
		return
	}

	conn.WriteMessage(websocket.TextMessage, msg)
}

func (s *Stream) sendUnsubscribe(params map[string]any) {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()

	if conn == nil {
		return
	}

	streamType := params["streamType"].(string)

	var msg []byte
	var err error

	if s.isQuickNode {
		msg, err = json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "hl_unsubscribe",
			"params":  map[string]any{"streamType": streamType},
			"id":      s.jsonrpcID.Add(1),
		})
	} else {
		msg, err = json.Marshal(map[string]any{
			"method":       "unsubscribe",
			"subscription": map[string]any{"type": streamType},
		})
	}

	if err != nil {
		return
	}

	conn.WriteMessage(websocket.TextMessage, msg)
}

func (s *Stream) connect() error {
	s.setState(ConnectionStateConnecting)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(s.wsURL, nil)
	if err != nil {
		return err
	}

	s.connMu.Lock()
	s.conn = conn
	s.connMu.Unlock()

	s.setState(ConnectionStateConnected)
	s.reconnectNum.Store(0)
	s.reconnectDelay = initialReconnectDelay
	s.lastPong.Store(time.Now().UnixMilli())

	// Resubscribe
	s.subMu.RLock()
	for _, info := range s.subscriptions {
		s.sendSubscribe(info.params)
	}
	s.subMu.RUnlock()

	if s.config.OnOpen != nil {
		s.config.OnOpen()
	}

	return nil
}

func (s *Stream) readPump() {
	defer func() {
		s.connMu.Lock()
		if s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
		s.connMu.Unlock()

		if s.running.Load() && s.config.Reconnect {
			s.scheduleReconnect()
		}
	}()

	for s.running.Load() {
		s.connMu.RLock()
		conn := s.conn
		s.connMu.RUnlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if s.config.OnError != nil && s.running.Load() {
				s.config.OnError(err)
			}
			return
		}

		var data map[string]any
		if err := json.Unmarshal(message, &data); err != nil {
			continue
		}

		// Handle pong
		if channel, _ := data["channel"].(string); channel == "pong" {
			s.lastPong.Store(time.Now().UnixMilli())
			continue
		}
		if dataType, _ := data["type"].(string); dataType == "pong" {
			s.lastPong.Store(time.Now().UnixMilli())
			continue
		}

		// Skip subscription confirmations
		if channel, _ := data["channel"].(string); channel == "subscriptionResponse" {
			continue
		}
		if _, ok := data["result"]; ok {
			continue
		}

		// Determine stream type and dispatch
		var streamType string
		if s.isQuickNode {
			if stream, _ := data["stream"].(string); len(stream) > 3 && stream[:3] == "hl." {
				streamType = stream[3:]
			}
		} else {
			streamType, _ = data["channel"].(string)
		}

		// Dispatch to callbacks
		s.subMu.RLock()
		callbacks := s.callbacks[streamType]
		callbacksCopy := make([]func(map[string]any), len(callbacks))
		copy(callbacksCopy, callbacks)
		s.subMu.RUnlock()

		for _, cb := range callbacksCopy {
			cb(data)
		}
	}
}

func (s *Stream) pingLoop() {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.pingDone:
			return
		case <-ticker.C:
			if s.getState() != ConnectionStateConnected {
				continue
			}

			// Check for stale connection
			lastPong := s.lastPong.Load()
			if lastPong > 0 && time.Now().UnixMilli()-lastPong > int64((s.config.PingInterval+pingTimeout).Milliseconds()) {
				s.connMu.Lock()
				if s.conn != nil {
					s.conn.Close()
				}
				s.connMu.Unlock()
				continue
			}

			// Send ping
			s.connMu.RLock()
			conn := s.conn
			s.connMu.RUnlock()

			if conn != nil {
				msg, _ := json.Marshal(map[string]any{"method": "ping"})
				conn.WriteMessage(websocket.TextMessage, msg)
			}
		}
	}
}

func (s *Stream) scheduleReconnect() {
	if !s.running.Load() {
		return
	}

	maxReconnect := s.config.MaxReconnect
	attempt := int(s.reconnectNum.Add(1))

	if maxReconnect > 0 && attempt > maxReconnect {
		s.running.Store(false)
		s.setState(ConnectionStateDisconnected)
		if s.config.OnClose != nil {
			s.config.OnClose()
		}
		return
	}

	s.setState(ConnectionStateReconnecting)

	if s.config.OnReconnect != nil {
		s.config.OnReconnect(attempt)
	}

	time.Sleep(s.reconnectDelay)
	if s.reconnectDelay < maxReconnectDelay {
		s.reconnectDelay = time.Duration(float64(s.reconnectDelay) * reconnectBackoff)
		if s.reconnectDelay > maxReconnectDelay {
			s.reconnectDelay = maxReconnectDelay
		}
	}

	if s.running.Load() {
		if err := s.connect(); err != nil {
			s.scheduleReconnect()
			return
		}
		go s.readPump()
	}
}

// Run runs the stream (blocking).
func (s *Stream) Run() error {
	s.running.Store(true)
	s.done = make(chan struct{})
	s.pingDone = make(chan struct{})

	if err := s.connect(); err != nil {
		return err
	}

	go s.pingLoop()
	s.readPump()

	return nil
}

// Start starts the stream in background.
func (s *Stream) Start() error {
	s.running.Store(true)
	s.done = make(chan struct{})
	s.pingDone = make(chan struct{})

	if err := s.connect(); err != nil {
		return err
	}

	go s.pingLoop()
	go s.readPump()

	return nil
}

// Stop stops the stream.
func (s *Stream) Stop() {
	s.running.Store(false)

	close(s.pingDone)

	s.connMu.Lock()
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.connMu.Unlock()

	s.setState(ConnectionStateDisconnected)

	if s.config.OnClose != nil {
		s.config.OnClose()
	}
}

// Connected returns true if the stream is connected.
func (s *Stream) Connected() bool {
	return s.getState() == ConnectionStateConnected
}

// State returns the current connection state.
func (s *Stream) State() ConnectionState {
	return s.getState()
}

// ═══════════════════════════════════════════════════════════════════════════════
// SUBSCRIBE OPTIONS
// ═══════════════════════════════════════════════════════════════════════════════

type subscribeParams struct {
	coins []string
	users []string
}

// SubscribeOption is an option for subscriptions.
type SubscribeOption func(*subscribeParams)

// SubscribeWithCoins filters by coins.
func SubscribeWithCoins(coins ...string) SubscribeOption {
	return func(p *subscribeParams) {
		p.coins = coins
	}
}

// SubscribeWithUsers filters by users.
func SubscribeWithUsers(users ...string) SubscribeOption {
	return func(p *subscribeParams) {
		p.users = users
	}
}
