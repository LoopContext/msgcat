package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	sub := os.Args[1]
	args := os.Args[2:]
	var err error
	switch sub {
	case "extract":
		cfg, e := parseExtractFlags(args)
		if e != nil {
			err = e
			break
		}
		err = runExtract(cfg)
	case "merge":
		cfg, e := parseMergeFlags(args)
		if e != nil {
			err = e
			break
		}
		err = runMerge(cfg)
	case "help", "-h", "--help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "msgcat: unknown subcommand %q\n", sub)
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "msgcat: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `msgcat - message catalog CLI for extract and merge workflow

usage: msgcat <command> [options] [paths]

commands:
  extract    Discover message keys from Go code; optionally sync into source YAML.
  merge      Produce translate.<lang>.yaml files from a source message file.

Use 'msgcat extract -h' or 'msgcat merge -h' for command-specific flags.
`)
}
