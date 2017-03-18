package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	data := []byte(`{"response": "after"}`)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("listening on port 8080")
}
