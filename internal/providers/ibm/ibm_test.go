package ibm

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
	if p.Name() != models.VendorIBM {
		t.Errorf("Expected vendor %s, got %s", models.VendorIBM, p.Name())
	}
}

func TestParseFilename(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		filename  string
		version   string
		arch      string
		imageType string
		url       string
	}{
		{
			"ibm-java-sdk-8.0-6.25-x86_64-archive.tgz",
			"8.0.6.25",
			"x86_64",
			models.ImageTypeJDK,
			"https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/8.0.6.25/linux/x86_64/ibm-java-sdk-8.0-6.25-x86_64-archive.tgz",
		},
		{
			"ibm-java-jre-8.0-6.25-x86_64-archive.tgz",
			"8.0.6.25",
			"x86_64",
			models.ImageTypeJRE,
			"https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/8.0.6.25/linux/x86_64/ibm-java-jre-8.0-6.25-x86_64-archive.tgz",
		},
		{
			"ibm-java-sdk-7.1-4.70-ppc64le-archive.tgz",
			"7.1.4.70",
			"ppc64le",
			models.ImageTypeJDK,
			"https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/7.1.4.70/linux/ppc64le/ibm-java-sdk-7.1-4.70-ppc64le-archive.tgz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			m := p.parseFilename(tt.filename, tt.url, tt.version, tt.arch)

			if m.Filename == "" {
				t.Error("Expected valid metadata, got empty filename")
			}
			if m.Filename != tt.filename {
				t.Errorf("Expected filename %s, got %s", tt.filename, m.Filename)
			}
			if m.Version != tt.version {
				t.Errorf("Expected version %s, got %s", tt.version, m.Version)
			}
			if m.ImageType != tt.imageType {
				t.Errorf("Expected image type %s, got %s", tt.imageType, m.ImageType)
			}
			if m.JVMImpl != models.JVMImplOpenJ9 {
				t.Errorf("Expected JVM impl %s, got %s", models.JVMImplOpenJ9, m.JVMImpl)
			}
			if m.OS != models.OSLinux {
				t.Errorf("Expected OS %s, got %s", models.OSLinux, m.OS)
			}
		})
	}
}

func TestProviderInterface(t *testing.T) {
	var _ interface {
		Name() string
		FetchReleases(context.Context) ([]models.Metadata, error)
	} = (*Provider)(nil)
}
