package providers

import (
	"errors"
	"testing"

	"github.com/joschi/java-metadata/internal/models"
)

// Mock provider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) FetchReleases() ([]models.Metadata, error) {
	return []models.Metadata{
		{
			Vendor:   m.name,
			Filename: "test.tar.gz",
		},
	}, nil
}

// Mock failing provider for testing
type failingProvider struct {
	name string
}

func (f *failingProvider) Name() string {
	return f.name
}

func (f *failingProvider) FetchReleases() ([]models.Metadata, error) {
	return nil, errors.New("intentional test failure")
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestRegisterProvider(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test-vendor"}

	r.Register(p)

	providers := r.All()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if providers[0].Name() != "test-vendor" {
		t.Errorf("Expected provider name 'test-vendor', got '%s'", providers[0].Name())
	}
}

func TestRegisterMultipleProviders(t *testing.T) {
	r := NewRegistry()
	p1 := &mockProvider{name: "vendor1"}
	p2 := &mockProvider{name: "vendor2"}
	p3 := &mockProvider{name: "vendor3"}

	r.Register(p1)
	r.Register(p2)
	r.Register(p3)

	providers := r.All()
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}
}

func TestGetAllProviders(t *testing.T) {
	r := NewRegistry()
	p1 := &mockProvider{name: "vendor1"}
	p2 := &mockProvider{name: "vendor2"}

	r.Register(p1)
	r.Register(p2)

	providers := r.All()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Verify both providers are present
	names := make(map[string]bool)
	for _, p := range providers {
		names[p.Name()] = true
	}

	if !names["vendor1"] {
		t.Error("vendor1 not found in registry")
	}
	if !names["vendor2"] {
		t.Error("vendor2 not found in registry")
	}
}

func TestProviderInterface(t *testing.T) {
	// Verify that mockProvider implements Provider interface
	var _ Provider = (*mockProvider)(nil)
}

func TestProviderFetchReleases(t *testing.T) {
	p := &mockProvider{name: "test-vendor"}
	metadata, err := p.FetchReleases()

	if err != nil {
		t.Fatalf("FetchReleases failed: %v", err)
	}

	if len(metadata) != 1 {
		t.Errorf("Expected 1 metadata entry, got %d", len(metadata))
	}

	if metadata[0].Vendor != "test-vendor" {
		t.Errorf("Expected vendor 'test-vendor', got '%s'", metadata[0].Vendor)
	}
}

func TestFailingProvider(t *testing.T) {
	p := &failingProvider{name: "failing-vendor"}
	_, err := p.FetchReleases()

	if err == nil {
		t.Error("Expected error from failing provider, got nil")
	}
}
