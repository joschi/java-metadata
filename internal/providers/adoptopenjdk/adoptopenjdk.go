package adoptopenjdk

import (
	"github.com/joschi/java-metadata/internal/models"
	"github.com/joschi/java-metadata/internal/providers/temurin"
)

// Provider implements the Provider interface for AdoptOpenJDK (legacy)
// AdoptOpenJDK was the predecessor to Eclipse Temurin (Adoptium)
// It uses the same API but with vendor=adoptopenjdk
type Provider struct {
	*temurin.Provider
}

// NewProvider creates a new AdoptOpenJDK provider
func NewProvider() *Provider {
	return &Provider{
		Provider: temurin.NewProviderWithVendor(models.VendorAdoptOpenJDK, "adoptopenjdk"),
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorAdoptOpenJDK
}
