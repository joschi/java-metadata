package semeru

import (
	"github.com/joschi/java-metadata/internal/models"
	"github.com/joschi/java-metadata/internal/providers/github"
)

// NewSemeru8Provider creates IBM Semeru 8 provider
func NewSemeru8Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru8-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru11Provider creates IBM Semeru 11 provider
func NewSemeru11Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru11-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru16Provider creates IBM Semeru 16 provider
func NewSemeru16Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru16-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru17Provider creates IBM Semeru 17 provider
func NewSemeru17Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru17-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru18Provider creates IBM Semeru 18 provider
func NewSemeru18Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru18-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru19Provider creates IBM Semeru 19 provider
func NewSemeru19Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru19-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru20Provider creates IBM Semeru 20 provider
func NewSemeru20Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru20-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru21Provider creates IBM Semeru 21 provider
func NewSemeru21Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru21-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru22Provider creates IBM Semeru 22 provider
func NewSemeru22Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru22-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru23Provider creates IBM Semeru 23 provider
func NewSemeru23Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru23-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru24Provider creates IBM Semeru 24 provider
func NewSemeru24Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru24-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru25Provider creates IBM Semeru 25 provider
func NewSemeru25Provider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru25-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru11CertifiedProvider creates IBM Semeru 11 Certified provider
func NewSemeru11CertifiedProvider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru11-certified-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru17CertifiedProvider creates IBM Semeru 17 Certified provider
func NewSemeru17CertifiedProvider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru17-certified-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}

// NewSemeru21CertifiedProvider creates IBM Semeru 21 Certified provider
func NewSemeru21CertifiedProvider() *github.GenericProvider {
	return github.NewGenericProvider(models.VendorSemeru, "ibmruntimes", "semeru21-certified-binaries",
		github.ParseSemeruFilename(models.VendorSemeru))
}
