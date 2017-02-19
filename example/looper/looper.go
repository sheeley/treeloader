package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sheeley/treeloader/example/sub"
)

func main() {
	identifier := rand.Int() % 10
	for {
		sub.Example()
		fmt.Printf("%d: %s\n", identifier, time.Now())
		time.Sleep(2 * time.Second)
	}
}
