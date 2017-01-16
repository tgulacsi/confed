package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
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
			if bytes.Equal(line, []byte("print")) {
				doPrint = true
				continue
			}
			if bytes.Equal(line, []byte("dump")) {
				m := cfg.AllSettings()
				log.Printf("DUMP %d", len(m))
				keys := make([]string, 0, len(m))
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("%s: %+v\n", k, cfg.Get(k))
				}
				return nil
			}
			log.Printf("%d. no command in %q", lineNo, line)
			continue
		}
		cmd, path := string(line[:i]), string(line[i+1:])
		defer os.Stdout.Close()
		switch cmd {
		case "print":
			doPrint = true
		case "get":
			v := cfg.Get(path)
			log.Printf("GET %q: %T", path, v)
			res := make(map[string]interface{})
			switch x := v.(type) {
			case error:
				log.Fatalf("ERROR: %#v", err)
			case map[string]interface{}:
				res = x
			case []map[string]interface{}:
				for i, m := range x {
					res[strconv.Itoa(i)] = m
				}
			default:
				res[path] = v
			}
			if err := enc.Encode(os.Stdout, config.New(res)); err != nil {
				return err
			}
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
