package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
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
	cfg := Config{Viper: viper.New()}
	cfg.SetConfigType("ini")
	for _, section := range f.Sections() {
		sectName := section.Name()
		for _, key := range section.Keys() {
			cfg.Set(fmt.Sprintf("%s%s%s", sectName, keyDelim, key.Name()), key.String())
		}
	}
	return cfg, nil
}
func (ed iniEncDec) Encode(w io.Writer, cfg Config) error {
	f := ini.Empty()
	for _, key := range cfg.AllKeys() {
		var section string
		k := key
		if i := strings.LastIndexByte(key, keyDelim[0]); i >= 0 {
			section, k = key[:i], key[i+1:]
		}
		f.Section(section).Key(k).SetValue(cfg.GetString(key))
	}
	_, err := f.WriteToIndent(w, "  ")
	return errors.Wrap(err, "WriteToIndent")
}
