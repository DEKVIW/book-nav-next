package version

// Version is injected at build time via -ldflags, e.g.:
//
//	-X github.com/booknav/book-nav/apps/server/internal/pkg/version.Version=0.1.0
//
// Defaults to a -dev label for local go run / untagged builds.
// Production Docker Hub images should pass VERSION without -dev (e.g. 0.1.0).
var Version = "0.1.0-dev"
