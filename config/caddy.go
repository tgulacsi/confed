package config

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/mholt/caddy/caddyfile"
	"github.com/pkg/errors"
)

var vedJSON = viperEncDec{Type: "json"}

type caddyEncDec struct{}

func (ed caddyEncDec) Decode(r io.Reader) (Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, err
	}
	if b, err = caddyfile.ToJSON(b); err != nil {
		return Config{}, errors.Wrap(err, "ToJSON")
	}
	cfg, err := vedJSON.Decode(io.MultiReader(
		strings.NewReader(`{"service_blocks":`),
		bytes.NewReader(b),
		strings.NewReader(`}`),
	))
	if err != nil {
		return cfg, errors.Wrap(err, string(b))
	}
	return cfg, nil
}
func (ed caddyEncDec) Encode(w io.Writer, cfg Config) error {
	var buf bytes.Buffer
	if err := vedJSON.Encode(&buf, cfg); err != nil {
		return err
	}
	b := bytes.TrimSpace(buf.Bytes())
	if bytes.HasPrefix(b, []byte(`{"service_blocks":[`)) && bytes.HasSuffix(b, []byte("]}")) {
		b = b[18 : len(b)-1]
	}
	c, err := caddyfile.FromJSON(b)
	if err != nil {
		return errors.Wrap(err, "FromJSON:\n"+string(b))
	}
	_, err = w.Write(c)
	return err
}
