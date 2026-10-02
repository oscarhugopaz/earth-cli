// Package version exposes build-time version metadata.
//
// The variables are intentionally mutable so the release process can inject
// real values with -ldflags (-X). Development builds keep the defaults.
package version

var (
	// Version is the semantic version of the build (for example "0.1.0").
	Version = "dev"
	// Commit is the short git commit the binary was built from.
	Commit = "none"
	// Date is the build/commit date in RFC3339 form.
	Date = "unknown"
)
