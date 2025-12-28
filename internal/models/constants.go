package models

// Vendor names as defined in the OpenAPI specification
const (
	VendorAdoptOpenJDK     = "adoptopenjdk"
	VendorBisheng          = "bisheng"
	VendorCorretto         = "corretto"
	VendorDragonwell       = "dragonwell"
	VendorGraalVM          = "graalvm"
	VendorGraalVMCommunity = "graalvm-community"
	VendorIBM              = "ibm"
	VendorJavaSERI         = "java-se-ri"
	VendorJetBrains        = "jetbrains"
	VendorKona             = "kona"
	VendorLiberica         = "liberica"
	VendorMandrel          = "mandrel"
	VendorMicrosoft        = "microsoft"
	VendorOpenJDK          = "openjdk"
	VendorOracle           = "oracle"
	VendorOracleGraalVM    = "oracle-graalvm"
	VendorRedHat           = "redhat"
	VendorSAPMachine       = "sapmachine"
	VendorSemeru           = "semeru"
	VendorTemurin          = "temurin"
	VendorTrava            = "trava"
	VendorZulu             = "zulu"
)

// Operating system names as defined in the OpenAPI specification
const (
	OSLinux   = "linux"
	OSMacOSX  = "macosx"
	OSWindows = "windows"
	OSSolaris = "solaris"
	OSAIX     = "aix"
)

// Architecture names as defined in the OpenAPI specification
const (
	ArchX86_64       = "x86_64"
	ArchI686         = "i686"
	ArchAArch64      = "aarch64"
	ArchARM32        = "arm32"
	ArchARM32VFPHFLT = "arm32-vfp-hflt"
	ArchPPC32        = "ppc32"
	ArchPPC64        = "ppc64"
	ArchPPC64LE      = "ppc64le"
	ArchS390         = "s390"
	ArchS390X        = "s390x"
	ArchSPARCV9      = "sparcv9"
	ArchRISCV64      = "riscv64"
)

// Release types as defined in the OpenAPI specification
const (
	ReleaseTypeEA = "ea" // Early Access
	ReleaseTypeGA = "ga" // General Availability
)

// Image types as defined in the OpenAPI specification
const (
	ImageTypeJRE = "jre"
	ImageTypeJDK = "jdk"
)

// JVM implementations as defined in the OpenAPI specification
const (
	JVMImplHotSpot = "hotspot"
	JVMImplOpenJ9  = "openj9"
	JVMImplGraalVM = "graalvm"
)

// Hash algorithm names
const (
	HashMD5    = "md5"
	HashSHA1   = "sha1"
	HashSHA256 = "sha256"
	HashSHA512 = "sha512"
)

// NormalizeOS converts various OS name formats to the canonical format
func NormalizeOS(os string) string {
	switch os {
	case "linux", "Linux", "alpine-linux":
		return OSLinux
	case "mac", "macos", "macosx", "osx", "darwin", "macOS":
		return OSMacOSX
	case "win", "windows", "Windows":
		return OSWindows
	case "solaris":
		return OSSolaris
	case "aix":
		return OSAIX
	default:
		return "unknown-os-" + os
	}
}

// NormalizeArchitecture converts various architecture formats to the canonical format
func NormalizeArchitecture(arch string) string {
	switch arch {
	case "amd64", "x64", "x86_64", "x86-64":
		return ArchX86_64
	case "x32", "x86", "x86_32", "x86-32", "i386", "i586", "i686":
		return ArchI686
	case "aarch64", "arm64":
		return ArchAArch64
	case "arm", "arm32", "armv7", "aarch32sf":
		return ArchARM32
	case "arm32-vfp-hflt", "aarch32hf":
		return ArchARM32VFPHFLT
	case "ppc":
		return ArchPPC32
	case "ppc64":
		return ArchPPC64
	case "ppc64le":
		return ArchPPC64LE
	case "s390":
		return ArchS390
	case "s390x":
		return ArchS390X
	case "sparcv9":
		return ArchSPARCV9
	case "riscv64":
		return ArchRISCV64
	default:
		return "unknown-architecture-" + arch
	}
}
