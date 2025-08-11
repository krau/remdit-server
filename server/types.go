package server

type FileSaveRequest struct {
	Content string `json:"content" binding:"required"`
}
