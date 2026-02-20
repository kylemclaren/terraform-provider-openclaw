package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileClient_GetConfig_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	cfg, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	if cfg.Raw != "{}" {
		t.Errorf("expected empty config '{}', got %q", cfg.Raw)
	}
	if cfg.Hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestFileClient_GetConfig_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	data := `{"gateway":{"port":18789}}`
	os.WriteFile(path, []byte(data), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	cfg, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	if cfg.Raw != data {
		t.Errorf("expected %q, got %q", data, cfg.Raw)
	}
}

func TestFileClient_PatchConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	os.WriteFile(path, []byte(`{"gateway":{"port":18789}}`), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	cfg, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	// Patch: add a new section
	patch := map[string]any{
		"channels": map[string]any{
			"whatsapp": map[string]any{
				"dmPolicy":  "pairing",
				"allowFrom": []string{"+15555550123"},
			},
		},
	}

	err = c.PatchConfig(context.Background(), patch, cfg.Hash)
	if err != nil {
		t.Fatalf("PatchConfig: %v", err)
	}

	// Read back
	cfg2, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig after patch: %v", err)
	}

	var result map[string]any
	json.Unmarshal([]byte(cfg2.Raw), &result)

	// Check gateway still exists
	gw, ok := result["gateway"].(map[string]any)
	if !ok {
		t.Fatal("gateway section missing after patch")
	}
	if gw["port"].(float64) != 18789 {
		t.Errorf("gateway port changed unexpectedly")
	}

	// Check new section
	channels, ok := result["channels"].(map[string]any)
	if !ok {
		t.Fatal("channels section missing after patch")
	}
	wa, ok := channels["whatsapp"].(map[string]any)
	if !ok {
		t.Fatal("channels.whatsapp missing after patch")
	}
	if wa["dmPolicy"] != "pairing" {
		t.Errorf("expected dmPolicy=pairing, got %v", wa["dmPolicy"])
	}
}

func TestFileClient_PatchConfig_IgnoresStaleHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	os.WriteFile(path, []byte(`{"gateway":{"port":18789}}`), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	cfg, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	// Simulate external change
	os.WriteFile(path, []byte(`{"gateway":{"port":9999}}`), 0o644)

	// In file mode, the mutex serializes access, so the stale hash is
	// intentionally ignored. PatchConfig re-reads the file fresh inside the
	// lock and applies the merge on top of the current content.
	err = c.PatchConfig(context.Background(), map[string]any{"test": true}, cfg.Hash)
	if err != nil {
		t.Fatalf("PatchConfig should succeed with stale hash in file mode: %v", err)
	}

	// Verify the patch was applied to the externally-modified file (port 9999)
	cfg2, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig after patch: %v", err)
	}
	var result map[string]any
	json.Unmarshal([]byte(cfg2.Raw), &result)

	gw := result["gateway"].(map[string]any)
	if gw["port"].(float64) != 9999 {
		t.Errorf("expected port 9999 (from external change), got %v", gw["port"])
	}
	if result["test"] != true {
		t.Errorf("expected test=true from patch, got %v", result["test"])
	}
}

func TestFileClient_PatchConfig_DeleteSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	os.WriteFile(path, []byte(`{"gateway":{"port":18789},"channels":{"whatsapp":{}}}`), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	cfg, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}

	// Delete channels by setting to null
	err = c.PatchConfig(context.Background(), map[string]any{"channels": nil}, cfg.Hash)
	if err != nil {
		t.Fatalf("PatchConfig delete: %v", err)
	}

	cfg2, err := c.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig after delete: %v", err)
	}

	var result map[string]any
	json.Unmarshal([]byte(cfg2.Raw), &result)

	if _, ok := result["channels"]; ok {
		t.Error("channels section should be deleted")
	}
	if _, ok := result["gateway"]; !ok {
		t.Error("gateway section should still exist")
	}
}

func TestFileClient_ApplyConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	raw := `{"gateway":{"port":9999}}`
	err = c.ApplyConfig(context.Background(), raw, "")
	if err != nil {
		t.Fatalf("ApplyConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != raw {
		t.Errorf("expected %q, got %q", raw, string(data))
	}
}

func TestFileClient_Health_Unsupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	_, err = c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for Health in file mode")
	}
}

func TestMergePatch(t *testing.T) {
	tests := []struct {
		name   string
		target map[string]any
		patch  map[string]any
		want   map[string]any
	}{
		{
			name:   "add key",
			target: map[string]any{"a": 1},
			patch:  map[string]any{"b": 2},
			want:   map[string]any{"a": 1, "b": 2},
		},
		{
			name:   "overwrite key",
			target: map[string]any{"a": 1},
			patch:  map[string]any{"a": 2},
			want:   map[string]any{"a": 2},
		},
		{
			name:   "delete key",
			target: map[string]any{"a": 1, "b": 2},
			patch:  map[string]any{"b": nil},
			want:   map[string]any{"a": 1},
		},
		{
			name:   "deep merge",
			target: map[string]any{"nested": map[string]any{"a": 1, "b": 2}},
			patch:  map[string]any{"nested": map[string]any{"b": 3, "c": 4}},
			want:   map[string]any{"nested": map[string]any{"a": 1, "b": 3, "c": 4}},
		},
		{
			name:   "nil target",
			target: nil,
			patch:  map[string]any{"a": 1},
			want:   map[string]any{"a": 1},
		},
		{
			name:   "replace non-map with map",
			target: map[string]any{"a": "string"},
			patch:  map[string]any{"a": map[string]any{"nested": true}},
			want:   map[string]any{"a": map[string]any{"nested": true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergePatch(tt.target, tt.patch)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("mergePatch() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestGetSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	os.WriteFile(path, []byte(`{"gateway":{"port":18789},"channels":{"whatsapp":{"dmPolicy":"pairing"}}}`), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	ctx := context.Background()

	// Get existing section
	section, hash, err := GetSection(ctx, c, "gateway")
	if err != nil {
		t.Fatalf("GetSection: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if section["port"].(float64) != 18789 {
		t.Errorf("expected port 18789, got %v", section["port"])
	}

	// Get missing section
	section, _, err = GetSection(ctx, c, "nonexistent")
	if err != nil {
		t.Fatalf("GetSection missing: %v", err)
	}
	if section != nil {
		t.Errorf("expected nil for missing section, got %v", section)
	}
}

func TestGetNestedSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	os.WriteFile(path, []byte(`{"channels":{"whatsapp":{"dmPolicy":"pairing"}}}`), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	ctx := context.Background()

	section, _, err := GetNestedSection(ctx, c, "channels", "whatsapp")
	if err != nil {
		t.Fatalf("GetNestedSection: %v", err)
	}
	if section["dmPolicy"] != "pairing" {
		t.Errorf("expected pairing, got %v", section["dmPolicy"])
	}

	// Missing nested path
	section, _, err = GetNestedSection(ctx, c, "channels", "telegram")
	if err != nil {
		t.Fatalf("GetNestedSection missing: %v", err)
	}
	if section != nil {
		t.Errorf("expected nil, got %v", section)
	}
}

func TestPatchNestedSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")

	os.WriteFile(path, []byte(`{}`), 0o644)

	c, err := NewFileClient(path)
	if err != nil {
		t.Fatalf("NewFileClient: %v", err)
	}

	ctx := context.Background()
	cfg, _ := c.GetConfig(ctx)

	err = PatchNestedSection(ctx, c, map[string]any{"dmPolicy": "open"}, cfg.Hash, "channels", "telegram")
	if err != nil {
		t.Fatalf("PatchNestedSection: %v", err)
	}

	section, _, err := GetNestedSection(ctx, c, "channels", "telegram")
	if err != nil {
		t.Fatalf("GetNestedSection after patch: %v", err)
	}
	if section["dmPolicy"] != "open" {
		t.Errorf("expected open, got %v", section["dmPolicy"])
	}
}
