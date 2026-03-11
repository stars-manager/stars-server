package version

import (
	"fmt"
	"runtime"
)

var (
	// Version 版本号（通过 ldflags 注入）
	Version = "dev"
	// GitCommit Git 提交 hash（通过 ldflags 注入）
	GitCommit = "unknown"
	// BuildTime 构建时间（通过 ldflags 注入）
	BuildTime = "unknown"
)

// Info 版本信息
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get 获取版本信息
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String 返回版本信息的字符串表示
func String() string {
	return fmt.Sprintf("Version: %s\nGitCommit: %s\nBuildTime: %s\nGoVersion: %s\nPlatform: %s",
		Version, GitCommit, BuildTime, runtime.Version(), fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}
