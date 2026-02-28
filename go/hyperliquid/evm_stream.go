package hyperliquid

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// EVMSubscriptionType represents EVM WebSocket subscription types.
type EVMSubscriptionType string

const (
	EVMSubscriptionNewHeads               EVMSubscriptionType = "newHeads"
	EVMSubscriptionLogs                   EVMSubscriptionType = "logs"
	EVMSubscriptionNewPendingTransactions EVMSubscriptionType = "newPendingTransactions"
)

// EVMStreamConfig holds configuration for EVM WebSocket streaming.
type EVMStreamConfig struct {
	OnError       func(error)
	OnClose       func()
	OnOpen        func()
	OnStateChange func(ConnectionState)
	Reconnect     bool
	MaxReconnect  int
	PingInterval  time.Duration
}

// DefaultEVMStreamConfig returns default EVM stream configuration.
func DefaultEVMStreamConfig() *EVMStreamConfig {
	return &EVMStreamConfig{
		Reconnect:    true,
		MaxReconnect: 10,
		PingInterval: 30 * time.Second,
	}
}

// EVMStream is a WebSocket client for EVM event streams.
type EVMStream struct {
	wsURL  string
	config *EVMStreamConfig

	conn          *websocket.Conn
	connMu        sync.RWMutex
	state         atomic.Int32
	running       atomic.Bool
	reconnectNum  atomic.Int32
	reconnectDelay time.Duration
	requestID     atomic.Int64

	pendingSubscriptions []evmSubscription
	activeSubscriptions  map[string]evmSubscription
	callbacks            map[string]func(map[string]any)
	pendingCallbacks     map[int64]func(map[string]any)
	subMu                sync.RWMutex

	done chan struct{}
}

type evmSubscription struct {
	subType  EVMSubscriptionType
	params   map[string]any
	callback func(map[string]any)
}

const (
	evmInitialReconnectDelay = 1 * time.Second
	evmMaxReconnectDelay     = 30 * time.Second
	evmReconnectBackoff      = 2.0
)

// NewEVMStream creates a new EVM WebSocket stream client.
func NewEVMStream(endpoint string, config *EVMStreamConfig) *EVMStream {
	if config == nil {
		config = DefaultEVMStreamConfig()
	}

	wsURL := buildEVMWebSocketURL(endpoint)

	s := &EVMStream{
		wsURL:                wsURL,
		config:               config,
		pendingSubscriptions: make([]evmSubscription, 0),
		activeSubscriptions:  make(map[string]evmSubscription),
		callbacks:            make(map[string]func(map[string]any)),
		pendingCallbacks:     make(map[int64]func(map[string]any)),
		reconnectDelay:       evmInitialReconnectDelay,
		done:                 make(chan struct{}),
	}

	return s
}

func buildEVMWebSocketURL(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	scheme := "wss"
	if parsed.Scheme == "http" {
		scheme = "ws"
	}

	host := parsed.Host

	// Extract token from path
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	token := ""
	for _, part := range pathParts {
		if part != "" && part != "info" && part != "hypercore" && part != "evm" && part != "nanoreth" && part != "ws" {
			token = part
			break
		}
	}

	if token != "" {
		return fmt.Sprintf("%s://%s/%s/nanoreth", scheme, host, token)
	}
	return fmt.Sprintf("%s://%s/nanoreth", scheme, host)
}

func (s *EVMStream) setState(state ConnectionState) {
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

func (s *EVMStream) getState() ConnectionState {
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

func (s *EVMStream) nextID() int64 {
	return s.requestID.Add(1)
}

// NewHeads subscribes to new block headers.
func (s *EVMStream) NewHeads(callback func(map[string]any)) *EVMStream {
	s.subMu.Lock()
	s.pendingSubscriptions = append(s.pendingSubscriptions, evmSubscription{
		subType:  EVMSubscriptionNewHeads,
		callback: callback,
	})
	s.subMu.Unlock()
	return s
}

// Logs subscribes to contract event logs.
func (s *EVMStream) Logs(filter map[string]any, callback func(map[string]any)) *EVMStream {
	s.subMu.Lock()
	s.pendingSubscriptions = append(s.pendingSubscriptions, evmSubscription{
		subType:  EVMSubscriptionLogs,
		params:   filter,
		callback: callback,
	})
	s.subMu.Unlock()
	return s
}

// NewPendingTransactions subscribes to pending transaction hashes.
func (s *EVMStream) NewPendingTransactions(callback func(map[string]any)) *EVMStream {
	s.subMu.Lock()
	s.pendingSubscriptions = append(s.pendingSubscriptions, evmSubscription{
		subType:  EVMSubscriptionNewPendingTransactions,
		callback: callback,
	})
	s.subMu.Unlock()
	return s
}

// Unsubscribe unsubscribes from a subscription.
func (s *EVMStream) Unsubscribe(subscriptionID string) bool {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()

	if conn == nil {
		return false
	}

	s.subMu.RLock()
	_, exists := s.activeSubscriptions[subscriptionID]
	s.subMu.RUnlock()

	if !exists {
		return false
	}

	reqID := s.nextID()
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_unsubscribe",
		"params":  []string{subscriptionID},
		"id":      reqID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return false
	}

	s.subMu.Lock()
	delete(s.activeSubscriptions, subscriptionID)
	delete(s.callbacks, subscriptionID)
	s.subMu.Unlock()

	return true
}

func (s *EVMStream) sendSubscriptions() {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()

	if conn == nil {
		return
	}

	s.subMu.RLock()
	subs := make([]evmSubscription, len(s.pendingSubscriptions))
	copy(subs, s.pendingSubscriptions)
	s.subMu.RUnlock()

	for _, sub := range subs {
		reqID := s.nextID()

		params := []any{string(sub.subType)}
		if sub.params != nil {
			params = append(params, sub.params)
		}

		msg := map[string]any{
			"jsonrpc": "2.0",
			"method":  "eth_subscribe",
			"params":  params,
			"id":      reqID,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		s.subMu.Lock()
		s.pendingCallbacks[reqID] = sub.callback
		s.subMu.Unlock()

		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *EVMStream) connect() error {
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
	s.reconnectDelay = evmInitialReconnectDelay

	s.sendSubscriptions()

	if s.config.OnOpen != nil {
		s.config.OnOpen()
	}

	return nil
}

func (s *EVMStream) readPump() {
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

		// Handle subscription confirmation
		if id, ok := data["id"].(float64); ok {
			if result, ok := data["result"].(string); ok {
				reqID := int64(id)
				s.subMu.Lock()
				if callback, exists := s.pendingCallbacks[reqID]; exists {
					s.callbacks[result] = callback
					s.activeSubscriptions[result] = evmSubscription{}
					delete(s.pendingCallbacks, reqID)
				}
				s.subMu.Unlock()
			}
			continue
		}

		// Handle subscription data
		if method, _ := data["method"].(string); method == "eth_subscription" {
			params, _ := data["params"].(map[string]any)
			if params == nil {
				continue
			}

			subID, _ := params["subscription"].(string)
			result, _ := params["result"].(map[string]any)

			if subID == "" {
				continue
			}

			// Handle string result (for newPendingTransactions)
			if result == nil {
				if txHash, ok := params["result"].(string); ok {
					result = map[string]any{"hash": txHash}
				}
			}

			if result == nil {
				continue
			}

			s.subMu.RLock()
			callback, exists := s.callbacks[subID]
			s.subMu.RUnlock()

			if exists && callback != nil {
				callback(result)
			}
		}
	}
}

func (s *EVMStream) scheduleReconnect() {
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

	time.Sleep(s.reconnectDelay)
	if s.reconnectDelay < evmMaxReconnectDelay {
		s.reconnectDelay = time.Duration(float64(s.reconnectDelay) * evmReconnectBackoff)
		if s.reconnectDelay > evmMaxReconnectDelay {
			s.reconnectDelay = evmMaxReconnectDelay
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
func (s *EVMStream) Run() error {
	s.running.Store(true)
	s.done = make(chan struct{})

	if err := s.connect(); err != nil {
		return err
	}

	s.readPump()

	return nil
}

// Start starts the stream in background.
func (s *EVMStream) Start() error {
	s.running.Store(true)
	s.done = make(chan struct{})

	if err := s.connect(); err != nil {
		return err
	}

	go s.readPump()

	return nil
}

// Stop stops the stream.
func (s *EVMStream) Stop() {
	s.running.Store(false)

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
func (s *EVMStream) Connected() bool {
	return s.getState() == ConnectionStateConnected
}

// State returns the current connection state.
func (s *EVMStream) State() ConnectionState {
	return s.getState()
}

// Subscriptions returns the list of active subscription IDs.
func (s *EVMStream) Subscriptions() []string {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	ids := make([]string, 0, len(s.activeSubscriptions))
	for id := range s.activeSubscriptions {
		ids = append(ids, id)
	}
	return ids
}
