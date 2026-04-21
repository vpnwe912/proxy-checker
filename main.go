package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type AppConfig struct {
	ListenAddr      string             `json:"listenAddr"`
	ProxyAPIURL     string             `json:"proxyApiUrl"`
	ProxyTypeMode   string             `json:"proxyTypeMode"`
	GeoLookup       bool               `json:"geoLookup"`
	GeoProvider     string             `json:"geoProvider"`
	GeoLookupURL    string             `json:"geoLookupUrl"`
	AutoImport      bool               `json:"autoImport"`
	AutoImportSec   int                `json:"autoImportSec"`
	AutoImportUnit  string             `json:"autoImportUnit"`
	TestURL         string             `json:"testUrl"`
	CheckTimeoutSec int                `json:"checkTimeoutSec"`
	MonitorEverySec int                `json:"monitorEverySec"`
	Workers         int                `json:"workers"`
	AllowInsecure   bool               `json:"allowInsecure"`
	UseCurl         bool               `json:"useCurl"`
	ThreeProxy      ThreeProxySettings `json:"threeProxy"`
}

type AppState struct {
	mu             sync.Mutex
	config         AppConfig
	proxies        []Proxy
	results        map[string]CheckResult
	activeProxy    *Proxy
	logs           []string
	monitorCancel  context.CancelFunc
	monitorRunning bool
	importCancel   context.CancelFunc
	importRunning  bool
	nextIndex      int
	threeProxy     ThreeProxyManager
}

type apiResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func runHTTPServer() {
	app := NewAppState()
	if err := app.LoadConfig(); err != nil {
		app.addLog("Settings were not loaded: " + err.Error())
	}
	mux := http.NewServeMux()
	app.routes(mux)
	addr := app.snapshotConfig().ListenAddr
	log.Printf("Proxy Checker started: http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func NewAppState() *AppState {
	return &AppState{
		config: AppConfig{
			ListenAddr:      "127.0.0.1:18080",
			ProxyTypeMode:   "auto",
			GeoLookup:       false,
			GeoProvider:     "auto",
			GeoLookupURL:    "https://ipinfo.io/json",
			AutoImportSec:   3600,
			AutoImportUnit:  "hour",
			TestURL:         "https://api.ipify.org",
			CheckTimeoutSec: 2,
			MonitorEverySec: 120,
			Workers:         20,
			AllowInsecure:   true,
			UseCurl:         true,
			ThreeProxy:      DefaultThreeProxySettings(),
		},
		results: map[string]CheckResult{},
	}
}

func (a *AppState) routes(mux *http.ServeMux) {
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/api/state", a.handleState)
	mux.HandleFunc("/api/settings", a.handleSettings)
	mux.HandleFunc("/api/load-url", a.handleLoadURL)
	mux.HandleFunc("/api/load-file", a.handleLoadFile)
	mux.HandleFunc("/api/check", a.handleCheck)
	mux.HandleFunc("/api/monitor/start", a.handleMonitorStart)
	mux.HandleFunc("/api/monitor/stop", a.handleMonitorStop)
	mux.HandleFunc("/api/3proxy/start", a.handleThreeProxyStart)
	mux.HandleFunc("/api/3proxy/stop", a.handleThreeProxyStop)
	mux.HandleFunc("/api/export-good", a.handleExportGood)
}

func (a *AppState) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTemplate.Execute(w, nil)
}

func (a *AppState) handleState(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	proxies := append([]Proxy(nil), a.proxies...)
	results := make([]CheckResult, 0, len(a.results))
	for _, result := range a.results {
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CheckedAt > results[j].CheckedAt
	})
	logs := append([]string(nil), a.logs...)
	cfg := a.config
	monitorRunning := a.monitorRunning
	importRunning := a.importRunning
	var active *Proxy
	if a.activeProxy != nil {
		p := *a.activeProxy
		active = &p
	}
	a.mu.Unlock()

	writeJSON(w, map[string]any{
		"config":         cfg,
		"proxies":        proxies,
		"results":        results,
		"logs":           logs,
		"activeProxy":    active,
		"monitorRunning": monitorRunning,
		"importRunning":  importRunning,
		"threeProxyRun":  a.threeProxy.Running(),
	})
}

func (a *AppState) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cfg AppConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	normalizeConfig(&cfg)
	a.mu.Lock()
	a.config = cfg
	a.mu.Unlock()
	if err := a.SaveConfig(); err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	if cfg.AutoImport {
		a.startAutoImport()
	} else {
		a.stopAutoImport()
	}
	a.addLog("Settings saved")
	writeJSON(w, apiResponse{OK: true, Message: "settings saved"})
}

func (a *AppState) handleLoadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	proxies, err := FetchProxies(ctx, strings.TrimSpace(payload.URL))
	if err != nil {
		a.addLog("API load failed: " + err.Error())
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	a.mu.Lock()
	a.config.ProxyAPIURL = strings.TrimSpace(payload.URL)
	a.mu.Unlock()
	_ = a.SaveConfig()
	a.setProxies(proxies)
	a.addLog(fmt.Sprintf("Loaded %d proxies from API", len(proxies)))
	writeJSON(w, apiResponse{OK: true, Message: fmt.Sprintf("loaded %d proxies", len(proxies))})
}

func (a *AppState) handleLoadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	proxies, err := ParseProxyList(data, header.Filename)
	if err != nil {
		a.addLog("File load failed: " + err.Error())
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	a.setProxies(proxies)
	a.addLog(fmt.Sprintf("Loaded %d proxies from file %s", len(proxies), header.Filename))
	writeJSON(w, apiResponse{OK: true, Message: fmt.Sprintf("loaded %d proxies", len(proxies))})
}

func (a *AppState) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	results := a.checkAll(r.Context())
	good := 0
	for _, result := range results {
		if result.OK {
			good++
		}
	}
	a.addLog(fmt.Sprintf("Check finished: %d good from %d", good, len(results)))
	writeJSON(w, apiResponse{OK: true, Message: fmt.Sprintf("good proxies: %d / %d", good, len(results))})
}

func (a *AppState) handleMonitorStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	if a.monitorRunning {
		a.mu.Unlock()
		writeJSON(w, apiResponse{OK: true, Message: "monitor already running"})
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.monitorCancel = cancel
	a.monitorRunning = true
	a.mu.Unlock()
	go a.monitorLoop(ctx)
	a.addLog("Monitor started")
	writeJSON(w, apiResponse{OK: true, Message: "monitor started"})
}

func (a *AppState) handleMonitorStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.stopMonitor()
	a.addLog("Monitor stopped")
	writeJSON(w, apiResponse{OK: true, Message: "monitor stopped"})
}

func (a *AppState) handleThreeProxyStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := a.snapshotConfig()
	cfg.ThreeProxy.ExePath = ResolveThreeProxyExe(cfg.ThreeProxy)
	if err := a.threeProxy.Start(cfg.ThreeProxy, func(err error) {
		if err != nil {
			a.addLog("3proxy exited: " + err.Error())
			return
		}
		a.addLog("3proxy exited")
	}); err != nil {
		a.addLog("3proxy start failed: " + err.Error())
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	a.addLog("3proxy started")
	writeJSON(w, apiResponse{OK: true, Message: "3proxy started"})
}

func (a *AppState) handleThreeProxyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := a.threeProxy.Stop(); err != nil {
		writeJSON(w, apiResponse{OK: false, Message: err.Error()})
		return
	}
	a.addLog("3proxy stopped")
	writeJSON(w, apiResponse{OK: true, Message: "3proxy stopped"})
}

func (a *AppState) handleExportGood(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	var lines []string
	for _, p := range a.proxies {
		if result, ok := a.results[p.Key()]; ok && result.OK {
			lines = append(lines, p.Display())
		}
	}
	a.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="good-proxies.txt"`)
	_, _ = w.Write([]byte(strings.Join(lines, "\r\n")))
}

func (a *AppState) monitorLoop(ctx context.Context) {
	defer func() {
		a.mu.Lock()
		a.monitorRunning = false
		a.monitorCancel = nil
		a.mu.Unlock()
	}()
	for {
		a.pickAndApplyParent(ctx)
		interval := time.Duration(a.snapshotConfig().MonitorEverySec) * time.Second
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (a *AppState) pickAndApplyParent(ctx context.Context) {
	cfg := a.snapshotConfig()
	a.mu.Lock()
	proxies := append([]Proxy(nil), a.proxies...)
	start := a.nextIndex
	a.mu.Unlock()
	if len(proxies) == 0 {
		_ = EnsureThreeProxyFiles(cfg.ThreeProxy, nil)
		_ = TouchReload(cfg.ThreeProxy)
		a.addLog("Monitor: no proxies loaded")
		return
	}
	type monitorJob struct {
		Order int
		Index int
		Proxy Proxy
	}
	type monitorResult struct {
		Order  int
		Index  int
		Result CheckResult
	}

	ordered := make([]monitorJob, 0, len(proxies))
	for i := 0; i < len(proxies); i++ {
		idx := (start + i) % len(proxies)
		ordered = append(ordered, monitorJob{Order: i, Index: idx, Proxy: proxies[idx]})
	}

	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}
	if workers > len(ordered) {
		workers = len(ordered)
	}
	jobs := make(chan monitorJob)
	results := make(chan monitorResult, len(ordered))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result := CheckProxy(ctx, job.Proxy, cfg.TestURL, time.Duration(cfg.CheckTimeoutSec)*time.Second, cfg.AllowInsecure, cfg.UseCurl, cfg.GeoLookup, cfg.GeoProvider, cfg.GeoLookupURL)
				results <- monitorResult{Order: job.Order, Index: job.Index, Result: result}
			}
		}()
	}
	go func() {
		for _, job := range ordered {
			if ctx.Err() != nil {
				break
			}
			jobs <- job
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var best *monitorResult
	for item := range results {
		result := item.Result
		a.saveResult(result)
		if result.OK {
			a.addLog(fmt.Sprintf("Monitor: ok %s HTTP %d", result.Proxy.Address(), result.StatusCode))
			if best == nil || item.Order < best.Order {
				copyItem := item
				best = &copyItem
			}
			continue
		}
		if ctx.Err() == nil {
			a.addLog(fmt.Sprintf("Monitor: dead %s %s", result.Proxy.Address(), result.Error))
		}
	}
	if ctx.Err() != nil {
		return
	}
	if best != nil {
		p := best.Result.Proxy
		_ = EnsureThreeProxyFiles(cfg.ThreeProxy, &p)
		_ = TouchReload(cfg.ThreeProxy)
		a.mu.Lock()
		a.activeProxy = &p
		a.nextIndex = (best.Index + 1) % len(proxies)
		a.mu.Unlock()
		a.addLog(fmt.Sprintf("Monitor: using %s HTTP %d", p.Address(), best.Result.StatusCode))
		return
	}

	_ = EnsureThreeProxyFiles(cfg.ThreeProxy, nil)
	_ = TouchReload(cfg.ThreeProxy)
	a.mu.Lock()
	a.activeProxy = nil
	a.mu.Unlock()
	a.addLog("Monitor: no alive proxies, fail-closed parent.cfg")
}

func (a *AppState) checkAll(ctx context.Context) []CheckResult {
	cfg := a.snapshotConfig()
	a.mu.Lock()
	proxies := append([]Proxy(nil), a.proxies...)
	a.mu.Unlock()
	if len(proxies) == 0 {
		return nil
	}
	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}
	if workers > len(proxies) {
		workers = len(proxies)
	}
	jobs := make(chan Proxy)
	results := make(chan CheckResult, len(proxies))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				results <- CheckProxy(ctx, p, cfg.TestURL, time.Duration(cfg.CheckTimeoutSec)*time.Second, cfg.AllowInsecure, cfg.UseCurl, cfg.GeoLookup, cfg.GeoProvider, cfg.GeoLookupURL)
			}
		}()
	}
	go func() {
		for _, p := range proxies {
			jobs <- p
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	var all []CheckResult
	for result := range results {
		a.saveResult(result)
		all = append(all, result)
	}
	return all
}

func (a *AppState) setProxies(proxies []Proxy) {
	a.mu.Lock()
	defer a.mu.Unlock()
	proxies = applyProxyTypeMode(proxies, a.config.ProxyTypeMode)
	a.proxies = proxies
	a.results = map[string]CheckResult{}
	a.activeProxy = nil
	a.nextIndex = 0
}

func (a *AppState) saveResult(result CheckResult) {
	a.mu.Lock()
	a.results[result.Proxy.Key()] = result
	a.mu.Unlock()
}

func (a *AppState) addLog(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	line := time.Now().Format("15:04:05") + " " + msg
	a.logs = append(a.logs, line)
	if len(a.logs) > 300 {
		a.logs = a.logs[len(a.logs)-300:]
	}
}

func (a *AppState) stopMonitor() {
	a.mu.Lock()
	cancel := a.monitorCancel
	a.monitorCancel = nil
	a.monitorRunning = false
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *AppState) startAutoImport() {
	a.mu.Lock()
	if a.importRunning {
		a.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.importCancel = cancel
	a.importRunning = true
	a.mu.Unlock()
	go a.autoImportLoop(ctx)
}

func (a *AppState) stopAutoImport() {
	a.mu.Lock()
	cancel := a.importCancel
	a.importCancel = nil
	a.importRunning = false
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *AppState) autoImportLoop(ctx context.Context) {
	defer func() {
		a.mu.Lock()
		a.importRunning = false
		a.importCancel = nil
		a.mu.Unlock()
	}()
	for {
		cfg := a.snapshotConfig()
		interval := time.Duration(cfg.AutoImportSec) * time.Second
		if interval < time.Second {
			interval = time.Second
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			a.importFromConfiguredAPI(ctx)
		}
	}
}

func (a *AppState) importFromConfiguredAPI(ctx context.Context) {
	cfg := a.snapshotConfig()
	apiURL := strings.TrimSpace(cfg.ProxyAPIURL)
	if apiURL == "" {
		a.addLog("Auto import skipped: API URL is empty")
		return
	}
	importCtx, cancel := context.WithTimeout(ctx, defaultNetworkTimeout)
	defer cancel()
	proxies, err := FetchProxies(importCtx, apiURL)
	if err != nil {
		a.addLog("Auto import failed: " + err.Error())
		return
	}
	a.setProxies(proxies)
	a.addLog(fmt.Sprintf("Auto import loaded proxies from API: %d", len(proxies)))
}

func (a *AppState) snapshotConfig() AppConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config
}

func (a *AppState) configPath() string {
	return filepath.Join(a.config.ThreeProxy.WorkDir, "settings.json")
}

func (a *AppState) LoadConfig() error {
	path := a.configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	normalizeConfig(&cfg)
	a.mu.Lock()
	a.config = cfg
	a.mu.Unlock()
	return nil
}

func (a *AppState) SaveConfig() error {
	cfg := a.snapshotConfig()
	if err := os.MkdirAll(cfg.ThreeProxy.WorkDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.ThreeProxy.WorkDir, "settings.json"), data, 0644)
}

func normalizeConfig(cfg *AppConfig) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:18080"
	}
	if cfg.TestURL == "" {
		cfg.TestURL = "https://api.ipify.org"
	}
	if strings.TrimSpace(cfg.GeoLookupURL) == "" {
		cfg.GeoLookupURL = "https://ipinfo.io/json"
	}
	if cfg.GeoProvider == "" {
		cfg.GeoProvider = inferGeoProvider(cfg.GeoLookupURL)
	}
	if !isSupportedGeoProvider(cfg.GeoProvider) {
		cfg.GeoProvider = "auto"
	}
	if cfg.ProxyTypeMode == "" {
		cfg.ProxyTypeMode = "auto"
	}
	if _, ok := proxyTypeFromScheme(cfg.ProxyTypeMode); !ok {
		cfg.ProxyTypeMode = "auto"
	}
	if cfg.AutoImportSec < 1 {
		cfg.AutoImportSec = 3600
	}
	if cfg.AutoImportUnit == "" {
		cfg.AutoImportUnit = "hour"
	}
	if cfg.CheckTimeoutSec < 1 {
		cfg.CheckTimeoutSec = 2
	}
	if cfg.MonitorEverySec < 5 {
		cfg.MonitorEverySec = 120
	}
	if cfg.Workers < 1 {
		cfg.Workers = 20
	}
	cfg.UseCurl = true
	def := DefaultThreeProxySettings()
	if cfg.ThreeProxy.WorkDir == "" {
		cfg.ThreeProxy.WorkDir = def.WorkDir
	}
	if cfg.ThreeProxy.InternalIP == "" {
		cfg.ThreeProxy.InternalIP = def.InternalIP
	}
	if cfg.ThreeProxy.ProxyPort == "" {
		cfg.ThreeProxy.ProxyPort = def.ProxyPort
	}
	if cfg.ThreeProxy.AdminPort == "" {
		cfg.ThreeProxy.AdminPort = def.AdminPort
	}
	if cfg.ThreeProxy.AllowedIP == "" {
		cfg.ThreeProxy.AllowedIP = def.AllowedIP
	}
	if cfg.ThreeProxy.ParentType == "" {
		cfg.ThreeProxy.ParentType = def.ParentType
	}
	if cfg.ThreeProxy.TimeoutLine == "" {
		cfg.ThreeProxy.TimeoutLine = def.TimeoutLine
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(value)
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Proxy Checker</title>
<style>
:root { color-scheme: light; --bg:#f4f7f5; --panel:#ffffff; --ink:#18201c; --muted:#5f6b63; --line:#ccd8d0; --green:#167d4a; --red:#b42318; --blue:#1664a5; --yellow:#966c00; }
* { box-sizing: border-box; }
body { margin:0; font-family: Segoe UI, Arial, sans-serif; background:var(--bg); color:var(--ink); }
header { padding:22px 28px; background:#20352a; color:#fff; }
header h1 { margin:0 0 4px; font-size:24px; font-weight:700; }
header p { margin:0; color:#d9e6de; }
main { padding:20px; max-width:1380px; margin:0 auto; }
.grid { display:grid; grid-template-columns: 360px 1fr; gap:16px; align-items:start; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:8px; padding:16px; }
.panel h2 { margin:0 0 12px; font-size:18px; }
label { display:block; margin:10px 0 4px; font-size:13px; color:var(--muted); }
input, button { font:inherit; }
input[type=text], input[type=number], input[type=file], select { width:100%; padding:9px 10px; border:1px solid var(--line); border-radius:6px; background:#fff; }
.row { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
button { border:0; border-radius:6px; padding:9px 12px; cursor:pointer; background:#244d3c; color:#fff; }
button.secondary { background:#1664a5; }
button.warn { background:#966c00; }
button.danger { background:#9f2a20; }
button:disabled { opacity:.55; cursor:not-allowed; }
.stats { display:grid; grid-template-columns: repeat(4, minmax(120px, 1fr)); gap:10px; margin-bottom:16px; }
.stat { background:#fff; border:1px solid var(--line); border-radius:8px; padding:12px; }
.stat b { display:block; font-size:24px; margin-bottom:2px; }
.muted { color:var(--muted); }
table { width:100%; border-collapse:collapse; background:#fff; border:1px solid var(--line); border-radius:8px; overflow:hidden; }
th, td { padding:9px; border-bottom:1px solid var(--line); text-align:left; font-size:13px; vertical-align:top; }
th { background:#edf3ef; color:#38463e; }
tr:last-child td { border-bottom:0; }
.ok { color:var(--green); font-weight:700; }
.fail { color:var(--red); font-weight:700; }
.warntext { color:var(--yellow); font-weight:700; }
pre { margin:0; white-space:pre-wrap; background:#121814; color:#e4f0e8; padding:12px; border-radius:8px; max-height:260px; overflow:auto; }
.split { display:grid; grid-template-columns:1fr 1fr; gap:16px; margin-top:16px; }
.small { font-size:12px; }
@media (max-width: 900px) { .grid, .split { grid-template-columns:1fr; } .stats { grid-template-columns:1fr 1fr; } }
</style>
</head>
<body>
<header>
  <h1>Proxy Checker</h1>
  <p>Загрузка прокси, проверка доступа к cabinet.tax.gov.ua и управление parent.cfg для 3proxy.</p>
</header>
<main>
  <div class="grid">
    <section class="panel">
      <h2>Загрузка прокси</h2>
      <label>API URL</label>
      <input id="apiUrl" type="text" placeholder="https://proxy-example.com/api/getproxy/?format=json...">
      <label>Тип прокси для строк без протокола</label>
      <select id="proxyTypeMode">
        <option value="auto">AUTO: проверить HTTP и SOCKS5</option>
        <option value="connect">HTTP</option>
        <option value="socks5">SOCKS5</option>
      </select>
      <div class="row" style="margin-top:8px"><button onclick="loadUrl()">Загрузить API</button></div>
      <label>Файл TXT или JSON</label>
      <input id="proxyFile" type="file">
      <div class="row" style="margin-top:8px"><button onclick="loadFile()">Загрузить файл</button></div>

      <h2 style="margin-top:22px">Проверка</h2>
      <label>URL проверки</label>
      <input id="testUrl" type="text">
      <label>Timeout, сек</label>
      <input id="checkTimeoutSec" type="number" min="1">
      <label>Потоков проверки</label>
      <input id="workers" type="number" min="1">
      <label>Интервал монитора, сек</label>
      <input id="monitorEverySec" type="number" min="5">
      <div class="row" style="margin-top:10px">
        <label><input id="allowInsecure" type="checkbox"> --insecure как в curl</label>
      </div>
      <div class="row" style="margin-top:10px">
        <button onclick="saveSettings()">Сохранить</button>
        <button class="secondary" onclick="checkAll()">Проверить все</button>
      </div>

      <h2 style="margin-top:22px">3proxy</h2>
      <label>Путь к 3proxy.exe</label>
      <input id="exePath" type="text" placeholder="C:\3proxy\bin\3proxy.exe">
      <label>Рабочая папка cfg/logs</label>
      <input id="workDir" type="text">
      <label>Local proxy port</label>
      <input id="proxyPort" type="text">
      <label>Admin port</label>
      <input id="adminPort" type="text">
      <label>Allowed IP</label>
      <input id="allowedIp" type="text">
      <div class="row" style="margin-top:10px">
        <button class="secondary" onclick="start3proxy()">Запустить 3proxy</button>
        <button class="danger" onclick="stop3proxy()">Остановить</button>
      </div>
      <div class="row" style="margin-top:8px">
        <button class="warn" onclick="startMonitor()">Старт монитор</button>
        <button class="danger" onclick="stopMonitor()">Стоп монитор</button>
      </div>
    </section>

    <section>
      <div class="stats">
        <div class="stat"><b id="total">0</b><span class="muted">Всего</span></div>
        <div class="stat"><b id="good">0</b><span class="muted">Рабочих</span></div>
        <div class="stat"><b id="monitor">off</b><span class="muted">Монитор</span></div>
        <div class="stat"><b id="threeproxy">off</b><span class="muted">3proxy</span></div>
      </div>
      <section class="panel">
        <div class="row" style="justify-content:space-between">
          <h2>Прокси</h2>
          <a href="/api/export-good">Скачать good-proxies.txt</a>
        </div>
        <p class="muted small" id="active">Активный parent: нет</p>
        <table>
          <thead><tr><th>Прокси</th><th>Тип</th><th>Логин</th><th>Статус</th><th>HTTP</th><th>Время</th></tr></thead>
          <tbody id="proxyRows"></tbody>
        </table>
      </section>
      <div class="split">
        <section class="panel">
          <h2>Лог</h2>
          <pre id="logs"></pre>
        </section>
        <section class="panel">
          <h2>Файлы</h2>
          <p class="small muted" id="paths"></p>
          <p class="small muted">После смены рабочего прокси программа перезаписывает parent.cfg и меняет reload.txt, чтобы 3proxy увидел monitor.</p>
        </section>
      </div>
    </section>
  </div>
</main>
<script>
let state = null;
async function api(path, body) {
  const res = await fetch(path, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body || {})});
  const data = await res.json();
  if (!data.ok) alert(data.message || 'Ошибка');
  await refresh();
  return data;
}
async function refresh() {
  const res = await fetch('/api/state');
  state = await res.json();
  fillSettings(state.config);
  const resultByKey = new Map();
  let good = 0;
  for (const r of state.results || []) {
    const key = [r.proxy.host, r.proxy.port, r.proxy.login || '', r.proxy.password || ''].join('|').toLowerCase();
    resultByKey.set(key, r);
    if (r.ok) good++;
  }
  document.getElementById('total').textContent = state.proxies.length;
  document.getElementById('good').textContent = good;
  document.getElementById('monitor').textContent = state.monitorRunning ? 'on' : 'off';
  document.getElementById('threeproxy').textContent = state.threeProxyRun ? 'on' : 'off';
  document.getElementById('active').textContent = state.activeProxy ? 'Активный parent: ' + state.activeProxy.host + ':' + state.activeProxy.port : 'Активный parent: нет';
  document.getElementById('logs').textContent = (state.logs || []).slice().reverse().join('\n');
  const cfg = state.config.threeProxy;
  document.getElementById('paths').textContent = '3proxy.cfg: ' + cfg.workDir + '\\3proxy.cfg\nparent.cfg: ' + cfg.workDir + '\\parent.cfg\nreload.txt: ' + cfg.workDir + '\\reload.txt';
  const rows = [];
  for (const p of state.proxies) {
    const key = [p.host, p.port, p.login || '', p.password || ''].join('|').toLowerCase();
    const r = resultByKey.get(key);
    let status = '<span class="muted">не проверен</span>';
    let code = '';
    let ms = '';
    if (r) {
      status = r.ok ? '<span class="ok">OK</span>' : '<span class="fail">FAIL</span>';
      code = r.statusCode || r.error || '';
      ms = r.duration ? Math.round(r.duration / 1000000) + ' ms' : '';
    }
    const effectiveType = r && r.proxy ? r.proxy.type : p.type;
    rows.push('<tr><td>' + esc(p.host + ':' + p.port) + '</td><td>' + esc(proxyTypeLabel(effectiveType)) + '</td><td>' + esc(p.login || '') + '</td><td>' + status + '</td><td>' + esc(String(code)) + '</td><td>' + esc(ms) + '</td></tr>');
  }
  document.getElementById('proxyRows').innerHTML = rows.join('');
}
function proxyTypeLabel(type) {
  switch ((type || 'auto').toLowerCase()) {
    case 'connect': return 'HTTP';
    case 'socks5': return 'SOCKS5';
    default: return 'AUTO';
  }
}
function fillSettings(cfg) {
  const tp = cfg.threeProxy;
  setValue('testUrl', cfg.testUrl);
  setValue('proxyTypeMode', cfg.proxyTypeMode || 'auto');
  setValue('checkTimeoutSec', cfg.checkTimeoutSec);
  setValue('workers', cfg.workers);
  setValue('monitorEverySec', cfg.monitorEverySec);
  document.getElementById('allowInsecure').checked = cfg.allowInsecure;
  setValue('exePath', tp.exePath);
  setValue('workDir', tp.workDir);
  setValue('proxyPort', tp.proxyPort);
  setValue('adminPort', tp.adminPort);
  setValue('allowedIp', tp.allowedIp);
}
function readSettings() {
  const cfg = state.config;
  cfg.proxyTypeMode = value('proxyTypeMode');
  cfg.testUrl = value('testUrl');
  cfg.checkTimeoutSec = num('checkTimeoutSec');
  cfg.workers = num('workers');
  cfg.monitorEverySec = num('monitorEverySec');
  cfg.allowInsecure = document.getElementById('allowInsecure').checked;
  cfg.threeProxy.exePath = value('exePath');
  cfg.threeProxy.workDir = value('workDir');
  cfg.threeProxy.proxyPort = value('proxyPort');
  cfg.threeProxy.adminPort = value('adminPort');
  cfg.threeProxy.allowedIp = value('allowedIp');
  return cfg;
}
function setValue(id, v) { const el = document.getElementById(id); if (document.activeElement !== el) el.value = v || ''; }
function value(id) { return document.getElementById(id).value.trim(); }
function num(id) { return parseInt(value(id), 10) || 0; }
function esc(s) { return s.replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
async function saveSettings() { await api('/api/settings', readSettings()); }
async function loadUrl() { await api('/api/load-url', {url:value('apiUrl')}); }
async function loadFile() {
  const file = document.getElementById('proxyFile').files[0];
  if (!file) return alert('Выберите файл');
  const form = new FormData();
  form.append('file', file);
  const res = await fetch('/api/load-file', {method:'POST', body:form});
  const data = await res.json();
  if (!data.ok) alert(data.message || 'Ошибка');
  await refresh();
}
async function checkAll() { await saveSettings(); await api('/api/check'); }
async function startMonitor() { await saveSettings(); await api('/api/monitor/start'); }
async function stopMonitor() { await api('/api/monitor/stop'); }
async function start3proxy() { await saveSettings(); await api('/api/3proxy/start'); }
async function stop3proxy() { await api('/api/3proxy/stop'); }
refresh();
setInterval(refresh, 2000);
</script>
</body>
</html>`))
