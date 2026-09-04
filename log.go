package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultOrigin = "https://app.3djobdesk.com"

var (
	statusMu sync.Mutex
	statusFn func(string)
)

func setStatusHandler(fn func(string)) {
	statusMu.Lock()
	statusFn = fn
	statusMu.Unlock()
}

func logPath() string {
	return filepath.Join(filepath.Dir(configPath()), "bridge.log")
}

func writeLog(msg string) {
	line := time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n"
	_ = os.MkdirAll(filepath.Dir(logPath()), 0o700)
	f, err := os.OpenFile(logPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func reportStatus(msg string) {
	writeLog(msg)
	statusMu.Lock()
	fn := statusFn
	statusMu.Unlock()
	if fn != nil {
		fn(msg)
		return
	}
	fmt.Println(msg)
}

func reportError(msg string) {
	writeLog(msg)
	statusMu.Lock()
	fn := statusFn
	statusMu.Unlock()
	if fn != nil {
		fn(msg)
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}
