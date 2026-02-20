package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileClient reads and writes the OpenClaw config file directly.
// This is the fallback for when no running Gateway is available
// (e.g. pre-provisioning a config before first boot).
type FileClient struct {
	path string
	mu   sync.Mutex
}

// NewFileClient creates a client that operates on the given config file path.
// The path is expanded (~ -> home dir) but the file need not exist yet.
func NewFileClient(path string) (*FileClient, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("expanding config path: %w", err)
	}
	return &FileClient{path: expanded}, nil
}

// GetConfig implements Client.
func (f *FileClient) GetConfig(_ context.Context) (*ConfigPayload, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getConfigLocked()
}

// getConfigLocked reads the config file. Caller must hold f.mu.
func (f *FileClient) getConfigLocked() (*ConfigPayload, error) {
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		// No config file yet -- return empty config.
		return &ConfigPayload{
			Raw:  "{}",
			Hash: hashBytes([]byte("{}")),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", f.path, err)
	}

	raw := string(data)

	return &ConfigPayload{
		Raw:  raw,
		Hash: hashBytes(data),
	}, nil
}

// PatchConfig implements Client.
// In file mode, concurrent access is serialized by the mutex, so the caller-
// provided baseHash is intentionally ignored. The mutex guarantees that no
// other goroutine can modify the file between our read and write, making
// optimistic-concurrency checks unnecessary (and counterproductive when
// Terraform applies multiple resources in parallel).
func (f *FileClient) PatchConfig(_ context.Context, patch map[string]any, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	cfg, err := f.getConfigLocked()
	if err != nil {
		return err
	}

	existing, err := parseRawJSON(cfg.Raw)
	if err != nil {
		return fmt.Errorf("parsing existing config: %w", err)
	}

	merged := mergePatch(existing, patch)

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := ensureDir(f.path); err != nil {
		return err
	}

	return os.WriteFile(f.path, out, 0o644)
}

// ApplyConfig implements Client.
// Like PatchConfig, the baseHash is ignored in file mode because the mutex
// serializes all access.
func (f *FileClient) ApplyConfig(_ context.Context, raw string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := ensureDir(f.path); err != nil {
		return err
	}

	return os.WriteFile(f.path, []byte(raw), 0o644)
}

// Health implements Client. Not supported in file mode.
func (f *FileClient) Health(_ context.Context) (*HealthPayload, error) {
	return nil, fmt.Errorf("health check not available in file mode (no running gateway)")
}

// Close implements Client.
func (f *FileClient) Close() error {
	return nil
}

// mergePatch applies RFC 7396 JSON Merge Patch semantics.
func mergePatch(target, patch map[string]any) map[string]any {
	if target == nil {
		target = make(map[string]any)
	}

	for key, patchVal := range patch {
		if patchVal == nil {
			delete(target, key)
			continue
		}

		patchMap, patchIsMap := patchVal.(map[string]any)
		if patchIsMap {
			targetVal, ok := target[key]
			if !ok {
				target[key] = patchMap
				continue
			}
			targetMap, targetIsMap := targetVal.(map[string]any)
			if targetIsMap {
				target[key] = mergePatch(targetMap, patchMap)
			} else {
				target[key] = patchMap
			}
		} else {
			target[key] = patchVal
		}
	}

	return target
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path), nil
}

func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return os.MkdirAll(dir, 0o755)
}
