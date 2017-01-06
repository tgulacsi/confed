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
		// first, we separate the directives' tokens
		var prev int
		for i, token := range tokens {
			if token.Text != k {
				continue
			}
			if prev == i {
				continue
			}
			// new group starts here
			cb[k] = append(cb[k], caddyParseTokenGroup(tokens[prev:i]))
			prev = i
		}
		cb[k] = append(cb[k], caddyParseTokenGroup(tokens[prev:]))
	}
	return cb
}

// then, convert groups to directives
func caddyParseTokenGroup(tokens []caddyfile.Token) caddyDirective {
	dir := caddyDirective{Main: caddyLine{Name: tokens[0].Text}}
	tokens = tokens[1:]
	ss := make([]string, 0, len(tokens))

	var inParams bool
	var prev int
	var param caddyLine
	for _, token := range tokens {
		if dir.Main.Name == "log" {
			fmt.Printf("%q dir=%#v ss=%q\n", token.Text, dir, ss)
		}
		if !inParams {
			if token.Text != "{" {
				ss = append(ss, token.Text)
				continue
			}
			inParams = true
			dir.Main.Args = ss[:len(ss):len(ss)]
			ss = ss[len(ss):]
			continue
		}
		if token.Text == "}" {
			break
		}
		if prev != token.Line { // new param
			prev = token.Line
			if param.Name != "" {
				dir.Params = append(dir.Params, param)
			}
		}
		if param.Name == "" {
			param.Name = token.Text
			continue
		}
		ss = append(ss, token.Text)
	}

	if !inParams {
		if len(ss) > 0 {
			dir.Main.Args = ss[:len(ss):len(ss)]
		}
	} else if param.Name != "" {
		param.Args = ss
		dir.Params = append(dir.Params, param)
	}
	return dir
}
