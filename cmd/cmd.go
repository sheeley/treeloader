package main

import (
	"flag"
	"log"
	"os"

	"github.com/sheeley/treeloader"
)

var (
	verbose    bool
	extensions = treeloader.StringSet{}
)

func main() {
	flag.BoolVar(&verbose, "v", true, "Enable verbose messaging")
	flag.BoolVar(&verbose, "verbose", true, "Enable verbose messaging")
	flag.Var(&extensions, "e", "Comma delimited list of file extensions to watch")
	flag.Var(&extensions, "extensions", "Comma delimited list of file extensions to watch")
	flag.Parse()
	cmd := flag.Arg(0)

	cmd = "example/http/main.go"

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
