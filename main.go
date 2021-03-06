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
	flagSep := flag.String("S", "/", "path separator")
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
					fmt.Printf("%s: %+v\n", k, cfg.Get(strings.Split(k, *flagSep)))
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
			v := cfg.Get(strings.Split(path, *flagSep))
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
			cfg, err := config.New(res)
			if err != nil {
				return err
			}
			if err = enc.Encode(os.Stdout, cfg); err != nil {
				return err
			}
		case "set":
			doPrint = true
			var value string
			if i = strings.IndexByte(path, ' '); i > 0 {
				path, value = path[:i], path[i+1:]
			}
			cfg.Set(strings.Split(path, *flagSep), value)
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
