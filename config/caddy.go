package config

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/mholt/caddy/caddyfile"
	toml "github.com/pelletier/go-toml"
)

type caddyEncDec struct{}

var caddyQuoteKey = strings.NewReplacer(".", "%2E").Replace
var caddyUnquoteKey = strings.NewReplacer("%2E", ".").Replace

func (ed caddyEncDec) Decode(r io.Reader) (Config, error) {
	var buf bytes.Buffer
	blocks, err := caddyfile.Parse("Caddyfile", io.TeeReader(r, &buf), nil)
	if err != nil {
		return Config{}, err
	}

	tt := toml.TreeFromMap(make(map[string]interface{}, len(blocks)*64))
	for _, block := range blocks {
		cb := convertCaddyBlock(block)
		log.Printf("cb=%#v", cb)
		// key: {directive:}
		for _, k := range block.Keys {
			k = caddyQuoteKey(k)
			path := []string{k, "", "", ""}[:1]
			for _, dirs := range cb {
				for _, dir := range dirs {
					path := append(path, dir.Main.Name)
					tt.SetPath(path, toIntfSlice(dir.Main.Args))
					if len(dir.Params) == 0 {
						continue
					}
					params := make(map[string][][]string, len(dir.Params))
					for _, sub := range dir.Params {
						params[sub.Name] = append(params[sub.Name], sub.Args)
					}
					for k, vv := range params {
						path := append(path, k)
						log.Println(path, vv)
						switch len(vv) {
						case 0:
							tt.SetPath(path, "")
						case 1:
							tt.SetPath(path, toIntfSlice(vv[0]))
						default:
							ss := make([][]interface{}, len(vv))
							for i, v := range vv {
								ss[i] = toIntfSlice(v)
							}
							tt.SetPath(path, ss)
						}
					}
				}
			}
		}
	}

	return Config{TomlTree: tt}, nil
}
func (ed caddyEncDec) Encode(w io.Writer, cfg Config) error {
	return ErrNotImplemented
}

type caddyBlock map[string][]caddyDirective

type caddyDirective struct {
	Main   caddyLine
	Params []caddyLine
}
type caddyLine struct {
	Name string
	Args []string
}

func toIntfSlice(ss []string) []interface{} {
	is := make([]interface{}, len(ss))
	for i, s := range ss {
		is[i] = s
	}
	return is
}
func asStringSlice(i interface{}) []string {
	if i == nil {
		return nil
	}
	if ss, ok := i.([]string); ok {
		return ss
	}
	if is, ok := i.([]interface{}); ok {
		ss := make([]string, len(is))
		for i, v := range is {
			if s, ok := v.(string); ok {
				ss[i] = s
			} else {
				ss[i] = fmt.Sprintf("%v", v)
			}
		}
		return ss
	}
	return nil
}
func convertCaddyBlock(block caddyfile.ServerBlock) caddyBlock {
	cb := make(caddyBlock, len(block.Tokens))
	for k, tokens := range block.Tokens {
		var dir caddyDirective
		var lastLine int
		for _, token := range tokens {
			prev := lastLine
			lastLine = token.Line
			if dir.Params == nil {
				if token.Text == "{" {
					dir.Params = make([]caddyLine, 0, 4)
					continue
				}
				if dir.Main.Name == "" {
					dir.Main.Name = token.Text
					continue
				}
				dir.Main.Args = append(dir.Main.Args, token.Text)
				continue
			}
			if token.Text == "}" {
				cb[k] = append(cb[k], dir)
				dir = caddyDirective{}
				continue
			}
			if prev != token.Line {
				dir.Params = append(dir.Params, caddyLine{Name: token.Text})
				continue
			}
			p := &dir.Params[len(dir.Params)-1]
			p.Args = append(p.Args, token.Text)
		}
		if dir.Params != nil {
			cb[k] = append(cb[k], dir)
		}
	}
	return cb
}
