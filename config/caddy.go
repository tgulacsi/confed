package config

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
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
					var written bool
					if len(dir.Main.Args) != 0 {
						tt.SetPath(append(path, "args"), toIntfSlice(dir.Main.Args))
						written = true
					}
					if len(dir.Params) == 0 {
						if !written {
							tt.SetPath(path, "")
						}
						continue
					}
					for _, vv := range dir.Params {
						written = true
						path := append(path, vv.Name)
						if len(vv.Args) == 0 {
							tt.SetPath(path, "")
						} else {
							tt.SetPath(path, toIntfSlice(vv.Args))
						}
					}
					if !written {
						tt.SetPath(path, "")
					}
				}
			}
		}
	}

	return Config{TomlTree: tt}, nil
}

func (ed caddyEncDec) Encode(w io.Writer, cfg Config) error {
	m0 := cfg.TomlTree.ToMap()
	for rK, rV := range m0 {
		m1, ok := rV.(map[string]interface{})
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%s {\n", caddyUnquoteKey(rK))
		for k, v := range m1 {
			m2, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			fmt.Fprintf(w, "\t%s ", k)
			args := asStringSlice(m2["args"])
			var minus int
			if len(args) != 0 {
				minus = 1
				quoteSlice(args, " ")
				fmt.Fprintf(w, "%s", strings.Join(args, " "))
			}
			if len(m2) == minus {
				fmt.Fprintf(w, "\n")
				continue
			}
			fmt.Fprintf(w, " {\n")
			for kk, vv := range m2 {
				if kk == "args" {
					continue
				}
				fmt.Fprintf(w, "\t\t%s", kk)
				args = asStringSlice(vv)
				if len(args) == 0 {
					continue
				}
				quoteSlice(args, " ")
				for i, s := range args {
					sep := " "
					if i == 0 {
						sep = "\t"
					}
					fmt.Fprintf(w, "%s%s", sep, s)
				}
				fmt.Fprintf(w, "\n")
			}
			fmt.Fprintf(w, "\t}\n")
		}
		fmt.Fprintf(w, "}\n")
	}
	return nil
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

	if dir.Main.Name == "rewrite" {
		fmt.Printf("%q: %#v", dir.Main.Name, tokens)
	}
	var inParams bool
	var prev int
	var param caddyLine
	for _, token := range tokens {
		if dir.Main.Name == "rewrite" {
			fmt.Printf("%q dir=%#v iP=%t ss=%q\n", token.Text, dir, inParams, ss)
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
			param.Args = ss[:len(ss):len(ss)]
			ss = ss[len(ss):]
			if param.Name != "" {
				dir.Params = append(dir.Params, param)
			}
			param = caddyLine{}
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

func quoteSlice(ss []string, sep string) {
	for i, s := range ss {
		if strings.Contains(s, sep) {
			ss[i] = strconv.Quote(s)
		}
	}
}
