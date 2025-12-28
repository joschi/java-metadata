package temurin

import (
	"context"
	"testing"

	"github.com/joschi/java-metadata/internal/models"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p == nil {
		t.Fatal("NewProvider returned nil")
	}
	if p.Name() != models.VendorTemurin {
		t.Errorf("Expected vendor %s, got %s", models.VendorTemurin, p.Name())
	}
}

func TestNewProviderWithVendor(t *testing.T) {
	p := NewProviderWithVendor(models.VendorAdoptOpenJDK, "adoptopenjdk")
	if p == nil {
		t.Fatal("NewProviderWithVendor returned nil")
	}
	if p.Name() != models.VendorAdoptOpenJDK {
		t.Errorf("Expected vendor %s, got %s", models.VendorAdoptOpenJDK, p.Name())
	}
}

// Note: private methods like parseVersion() and parseFeatures() are tested indirectly through FetchReleases()
// Testing private methods directly would require exposing them, which breaks encapsulation

func TestProviderInterface(t *testing.T) {
	// Verify that Provider implements the Provider interface
	var _ interface {
		Name() string
		FetchReleases(context.Context) ([]models.Metadata, error)
	} = (*Provider)(nil)
}
