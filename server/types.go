package server

type FileInfoMessagePayload struct {
	FileID  string
	EditUrl string
}

type SessionState uint

const (
	SessionStateNone SessionState = iota
	SessionStateFileUpload
	SessionStateFileInfo
	SessionStateListen
)
