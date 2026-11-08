package health

type Status struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
	Time     int64  `json:"time"`
}
