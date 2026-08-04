package buildinfo

// Version, ShortCommit, and BuildDate are set at link time via ldflags (e.g. Goreleaser).
var (
	Version    string
	ShortCommit string
	BuildDate   string
)
