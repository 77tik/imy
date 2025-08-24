package utils

import (
	"fmt"
	"runtime"
)

var (
	Version   = "unknown"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

func GetVersionInfo() string {
	return fmt.Sprintf("Version: %s\nGit Commit: %s\nBuild Time: %s\nGo Version: %s\nPlatform: %s",
		Version,
		GitCommit,
		BuildTime,
		GoVersion,
		Platform,
	)
}
