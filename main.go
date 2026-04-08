package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
)

func main() {
	port := flag.Int("port", 7777, "Port to listen on")
	open := flag.Bool("open", true, "Auto-open browser")
	flag.Parse()

	hub := NewHub()
	go hub.Run()

	mgr := NewManager(8, func(data []byte) {
		hub.broadcast <- data
	})

	handler := SetupHTTP(hub, mgr)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	url := fmt.Sprintf("http://%s", addr)

	log.Printf("KISS-TMUX serving at %s", url)

	if *open {
		go openBrowser(url)
	}

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}
