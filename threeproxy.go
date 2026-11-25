package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ThreeProxySettings struct {
	ExePath     string `json:"exePath"`
	WorkDir     string `json:"workDir"`
	InternalIP  string `json:"internalIp"`
	ProxyPort   string `json:"proxyPort"`
	AdminPort   string `json:"adminPort"`
	UseDaemon   bool   `json:"useDaemon"`
	AuthIPOnly  bool   `json:"authIpOnly"`
	AllowedIP   string `json:"allowedIp"`
	ParentType  string `json:"parentType"`
	TimeoutLine string `json:"timeoutLine"`
}

type ThreeProxyManager struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	started time.Time
}

func DefaultThreeProxySettings() ThreeProxySettings {
	cwd, _ := os.Getwd()
	workDir := filepath.Join(cwd, "runtime")
	return ThreeProxySettings{
		WorkDir:     workDir,
		InternalIP:  "127.0.0.1",
		ProxyPort:   "8080",
		AdminPort:   "8081",
		UseDaemon:   false,
		AuthIPOnly:  true,
		AllowedIP:   "127.0.0.1",
		ParentType:  "connect",
		TimeoutLine: "1 5 30 60 1 1 1",
	}
}

func (s ThreeProxySettings) ConfigPath() string {
	return filepath.Join(s.WorkDir, "3proxy.cfg")
}

func (s ThreeProxySettings) ParentPath() string {
	return filepath.Join(s.WorkDir, "parent.cfg")
}

func (s ThreeProxySettings) ReloadPath() string {
	return filepath.Join(s.WorkDir, "reload.txt")
}

func (s ThreeProxySettings) LogPath() string {
	return filepath.Join(s.WorkDir, "logs", "3proxy.log")
}

func EnsureThreeProxyFiles(s ThreeProxySettings, parent *Proxy) error {
	if err := os.MkdirAll(filepath.Join(s.WorkDir, "logs"), 0755); err != nil {
		return err
	}
	if err := WriteMainConfigWithParent(s, parent); err != nil {
		return err
	}
	if _, err := os.Stat(s.ReloadPath()); os.IsNotExist(err) {
		if err := TouchReload(s); err != nil {
			return err
		}
	}
	return nil
}

func EnsureThreeProxyRuntimeFiles(s ThreeProxySettings) error {
	if err := os.MkdirAll(filepath.Join(s.WorkDir, "logs"), 0755); err != nil {
		return err
	}
	if err := WriteMainConfigWithParent(s, nil); err != nil {
		return err
	}
	if _, err := os.Stat(s.ReloadPath()); os.IsNotExist(err) {
		if err := TouchReload(s); err != nil {
			return err
		}
	}
	return nil
}

func WriteMainConfig(s ThreeProxySettings) error {
	return WriteMainConfigWithParent(s, nil)
}

func WriteMainConfigWithParent(s ThreeProxySettings, p *Proxy) error {
	// Конвертируем пути в короткий 8.3 формат для совместимости с кириллицей
	shortReload := ToShortPath(s.ReloadPath())
	shortLog := ToShortPath(s.LogPath())
	lines := []string{
		fmt.Sprintf("monitor %s", quote3proxyPath(shortReload)),
		"nscache 65536",
		"",
		fmt.Sprintf("log %s D", quote3proxyPath(shortLog)),
		`logformat "L%Y-%m-%d %H:%M:%S %N %E %C:%c %R:%r %T"`,
		"",
		fmt.Sprintf("internal %s", valueOrDefault(s.InternalIP, "127.0.0.1")),
	}
	if s.UseDaemon {
		lines = append([]string{"daemon"}, lines...)
	}
	if s.AuthIPOnly {
		lines = append(lines, "auth iponly")
	} else {
		lines = append(lines, "auth none")
	}

	parentLine := buildParentLine(s, p)

	lines = append(lines,
		"",
		fmt.Sprintf("timeouts %s", valueOrDefault(s.TimeoutLine, "1 5 30 60 1 1 1")),
		"",
		fmt.Sprintf("admin -p%s", valueOrDefault(s.AdminPort, "8081")),
		"",
		fmt.Sprintf("allow * %s", valueOrDefault(s.AllowedIP, "127.0.0.1")),
	)
	if parentLine != "" {
		lines = append(lines, parentLine)
		lines = append(lines,
			fmt.Sprintf("proxy -a -p%s", valueOrDefault(s.ProxyPort, "8080")),
			"",
			"deny *",
			"flush",
			"",
		)
	} else {
		// Нет parent — блокируем весь трафик, чтобы не пропускать через локальный IP
		lines = append(lines,
			"deny *",
			"flush",
			"",
		)
	}
	return os.WriteFile(s.ConfigPath(), []byte(strings.Join(lines, "\r\n")), 0644)
}

func buildParentLine(s ThreeProxySettings, p *Proxy) string {
	if p == nil {
		return ""
	}
	parentType := p.Type
	if parentType == "" || parentType == "auto" {
		parentType = valueOrDefault(s.ParentType, "connect")
	}
	line := fmt.Sprintf("parent 1000 %s %s %s", parentType, p.Host, p.Port)
	if p.Login != "" || p.Password != "" {
		line += fmt.Sprintf(" %s %s", p.Login, p.Password)
	}
	return line
}

func quote3proxyPath(path string) string {
	return `"` + strings.ReplaceAll(path, `"`, "") + `"`
}

func WriteParentConfig(s ThreeProxySettings, p *Proxy) error {
	path := s.ParentPath()
	if p == nil {
		return os.WriteFile(path, []byte(""), 0644)
	}
	parentType := p.Type
	if parentType == "" || parentType == "auto" {
		parentType = valueOrDefault(s.ParentType, "connect")
	}
	line := fmt.Sprintf("parent 1000 %s %s %s", parentType, p.Host, p.Port)
	if p.Login != "" || p.Password != "" {
		line += fmt.Sprintf(" %s %s", p.Login, p.Password)
	}
	return os.WriteFile(path, []byte(line+"\r\n"), 0644)
}

func TouchReload(s ThreeProxySettings) error {
	text := fmt.Sprintf("Reload logic: %d %s\r\n", time.Now().UnixNano(), time.Now().Format("2006-01-02 15:04:05"))
	return os.WriteFile(s.ReloadPath(), []byte(text), 0644)
}

func (m *ThreeProxyManager) Start(s ThreeProxySettings, onExit func(error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		return nil
	}
	if strings.TrimSpace(s.ExePath) == "" {
		return fmt.Errorf("path to 3proxy executable is empty")
	}
	// Ждём освобождения порта (TIME_WAIT после предыдущей остановки)
	if !waitPortFree(s.ProxyPort, 10*time.Second) {
		return fmt.Errorf("port %s is still in use (TIME_WAIT), try again in a moment", s.ProxyPort)
	}
	if err := EnsureThreeProxyRuntimeFiles(s); err != nil {
		return err
	}
	exePath := ToShortPath(s.ExePath)
	cfgPath := ToShortPath(s.ConfigPath())
	workDir := ToShortPath(s.WorkDir)
	cmd := exec.Command(exePath, cfgPath)
	hideChildProcessWindow(cmd)
	cmd.Dir = workDir
	var stderrBuf bytes.Buffer
	startLog, err := os.OpenFile(filepath.Join(s.WorkDir, "logs", "3proxy-start.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		_, _ = startLog.WriteString(time.Now().Format("2006-01-02 15:04:05") + " starting " + s.ExePath + " " + s.ConfigPath() + "\r\n")
		combined := io.MultiWriter(startLog, &stderrBuf)
		cmd.Stdout = combined
		cmd.Stderr = combined
		defer startLog.Close()
	} else {
		cmd.Stdout = &stderrBuf
		cmd.Stderr = &stderrBuf
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	m.started = time.Now()
	exitCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		exitCh <- err
		m.mu.Lock()
		if m.cmd == cmd {
			m.cmd = nil
		}
		m.mu.Unlock()
		if onExit != nil {
			onExit(err)
		}
	}()
	select {
	case err := <-exitCh:
		output := strings.TrimSpace(stderrBuf.String())
		if output != "" {
			if err != nil {
				return fmt.Errorf("3proxy exited immediately: %w\nOutput: %s", err, output)
			}
			return fmt.Errorf("3proxy exited immediately\nOutput: %s", output)
		}
		if err != nil {
			return fmt.Errorf("3proxy exited immediately: %w", err)
		}
		return fmt.Errorf("3proxy exited immediately")
	case <-time.After(800 * time.Millisecond):
	}
	return nil
}

func (m *ThreeProxyManager) Stop() error {
	m.mu.Lock()
	if m.cmd == nil || m.cmd.Process == nil {
		m.cmd = nil
		m.mu.Unlock()
		return nil
	}
	err := m.cmd.Process.Kill()
	m.cmd = nil
	m.mu.Unlock()
	return err
}

// waitPortFree ждёт до timeout пока TCP-порт на localhost не перестанет быть занятым.
// Нужно для обхода TIME_WAIT после остановки 3proxy на Windows.
func waitPortFree(port string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort("127.0.0.1", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			// порт не отвечает — свободен
			return true
		}
		conn.Close()
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

func (m *ThreeProxyManager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cmd != nil && m.cmd.Process != nil
}

func valueOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
