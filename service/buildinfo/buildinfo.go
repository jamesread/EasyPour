package buildinfo

// Version, ShortCommit, and BuildDate are set at link time via ldflags (e.g. Goreleaser).
// Version defaults to "dev" so local builds still report a non-empty version.
var (
	Version     = "dev"
	ShortCommit string
	BuildDate   string
)
