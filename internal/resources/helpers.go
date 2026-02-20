package resources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// isConnectionClosed returns true if the error indicates the WebSocket
// connection was closed — typically because the gateway restarted after
// a config write. This is expected and not a real failure.
func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection closed") ||
		strings.Contains(msg, "websocket: close") ||
		strings.Contains(msg, "use of closed network connection")
}

// ── Model → Map helpers (for writing config) ────────────────

func setIfString(m map[string]any, key string, val types.String) {
	if !val.IsNull() && !val.IsUnknown() {
		m[key] = val.ValueString()
	}
}

func setIfBool(m map[string]any, key string, val types.Bool) {
	if !val.IsNull() && !val.IsUnknown() {
		m[key] = val.ValueBool()
	}
}

func setIfInt64(m map[string]any, key string, val types.Int64) {
	if !val.IsNull() && !val.IsUnknown() {
		m[key] = val.ValueInt64()
	}
}

func setIfStringList(ctx context.Context, m map[string]any, key string, val types.List) {
	if !val.IsNull() && !val.IsUnknown() {
		var strs []string
		val.ElementsAs(ctx, &strs, false)
		m[key] = strs
	}
}

// ── Map → Model helpers (for reading config) ────────────────

func readString(m map[string]any, key string, target *types.String) {
	if v, ok := m[key].(string); ok {
		*target = types.StringValue(v)
	}
}

func readBool(m map[string]any, key string, target *types.Bool) {
	if v, ok := m[key].(bool); ok {
		*target = types.BoolValue(v)
	}
}

func readFloat64AsInt64(m map[string]any, key string, target *types.Int64) {
	if v, ok := m[key].(float64); ok {
		*target = types.Int64Value(int64(v))
	}
}

func readStringList(ctx context.Context, m map[string]any, key string, target *types.List) {
	if v, ok := m[key].([]any); ok {
		strs := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				strs = append(strs, str)
			}
		}
		list, _ := types.ListValueFrom(ctx, types.StringType, strs)
		*target = list
	}
}
