package health

type Status struct {
	Status      string `json:"status"`
	Database    string `json:"database"`
	JournalMode string `json:"journalMode"`
	Version     string `json:"version"`
	Time        int64  `json:"time"`
}
