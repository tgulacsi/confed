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

// Config is the read, modifyable configuration, based on *toml.TomlTree.
type Config struct {
	*toml.TomlTree
	// tbd is the list of keys to be deleted (at Encode).
	tbd []string
}

// New returns a new Config from the given map.
func New(m map[string]interface{}) Config {
	return Config{TomlTree: toml.TreeFromMap(m)}
}

func (cfg Config) String() string {
	var buf bytes.Buffer
	if _, err := cfg.TomlTree.WriteToToml(&buf, "", ""); err != nil {
		panic(err)
	}
	return buf.String()
}

// Type represents a configuration file format.
type Type string

var encdecMu sync.RWMutex
var encdec = map[Type]EncoderDecoder{
	"caddy":      caddyEncDec{},
	"hcl":        defaultEncDec{Type: "hcl"},
	"ini":        iniEncDec{},
	"json":       defaultEncDec{Type: "json"},
	"properties": defaultEncDec{Type: "properties"},
	"toml":       defaultEncDec{Type: "toml"},
	"yaml":       defaultEncDec{Type: "yaml"},
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

func (ved defaultEncDec) Decode(r io.Reader) (Config, error) {
	m := make(map[string]interface{})
	var b []byte
	var cfg Config
	switch ved.Type {
	case "yaml", "hcl", "properties":
		var err error
		if b, err = ioutil.ReadAll(r); err != nil {
			return cfg, err
		}
	}

	switch ved.Type {
	case "toml":
		tt, err := toml.LoadReader(r)
		return Config{TomlTree: tt}, err

	case "yaml":
		if err := yaml.Unmarshal(b, m); err != nil {
			return cfg, err
		}
	case "json":
		if err := json.NewDecoder(r).Decode(m); err != nil {
			return cfg, err
		}
	case "hcl":
		if err := hcl.Unmarshal(b, m); err != nil {
			return cfg, err
		}
	case "properties":
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
	return Config{TomlTree: toml.TreeFromMap(m)}, nil
}
func (ved defaultEncDec) Encode(w io.Writer, cfg Config) error {
	m := cfg.AllSettings()
	switch ved.Type {
	case "yaml":
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
	case "hcl":
		return errors.Wrap(ErrNotImplemented, "hcl")
	case "toml":
		_, err := io.WriteString(w, toml.TreeFromMap(m).String())
		return err
	case "properties":
		p := properties.NewProperties()
		for k := range m {
			p.Set(k, fmt.Sprintf("%v", cfg.Get(k)))
		}
		_, err := p.Write(w, properties.UTF8)
		return err
	default:
		return errors.Wrap(ErrUnknownType, ved.Type)
	}
}

const keyDelim = "."

func (cfg *Config) Del(key string) {
	cfg.tbd = append(cfg.tbd, key)
}

// AllSettings returns all settings, excluding the deleted keys.
func (cfg Config) AllSettings() map[string]interface{} {
	m := cfg.TomlTree.ToMap()
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
func (cfg Config) Get(key string) interface{} {
	if !strings.HasPrefix(key, "$") {
		return cfg.TomlTree.Get(key)
	}
	//tt := toml.TreeFromMap(cfg.AllSettings())
	//qr, err := tt.Query(key)
	qr, err := cfg.TomlTree.Query(key)
	if qr == nil {
		return err
	}
	result := make([]map[string]interface{}, len(qr.Values()))
	for i, v := range qr.Values() {
		switch x := v.(type) {
		case *toml.TomlTree:
			result[i] = x.ToMap()
		case map[string]interface{}:
			result[i] = x
		default:
			result[i] = map[string]interface{}{"error": errors.Errorf("cannot convert %T to map[string]interface{}: %v", v, v)}
		}
	}
	return result
}
