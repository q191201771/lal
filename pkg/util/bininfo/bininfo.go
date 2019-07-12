package bininfo

import "fmt"

// 编译时通过如下方式传入编译时信息
// go build -ldflags " \
//   -X 'github.com/q191201771/lal/pkg/bininfo.BuildTime=`date +'%Y.%m.%d.%H%M%S'`' \
//   -X 'github.com/q191201771/lal/pkg/bininfo.GitCommitID=`git log --pretty=format:'%h' -n 1`' \
//   -X 'github.com/q191201771/lal/pkg/bininfo.GoVersion=`go version`' \
// "

var (
	BuildTime   string
	GitCommitID string
	GoVersion   string
)

func StringifySingleLine() string {
	return fmt.Sprintf("BuildTime: %s. GitCommitID: %s. GoVersion: %s.", BuildTime, GitCommitID, GoVersion)
}

func StringifyMultiLine() string {
	return fmt.Sprintf("BuildTime: %s\nGitCommitID: %s\nGoVersion: %s\n", BuildTime, GitCommitID, GoVersion)
}
