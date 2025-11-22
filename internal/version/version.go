package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set via ldflags)
	Version = "dev"

	// Commit is the git commit SHA (set via ldflags)
	Commit = "unknown"

	// Date is the build date (set via ldflags)
	Date = "unknown"
)

// Info holds version information
type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Platform  string
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("claude-sync %s", i.Version)
}

// Verbose returns a detailed version string
func (i Info) Verbose() string {
	return fmt.Sprintf(`claude-sync version information:
  Version:    %s
  Commit:     %s
  Built:      %s
  Go version: %s
  Platform:   %s`,
		i.Version,
		i.Commit,
		i.Date,
		i.GoVersion,
		i.Platform,
	)
}

// ShortCommit returns the short commit hash (first 7 chars)
func (i Info) ShortCommit() string {
	if len(i.Commit) >= 7 {
		return i.Commit[:7]
	}
	return i.Commit
}
