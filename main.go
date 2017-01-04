package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

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
	flagNoCommands := flag.Bool("n", false, "don't read commands from stdin")
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

	if *flagNoCommands {
		return enc.Encode(os.Stdout, cfg)
	}

	var doPrint bool

	// read commands from stdin, and execute them!
	scanner := bufio.NewScanner(os.Stdin)
	var lineNo int
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		i := bytes.IndexByte(line, ' ')
		if i < 0 {
			log.Printf("%d. no command in %q", lineNo, line)
			continue
		}
		cmd, path := string(line[:i]), string(line[i+1:])
		switch cmd {
		case "print":
			doPrint = true
		case "get":
			v := cfg.Get(path)
			b, err := json.Marshal(map[string]interface{}{path: v})
			if err != nil {
				log.Println(err)
			}
			log.Printf("%#v", v)
			os.Stdout.Write(b)
		case "set":
			doPrint = true
			var value string
			if i = strings.IndexByte(path, ' '); i > 0 {
				path, value = path[:i], path[i+1:]
			}
			cfg.Set(path, value)
		case "rm", "del":
			doPrint = true
			cfg.Del(path)
		default:
			log.Printf("%d. unknown command %q", lineNo, cmd)
			continue
		}
	}
	if !doPrint && lineNo >= 1 {
		return nil
	}
	return enc.Encode(os.Stdout, cfg)
}
