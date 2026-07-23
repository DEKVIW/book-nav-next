package version

// 构建时可通过 -ldflags 注入。
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
