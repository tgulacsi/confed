package config

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mholt/caddy/caddyfile"
)

func TestCaddyParse(t *testing.T) {
	var ed caddyEncDec
	cfg, err := ed.Decode(strings.NewReader(caddyTest1))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(cfg.String())
}
func TestCaddyBlocks(t *testing.T) {
	os.Setenv("BRUNO_HOME", "{BRUNO_HOME}")
	blocks, err := caddyfile.Parse("Caddyfile", strings.NewReader(caddyTest1), nil)
	if err != nil {
		t.Fatal(err)
	}
	cb := convertCaddyBlock(blocks[0])
	t.Logf("cb:%#v", cb)

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
	} {
		if got := cb[key]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %s, wanted %s.", key, got, want)
		}
	}

	proxy := cb["proxy"]
	if len(proxy) != 6 {
		t.Errorf("Got %d proxy directives, wanted 6.", len(proxy))
	}
	t.Logf("proxy=%#v", proxy)
}

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
