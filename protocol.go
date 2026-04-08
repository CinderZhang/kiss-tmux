package main

// ClientMsg is the envelope for all client→server WebSocket messages.
type ClientMsg struct {
	Type    string `json:"type"`
	Session string `json:"session,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
	Cwd     string `json:"cwd,omitempty"`
	Name    string `json:"name,omitempty"`
	Data    string `json:"data,omitempty"`
	Cols    uint16 `json:"cols,omitempty"`
	Rows    uint16 `json:"rows,omitempty"`
}

// ServerMsg is the envelope for all server→client WebSocket messages.
type ServerMsg struct {
	Type     string        `json:"type"`
	Session  string        `json:"session,omitempty"`
	Data     string        `json:"data,omitempty"`
	Code     int           `json:"code,omitempty"`
	Error    string        `json:"error,omitempty"`
	Sessions []SessionInfo `json:"sessions,omitempty"`
}

// SessionInfo describes a session in the sessions list broadcast.
type SessionInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Cmd     string `json:"cmd"`
	Running bool   `json:"running"`
}
