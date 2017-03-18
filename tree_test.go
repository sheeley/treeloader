package treeloader_test

import (
	"fmt"

	"io/ioutil"

	"net/http"

	"strings"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sheeley/treeloader"
)

func copy(src string, dst string) {
	// Read all content of src to data
	data, err := ioutil.ReadFile(src)
	if err != nil {
		panic(err)
	}
	// Write data to dst
	err = ioutil.WriteFile(dst, data, 0644)
	if err != nil {
		panic(err)
	}
}

var _ = Describe("Tree", func() {
	It("should properly calculate dependencies", func() {
		deps, err := treeloader.DirsToWatch("example/main.go")
		Expect(err).To(BeNil())
		Expect(deps).To(BeEquivalentTo(map[string]int{
			"github.com/sheeley/treeloader/example":             1,
			"github.com/sheeley/treeloader/example/sub2/subsub": 1,
			"github.com/sheeley/treeloader/example/sub":         1,
			"github.com/sheeley/treeloader/example/sub2":        1,
		}))
	})

	It("should restart a http server", func(done Done) {
		// tmpdir, err := ioutil.TempDir("", "httptest")
		// if err != nil {
		// 	panic(err)
		// }
		goFile := "example/http/main.go"
		copy("example/http/source/before.go", goFile)
		reloaded := make(chan string, 1)
		runner, err := treeloader.New(&treeloader.Options{
			Reloaded: reloaded,
			CmdPath:  "example/http/main.go",
			Verbose:  true,
		})
		Expect(err).To(BeNil())
		defer runner.Close()

		// start, check response for initial value
		go func() {
			if err := runner.Run(); err != nil {
				panic(err)
			}
		}()
		Expect(<-reloaded).To(BeEquivalentTo(""))
		resp, err := http.Get("http://127.0.0.1:8080")
		if err != nil {
			fmt.Println(err)
		}
		Expect(err).To(BeNil())
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(body).To(BeEquivalentTo([]byte(`{"response": "before"}`)))

		time.Sleep(1 * time.Second)
		copy("example/http/source/after.go", goFile)
		changedFile := <-reloaded
		Expect(strings.HasSuffix(changedFile, goFile)).To(BeTrue())
		resp, err = http.Get("http://127.0.0.1:8080")
		Expect(err).To(BeNil())
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(body).To(BeEquivalentTo([]byte(`{"response": "after"}`)))

		close(done)
	}, 3)

	// It("should restart a http server when json changes", func(done Done) {
	// 	configFile := "example/http/config.json"
	// 	ioutil.WriteFile(configFile, beforebyte, 600)
	// 	reloaded := make(chan string, 1)
	// 	opts := &Options{
	// 		Reloaded:   reloaded,
	// 		CmdPath:    "example/http/main.go",
	// 		Extensions: StringSet{"json": 1},
	// 	}
	// 	runner, err := New(opts)
	// 	Expect(err).To(BeNil())
	// 	defer runner.Close()

	// 	// start, check response for initial value
	// 	go runner.Run()
	// 	Expect(<-reloaded).To(BeEquivalentTo(""))
	// 	resp, err := http.Get("http://127.0.0.1:8080")
	// 	Expect(err).To(BeNil())
	// 	defer resp.Body.Close()
	// 	body, err := ioutil.ReadAll(resp.Body)
	// 	Expect(err).To(BeNil())
	// 	Expect(body).To(BeEquivalentTo(beforebyte))

	// 	time.Sleep(1 * time.Second)
	// 	// change config.json, check http response
	// 	ioutil.WriteFile(configFile, afterbyte, 600)
	// 	changedFile := <-reloaded
	// 	Expect(strings.HasSuffix(changedFile, configFile)).To(BeTrue())
	// 	resp, err = http.Get("http://127.0.0.1:8080")
	// 	Expect(err).To(BeNil())
	// 	defer resp.Body.Close()
	// 	body, err = ioutil.ReadAll(resp.Body)
	// 	Expect(err).To(BeNil())
	// 	Expect(body).To(BeEquivalentTo(afterbyte))

	// 	close(done)
	// }, 3)
})
