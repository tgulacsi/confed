package main

import (
	"flag"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/tgulacsi/confed/config"
)

func main() {
	if err := Main(); err != nil {
		log.Fatalln(err)
	}
}

func Main() error {
	flagTypeIn := flag.String("f", "json", "Type of input")
	flagTypeOut := flag.String("t", "json", "Type of output")
	flag.Parse()
	fn := flag.Arg(0)
	inp, err := os.Open(fn)
	if err != nil {
		return errors.Wrap(err, fn)
	}
	defer inp.Close()

	dec := config.Parser(config.Type(*flagTypeIn))
	enc := config.Dumper(config.Type(*flagTypeOut))
	log.Printf("Input: %#v, Output: %#v", dec, enc)

	defer os.Stdout.Close()
	cfg, err := dec.Decode(inp)
	inp.Close()
	if err != nil {
		return errors.Wrap(err, "decode "+inp.Name())
	}

	// TODO(tgulacsi): read commands from stdin, and execute them!

	return enc.Encode(os.Stdout, cfg)
}
