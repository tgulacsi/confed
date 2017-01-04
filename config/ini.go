package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	ini "gopkg.in/ini.v1"
)

type iniEncDec struct{}

func (ed iniEncDec) Decode(r io.Reader) (Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, err
	}
	f, err := ini.InsensitiveLoad(b)
	if err != nil {
		return Config{}, errors.Wrap(err, string(b))
	}
	tt := toml.TreeFromMap(make(map[string]interface{}))
	for _, section := range f.Sections() {
		path := []string{section.Name(), ""}[:1]
		for _, key := range section.Keys() {
			tt.SetPath(append(path, key.Name()), key.String())
		}
	}
	return Config{TomlTree: tt}, nil
}
func (ed iniEncDec) Encode(w io.Writer, cfg Config) error {
	f := ini.Empty()
	for _, key := range cfg.Keys() {
		var section string
		k := key
		if i := strings.LastIndexByte(key, keyDelim[0]); i >= 0 {
			section, k = key[:i], key[i+1:]
		}
		f.Section(section).Key(k).SetValue(fmt.Sprintf("%v", cfg.Get(key)))
	}
	_, err := f.WriteToIndent(w, "  ")
	return errors.Wrap(err, "WriteToIndent")
}
