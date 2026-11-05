package proxy

import (
	"strings"
	"testing"
)

func TestCleanHostPort(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"socks5://127.0.0.1:10808", "127.0.0.1:10808"},
		{"http://127.0.0.1:7890/", "127.0.0.1:7890"},
		{"socks5h://127.0.0.1:2080", "127.0.0.1:2080"},
		{"127.0.0.1:10808", "127.0.0.1:10808"},
	}

	for _, c := range cases {
		res := CleanHostPort(c.input)
		if res != c.expected {
			t.Errorf("CleanHostPort(%s) = %s; want %s", c.input, res, c.expected)
		}
	}
}

func TestBuildProxyEnv(t *testing.T) {
	env := BuildProxyEnv("socks5h://127.0.0.1:10808")
	var foundAllProxy bool
	for _, e := range env {
		if strings.HasPrefix(e, "ALL_PROXY=") {
			foundAllProxy = true
			if !strings.Contains(e, "127.0.0.1:10808") {
				t.Errorf("ALL_PROXY does not contain target proxy: %s", e)
			}
		}
	}
	if !foundAllProxy {
		t.Errorf("BuildProxyEnv did not produce ALL_PROXY")
	}
}

func TestDirectConnectivityAndVpnDetection(t *testing.T) {
	// 1. HasActiveVpnInterface should execute safely without crashing
	vpnActive := HasActiveVpnInterface()
	t.Logf("HasActiveVpnInterface returned: %v", vpnActive)

	// 2. DirectEndpoint definition check
	if DirectEndpoint.Type != "direct" || DirectEndpoint.URL != "direct://" {
		t.Errorf("DirectEndpoint invalid: %+v", DirectEndpoint)
	}

	// 3. CheckDirectConnectivity execution
	// Should execute with timeout without panicking
	reachable := CheckDirectConnectivity(800 * 1000 * 1000) // 800ms
	t.Logf("CheckDirectConnectivity reachable: %v", reachable)

	// 4. GetBestProxy returns a valid endpoint if reachable/VPN or local proxy
	best, err := GetBestProxy()
	if err == nil {
		t.Logf("GetBestProxy discovered: %+v", best)
		if best.URL == "" {
			t.Errorf("GetBestProxy returned empty URL")
		}
	} else {
		t.Logf("GetBestProxy returned error (no proxy or direct connectivity): %v", err)
	}
}

func BenchmarkCleanHostPort(b *testing.B) {
	input := "socks5h://127.0.0.1:10808"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res := CleanHostPort(input)
		if res == "" {
			b.Fatal("empty host port")
		}
	}
}

func BenchmarkBuildProxyEnv(b *testing.B) {
	input := "socks5h://127.0.0.1:10808"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		env := BuildProxyEnv(input)
		if len(env) == 0 {
			b.Fatal("empty proxy env")
		}
	}
}
