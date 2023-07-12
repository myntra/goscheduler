package store

type App struct {
	AppId      string `json:"appId"`
	Partitions uint32 `json:"partitions"`
	Active     bool   `json:"active"`
}

type AppErrorResponse struct {
	Errors []string `json:"errors"`
}
