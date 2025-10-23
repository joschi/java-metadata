package github

import (
	"testing"

	"github.com/joschi/java-metadata/internal/models"
)

func TestNewGenericProvider(t *testing.T) {
	parser := func(filename, url, tagName string) models.Metadata {
		return models.Metadata{Filename: filename}
	}

	p := NewGenericProvider(models.VendorGraalVM, "graalvm", "graalvm-ce-builds", parser)
	if p == nil {
		t.Fatal("NewGenericProvider returned nil")
	}
	if p.Name() != models.VendorGraalVM {
		t.Errorf("Expected vendor %s, got %s", models.VendorGraalVM, p.Name())
	}
}

func TestParseSemeruFilename(t *testing.T) {
	parser := ParseSemeruFilename(models.VendorSemeru)

	tests := []struct {
		filename string
		tagName  string
		valid    bool
		version  string
		os       string
		arch     string
	}{
		{
			"ibm-semeru-open-jdk_x64_linux_11.0.13_8_openj9-0.29.0.tar.gz",
			"jdk-11.0.13+8_openj9-0.29.0",
			true,
			"11.0.13+8_openj9-0.29.0",
			models.OSLinux,
			models.ArchX86_64,
		},
		{
			"ibm-semeru-open-jre_aarch64_mac_11.0.13_8_openj9-0.29.0.tar.gz",
			"jdk-11.0.13+8_openj9-0.29.0",
			true,
			"11.0.13+8_openj9-0.29.0",
			models.OSMacOSX,
			models.ArchAArch64,
		},
		{
			"invalid-filename.tar.gz",
			"jdk-11.0.13+8",
			false,
			"",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			m := parser(tt.filename, "https://example.com/"+tt.filename, tt.tagName)
			if tt.valid {
				if m.Filename == "" {
					t.Error("Expected valid metadata, got empty filename")
				}
				if m.Version != tt.version {
					t.Errorf("Expected version %s, got %s", tt.version, m.Version)
				}
				if m.OS != tt.os {
					t.Errorf("Expected OS %s, got %s", tt.os, m.OS)
				}
				if m.Architecture != tt.arch {
					t.Errorf("Expected arch %s, got %s", tt.arch, m.Architecture)
				}
			} else {
				if m.Filename != "" {
					t.Error("Expected empty metadata for invalid filename")
				}
			}
		})
	}
}

func TestParseGraalVMFilename(t *testing.T) {
	parser := ParseGraalVMFilename(models.VendorGraalVM)

	tests := []struct {
		filename string
		tagName  string
		valid    bool
	}{
		{
			"graalvm-ce-java17-linux-amd64-22.3.0.tar.gz",
			"vm-22.3.0",
			true,
		},
		{
			"graalvm-ce-java11-darwin-aarch64-22.3.0.tar.gz",
			"vm-22.3.0",
			true,
		},
		{
			"invalid-filename.tar.gz",
			"vm-22.3.0",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			m := parser(tt.filename, "https://example.com/"+tt.filename, tt.tagName)
			if tt.valid {
				if m.Filename == "" {
					t.Error("Expected valid metadata, got empty filename")
				}
				if m.Version != tt.tagName {
					t.Errorf("Expected version %s, got %s", tt.tagName, m.Version)
				}
			} else {
				if m.Filename != "" {
					t.Error("Expected empty metadata for invalid filename")
				}
			}
		})
	}
}

func TestParseDragonwellFilename(t *testing.T) {
	parser := ParseDragonwellFilename(models.VendorDragonwell)

	tests := []struct {
		filename string
		valid    bool
		os       string
		arch     string
	}{
		{
			"Alibaba_Dragonwell_Extended_11.0.13.8_x64_linux_jdk.tar.gz",
			true,
			models.OSLinux,
			models.ArchX86_64,
		},
		{
			"Alibaba_Dragonwell_Standard_11.0.13.8_linux_aarch64_jdk.tar.gz",
			true,
			models.OSLinux,
			models.ArchAArch64,
		},
		{
			"invalid-filename.tar.gz",
			false,
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			m := parser(tt.filename, "https://example.com/"+tt.filename, "dragonwell-11.0.13.8_jdk")
			if tt.valid {
				if m.Filename == "" {
					t.Error("Expected valid metadata, got empty filename")
				}
				if m.OS != tt.os {
					t.Errorf("Expected OS %s, got %s", tt.os, m.OS)
				}
				if m.Architecture != tt.arch {
					t.Errorf("Expected arch %s, got %s", tt.arch, m.Architecture)
				}
			} else {
				if m.Filename != "" {
					t.Error("Expected empty metadata for invalid filename")
				}
			}
		})
	}
}

func TestParseKonaFilename(t *testing.T) {
	parser := ParseKonaFilename(models.VendorKona)

	tests := []struct {
		filename string
		valid    bool
		features []string
	}{
		{
			"TencentKona-11.0.13_jdk_linux-x86_64.tar.gz",
			true,
			[]string{},
		},
		{
			"TencentKona-11.0.13_jdk_fiber_linux-x86_64.tar.gz",
			true,
			[]string{"fiber"},
		},
		{
			"invalid-filename.tar.gz",
			false,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			m := parser(tt.filename, "https://example.com/"+tt.filename, "TencentKona-11.0.13")
			if tt.valid {
				if m.Filename == "" {
					t.Error("Expected valid metadata, got empty filename")
				}
				if len(m.Features) != len(tt.features) {
					t.Errorf("Expected %d features, got %d", len(tt.features), len(m.Features))
				}
			} else {
				if m.Filename != "" {
					t.Error("Expected empty metadata for invalid filename")
				}
			}
		})
	}
}

// Note: GitHub provider is tested through NewGenericProvider in integration tests
