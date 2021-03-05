package version

const (
	version = "v0.0.1"
)

var (
	Tag       = ""
	Branch    = ""
	CommitId  = ""
	BuildTime = ""
)

type Info struct {
	Version   string `json:"version"`
	Tag       string `json:"tag"`
	Branch    string `json:"branch"`
	CommitId  string `json:"commit_id"`
	BuildTime string `json:"build_time"`
}

func GetVersion() *Info {
	return &Info{
		Version:   version,
		Tag:       Tag,
		Branch:    Branch,
		CommitId:  CommitId,
		BuildTime: BuildTime,
	}
}
