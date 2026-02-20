// Package client provides the transport layer for communicating with an
// OpenClaw Gateway -- either over its WebSocket RPC API or by reading/writing
// the JSON5 config file directly.
package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// ConfigPayload represents the response from config.get.
type ConfigPayload struct {
	// Raw is the full JSON5 config string.
	Raw string `json:"raw"`
	// Hash is an opaque string used for optimistic concurrency (baseHash).
	Hash string `json:"hash"`
	// Parsed is the unmarshalled config as a generic map.
	Parsed map[string]any `json:"-"`
}

// HealthPayload represents the response from the health RPC.
type HealthPayload struct {
	OK             bool   `json:"ok"`
	Timestamp      int64  `json:"ts"`
	DurationMs     int64  `json:"durationMs"`
	DefaultAgentID string `json:"defaultAgentId"`
	HeartbeatSecs  int64  `json:"heartbeatSeconds"`
}

// Client is the interface that both the WebSocket and file-based backends
// implement. Every Terraform CRUD operation ultimately calls one of these.
type Client interface {
	// GetConfig retrieves the full OpenClaw configuration.
	GetConfig(ctx context.Context) (*ConfigPayload, error)

	// PatchConfig applies a partial JSON merge-patch to the config.
	// The baseHash must match the hash from the last GetConfig call
	// (optimistic concurrency).
	PatchConfig(ctx context.Context, patch map[string]any, baseHash string) error

	// ApplyConfig replaces the entire config.
	ApplyConfig(ctx context.Context, raw string, baseHash string) error

	// Health returns gateway health info. Only supported over WS.
	Health(ctx context.Context) (*HealthPayload, error)

	// Close tears down the underlying connection/resources.
	Close() error
}

// GetSection is a helper that reads a top-level config section as a typed map.
func GetSection(ctx context.Context, c Client, key string) (map[string]any, string, error) {
	cfg, err := c.GetConfig(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("reading config: %w", err)
	}

	parsed, err := parseRawJSON(cfg.Raw)
	if err != nil {
		return nil, cfg.Hash, fmt.Errorf("parsing config JSON: %w", err)
	}

	section, ok := parsed[key]
	if !ok {
		return nil, cfg.Hash, nil // section doesn't exist yet
	}

	m, ok := section.(map[string]any)
	if !ok {
		return nil, cfg.Hash, fmt.Errorf("config key %q is not an object", key)
	}

	return m, cfg.Hash, nil
}

// GetNestedSection reads a nested config path like "channels.whatsapp".
func GetNestedSection(ctx context.Context, c Client, keys ...string) (map[string]any, string, error) {
	cfg, err := c.GetConfig(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("reading config: %w", err)
	}

	parsed, err := parseRawJSON(cfg.Raw)
	if err != nil {
		return nil, cfg.Hash, fmt.Errorf("parsing config JSON: %w", err)
	}

	current := parsed
	for i, key := range keys {
		val, ok := current[key]
		if !ok {
			return nil, cfg.Hash, nil
		}
		m, ok := val.(map[string]any)
		if !ok {
			return nil, cfg.Hash, fmt.Errorf("config path %v at index %d is not an object", keys[:i+1], i)
		}
		current = m
	}

	return current, cfg.Hash, nil
}

// PatchSection writes a single top-level section via merge-patch.
func PatchSection(ctx context.Context, c Client, key string, value any, baseHash string) error {
	patch := map[string]any{key: value}
	return c.PatchConfig(ctx, patch, baseHash)
}

// PatchNestedSection writes a nested section via merge-patch.
// e.g. PatchNestedSection(ctx, c, val, hash, "channels", "whatsapp")
func PatchNestedSection(ctx context.Context, c Client, value any, baseHash string, keys ...string) error {
	// Build nested map from inside out.
	var patch any = value
	for i := len(keys) - 1; i >= 0; i-- {
		patch = map[string]any{keys[i]: patch}
	}
	return c.PatchConfig(ctx, patch.(map[string]any), baseHash)
}

// DeleteSection removes a top-level section by patching it to null.
func DeleteSection(ctx context.Context, c Client, key string, baseHash string) error {
	patch := map[string]any{key: nil}
	return c.PatchConfig(ctx, patch, baseHash)
}

// parseRawJSON parses a JSON (or JSON5-compatible subset) string into a map.
// OpenClaw's config.get RPC returns standard JSON even though the file is JSON5.
func parseRawJSON(raw string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return result, nil
}
