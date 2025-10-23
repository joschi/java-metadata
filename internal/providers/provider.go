package providers

import (
	"github.com/joschi/java-metadata/internal/models"
)

// Provider is the interface that all vendor-specific providers must implement
type Provider interface {
	// Name returns the vendor name (e.g., "temurin", "zulu")
	Name() string

	// FetchReleases fetches all available releases for this vendor
	// It returns metadata entries without checksums/sizes (those are computed separately)
	FetchReleases() ([]models.Metadata, error)
}

// Registry maintains a list of all available providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider with the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	provider, exists := r.providers[name]
	return provider, exists
}

// All returns all registered providers
func (r *Registry) All() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// Names returns all registered provider names
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
