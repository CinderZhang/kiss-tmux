package main

import (
	"bytes"
	"testing"
)

func TestRingBufferEmpty(t *testing.T) {
	rb := NewRingBuffer(64)
	got := rb.Bytes()
	if len(got) != 0 {
		t.Errorf("empty buffer returned %d bytes", len(got))
	}
}

func TestRingBufferSmallWrite(t *testing.T) {
	rb := NewRingBuffer(64)
	rb.Write([]byte("hello"))
	got := rb.Bytes()
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestRingBufferMultipleWrites(t *testing.T) {
	rb := NewRingBuffer(64)
	rb.Write([]byte("hello "))
	rb.Write([]byte("world"))
	got := rb.Bytes()
	if string(got) != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestRingBufferExactFill(t *testing.T) {
	rb := NewRingBuffer(8)
	rb.Write([]byte("12345678"))
	got := rb.Bytes()
	if string(got) != "12345678" {
		t.Errorf("got %q, want %q", got, "12345678")
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(8)
	rb.Write([]byte("12345678"))
	rb.Write([]byte("ab"))
	got := rb.Bytes()
	// Should keep last 8 bytes: "345678ab"
	if string(got) != "345678ab" {
		t.Errorf("got %q, want %q", got, "345678ab")
	}
}

func TestRingBufferLargeOverflow(t *testing.T) {
	rb := NewRingBuffer(8)
	// Write 20 bytes — only last 8 should remain
	rb.Write([]byte("12345678901234567890"))
	got := rb.Bytes()
	if string(got) != "34567890" {
		t.Errorf("got %q, want %q", got, "34567890")
	}
}

func TestRingBufferMultipleOverflows(t *testing.T) {
	rb := NewRingBuffer(8)
	for i := 0; i < 100; i++ {
		rb.Write([]byte("x"))
	}
	got := rb.Bytes()
	if len(got) != 8 {
		t.Errorf("len = %d, want 8", len(got))
	}
	if string(got) != "xxxxxxxx" {
		t.Errorf("got %q, want %q", got, "xxxxxxxx")
	}
}

func TestRingBuffer64KB(t *testing.T) {
	rb := NewRingBuffer(65536)
	// Write 100KB of data
	chunk := bytes.Repeat([]byte("A"), 1024)
	for i := 0; i < 100; i++ {
		rb.Write(chunk)
	}
	got := rb.Bytes()
	if len(got) != 65536 {
		t.Errorf("len = %d, want 65536", len(got))
	}
}

func TestManagerSpawnAndList(t *testing.T) {
	var lastBroadcast []byte
	mgr := NewManager(8, func(data []byte) {
		lastBroadcast = data
	})

	id, err := mgr.Spawn("cmd.exe", "", "test-session", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("spawn returned empty ID")
	}

	sessions := mgr.List()
	if len(sessions) != 1 {
		t.Fatalf("list returned %d sessions, want 1", len(sessions))
	}
	if sessions[0].ID != id {
		t.Errorf("id = %q, want %q", sessions[0].ID, id)
	}
	if sessions[0].Name != "test-session" {
		t.Errorf("name = %q, want test-session", sessions[0].Name)
	}
	if !sessions[0].Running {
		t.Error("session should be running")
	}

	if lastBroadcast == nil {
		t.Error("no broadcast after spawn")
	}

	mgr.Kill(id)
}

func TestManagerKill(t *testing.T) {
	mgr := NewManager(8, func(data []byte) {})

	id, err := mgr.Spawn("cmd.exe", "", "kill-test", 80, 24)
	if err != nil {
		t.Fatal(err)
	}

	mgr.Kill(id)

	sessions := mgr.List()
	if len(sessions) != 1 {
		t.Fatalf("killed session should remain in list, got %d", len(sessions))
	}
	if sessions[0].Running {
		t.Error("killed session should not be running")
	}
}

func TestManagerRename(t *testing.T) {
	mgr := NewManager(8, func(data []byte) {})

	id, err := mgr.Spawn("cmd.exe", "", "old-name", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Kill(id)

	mgr.Rename(id, "new-name")

	sessions := mgr.List()
	if sessions[0].Name != "new-name" {
		t.Errorf("name = %q, want new-name", sessions[0].Name)
	}
}

func TestManagerMaxSessions(t *testing.T) {
	mgr := NewManager(2, func(data []byte) {})

	id1, err := mgr.Spawn("cmd.exe", "", "s1", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Kill(id1)

	id2, err := mgr.Spawn("cmd.exe", "", "s2", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Kill(id2)

	_, err = mgr.Spawn("cmd.exe", "", "s3", 80, 24)
	if err == nil {
		t.Error("expected error when exceeding max sessions")
	}
}
