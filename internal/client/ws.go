package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// wsFrame is the wire format for OpenClaw Gateway WebSocket messages.
type wsFrame struct {
	Type    string `json:"type"`              // "req", "res", "event"
	ID      string `json:"id,omitempty"`      // request/response correlation
	Method  string `json:"method,omitempty"`  // for requests
	Params  any    `json:"params,omitempty"`  // for requests
	OK      *bool  `json:"ok,omitempty"`      // for responses
	Payload any    `json:"payload,omitempty"` // for responses
	Error   any    `json:"error,omitempty"`   // for error responses
	Event   string `json:"event,omitempty"`   // for events
}

// WSClient communicates with the OpenClaw Gateway over WebSocket.
type WSClient struct {
	conn      *websocket.Conn
	url       string
	token     string
	mu        sync.Mutex
	pending   map[string]chan wsFrame
	challenge chan wsFrame // receives the connect.challenge event
	nextID    atomic.Int64
	connected bool
	done      chan struct{}
}

// WSClientConfig holds connection parameters.
type WSClientConfig struct {
	URL   string
	Token string
}

// NewWSClient dials the Gateway and performs the connect handshake.
// It retries with exponential backoff to tolerate gateway restarts
// (e.g. after a config.patch triggers a reload/restart cycle).
func NewWSClient(ctx context.Context, cfg WSClientConfig) (*WSClient, error) {
	const maxRetries = 5
	backoff := 1 * time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff):
				backoff = backoff * 2
				if backoff > 10*time.Second {
					backoff = 10 * time.Second
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("ws connect cancelled after %d attempts: %w (last error: %v)", attempt, ctx.Err(), lastErr)
			}
		}

		c, err := dialAndHandshake(ctx, cfg)
		if err == nil {
			return c, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

func dialAndHandshake(ctx context.Context, cfg WSClientConfig) (*WSClient, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("ws dial %s: %w", cfg.URL, err)
	}

	c := &WSClient{
		conn:      conn,
		url:       cfg.URL,
		token:     cfg.Token,
		pending:   make(map[string]chan wsFrame),
		challenge: make(chan wsFrame, 1),
		done:      make(chan struct{}),
	}

	// Start the read pump before handshake so we can receive the response.
	go c.readPump()

	// Perform the mandatory connect handshake.
	if err := c.handshake(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ws handshake: %w", err)
	}

	c.connected = true
	return c, nil
}

func (c *WSClient) handshake(ctx context.Context) error {
	// Wait for the gateway's connect.challenge event (sent immediately on WS open).
	var challengeNonce string
	select {
	case frame := <-c.challenge:
		if p, ok := frame.Payload.(map[string]any); ok {
			if n, ok := p["nonce"].(string); ok {
				challengeNonce = n
			}
		}
	case <-time.After(5 * time.Second):
		// Some gateways may not send a challenge; proceed without it.
	case <-ctx.Done():
		return ctx.Err()
	}

	// Generate an ephemeral Ed25519 keypair for device identity.
	// The gateway auto-pairs local (loopback) connections silently.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate device key: %w", err)
	}

	// Device ID = SHA-256 hex of the raw 32-byte public key.
	idHash := sha256.Sum256(pub)
	deviceID := fmt.Sprintf("%x", idHash)

	// Public key as base64url (raw 32 bytes, no padding).
	publicKeyB64 := base64.RawURLEncoding.EncodeToString(pub)

	signedAt := time.Now().UnixMilli()

	clientID := "cli"
	clientMode := "cli"
	role := "operator"
	scopes := []string{"operator.read", "operator.write", "operator.admin"}

	// Build the signed payload per the OpenClaw device auth protocol.
	// v1 format (local, no nonce): v1|deviceId|clientId|clientMode|role|scopes|signedAt|token
	// v2 format (with nonce):      v2|deviceId|clientId|clientMode|role|scopes|signedAt|token|nonce
	// The token in the signed payload is the auth token (shared secret or device token),
	// or empty string if none provided.
	authToken := c.token
	scopeStr := strings.Join(scopes, ",")
	var version, signedPayload string
	if challengeNonce != "" {
		version = "v2"
		signedPayload = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s|%s",
			version, deviceID, clientID, clientMode, role, scopeStr, signedAt, authToken, challengeNonce)
	} else {
		version = "v1"
		signedPayload = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s",
			version, deviceID, clientID, clientMode, role, scopeStr, signedAt, authToken)
	}

	sig := ed25519.Sign(priv, []byte(signedPayload))
	signatureB64 := base64.RawURLEncoding.EncodeToString(sig)

	device := map[string]any{
		"id":        deviceID,
		"publicKey": publicKeyB64,
		"signature": signatureB64,
		"signedAt":  signedAt,
	}
	if challengeNonce != "" {
		device["nonce"] = challengeNonce
	}

	params := map[string]any{
		"minProtocol": 3,
		"maxProtocol": 3,
		"client": map[string]any{
			"id":       clientID,
			"version":  "dev",
			"platform": "linux",
			"mode":     clientMode,
		},
		"role":        role,
		"scopes":      scopes,
		"caps":        []string{},
		"commands":    []string{},
		"permissions": map[string]any{},
		"locale":      "en-US",
		"userAgent":   "terraform-provider-openclaw/dev",
		"device":      device,
	}
	if c.token != "" {
		params["auth"] = map[string]any{
			"token": c.token,
		}
	}

	resp, err := c.call(ctx, "connect", params)
	if err != nil {
		return err
	}

	if resp.OK == nil || !*resp.OK {
		errBytes, _ := json.Marshal(resp.Error)
		return fmt.Errorf("connect rejected: %s", string(errBytes))
	}

	return nil
}

func (c *WSClient) call(ctx context.Context, method string, params any) (wsFrame, error) {
	id := fmt.Sprintf("tf-%d", c.nextID.Add(1))
	ch := make(chan wsFrame, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	frame := wsFrame{
		Type:   "req",
		ID:     id,
		Method: method,
		Params: params,
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return wsFrame{}, fmt.Errorf("marshal request: %w", err)
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()
	if err != nil {
		return wsFrame{}, fmt.Errorf("ws write: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return wsFrame{}, ctx.Err()
	case <-c.done:
		return wsFrame{}, fmt.Errorf("connection closed")
	}
}

func (c *WSClient) readPump() {
	defer close(c.done)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var frame wsFrame
		if err := json.Unmarshal(message, &frame); err != nil {
			continue
		}

		// Route responses to pending callers.
		if frame.Type == "res" && frame.ID != "" {
			c.mu.Lock()
			ch, ok := c.pending[frame.ID]
			c.mu.Unlock()
			if ok {
				ch <- frame
			}
		}

		// Route the connect.challenge event.
		if frame.Type == "event" && frame.Event == "connect.challenge" {
			select {
			case c.challenge <- frame:
			default:
			}
		}
	}
}

// GetConfig implements Client.
func (c *WSClient) GetConfig(ctx context.Context) (*ConfigPayload, error) {
	resp, err := c.call(ctx, "config.get", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.OK == nil || !*resp.OK {
		return nil, fmt.Errorf("config.get failed: %v", resp.Error)
	}

	payloadBytes, err := json.Marshal(resp.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	var result struct {
		Raw    *string        `json:"raw"`
		Hash   string         `json:"hash"`
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(payloadBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal config payload: %w", err)
	}

	raw := ""
	if result.Raw != nil {
		raw = *result.Raw
	} else if result.Config != nil {
		// When the config file doesn't exist yet, raw is null but the
		// gateway returns the effective (default) config under "config".
		configBytes, err := json.MarshalIndent(result.Config, "", "  ")
		if err == nil {
			raw = string(configBytes)
		}
	}

	return &ConfigPayload{
		Raw:  raw,
		Hash: result.Hash,
	}, nil
}

// PatchConfig implements Client.
func (c *WSClient) PatchConfig(ctx context.Context, patch map[string]any, baseHash string) error {
	rawBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch: %w", err)
	}

	params := map[string]any{
		"raw":      string(rawBytes),
		"baseHash": baseHash,
	}

	resp, err := c.call(ctx, "config.patch", params)
	if err != nil {
		return err
	}
	if resp.OK == nil || !*resp.OK {
		return fmt.Errorf("config.patch failed: %v", resp.Error)
	}
	return nil
}

// ApplyConfig implements Client.
func (c *WSClient) ApplyConfig(ctx context.Context, raw string, baseHash string) error {
	params := map[string]any{
		"raw": raw,
	}
	if baseHash != "" {
		params["baseHash"] = baseHash
	}

	resp, err := c.call(ctx, "config.apply", params)
	if err != nil {
		return err
	}
	if resp.OK == nil || !*resp.OK {
		return fmt.Errorf("config.apply failed: %v", resp.Error)
	}
	return nil
}

// Health implements Client.
func (c *WSClient) Health(ctx context.Context) (*HealthPayload, error) {
	resp, err := c.call(ctx, "health", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.OK == nil || !*resp.OK {
		return nil, fmt.Errorf("health failed: %v", resp.Error)
	}

	payloadBytes, err := json.Marshal(resp.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal health payload: %w", err)
	}

	var health HealthPayload
	if err := json.Unmarshal(payloadBytes, &health); err != nil {
		return nil, fmt.Errorf("unmarshal health: %w", err)
	}

	return &health, nil
}

// Close implements Client.
func (c *WSClient) Close() error {
	return c.conn.Close()
}
