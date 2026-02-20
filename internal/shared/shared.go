// Package shared holds types shared between the provider and resource/datasource packages
// to avoid import cycles.
package shared

import "github.com/kylemclaren/terraform-provider-openclaw/internal/client"

// ProviderData is passed from Configure to all resources and data sources.
type ProviderData struct {
	Client client.Client
}
