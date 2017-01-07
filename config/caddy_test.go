package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/diff"
	"github.com/mholt/caddy/caddyfile"
)

func init() {
	for _, s := range []string{
		"BRUNO_HOME",
		"portof_dealer_szerzodesek",
		"portof_splprn_admin_https",
		"portof_splprn_admin",
		"portof_aodb_https",
		"portof_aodb_http",
		"portof_mevv",
		"portof_ws",
	} {
		if os.Getenv(s) == "" {
			os.Setenv(s, fmt.Sprintf("{%s}", s))
		}
	}
}

func TestCaddyParse(t *testing.T) {
	var ed caddyEncDec
	cfg, err := ed.Decode(strings.NewReader(caddyTest1))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(cfg.String())
}
func TestCaddyBlocks(t *testing.T) {
	blocks, err := caddyfile.Parse("Caddyfile", strings.NewReader(caddyTest1), nil)
	if err != nil {
		t.Fatal(err)
	}
	cb := convertCaddyBlock(blocks[0])

	var gBuf, wBuf bytes.Buffer
	gE := json.NewEncoder(&gBuf)
	gE.SetIndent("", "  ")
	wE := json.NewEncoder(&wBuf)
	wE.SetIndent("", "  ")
	for key, want := range map[string][]caddyDirective{
		"tls": {{
			Main: caddyLine{
				Name: "tls",
				Args: []string{"{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem",
					"{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem"}},
			Params: []caddyLine{
				caddyLine{Name: "protocols", Args: []string{"tls1.0", "tls1.2"}},
			},
		}},

		"log": {{Main: caddyLine{Name: "log", Args: []string{"{BRUNO_HOME}/data/mai/log/ws-proxy.log"}}}},

		"rewrite": {
			{Main: caddyLine{Name: "rewrite"},
				Params: []caddyLine{
					caddyLine{Name: "r", Args: []string{"/Dealer/(.*)"}},
					caddyLine{Name: "to", Args: []string{"/letme/Dealer/{1}"}},
				}},
			{Main: caddyLine{Name: "rewrite"},
				Params: []caddyLine{
					caddyLine{Name: "r", Args: []string{"^(/letme)?/Dealer/Dealer/(.*)"}},
					caddyLine{Name: "to", Args: []string{"/letme/Dealer/{1}"}},
				}},
		},
	} {
		gBuf.Reset()
		if err := gE.Encode(cb[key]); err != nil {
			t.Fatalf("encode %#v: %v", cb[key], err)
		}
		wBuf.Reset()
		if err := wE.Encode(want); err != nil {
			t.Fatalf("encode %#v: %v", want, err)
		}
		if d := diff.Diff(
			string(bytes.Replace(wBuf.Bytes(), []byte(": []"), []byte(": null"), -1)),
			string(bytes.Replace(gBuf.Bytes(), []byte(": []"), []byte(": null"), -1)),
		); d != "" {
			t.Errorf("%s:\n%s", key, d)
		}
	}

	proxy := cb["proxy"]
	if len(proxy) != 6 {
		t.Errorf("Got %d proxy directives, wanted 6.", len(proxy))
	}
	t.Logf("proxy=%#v", proxy)
}

func TestConvertCT(t *testing.T) {
	var ed caddyEncDec
	cfg, err := ed.Decode(strings.NewReader(caddyTest1))
	if err != nil {
		t.Fatal(err)
	}
	want, got := caddyTest1TOML, cfg.String()
	t.Log("Got:")
	t.Log(got)
	if d := diff.Diff(want, got); d != "" {
		t.Error(d)
	}
}

const caddyTest1TOML = `
[http://0%2E0%2E0%2E0:4444]

  [http://0%2E0%2E0%2E0:4444.log]
    args = ["{BRUNO_HOME}/data/mai/log/grafana-proxy.log"]

  [http://0%2E0%2E0%2E0:4444.proxy]
    args = ["/","http://192.168.3.110:3000"]

    [http://0%2E0%2E0%2E0:4444.proxy.params]
      header_upstream = ["-Proxy",""]
      without = ["/metrics/mabisz"]

[http://0%2E0%2E0%2E0:{portof_aodb_http}]

  [http://0%2E0%2E0%2E0:{portof_aodb_http}.log]
    args = ["{BRUNO_HOME}/data/mai/log/aodb-proxy.log"]

  [http://0%2E0%2E0%2E0:{portof_aodb_http}.proxy]
    args = ["/","unix:{BRUNO_HOME}/data/ws/aodb.socket"]

    [http://0%2E0%2E0%2E0:{portof_aodb_http}.proxy.params]
      fail_timeout = ["1s"]
      header_upstream = ["-Proxy",""]
      max_fails = ["3"]
      policy = ["least_conn"]
      transparent = ""
      try_duration = ["3s"]
      without = ["/_macroexpert"]

[https://0%2E0%2E0%2E0:{portof_aodb_https}]

  [https://0%2E0%2E0%2E0:{portof_aodb_https}.log]
    args = ["{BRUNO_HOME}/data/mai/log/aodb-proxy-https.log"]

  [https://0%2E0%2E0%2E0:{portof_aodb_https}.proxy]
    args = ["/","http://localhost:"]

    [https://0%2E0%2E0%2E0:{portof_aodb_https}.proxy.params]
      header_upstream = ["-Proxy",""]
      without = ["/_macroexpert"]

  [https://0%2E0%2E0%2E0:{portof_aodb_https}.tls]
    args = ["{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem","{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem"]

    [https://0%2E0%2E0%2E0:{portof_aodb_https}.tls.params]
      protocols = ["tls1.0","tls1.2"]

[https://0%2E0%2E0%2E0:{portof_splprn_admin_https}]

  [https://0%2E0%2E0%2E0:{portof_splprn_admin_https}.log]
    args = ["{BRUNO_HOME}/data/mai/log/splprn_admin-proxy.log"]

  [https://0%2E0%2E0%2E0:{portof_splprn_admin_https}.proxy]
    args = ["/","http://localhost:{portof_splprn_admin}"]

    [https://0%2E0%2E0%2E0:{portof_splprn_admin_https}.proxy.params]
      header_upstream = ["-Proxy",""]

  [https://0%2E0%2E0%2E0:{portof_splprn_admin_https}.tls]
    args = ["{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem","{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem"]

    [https://0%2E0%2E0%2E0:{portof_splprn_admin_https}.tls.params]
      protocols = ["tls1.0","tls1.2"]

[https://0%2E0%2E0%2E0:{portof_ws}]

  [https://0%2E0%2E0%2E0:{portof_ws}.log]
    args = ["{BRUNO_HOME}/data/mai/log/ws-proxy.log"]

  [https://0%2E0%2E0%2E0:{portof_ws}.proxy]
    args = ["/","unix:{BRUNO_HOME}/data/ws/ws-1.socket","unix:{BRUNO_HOME}/data/ws/ws-1.socket"]

    [https://0%2E0%2E0%2E0:{portof_ws}.proxy.params]
      fail_timeout = ["9s"]
      header_upstream = ["X-Forwarded-For","{remote}"]
      max_fails = ["1"]
      policy = ["least_conn"]
      transparent = ""
      try_duration = ["10s"]
      without = ["/letme"]

  [https://0%2E0%2E0%2E0:{portof_ws}.rewrite]

    [https://0%2E0%2E0%2E0:{portof_ws}.rewrite.params]
      r = ["^(/letme)?/Dealer/Dealer/(.*)"]
      to = ["/letme/Dealer/{1}"]

  [https://0%2E0%2E0%2E0:{portof_ws}.tls]
    args = ["{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem","{BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem"]

    [https://0%2E0%2E0%2E0:{portof_ws}.tls.params]
      protocols = ["tls1.0","tls1.2"]
`

const caddyTest1 = `## WS-HTTP
#http://0.0.0.0:{$portof_ws_http} {
#	log {$BRUNO_HOME}/data/mai/log/ws-proxy-http.log
#
#	proxy /ungzip http://localhost:{$portof_ws_1} http://localhost:{$portof_ws_2} {
#		header_upstream -Proxy ""
#		header_upstream X-Forwarded-For {remote}
#	}
#}

# WS
https://0.0.0.0:{$portof_ws} {
	log {$BRUNO_HOME}/data/mai/log/ws-proxy.log
	tls {$BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem {$BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem {
		protocols tls1.0 tls1.2
	}

	proxy /dealer/allomany http://localhost:{$portof_dealer_szerzodesek} {
		header_upstream X-Forward-For {remote}
		without /dealer/allomany
	}

	proxy /inphone unix:{$BRUNO_HOME}/data/ws/callcenter.socket {
		without /inphone
	}
	proxy /call_center unix:{$BRUNO_HOME}/data/ws/callcenter.socket {
		without /call_center
	}

	proxy /test http://localhost:{$portof_ws_1} {
		header_upstream -Proxy ""
	}

	rewrite {
		r /Dealer/(.*)
		to /letme/Dealer/{1}
	}
	rewrite {
		r ^(/letme)?/Dealer/Dealer/(.*)
		to /letme/Dealer/{1}
	}
	proxy /letme http://127.0.0.1:8081 {
		without /letme
	}

	#proxy / http://localhost:{$portof_ws_1} http://localhost:{$portof_ws_2} {
	proxy / unix:{$BRUNO_HOME}/data/ws/ws-1.socket unix:{$BRUNO_HOME}/data/ws/ws-1.socket {
		header_upstream -Proxy ""
		header_upstream X-Forwarded-For {remote}
        fail_timeout 9s
        max_fails 1
        try_duration 10s
        policy least_conn
        transparent
	}
}

# AODB-HTTPS
https://0.0.0.0:{$portof_aodb_https} {
	log {$BRUNO_HOME}/data/mai/log/aodb-proxy-https.log
	tls {$BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem {$BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem {
		protocols tls1.0 tls1.2
	}

	proxy /_koord http://localhost:{$portof_mevv} {
		header_upstream -Proxy ""
		without /_koord
	}
	proxy /_macroexpert http://localhost:{$portof_mevv} {
		header_upstream -Proxy ""
		without /_macroexpert
	}

	proxy / http://localhost:{$portof_aodb_local} {
		header_upstream -Proxy ""
	}
}

# AODB
http://0.0.0.0:{$portof_aodb_http} {
	log {$BRUNO_HOME}/data/mai/log/aodb-proxy.log

	proxy /_koord http://localhost:{$portof_mevv} {
		header_upstream -Proxy ""
		without /_koord
	}
	proxy /_macroexpert http://localhost:{$portof_mevv} {
		header_upstream -Proxy ""
		without /_macroexpert
	}

	proxy / unix:{$BRUNO_HOME}/data/ws/aodb.socket {
		header_upstream -Proxy ""
        fail_timeout 1s
        max_fails 3
        try_duration 3s
        policy least_conn
        transparent
	}
}

http://0.0.0.0:4444 {
	log {$BRUNO_HOME}/data/mai/log/grafana-proxy.log

	proxy /metrics/oracle http://localhost:{$portof_oracle_metrics}/metrics {
		header_upstream -Proxy ""
		without /metrics/oracle
	}
	proxy /metrics/ws-1 http://localhost:{$portof_ws_1}/metrics {
		header_upstream -Proxy ""
		without /metrics/ws-1
	}
	proxy /metrics/ws-2 http://localhost:{$portof_ws_2}/metrics {
		header_upstream -Proxy ""
		without /metrics/ws-2
	}
	proxy /metrics/ktny http://localhost:{$portof_ktny}/metrics {
		header_upstream -Proxy ""
		without /metrics/ktny
	}
	proxy /metrics/mabisz http://localhost:{$portof_mabisz}/metrics {
		header_upstream -Proxy ""
		without /metrics/mabisz
	}

	proxy / http://192.168.3.110:3000 {
		header_upstream -Proxy ""
	}
}

# splprn admin
https://0.0.0.0:{$portof_splprn_admin_https} {
	log {$BRUNO_HOME}/data/mai/log/splprn_admin-proxy.log
	tls {$BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.crt.pem {$BRUNO_HOME}/../admin/ssl/lnx-dev-kbe.unosoft.local.key.pem {
		protocols tls1.0 tls1.2
	}

	proxy / http://localhost:{$portof_splprn_admin} {
		header_upstream -Proxy ""
	}
}
`
