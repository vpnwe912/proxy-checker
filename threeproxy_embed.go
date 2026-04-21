package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Put 3proxy.exe and required DLL files into embedded/3proxy before wails build.
//
//go:embed embedded/3proxy/*
var embeddedThreeProxy embed.FS

func EnsureEmbeddedThreeProxy(settings ThreeProxySettings) error {
	targetDir := EmbeddedThreeProxyTargetDir(settings)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	return fs.WalkDir(embeddedThreeProxy, "embedded/3proxy", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		if strings.EqualFold(name, ".keep") || strings.EqualFold(name, "README.txt") {
			return nil
		}
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); err == nil {
			return nil
		}
		data, err := embeddedThreeProxy.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0755)
	})
}

func EmbeddedThreeProxyTargetDir(settings ThreeProxySettings) string {
	return filepath.Join(settings.WorkDir, "embedded-3proxy")
}

func ResolveThreeProxyExe(settings ThreeProxySettings) string {
	if strings.TrimSpace(settings.ExePath) != "" {
		return settings.ExePath
	}
	candidate := filepath.Join(EmbeddedThreeProxyTargetDir(settings), "3proxy.exe")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}
