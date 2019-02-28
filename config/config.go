// Copyright 2019 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/hashicorp/hcl"
	"github.com/magiconair/properties"
	"github.com/pelletier/go-toml"
	"github.com/pelletier/go-toml/query"
	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

var (
	ErrUnknownType    = errors.New("unknown type")
	ErrNotImplemented = errors.New("not implemented")
)

// Encoder is an interface for encoding a Config.
type Encoder interface {
	// Encode the Config into the Writer.
	Encode(io.Writer, Config) error
}

// Decoder is an interface for decoding from a Reader.
type Decoder interface {
	// Decode the Config from the Reader.
	Decode(io.Reader) (Config, error)
}

// EncoderDecoder can Encode and Decode, too.
type EncoderDecoder interface {
	Encoder
	Decoder
}

// Config is the read, modifyable configuration, based on *toml.Tree.
type Config struct {
	*toml.Tree
	// tbd is the list of keys to be deleted (at Encode).
	tbd []string
}

// New returns a new Config from the given map.
func New(m map[string]interface{}) (Config, error) {
	tt, err := toml.TreeFromMap(m)
	return Config{Tree: tt}, err
}

func (cfg Config) String() string {
	var buf bytes.Buffer
	if _, err := cfg.Tree.WriteTo(&buf); err != nil {
		panic(err)
	}
	return buf.String()
}

// Type represents a configuration file format.
type Type string

var encdecMu sync.RWMutex
var encdec = map[Type]EncoderDecoder{
	"caddy":       caddyEncDec{},
	hclEnc:        defaultEncDec{Type: hclEnc},
	"ini":         iniEncDec{},
	"json":        defaultEncDec{Type: "json"},
	propertiesEnc: defaultEncDec{Type: propertiesEnc},
	"toml":        defaultEncDec{Type: "toml"},
	yamlEnc:       defaultEncDec{Type: yamlEnc},
}

// Register a new EncoderDecoder. Will panic if typ already registered.
func Register(typ Type, ed EncoderDecoder) {
	encdecMu.Lock()
	defer encdecMu.Unlock()
	if _, ok := encdec[typ]; ok {
		panic(typ + " already registered!")
	}
	encdec[typ] = ed
}

// Parser returns a Decoder for the given type.
func Parser(typ Type) Decoder {
	encdecMu.RLock()
	defer encdecMu.RUnlock()
	return encdec[typ]
}

// Dumper returns an Encoder for the given type.
func Dumper(typ Type) Encoder {
	encdecMu.RLock()
	defer encdecMu.RUnlock()
	return encdec[typ]
}

type defaultEncDec struct{ Type string }

const yamlEnc, hclEnc, propertiesEnc = "yaml", "hcl", "properties"

func (ved defaultEncDec) Decode(r io.Reader) (Config, error) {
	m := make(map[string]interface{})
	var b []byte
	var cfg Config
	switch ved.Type {
	case yamlEnc, hclEnc, propertiesEnc:
		var err error
		if b, err = ioutil.ReadAll(r); err != nil {
			return cfg, err
		}
	}

	switch ved.Type {
	case "toml":
		tt, err := toml.LoadReader(r)
		return Config{Tree: tt}, err

	case yamlEnc:
		if err := yaml.Unmarshal(b, m); err != nil {
			return cfg, err
		}
	case "json":
		if err := json.NewDecoder(r).Decode(m); err != nil {
			return cfg, err
		}
	case hclEnc:
		if err := hcl.Unmarshal(b, m); err != nil {
			return cfg, err
		}
	case propertiesEnc:
		props, err := properties.Load(b, properties.UTF8)
		if err != nil {
			return cfg, err
		}
		for _, k := range props.Keys() {
			m[k], _ = props.Get(k)
		}
	default:
		return cfg, errors.Wrap(ErrUnknownType, ved.Type)
	}
	tt, err := toml.TreeFromMap(m)
	return Config{Tree: tt}, err
}
func (ved defaultEncDec) Encode(w io.Writer, cfg Config) error {
	m := cfg.AllSettings()
	switch ved.Type {
	case yamlEnc:
		b, err := yaml.Marshal(m)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	case hclEnc:
		return errors.Wrap(ErrNotImplemented, hclEnc)
	case "toml":
		tt, err := toml.TreeFromMap(m)
		if _, wErr := io.WriteString(w, tt.String()); wErr != nil && err == nil {
			return wErr
		}
		return err
	case propertiesEnc:
		p := properties.NewProperties()
		for k := range m {
			p.Set(k, fmt.Sprintf("%v", cfg.Get([]string{k})))
		}
		_, err := p.Write(w, properties.UTF8)
		return err
	default:
		return errors.Wrap(ErrUnknownType, ved.Type)
	}
}

const keyDelim = "."

func (cfg *Config) Del(key ...string) {
	cfg.tbd = append(cfg.tbd, key...)
}

// AllSettings returns all settings, excluding the deleted keys.
func (cfg Config) AllSettings() map[string]interface{} {
	m := cfg.Tree.ToMap()
	if len(cfg.tbd) == 0 {
		return m
	}
	for _, key := range cfg.tbd {
		delete(m, key)
		key += keyDelim
		for k := range m {
			if strings.HasPrefix(k, key) {
				delete(m, key)
			}
		}
	}
	return m
}

// Get returns the value for the key.
//
// If the key starts with "$", then a JSONPath-like TOML Query is compiled and executed.
func (cfg Config) Get(key []string) interface{} {
	if !strings.HasPrefix(key[0], "$") {
		return cfg.Tree.GetPath(key)
	}
	//tt := toml.TreeFromMap(cfg.AllSettings())
	//qr, err := tt.Query(key)
	qry, err := query.Compile(key[0])
	if qry == nil {
		return err
	}
	qr := qry.Execute(cfg.Tree)
	result := make([]map[string]interface{}, len(qr.Values()))
	for i, v := range qr.Values() {
		switch x := v.(type) {
		case *toml.Tree:
			result[i] = x.ToMap()
		case map[string]interface{}:
			result[i] = x
		default:
			result[i] = map[string]interface{}{"error": errors.Errorf("cannot convert %T to map[string]interface{}: %v", v, v)}
		}
	}
	return result
}

func (cfg Config) Set(key []string, value interface{}) {
	cfg.Tree.SetPath(key, value)
}
