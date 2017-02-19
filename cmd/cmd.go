package main

import (
	"flag"
	"log"
	"os"

	"github.com/sheeley/treeloader"
)

var (
	cmd        string
	verbose    bool
	extensions = treeloader.DefaultExtensions
)

func main() {
	flag.BoolVar(&verbose, "v", true, "Enable verbose messaging")
	flag.BoolVar(&verbose, "verbose", true, "Enable verbose messaging")
	flag.Var(&extensions, "e", "Comma delimited list of file extensions to watch (defaults to .go)")
	flag.Var(&extensions, "extensions", "Comma delimited list of file extensions to watch (defaults to .go)")
	flag.Parse()
	cmd = flag.Arg(0)

	cmd = "example/looper/looper.go"

	if cmd == "" {
		flag.Usage()
		os.Exit(1)
	}

	loader, err := treeloader.New(&treeloader.Options{
		CmdPath:    cmd,
		Verbose:    verbose,
		Extensions: extensions,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(loader.Run())
}
