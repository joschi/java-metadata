package models

// Metadata represents the metadata for a JRE/JDK distribution artifact.
// This struct matches the OpenAPI schema definition exactly.
type Metadata struct {
	Vendor       string   `json:"vendor"`
	Filename     string   `json:"filename"`
	ReleaseType  string   `json:"release_type"`
	Version      string   `json:"version"`
	JavaVersion  string   `json:"java_version"`
	JVMImpl      string   `json:"jvm_impl"`
	OS           string   `json:"os"`
	Architecture string   `json:"architecture"`
	FileType     string   `json:"file_type"`
	ImageType    string   `json:"image_type"`
	Features     []string `json:"features"`
	URL          string   `json:"url"`
	MD5          string   `json:"md5"`
	MD5File      string   `json:"md5_file"`
	SHA1         string   `json:"sha1"`
	SHA1File     string   `json:"sha1_file"`
	SHA256       string   `json:"sha256"`
	SHA256File   string   `json:"sha256_file"`
	SHA512       string   `json:"sha512"`
	SHA512File   string   `json:"sha512_file"`
	Size         int64    `json:"size"`
}

// MetadataList is a collection of metadata entries.
type MetadataList []Metadata
