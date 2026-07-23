package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Set at link time via:
//
//	-ldflags "-X github.com/stifer/agent-hub/internal/version.Version=v0.2.0 -X github.com/stifer/agent-hub/internal/version.Commit=abc -X github.com/stifer/agent-hub/internal/version.BuildTime=..."
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// String is a one-line human summary.
func String() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		v = "dev"
	}
	c := Commit
	if len(c) > 8 {
		c = c[:8]
	}
	return fmt.Sprintf("%s (commit %s, built %s, %s/%s)", v, c, BuildTime, runtime.GOOS, runtime.GOARCH)
}

// Info is JSON-serializable build metadata.
func Info() map[string]string {
	return map[string]string{
		"version":    strings.TrimSpace(Version),
		"commit":     Commit,
		"build_time": BuildTime,
		"go":         runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
}
