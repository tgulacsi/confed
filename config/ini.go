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
	"fmt"
	"io"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	ini "gopkg.in/ini.v1"
)

type iniEncDec struct{}

func (ed iniEncDec) Decode(r io.Reader) (Config, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return Config{}, err
	}
	f, err := ini.InsensitiveLoad(b)
	if err != nil {
		return Config{}, errors.Wrap(err, string(b))
	}
	tt, err := toml.TreeFromMap(make(map[string]interface{}))
	if err != nil {
		return Config{Tree: tt}, err
	}
	for _, section := range f.Sections() {
		path := []string{section.Name(), ""}[:1]
		for _, key := range section.Keys() {
			tt.SetPath(append(path, key.Name()), key.String())
		}
	}
	return Config{Tree: tt}, nil
}
func (ed iniEncDec) Encode(w io.Writer, cfg Config) error {
	f := ini.Empty()
	for _, key := range cfg.Keys() {
		var section string
		k := key
		if i := strings.LastIndexByte(key, keyDelim[0]); i >= 0 {
			section, k = key[:i], key[i+1:]
		}
		f.Section(section).Key(k).SetValue(fmt.Sprintf("%v", cfg.Get([]string{key})))
	}
	_, err := f.WriteToIndent(w, "  ")
	return errors.Wrap(err, "WriteToIndent")
}
