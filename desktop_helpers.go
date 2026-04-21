package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultNetworkTimeout = 30 * time.Second

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func (a *AppState) ExportGoodProxies() (string, int, error) {
	cfg := a.snapshotConfig()
	if err := os.MkdirAll(cfg.ThreeProxy.WorkDir, 0755); err != nil {
		return "", 0, err
	}

	a.mu.Lock()
	var lines []string
	for _, p := range a.proxies {
		if result, ok := a.results[p.Key()]; ok && result.OK {
			lines = append(lines, p.Display())
		}
	}
	a.mu.Unlock()

	path := filepath.Join(cfg.ThreeProxy.WorkDir, "good-proxies.txt")
	text := strings.Join(lines, "\r\n")
	if text != "" {
		text += "\r\n"
	}
	return path, len(lines), os.WriteFile(path, []byte(text), 0644)
}
