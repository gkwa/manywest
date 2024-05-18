package version

import "fmt"

type BuildInfo struct {
	Date        string
	FullGitSHA  string
	GoVersion   string
	ShortGitSHA string
	Version     string
}

func (bi BuildInfo) String() string {
	return fmt.Sprintf(`Version: %s, %s
Build Date: %s
Go Version: %s`, bi.Version, bi.FullGitSHA, bi.Date, bi.GoVersion)
}

var (
	Date        string
	FullGitSHA  string
	GoVersion   string
	ShortGitSHA string
	Version     string
)

func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Date:        Date,
		FullGitSHA:  FullGitSHA,
		GoVersion:   GoVersion,
		ShortGitSHA: ShortGitSHA,
		Version:     Version,
	}
}
