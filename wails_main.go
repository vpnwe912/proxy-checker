package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewDesktopApp()

	err := wails.Run(&options.App{
		Title:  "Proxy Checker",
		Width:  1280,
		Height: 820,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

type DesktopApp struct {
	ctx   context.Context
	state *AppState
}

type StateSnapshot struct {
	Config         AppConfig     `json:"config"`
	Proxies        []Proxy       `json:"proxies"`
	Results        []CheckResult `json:"results"`
	Logs           []string      `json:"logs"`
	ActiveProxy    *Proxy        `json:"activeProxy"`
	MonitorRunning bool          `json:"monitorRunning"`
	ImportRunning  bool          `json:"importRunning"`
	ThreeProxyRun  bool          `json:"threeProxyRun"`
}

func NewDesktopApp() *DesktopApp {
	return &DesktopApp{state: NewAppState()}
}

func (a *DesktopApp) Startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.state.LoadConfig(); err != nil {
		a.state.addLog("Settings were not loaded: " + err.Error())
	}
	if err := EnsureEmbeddedThreeProxy(a.state.snapshotConfig().ThreeProxy); err != nil {
		a.state.addLog("Embedded 3proxy was not prepared: " + err.Error())
	}
	if a.state.snapshotConfig().AutoImport {
		a.state.startAutoImport()
		go a.state.importFromConfiguredAPI(context.Background())
	}
}

func (a *DesktopApp) Shutdown(ctx context.Context) {
	a.state.stopMonitor()
	a.state.stopAutoImport()
	_ = a.state.threeProxy.Stop()
}

func (a *DesktopApp) GetState() StateSnapshot {
	a.state.mu.Lock()
	proxies := append([]Proxy(nil), a.state.proxies...)
	results := make([]CheckResult, 0, len(a.state.results))
	for _, result := range a.state.results {
		results = append(results, result)
	}
	logs := append([]string(nil), a.state.logs...)
	cfg := a.state.config
	monitorRunning := a.state.monitorRunning
	importRunning := a.state.importRunning
	var active *Proxy
	if a.state.activeProxy != nil {
		p := *a.state.activeProxy
		active = &p
	}
	a.state.mu.Unlock()

	return StateSnapshot{
		Config:         cfg,
		Proxies:        proxies,
		Results:        results,
		Logs:           logs,
		ActiveProxy:    active,
		MonitorRunning: monitorRunning,
		ImportRunning:  importRunning,
		ThreeProxyRun:  a.state.threeProxy.Running(),
	}
}

func (a *DesktopApp) SaveSettings(cfg AppConfig) apiResponse {
	normalizeConfig(&cfg)
	a.state.mu.Lock()
	a.state.config = cfg
	a.state.mu.Unlock()
	if err := EnsureEmbeddedThreeProxy(cfg.ThreeProxy); err != nil {
		a.state.addLog("Embedded 3proxy was not prepared: " + err.Error())
	}
	if err := a.state.SaveConfig(); err != nil {
		return apiResponse{OK: false, Message: err.Error()}
	}
	if cfg.AutoImport {
		a.state.startAutoImport()
	} else {
		a.state.stopAutoImport()
	}
	a.state.addLog("Settings saved")
	return apiResponse{OK: true, Message: "settings saved"}
}

func (a *DesktopApp) LoadURL(apiURL string) apiResponse {
	ctx, cancel := context.WithTimeout(a.context(), defaultNetworkTimeout)
	defer cancel()
	proxies, err := FetchProxies(ctx, apiURL)
	if err != nil {
		a.state.addLog("API load failed: " + err.Error())
		return apiResponse{OK: false, Message: err.Error()}
	}
	a.state.mu.Lock()
	a.state.config.ProxyAPIURL = apiURL
	a.state.mu.Unlock()
	_ = a.state.SaveConfig()
	a.state.setProxies(proxies)
	a.state.addLog("Loaded proxies from API: " + itoa(len(proxies)))
	return apiResponse{OK: true, Message: "loaded " + itoa(len(proxies)) + " proxies"}
}

func (a *DesktopApp) LoadFilePath(path string) apiResponse {
	data, err := osReadFile(path)
	if err != nil {
		a.state.addLog("File load failed: " + err.Error())
		return apiResponse{OK: false, Message: err.Error()}
	}
	proxies, err := ParseProxyList(data, path)
	if err != nil {
		a.state.addLog("File parse failed: " + err.Error())
		return apiResponse{OK: false, Message: err.Error()}
	}
	a.state.setProxies(proxies)
	a.state.addLog("Loaded proxies from file: " + itoa(len(proxies)))
	return apiResponse{OK: true, Message: "loaded " + itoa(len(proxies)) + " proxies"}
}

func (a *DesktopApp) SelectProxyFile() string {
	path, err := wailsruntime.OpenFileDialog(a.context(), wailsruntime.OpenDialogOptions{
		Title: "Select proxy file",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Proxy files (*.txt, *.json)", Pattern: "*.txt;*.json"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		a.state.addLog("File dialog failed: " + err.Error())
		return ""
	}
	return path
}

func (a *DesktopApp) CheckAll() apiResponse {
	results := a.state.checkAll(a.context())
	good := 0
	for _, result := range results {
		if result.OK {
			good++
		}
	}
	a.state.addLog("Check finished: " + itoa(good) + " good from " + itoa(len(results)))
	return apiResponse{OK: true, Message: "good proxies: " + itoa(good) + " / " + itoa(len(results))}
}

func (a *DesktopApp) StartMonitor() apiResponse {
	a.state.mu.Lock()
	if a.state.monitorRunning {
		a.state.mu.Unlock()
		return apiResponse{OK: true, Message: "monitor already running"}
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.state.monitorCancel = cancel
	a.state.monitorRunning = true
	a.state.mu.Unlock()
	go a.state.monitorLoop(ctx)
	a.state.addLog("Monitor started")
	return apiResponse{OK: true, Message: "monitor started"}
}

func (a *DesktopApp) StopMonitor() apiResponse {
	a.state.stopMonitor()
	a.state.addLog("Monitor stopped")
	return apiResponse{OK: true, Message: "monitor stopped"}
}

func (a *DesktopApp) StartThreeProxy() apiResponse {
	cfg := a.state.snapshotConfig()
	if err := EnsureEmbeddedThreeProxy(cfg.ThreeProxy); err != nil {
		a.state.addLog("Embedded 3proxy failed: " + err.Error())
		return apiResponse{OK: false, Message: err.Error()}
	}
	cfg.ThreeProxy.ExePath = ResolveThreeProxyExe(cfg.ThreeProxy)
	if err := a.state.threeProxy.Start(cfg.ThreeProxy, func(err error) {
		if err != nil {
			a.state.addLog("3proxy exited: " + err.Error())
			return
		}
		a.state.addLog("3proxy exited")
	}); err != nil {
		a.state.addLog("3proxy start failed: " + err.Error())
		return apiResponse{OK: false, Message: err.Error()}
	}
	a.state.addLog("3proxy started")
	return apiResponse{OK: true, Message: "3proxy started"}
}

func (a *DesktopApp) StopThreeProxy() apiResponse {
	if err := a.state.threeProxy.Stop(); err != nil {
		return apiResponse{OK: false, Message: err.Error()}
	}
	a.state.addLog("3proxy stopped")
	return apiResponse{OK: true, Message: "3proxy stopped"}
}

func (a *DesktopApp) ExportGoodProxies() apiResponse {
	path, count, err := a.state.ExportGoodProxies()
	if err != nil {
		return apiResponse{OK: false, Message: err.Error()}
	}
	return apiResponse{OK: true, Message: "saved " + itoa(count) + " proxies to " + path}
}

func (a *DesktopApp) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *DesktopApp) GetVersion() map[string]interface{} {
	return GetFullVersionInfo()
}
