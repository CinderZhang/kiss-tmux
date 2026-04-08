package main

import (
	"encoding/json"
	"testing"
)

func TestClientMsgSpawn(t *testing.T) {
	raw := `{"type":"spawn","cmd":"claude","cwd":"C:\\Users","name":"main"}`
	var msg ClientMsg
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != "spawn" {
		t.Errorf("type = %q, want spawn", msg.Type)
	}
	if msg.Cmd != "claude" {
		t.Errorf("cmd = %q, want claude", msg.Cmd)
	}
	if msg.Cwd != `C:\Users` {
		t.Errorf("cwd = %q, want C:\\Users", msg.Cwd)
	}
	if msg.Name != "main" {
		t.Errorf("name = %q, want main", msg.Name)
	}
}

func TestClientMsgInput(t *testing.T) {
	raw := `{"type":"input","session":"abc123","data":"ls\r\n"}`
	var msg ClientMsg
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != "input" || msg.Session != "abc123" || msg.Data != "ls\r\n" {
		t.Errorf("unexpected: %+v", msg)
	}
}

func TestClientMsgResize(t *testing.T) {
	raw := `{"type":"resize","session":"abc123","cols":120,"rows":40}`
	var msg ClientMsg
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Cols != 120 || msg.Rows != 40 {
		t.Errorf("cols=%d rows=%d, want 120,40", msg.Cols, msg.Rows)
	}
}

func TestServerMsgOutput(t *testing.T) {
	msg := ServerMsg{Type: "output", Session: "abc123", Data: "hello\x1b[32mworld\x1b[0m"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ServerMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Type != "output" || decoded.Session != "abc123" {
		t.Errorf("unexpected: %+v", decoded)
	}
	if decoded.Data != msg.Data {
		t.Errorf("data mismatch: got %q, want %q", decoded.Data, msg.Data)
	}
}

func TestServerMsgSessions(t *testing.T) {
	msg := ServerMsg{
		Type: "sessions",
		Sessions: []SessionInfo{
			{ID: "a1", Name: "main", Cmd: "claude", Running: true},
			{ID: "b2", Name: "test", Cmd: "cmd.exe", Running: false},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ServerMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Sessions) != 2 {
		t.Fatalf("sessions count = %d, want 2", len(decoded.Sessions))
	}
	if decoded.Sessions[0].Running != true || decoded.Sessions[1].Running != false {
		t.Errorf("running flags wrong: %+v", decoded.Sessions)
	}
}

func TestServerMsgOmitsEmpty(t *testing.T) {
	msg := ServerMsg{Type: "exited", Session: "abc123", Code: 0}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, ok := raw["data"]; ok {
		t.Error("data field should be omitted when empty")
	}
	if _, ok := raw["error"]; ok {
		t.Error("error field should be omitted when empty")
	}
}
