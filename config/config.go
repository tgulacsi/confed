package config

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/magiconair/properties"
	"github.com/pelletier/go-toml"
	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
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

// Config is the read, modifyable configuration, based on *viper.Viper.
type Config struct {
	*viper.Viper
}

// Type represents a configuration file format.
type Type string

var encdecMu sync.RWMutex
var encdec = map[Type]EncoderDecoder{
	"caddy":      caddyEncDec{},
	"hcl":        viperEncDec{Type: "hcl"},
	"ini":        iniEncDec{},
	"json":       viperEncDec{Type: "json"},
	"properties": viperEncDec{Type: "properties"},
	"toml":       viperEncDec{Type: "toml"},
	"yaml":       viperEncDec{Type: "yaml"},
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

type viperEncDec struct{ Type string }

func (ved viperEncDec) Decode(r io.Reader) (Config, error) {
	cfg := Config{Viper: viper.New()}
	switch ved.Type {
	case "yaml", "json", "hcl", "toml", "properties":
		cfg.SetConfigType(ved.Type)
		err := cfg.ReadConfig(r)
		return cfg, err
	default:
		return cfg, errors.Wrap(ErrUnknownType, ved.Type)
	}
}
func (ved viperEncDec) Encode(w io.Writer, cfg Config) error {
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
		return json.NewEncoder(w).Encode(m)
	case "hcl":
		return errors.Wrap(ErrNotImplemented, "hcl")
	case "toml":
		_, err := io.WriteString(w, toml.TreeFromMap(m).String())
		return err
	case "properties":
		p := properties.NewProperties()
		for k := range m {
			p.Set(k, cfg.GetString(k))
		}
		_, err := p.Write(w, properties.UTF8)
		return err
	default:
		return errors.Wrap(ErrUnknownType, ved.Type)
	}
}

const keyDelim = "."
