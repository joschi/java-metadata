package models

import "testing"

func TestNormalizeOS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"linux", OSLinux},
		{"Linux", OSLinux},
		{"mac", OSMacOSX},
		{"macos", OSMacOSX},
		{"macosx", OSMacOSX},
		{"osx", OSMacOSX},
		{"darwin", OSMacOSX},
		{"macOS", OSMacOSX},
		{"windows", OSWindows},
		{"Windows", OSWindows},
		{"win", OSWindows},
		{"solaris", OSSolaris},
		{"aix", OSAIX},
		{"unknown", "unknown-os-unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeOS(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeOS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeArchitecture(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x64", ArchX86_64},
		{"x86_64", ArchX86_64},
		{"amd64", ArchX86_64},
		{"x86-64", ArchX86_64},
		{"x32", ArchI686},
		{"x86_32", ArchI686},
		{"x86", ArchI686},
		{"i386", ArchI686},
		{"i586", ArchI686},
		{"i686", ArchI686},
		{"aarch64", ArchAArch64},
		{"arm64", ArchAArch64},
		{"arm", ArchARM32},
		{"arm32", ArchARM32},
		{"ppc64", ArchPPC64},
		{"ppc64le", ArchPPC64LE},
		{"s390x", ArchS390X},
		{"riscv64", ArchRISCV64},
		{"sparcv9", ArchSPARCV9},
		{"unknown", "unknown-architecture-unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeArchitecture(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeArchitecture(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMetadataValidation(t *testing.T) {
	// Test that a valid metadata struct can be created
	m := Metadata{
		Vendor:       VendorTemurin,
		Filename:     "test.tar.gz",
		ReleaseType:  ReleaseTypeGA,
		Version:      "17.0.1",
		JavaVersion:  "17.0.1+12",
		JVMImpl:      JVMImplHotSpot,
		OS:           OSLinux,
		Architecture: ArchX86_64,
		FileType:     "tar.gz",
		ImageType:    ImageTypeJDK,
		Features:     []string{},
		URL:          "https://example.com/test.tar.gz",
	}

	if m.Vendor != VendorTemurin {
		t.Errorf("Expected vendor %s, got %s", VendorTemurin, m.Vendor)
	}
	if m.OS != OSLinux {
		t.Errorf("Expected OS %s, got %s", OSLinux, m.OS)
	}
}
