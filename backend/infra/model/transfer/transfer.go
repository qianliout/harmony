// Package transfer 定义导入导出用的行结构与预览结果。
package transfer

type Row struct {
	Date          string `json:"date"`
	PersonName    string `json:"personName"`
	Relation      string `json:"relation"`
	EventName     string `json:"eventName"`
	EventType     string `json:"eventType"`
	EventDate     string `json:"eventDate"`
	BookName      string `json:"bookName"`
	ItemKind      string `json:"itemKind"`
	Direction     string `json:"direction"`
	Amount        string `json:"amount"`
	GiftName      string `json:"giftName"`
	GiftValue     string `json:"giftValue"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
}

type PreviewRow struct {
	Line   int      `json:"line"`
	OK     bool     `json:"ok"`
	Errors []string `json:"errors"`
	Row    Row      `json:"row"`
}

type Preview struct {
	Rows       []PreviewRow `json:"rows"`
	Total      int          `json:"total"`
	OKCount    int          `json:"okCount"`
	ErrorCount int          `json:"errorCount"`
}

type CommitParam struct {
	Rows []Row `json:"rows"`
}

type CommitResult struct {
	Count      int `json:"count"`
	NewPersons int `json:"newPersons"`
	NewEvents  int `json:"newEvents"`
	NewBooks   int `json:"newBooks"`
}
