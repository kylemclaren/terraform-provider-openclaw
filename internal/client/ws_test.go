package client

import (
	"context"
	"os"
	"testing"
	"time"
)

// These tests require a running OpenClaw gateway at OPENCLAW_GATEWAY_URL.
// Skip if not available.

func getWSClient(t *testing.T) *WSClient {
	t.Helper()

	url := os.Getenv("OPENCLAW_GATEWAY_URL")
	if url == "" {
		url = "ws://127.0.0.1:18789"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := NewWSClient(ctx, WSClientConfig{
		URL:   url,
		Token: os.Getenv("OPENCLAW_GATEWAY_TOKEN"),
	})
	if err != nil {
		t.Skipf("Skipping WS test: cannot connect to gateway at %s: %v", url, err)
	}

	t.Cleanup(func() { c.Close() })
	return c
}

func TestWSClient_GetConfig(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Set TF_ACC=1 to run acceptance tests")
	}

	c := getWSClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := c.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	if cfg.Raw == "" {
		t.Error("expected non-empty raw config")
	}
	if cfg.Hash == "" {
		t.Error("expected non-empty hash")
	}

	t.Logf("Config hash: %s", cfg.Hash)
	t.Logf("Config length: %d bytes", len(cfg.Raw))
}

func TestWSClient_PatchConfig(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Set TF_ACC=1 to run acceptance tests")
	}

	c := getWSClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Read current config to get hash
	cfg, err := c.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	// Patch: add a test section under channels
	patch := map[string]any{
		"channels": map[string]any{
			"signal": map[string]any{
				"reactionNotifications": "own",
			},
		},
	}

	err = c.PatchConfig(ctx, patch, cfg.Hash)
	if err != nil {
		t.Fatalf("PatchConfig: %v", err)
	}

	// Verify the patch took effect
	cfg2, err := c.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig after patch: %v", err)
	}

	if cfg2.Hash == cfg.Hash {
		t.Error("expected hash to change after patch")
	}

	// Clean up: remove the test signal section
	cleanup := map[string]any{
		"channels": map[string]any{
			"signal": nil,
		},
	}
	c.PatchConfig(ctx, cleanup, cfg2.Hash)
}

func TestWSClient_Health(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Set TF_ACC=1 to run acceptance tests")
	}

	c := getWSClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := c.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}

	if !health.OK {
		t.Error("expected health.OK to be true")
	}
	if health.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	t.Logf("Health OK: %v", health.OK)
	t.Logf("Timestamp: %d", health.Timestamp)
	t.Logf("Default Agent: %s", health.DefaultAgentID)
	t.Logf("Heartbeat: %d seconds", health.HeartbeatSecs)
}
