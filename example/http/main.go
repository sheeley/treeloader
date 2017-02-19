package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := path.Dir(ex)
	data, err := ioutil.ReadFile(path.Join(exPath, "config.json"))
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
