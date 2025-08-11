package server

type FileSaveRequest struct {
	Content string `json:"content" binding:"required"`
}

type SaveResultMessage struct {
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

type SaveResult struct {
	Success bool
	Reason  string
}
