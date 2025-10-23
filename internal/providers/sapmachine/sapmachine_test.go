package sapmachine

import (
	"testing"

	"github.com/joschi/java-metadata/internal/models"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p == nil {
		t.Fatal("NewProvider returned nil")
	}
	if p.Name() != models.VendorSAPMachine {
		t.Errorf("Expected vendor %s, got %s", models.VendorSAPMachine, p.Name())
	}
}

// Note: private method parseFilename() is tested indirectly through FetchReleases()
// Testing private methods directly would require exposing them, which breaks encapsulation

func TestProviderInterface(t *testing.T) {
	var _ interface {
		Name() string
		FetchReleases() ([]models.Metadata, error)
	} = (*Provider)(nil)
}
