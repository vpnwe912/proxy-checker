package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type Proxy struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	Type     string `json:"type"`
	Source   string `json:"source,omitempty"`
}

type CheckResult struct {
	Proxy      Proxy         `json:"proxy"`
	OK         bool          `json:"ok"`
	StatusCode int           `json:"statusCode"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	CheckedAt  string        `json:"checkedAt"`
}

func (p Proxy) Key() string {
	return strings.ToLower(strings.Join([]string{p.Host, p.Port, p.Login, p.Password, normalizeProxyType(p.Type)}, "|"))
}

func (p Proxy) Address() string {
	return p.Host + ":" + p.Port
}

func (p Proxy) Display() string {
	prefix := ""
	switch normalizeProxyType(p.Type) {
	case "socks5":
		prefix = "socks5://"
	case "connect":
		prefix = "http://"
	}
	if p.Login != "" || p.Password != "" {
		return prefix + p.Login + ":" + p.Password + "@" + p.Host + ":" + p.Port
	}
	return prefix + p.Host + ":" + p.Port
}

func (p Proxy) URL() (*url.URL, error) {
	if p.Host == "" || p.Port == "" {
		return nil, errors.New("empty proxy host or port")
	}
	u := &url.URL{
		Scheme: proxyURLScheme(p.Type),
		Host:   p.Host + ":" + p.Port,
	}
	if p.Login != "" || p.Password != "" {
		u.User = url.UserPassword(p.Login, p.Password)
	}
	return u, nil
}

func (p Proxy) URLWithoutAuth() (*url.URL, error) {
	if p.Host == "" || p.Port == "" {
		return nil, errors.New("empty proxy host or port")
	}
	return &url.URL{
		Scheme: proxyURLScheme(p.Type),
		Host:   p.Host + ":" + p.Port,
	}, nil
}

func ParseProxyList(data []byte, source string) ([]Proxy, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		proxies, err := parseJSONProxies(trimmed, source)
		if err == nil && len(proxies) > 0 {
			return dedupeProxies(proxies), nil
		}
		proxies, err = parseJSONLines(string(data), source)
		if err == nil && len(proxies) > 0 {
			return dedupeProxies(proxies), nil
		}
	}
	proxies, err := parseTextProxies(string(data), source)
	if err != nil {
		return nil, err
	}
	return dedupeProxies(proxies), nil
}

func parseJSONLines(input string, source string) ([]Proxy, error) {
	var proxies []Proxy
	var firstErr error
	for _, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "[") {
			continue
		}
		parsed, err := parseJSONProxies([]byte(line), source)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		proxies = append(proxies, parsed...)
	}
	if len(proxies) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return proxies, nil
}

func parseTextProxies(input string, source string) ([]Proxy, error) {
	var proxies []Proxy
	var bad []string
	for _, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := parseProxyLine(line, source)
		if err != nil {
			bad = append(bad, line)
			continue
		}
		proxies = append(proxies, p)
	}
	if len(proxies) == 0 && len(bad) > 0 {
		return nil, fmt.Errorf("no valid proxies found; first invalid line: %s", bad[0])
	}
	return proxies, nil
}

func parseProxyLine(line string, source string) (Proxy, error) {
	p := Proxy{Type: defaultProxyType(source), Source: source}
	if parsed, ok, err := parseProxyURLLine(line, source); ok || err != nil {
		return parsed, err
	}
	if strings.Contains(line, "@") {
		left, right, ok := strings.Cut(line, "@")
		if !ok {
			return p, errors.New("invalid auth separator")
		}
		if host, port, err := splitHostPort(left); err == nil && isValidPort(port) {
			login, pass := splitLoginPassword(right)
			p.Host, p.Port, p.Login, p.Password = host, port, login, pass
			return normalizeProxy(p)
		}
		if host, port, err := splitHostPort(right); err == nil && isValidPort(port) {
			login, pass := splitLoginPassword(left)
			p.Host, p.Port, p.Login, p.Password = host, port, login, pass
			return normalizeProxy(p)
		}
		if strings.Count(left, ":") == 1 {
			host, port, err := splitHostPort(left)
			if err != nil {
				return p, err
			}
			login, pass := splitLoginPassword(right)
			p.Host, p.Port, p.Login, p.Password = host, port, login, pass
			return normalizeProxy(p)
		}
		login, pass := splitLoginPassword(left)
		host, port, err := splitHostPort(right)
		if err != nil {
			return p, err
		}
		p.Host, p.Port, p.Login, p.Password = host, port, login, pass
		return normalizeProxy(p)
	}

	parts := strings.Split(line, ":")
	switch len(parts) {
	case 2:
		p.Host = strings.TrimSpace(parts[0])
		p.Port = strings.TrimSpace(parts[1])
	case 4:
		p.Host = strings.TrimSpace(parts[0])
		p.Port = strings.TrimSpace(parts[1])
		p.Login = strings.TrimSpace(parts[2])
		p.Password = strings.TrimSpace(parts[3])
	default:
		return p, errors.New("expected host:port, http://host:port, socks5://host:port, host:port@login:pass, login:pass@host:port or host:port:login:pass")
	}
	return normalizeProxy(p)
}

func parseProxyURLLine(line string, source string) (Proxy, bool, error) {
	if !strings.Contains(line, "://") {
		return Proxy{}, false, nil
	}
	u, err := url.Parse(line)
	if err != nil {
		return Proxy{}, true, err
	}
	proxyType, ok := proxyTypeFromScheme(u.Scheme)
	if !ok {
		return Proxy{}, true, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" || port == "" {
		return Proxy{}, true, errors.New("proxy URL must contain host and port")
	}
	p := Proxy{Host: host, Port: port, Type: proxyType, Source: source}
	if u.User != nil {
		p.Login = u.User.Username()
		p.Password, _ = u.User.Password()
	}
	p, err = normalizeProxy(p)
	return p, true, err
}

func isValidPort(value string) bool {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	return err == nil && port >= 1 && port <= 65535
}

func splitHostPort(value string) (string, string, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return "", "", errors.New("expected host:port")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func splitLoginPassword(value string) (string, string) {
	login, pass, ok := strings.Cut(value, ":")
	if !ok {
		return strings.TrimSpace(value), ""
	}
	return strings.TrimSpace(login), strings.TrimSpace(pass)
}

func normalizeProxy(p Proxy) (Proxy, error) {
	p.Host = strings.TrimSpace(p.Host)
	p.Port = strings.TrimSpace(p.Port)
	p.Login = strings.TrimSpace(p.Login)
	p.Password = strings.TrimSpace(p.Password)
	if p.Type == "" {
		p.Type = "auto"
	}
	proxyType, err := normalizeProxyTypeWithError(p.Type)
	if err != nil {
		return p, err
	}
	p.Type = proxyType
	if p.Host == "" || p.Port == "" {
		return p, errors.New("empty host or port")
	}
	port, err := strconv.Atoi(p.Port)
	if err != nil || port < 1 || port > 65535 {
		return p, fmt.Errorf("invalid port: %s", p.Port)
	}
	return p, nil
}

func parseJSONProxies(data []byte, source string) ([]Proxy, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	var proxies []Proxy
	collectJSONProxies(value, source, &proxies)
	if len(proxies) == 0 {
		return nil, errors.New("json contains no proxy objects with host and port")
	}
	return proxies, nil
}

func collectJSONProxies(value any, source string, out *[]Proxy) {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			collectJSONProxies(item, source, out)
		}
	case string:
		p, err := parseProxyLine(v, source)
		if err == nil {
			*out = append(*out, p)
		}
	case map[string]any:
		for _, key := range []string{"proxy", "address", "addr"} {
			if line := stringField(v, key); line != "" {
				p, err := parseProxyLine(line, source)
				if err == nil {
					*out = append(*out, p)
					return
				}
			}
		}
		host := stringField(v, "host")
		port := stringField(v, "port")
		if host != "" && port != "" {
			p := Proxy{
				Host:     host,
				Port:     port,
				Login:    firstStringField(v, "login", "user", "username"),
				Password: firstStringField(v, "password", "pass"),
				Type:     proxyTypeFromFields(v, source),
				Source:   source,
			}
			if normalized, err := normalizeProxy(p); err == nil {
				*out = append(*out, normalized)
			}
			return
		}
		for _, item := range v {
			collectJSONProxies(item, source, out)
		}
	}
}

func firstStringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringField(m, key); value != "" {
			return value
		}
	}
	return ""
}

func stringField(m map[string]any, key string) string {
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func dedupeProxies(proxies []Proxy) []Proxy {
	seen := map[string]struct{}{}
	var result []Proxy
	for _, p := range proxies {
		key := p.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, p)
	}
	return result
}

func FetchProxies(ctx context.Context, apiURL string) ([]Proxy, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ParseProxyList(body, apiURL)
}

func CheckProxy(ctx context.Context, p Proxy, testURL string, timeout time.Duration, allowInsecure bool, useCurl bool) CheckResult {
	if useCurl {
		if result, ok := CheckProxyWithCurl(ctx, p, testURL, timeout, allowInsecure); ok {
			return result
		}
	}
	return CheckProxyWithGo(ctx, p, testURL, timeout, allowInsecure)
}

func CheckProxyWithCurl(ctx context.Context, p Proxy, testURL string, timeout time.Duration, allowInsecure bool) (CheckResult, bool) {
	if normalizeProxyType(p.Type) == "auto" {
		httpResult, ok := CheckProxyWithCurl(ctx, withProxyType(p, "connect"), testURL, timeout, allowInsecure)
		if !ok || httpResult.OK {
			return httpResult, ok
		}
		socksResult, ok := CheckProxyWithCurl(ctx, withProxyType(p, "socks5"), testURL, timeout, allowInsecure)
		if !ok || socksResult.OK {
			return socksResult, ok
		}
		return combinedAutoResult(p, httpResult, socksResult), true
	}
	start := time.Now()
	result := CheckResult{Proxy: p, CheckedAt: start.Format(time.RFC3339)}
	proxyURL, err := p.URLWithoutAuth()
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result, true
	}
	if _, err := exec.LookPath("curl.exe"); err != nil {
		return result, false
	}
	proxyFlag := "-x"
	proxyAddress := proxyURL.String()
	if normalizeProxyType(p.Type) == "socks5" {
		proxyFlag = "--socks5"
		proxyAddress = proxyURL.Host
	}
	args := []string{
		proxyFlag, proxyAddress,
		"--max-time", strconv.Itoa(max(1, int(timeout.Seconds()))),
		"--connect-timeout", strconv.Itoa(max(1, int(timeout.Seconds()))),
		"--location",
		"--silent",
		"--show-error",
		"--output", "NUL",
		"--write-out", "%{http_code}",
	}
	if p.Login != "" || p.Password != "" {
		args = append(args, "--proxy-user", p.Login+":"+p.Password, "--proxy-basic")
	}
	if allowInsecure {
		args = append(args, "--insecure")
	}
	args = append(args, testURL)
	cmd := exec.CommandContext(ctx, "curl.exe", args...)
	hideChildProcessWindow(cmd)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	result.Duration = time.Since(start)
	httpCode := strings.TrimSpace(stdout.String())
	if code, convErr := strconv.Atoi(httpCode); convErr == nil {
		result.StatusCode = code
		result.OK = code == http.StatusOK || code == http.StatusMovedPermanently || code == http.StatusFound
		if !result.OK && err == nil {
			result.Error = "HTTP " + httpCode
		}
	}
	if err != nil {
		result.Error = strings.TrimSpace(stderr.String())
		if result.Error == "" {
			result.Error = strings.TrimSpace(stdout.String())
		}
		if result.Error == "" {
			result.Error = err.Error()
		} else {
			result.Error += " (" + err.Error() + ")"
		}
	}
	return result, true
}

func CheckProxyWithGo(ctx context.Context, p Proxy, testURL string, timeout time.Duration, allowInsecure bool) CheckResult {
	if normalizeProxyType(p.Type) == "auto" {
		httpResult := CheckProxyWithGo(ctx, withProxyType(p, "connect"), testURL, timeout, allowInsecure)
		if httpResult.OK {
			return httpResult
		}
		socksResult := CheckProxyWithGo(ctx, withProxyType(p, "socks5"), testURL, timeout, allowInsecure)
		if socksResult.OK {
			return socksResult
		}
		return combinedAutoResult(p, httpResult, socksResult)
	}
	start := time.Now()
	result := CheckResult{Proxy: p, CheckedAt: start.Format(time.RFC3339)}
	proxyURL, err := p.URL()
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: allowInsecure,
		},
	}
	if normalizeProxyType(p.Type) == "socks5" {
		dialer, err := socks5Dialer(p)
		if err != nil {
			result.Error = err.Error()
			result.Duration = time.Since(start)
			return result
		}
		transport.DialContext = dialer.DialContext
	} else {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	req.Header.Set("User-Agent", "ProxyChecker/1.0")
	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	result.StatusCode = resp.StatusCode
	result.OK = resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound
	result.Duration = time.Since(start)
	return result
}

func withProxyType(p Proxy, proxyType string) Proxy {
	p.Type = proxyType
	return p
}

func applyProxyTypeMode(proxies []Proxy, mode string) []Proxy {
	proxyType := normalizeProxyType(mode)
	if proxyType == "auto" {
		return proxies
	}
	result := append([]Proxy(nil), proxies...)
	for i := range result {
		if normalizeProxyType(result[i].Type) == "auto" {
			result[i].Type = proxyType
		}
	}
	return result
}

func combinedAutoResult(p Proxy, httpResult CheckResult, socksResult CheckResult) CheckResult {
	result := socksResult
	result.Proxy = p
	result.StatusCode = 0
	result.OK = false
	result.Duration = httpResult.Duration + socksResult.Duration
	result.CheckedAt = httpResult.CheckedAt
	result.Error = "HTTP: " + resultErrorText(httpResult) + "; SOCKS5: " + resultErrorText(socksResult)
	return result
}

func resultErrorText(result CheckResult) string {
	if result.Error != "" {
		return result.Error
	}
	if result.StatusCode != 0 {
		return "HTTP " + strconv.Itoa(result.StatusCode)
	}
	return "failed"
}

func proxyTypeFromFields(m map[string]any, source string) string {
	for _, key := range []string{"type", "protocol", "scheme"} {
		if proxyType, ok := proxyTypeFromScheme(stringField(m, key)); ok {
			return proxyType
		}
	}
	return defaultProxyType(source)
}

func defaultProxyType(source string) string {
	lower := strings.ToLower(source)
	if strings.Contains(lower, "socks5") || strings.Contains(lower, "socks") {
		return "socks5"
	}
	if strings.Contains(lower, "http_auth") || strings.Contains(lower, "https_auth") || strings.Contains(lower, "type=http") || strings.Contains(lower, "protocol=http") {
		return "connect"
	}
	return "auto"
}

func proxyTypeFromScheme(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return "auto", true
	case "http", "https", "connect":
		return "connect", true
	case "socks5", "socks":
		return "socks5", true
	default:
		return "", false
	}
}

func normalizeProxyType(value string) string {
	proxyType, err := normalizeProxyTypeWithError(value)
	if err != nil {
		return "auto"
	}
	return proxyType
}

func normalizeProxyTypeWithError(value string) (string, error) {
	if proxyType, ok := proxyTypeFromScheme(value); ok {
		return proxyType, nil
	}
	return "", fmt.Errorf("unsupported proxy type: %s", value)
}

func proxyURLScheme(proxyType string) string {
	if normalizeProxyType(proxyType) == "socks5" {
		return "socks5"
	}
	return "http"
}

type contextDialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

type contextDialerFunc func(ctx context.Context, network string, address string) (net.Conn, error)

func (f contextDialerFunc) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func socks5Dialer(p Proxy) (contextDialer, error) {
	var auth *proxy.Auth
	if p.Login != "" || p.Password != "" {
		auth = &proxy.Auth{User: p.Login, Password: p.Password}
	}
	dialer, err := proxy.SOCKS5("tcp", p.Address(), auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	if contextDialer, ok := dialer.(contextDialer); ok {
		return contextDialer, nil
	}
	return contextDialerFunc(func(ctx context.Context, network string, address string) (net.Conn, error) {
		type dialResult struct {
			conn net.Conn
			err  error
		}
		resultCh := make(chan dialResult, 1)
		go func() {
			conn, err := dialer.Dial(network, address)
			resultCh <- dialResult{conn: conn, err: err}
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-resultCh:
			return result.conn, result.err
		}
	}), nil
}
