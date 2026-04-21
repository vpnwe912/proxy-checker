package main

import "testing"

func TestParseProxyFormats(t *testing.T) {
	input := []byte(`127.0.0.1:8085
127.0.0.1:8080@USER:PWD
USER:PWD@127.0.0.1:8080
127.0.0.1:8080:USER:PWD
http://127.0.0.1:8080
socks5://USER:PWD@127.0.0.1:1080`)
	proxies, err := ParseProxyList(input, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 6 {
		t.Fatalf("expected 6 proxies, got %d", len(proxies))
	}
	if proxies[1].Login != "USER" || proxies[1].Password != "PWD" {
		t.Fatalf("auth was not parsed: %+v", proxies[1])
	}
	if proxies[0].Type != "auto" {
		t.Fatalf("bare proxy should use auto detection: %+v", proxies[0])
	}
	if proxies[4].Type != "connect" {
		t.Fatalf("http scheme should map to connect parent: %+v", proxies[4])
	}
	if proxies[5].Type != "socks5" || proxies[5].Login != "USER" || proxies[5].Password != "PWD" {
		t.Fatalf("socks5 URL was not parsed: %+v", proxies[5])
	}
}

func TestParseJSONProxyFormats(t *testing.T) {
	input := []byte(`[
{"host":"127.0.0.1","port":"8085"},
{"host":"127.0.0.1","port":"8080","login":"USER","password":"PWD"},
{"host":"127.0.0.1","port":"1080","login":"USER","password":"PWD","type":"socks5"}
]`)
	proxies, err := ParseProxyList(input, "api")
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 3 {
		t.Fatalf("expected 3 proxies, got %d", len(proxies))
	}
	if proxies[1].Login == "" || proxies[1].Password == "" {
		t.Fatalf("json auth was not parsed: %+v", proxies[1])
	}
	if proxies[2].Type != "socks5" {
		t.Fatalf("json type was not parsed: %+v", proxies[2])
	}
}

func TestParseJSONLinesProxyFormat(t *testing.T) {
	input := []byte(`{"host":"127.0.0.1","port":"8085"}
{"host":"127.0.0.1","port":"8080","login":"USER","password":"PWD"}`)
	proxies, err := ParseProxyList(input, "api")
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}
	if proxies[0].Port != "8085" || proxies[1].Login != "USER" {
		t.Fatalf("json lines were not parsed: %+v", proxies)
	}
}

func TestParseJSONStringProxyFormatsUseAutoDetection(t *testing.T) {
	input := []byte(`[
"127.0.0.1:8080@USER:PWD",
"127.0.0.1:1080@USER:PWD",
{"proxy":"127.0.0.1:1080@USER:PWD"}
]`)
	proxies, err := ParseProxyList(input, "api")
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 3 {
		t.Fatalf("expected 3 proxies, got %d", len(proxies))
	}
	for _, p := range proxies {
		if p.Type != "auto" {
			t.Fatalf("bare proxy should use auto detection: %+v", p)
		}
		if p.Login != "USER" || p.Password != "PWD" {
			t.Fatalf("auth was not parsed: %+v", p)
		}
	}
}

func TestParseAPITypeHintSetsHTTPProxyType(t *testing.T) {
	input := []byte(`127.0.0.1:8080@USER:PWD
127.0.0.1:1080@USER:PWD`)
	proxies, err := ParseProxyList(input, "https://proxy-example.com/api/getproxy/?format=txt&type=http_auth")
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}
	for _, p := range proxies {
		if p.Type != "connect" {
			t.Fatalf("http_auth API proxies should be HTTP/connect: %+v", p)
		}
	}
}

func TestApplyProxyTypeModeOnlyChangesAutoProxies(t *testing.T) {
	proxies := []Proxy{
		{Host: "1.1.1.1", Port: "8080", Type: "auto"},
		{Host: "2.2.2.2", Port: "1080", Type: "socks5"},
	}
	result := applyProxyTypeMode(proxies, "connect")
	if result[0].Type != "connect" {
		t.Fatalf("auto proxy should be forced to connect: %+v", result[0])
	}
	if result[1].Type != "socks5" {
		t.Fatalf("explicit socks5 proxy should stay socks5: %+v", result[1])
	}
	if proxies[0].Type != "auto" {
		t.Fatalf("input slice should not be mutated: %+v", proxies[0])
	}
}
