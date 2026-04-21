package main

import "time"
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
	if len(proxies) != 4 {
		t.Fatalf("expected 4 proxies after dedupe, got %d", len(proxies))
	}
	if proxies[1].Login != "USER" || proxies[1].Password != "PWD" {
		t.Fatalf("auth was not parsed: %+v", proxies[1])
	}
	if proxies[0].Type != "auto" {
		t.Fatalf("bare proxy should use auto detection: %+v", proxies[0])
	}
	if proxies[2].Type != "connect" {
		t.Fatalf("http scheme should map to connect parent: %+v", proxies[2])
	}
	if proxies[3].Type != "socks5" || proxies[3].Login != "USER" || proxies[3].Password != "PWD" {
		t.Fatalf("socks5 URL was not parsed: %+v", proxies[3])
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
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies after dedupe, got %d", len(proxies))
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

func TestParseIPInfoGeo(t *testing.T) {
	body := []byte(`{
  "ip": "170.168.31.178",
  "region": "Cherkasy",
  "country": "UA",
  "timezone": "Europe/Kyiv"
}`)
	geo, err := parseGeoResponse("ipinfo", "https://ipinfo.io/json", body)
	if err != nil {
		t.Fatal(err)
	}
	if geo.Country != "UA" || geo.Region != "Cherkasy" || geo.Timezone != "Europe/Kyiv" {
		t.Fatalf("unexpected geo info: %+v", geo)
	}
}

func TestParseIPAPIGeo(t *testing.T) {
	body := []byte(`{
  "status":"success",
  "country":"Ukraine",
  "countryCode":"UA",
  "region":"30",
  "regionName":"Kyiv",
  "city":"Kyiv",
  "timezone":"Europe/Kyiv",
  "query":"170.168.31.178"
}`)
	geo, err := parseGeoResponse("ipapi", "http://ip-api.com/json/", body)
	if err != nil {
		t.Fatal(err)
	}
	if geo.IP != "170.168.31.178" || geo.Country != "UA" || geo.Region != "Kyiv" || geo.Timezone != "Europe/Kyiv" {
		t.Fatalf("unexpected ip-api geo info: %+v", geo)
	}
}

func TestParseIPWhoIsGeo(t *testing.T) {
	body := []byte(`{
  "success": true,
  "ip": "170.168.31.178",
  "country_code": "UA",
  "region": "Kyiv",
  "timezone": {"id":"Europe/Kyiv"}
}`)
	geo, err := parseGeoResponse("ipwhois", "https://ipwho.is/", body)
	if err != nil {
		t.Fatal(err)
	}
	if geo.Country != "UA" || geo.Region != "Kyiv" || geo.Timezone != "Europe/Kyiv" {
		t.Fatalf("unexpected ipwho.is geo info: %+v", geo)
	}
}

func TestParse2IPJSONGeo(t *testing.T) {
	body := []byte(`{
  "ip":"170.168.31.178",
  "country_code":"UA",
  "city":"Kyiv"
}`)
	geo, err := parseGeoResponse("2ip_json", "https://2ip.ua/json", body)
	if err != nil {
		t.Fatal(err)
	}
	if geo.Country != "UA" || geo.Region != "Kyiv" {
		t.Fatalf("unexpected 2ip json geo info: %+v", geo)
	}
}

func TestParse2IPGeo(t *testing.T) {
	body := []byte(` ip             : 170.168.31.242
 provider       : Alex Largman
 location       : Ukraine (UA), Kyiv
`)
	geo, err := parseGeoResponse("2ip", "https://2ip.ua", body)
	if err != nil {
		t.Fatal(err)
	}
	if geo.IP != "170.168.31.242" {
		t.Fatalf("unexpected IP: %+v", geo)
	}
	if geo.Country != "UA" {
		t.Fatalf("unexpected country: %+v", geo)
	}
	if geo.Region != "Kyiv" {
		t.Fatalf("unexpected region: %+v", geo)
	}
	if geo.Timezone != "" {
		t.Fatalf("2ip parser should not invent timezone: %+v", geo)
	}
}

func TestInferGeoProvider(t *testing.T) {
	if inferGeoProvider("https://2ip.ua") != "2ip" {
		t.Fatal("expected 2ip provider")
	}
	if inferGeoProvider("https://ipinfo.io/json") != "ipinfo" {
		t.Fatal("expected ipinfo provider")
	}
	if inferGeoProvider("https://2ip.ua/json") != "2ip_json" {
		t.Fatal("expected 2ip_json provider")
	}
	if inferGeoProvider("http://ip-api.com/json/") != "ipapi" {
		t.Fatal("expected ipapi provider")
	}
	if inferGeoProvider("https://ifconfig.co/json") != "ifconfig" {
		t.Fatal("expected ifconfig provider")
	}
	if inferGeoProvider("https://ipwho.is/") != "ipwhois" {
		t.Fatal("expected ipwhois provider")
	}
	if inferGeoProvider("https://ipapi.co/json/") != "ipapi_co" {
		t.Fatal("expected ipapi_co provider")
	}
	if inferGeoProvider("https://api.ip.sb/geoip") != "ip_sb" {
		t.Fatal("expected ip_sb provider")
	}
	if inferGeoProvider("") != "auto" {
		t.Fatal("expected auto provider for empty URL")
	}
}

func TestMergeGeoInfo(t *testing.T) {
	base := GeoInfo{Country: "UA", Region: "Kyiv"}
	next := GeoInfo{IP: "1.2.3.4", Country: "FI", Region: "Uusimaa", Timezone: "Europe/Kyiv"}
	merged := mergeGeoInfo(base, next)
	if merged.Country != "UA" || merged.Region != "Kyiv" {
		t.Fatalf("base values should win: %+v", merged)
	}
	if merged.IP != "1.2.3.4" || merged.Timezone != "Europe/Kyiv" {
		t.Fatalf("missing values should be filled: %+v", merged)
	}
}

func TestMergeProxiesAddsOnlyNewEntries(t *testing.T) {
	state := NewAppState()
	state.setProxies([]Proxy{
		{Host: "1.1.1.1", Port: "8080", Type: "connect"},
	})
	added, skipped := state.mergeProxies([]Proxy{
		{Host: "1.1.1.1", Port: "8080", Type: "connect"},
		{Host: "2.2.2.2", Port: "8080", Type: "connect"},
	})
	if added != 1 || skipped != 1 {
		t.Fatalf("unexpected merge result: added=%d skipped=%d", added, skipped)
	}
	if len(state.proxies) != 2 {
		t.Fatalf("expected 2 proxies after merge, got %d", len(state.proxies))
	}
}

func TestRemoveDeadProxiesKeepsUncheckedAndAlive(t *testing.T) {
	state := NewAppState()
	state.setProxies([]Proxy{
		{Host: "1.1.1.1", Port: "8080", Type: "connect"},
		{Host: "2.2.2.2", Port: "8080", Type: "connect"},
		{Host: "3.3.3.3", Port: "8080", Type: "connect"},
	})
	state.results[state.proxies[0].Key()] = CheckResult{Proxy: state.proxies[0], OK: true, CheckedAt: time.Now().Format(time.RFC3339)}
	state.results[state.proxies[1].Key()] = CheckResult{Proxy: state.proxies[1], OK: false, CheckedAt: time.Now().Format(time.RFC3339)}
	removed := state.removeDeadProxies()
	if removed != 1 {
		t.Fatalf("expected 1 removed proxy, got %d", removed)
	}
	if len(state.proxies) != 2 {
		t.Fatalf("expected 2 proxies after cleanup, got %d", len(state.proxies))
	}
	for _, proxy := range state.proxies {
		if proxy.Host == "2.2.2.2" {
			t.Fatalf("dead proxy was not removed")
		}
	}
}
