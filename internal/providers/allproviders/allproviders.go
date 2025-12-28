package allproviders

import (
	"github.com/joschi/java-metadata/internal/models"
	"github.com/joschi/java-metadata/internal/providers"
	"github.com/joschi/java-metadata/internal/providers/adoptopenjdk"
	"github.com/joschi/java-metadata/internal/providers/corretto"
	"github.com/joschi/java-metadata/internal/providers/github"
	"github.com/joschi/java-metadata/internal/providers/ibm"
	"github.com/joschi/java-metadata/internal/providers/javase"
	"github.com/joschi/java-metadata/internal/providers/microsoft"
	"github.com/joschi/java-metadata/internal/providers/oracle"
	"github.com/joschi/java-metadata/internal/providers/oraclegraalvm"
	"github.com/joschi/java-metadata/internal/providers/sapmachine"
	"github.com/joschi/java-metadata/internal/providers/semeru"
	"github.com/joschi/java-metadata/internal/providers/temurin"
	"github.com/joschi/java-metadata/internal/providers/zulu"
)

// RegisterAll registers all available providers with the registry
func RegisterAll(registry *providers.Registry) {
	// Core providers
	registry.Register(adoptopenjdk.NewProvider())
	registry.Register(corretto.NewProvider())
	registry.Register(microsoft.NewProvider())
	registry.Register(sapmachine.NewProvider())
	registry.Register(temurin.NewProvider())
	registry.Register(zulu.NewProvider())

	// IBM Semeru (15 variants)
	registry.Register(semeru.NewSemeru8Provider())
	registry.Register(semeru.NewSemeru11Provider())
	registry.Register(semeru.NewSemeru11CertifiedProvider())
	registry.Register(semeru.NewSemeru16Provider())
	registry.Register(semeru.NewSemeru17Provider())
	registry.Register(semeru.NewSemeru17CertifiedProvider())
	registry.Register(semeru.NewSemeru18Provider())
	registry.Register(semeru.NewSemeru19Provider())
	registry.Register(semeru.NewSemeru20Provider())
	registry.Register(semeru.NewSemeru21Provider())
	registry.Register(semeru.NewSemeru21CertifiedProvider())
	registry.Register(semeru.NewSemeru22Provider())
	registry.Register(semeru.NewSemeru23Provider())
	registry.Register(semeru.NewSemeru24Provider())
	registry.Register(semeru.NewSemeru25Provider())
	registry.Register(semeru.NewSemeru25CertifiedProvider())

	// GraalVM variants (using GitHub)
	registry.Register(github.NewGenericProvider(models.VendorGraalVM, "graalvm", "graalvm-ce-builds",
		github.ParseGraalVMFilename("graalvm")))
	registry.Register(github.NewGenericProvider(models.VendorGraalVMCommunity, "graalvm", "graalvm-ce-builds",
		github.ParseGraalVMFilename("graalvm-community")))

	// Mandrel
	registry.Register(github.NewGenericProvider(models.VendorMandrel, "graalvm", "mandrel",
		github.ParseMandrelFilename()))

	// Dragonwell variants (4)
	registry.Register(github.NewGenericProvider(models.VendorDragonwell, "alibaba", "dragonwell8",
		github.ParseDragonwellFilename(models.VendorDragonwell)))
	registry.Register(github.NewGenericProvider(models.VendorDragonwell, "alibaba", "dragonwell11",
		github.ParseDragonwellFilename(models.VendorDragonwell)))
	registry.Register(github.NewGenericProvider(models.VendorDragonwell, "alibaba", "dragonwell17",
		github.ParseDragonwellFilename(models.VendorDragonwell)))
	registry.Register(github.NewGenericProvider(models.VendorDragonwell, "alibaba", "dragonwell21",
		github.ParseDragonwellFilename(models.VendorDragonwell)))

	// Trava variants (2)
	registry.Register(github.NewGenericProvider(models.VendorTrava, "TravaOpenJDK", "trava-jdk-8-dcevm",
		github.ParseTravaFilename(models.VendorTrava)))
	registry.Register(github.NewGenericProvider(models.VendorTrava, "TravaOpenJDK", "trava-jdk-11-dcevm",
		github.ParseTravaFilename(models.VendorTrava)))

	// JetBrains Runtime
	registry.Register(github.NewGenericProvider(models.VendorJetBrains, "JetBrains", "JetBrainsRuntime",
		github.ParseJetBrainsFilename()))

	// Tencent Kona variants (4)
	registry.Register(github.NewGenericProvider(models.VendorKona, "Tencent", "TencentKona-8",
		github.ParseKonaFilename(models.VendorKona)))
	registry.Register(github.NewGenericProvider(models.VendorKona, "Tencent", "TencentKona-11",
		github.ParseKonaFilename(models.VendorKona)))
	registry.Register(github.NewGenericProvider(models.VendorKona, "Tencent", "TencentKona-17",
		github.ParseKonaFilename(models.VendorKona)))
	registry.Register(github.NewGenericProvider(models.VendorKona, "Tencent", "TencentKona-21",
		github.ParseKonaFilename(models.VendorKona)))

	// OpenJDK variants (4)
	registry.Register(github.NewGenericProvider(models.VendorOpenJDK, "openjdk", "jdk",
		github.ParseOpenJDKFilename(models.VendorOpenJDK)))
	registry.Register(github.NewGenericProvider(models.VendorOpenJDK, "openjdk", "leyden",
		github.ParseOpenJDKFilename(models.VendorOpenJDK)))
	registry.Register(github.NewGenericProvider(models.VendorOpenJDK, "openjdk", "loom",
		github.ParseOpenJDKFilename(models.VendorOpenJDK)))
	registry.Register(github.NewGenericProvider(models.VendorOpenJDK, "openjdk", "valhalla",
		github.ParseOpenJDKFilename(models.VendorOpenJDK)))

	// GraalVM Legacy
	registry.Register(github.NewGenericProvider(models.VendorGraalVM, "graalvm", "graalvm-ce-builds",
		github.ParseGraalVMFilename(models.VendorGraalVM)))

	// Red Hat OpenJDK
	registry.Register(github.NewGenericProvider(models.VendorRedHat, "redhat-openjdk", "openjdk",
		github.ParseRedHatFilename()))

	// BiSheng JDK
	registry.Register(github.NewGenericProvider(models.VendorBisheng, "openeuler-mirror", "bishengjdk-11",
		github.ParseBiShengFilename()))

	// BellSoft Liberica
	registry.Register(github.NewGenericProvider(models.VendorLiberica, "bell-sw", "Liberica",
		github.ParseLibericaFilename()))

	// Java SE Reference Implementation
	registry.Register(javase.NewProvider())

	// Oracle JDK
	registry.Register(oracle.NewProvider())

	// Oracle GraalVM
	registry.Register(oraclegraalvm.NewProvider())
	registry.Register(oraclegraalvm.NewEAProvider())

	// IBM JDK (legacy)
	registry.Register(ibm.NewProvider())
}
