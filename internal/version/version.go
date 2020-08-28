package version

import (
	"fmt"
	"runtime"
)

var (
	GitVersion    = "unknown"
	GitCommit     = "unknown"
	GitCommitTime = "unknown"
	GitTreeState  = "unknown"
	GoVersion     = runtime.Version()
	Compiler      = runtime.Compiler
	Platform      = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

type Info struct {
	GitVersion    string
	GitCommit     string
	GitCommitTime string
	GitTreeState  string
	GoVersion     string
	Compiler      string
	Platform      string
}

var Version Info

func init() {
	Version = Info{
		GitVersion:    GitVersion,
		GitCommit:     GitCommit,
		GitCommitTime: GitCommitTime,
		GitTreeState:  GitTreeState,
		GoVersion:     GoVersion,
		Compiler:      Compiler,
		Platform:      Platform,
	}
}
