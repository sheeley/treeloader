package treeloader_test

import (
	. "github.com/sheeley/treeloader"

	"io/ioutil"

	"net/http"

	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	beforebyte = []byte(`{"response": "before"}`)
	afterbyte  = []byte(`{"response": "after"}`)
)

var _ = Describe("Tree", func() {
	It("should properly calculate dependencies", func() {
		deps, err := DirsToWatch("example/main.go")
		Expect(err).To(BeNil())
		Expect(deps).To(BeEquivalentTo(map[string]int{
			"github.com/sheeley/treeloader/example":             1,
			"github.com/sheeley/treeloader/example/sub2/subsub": 1,
			"github.com/sheeley/treeloader/example/sub":         1,
			"github.com/sheeley/treeloader/example/sub2":        1,
		}))
	})

	It("should restart a http server", func(done Done) {
		configFile := "example/http/config.json"
		ioutil.WriteFile(configFile, beforebyte, 600)
		reloaded := make(chan string, 1)
		opts := &Options{
			Reloaded:   reloaded,
			CmdPath:    "example/http/main.go",
			Extensions: StringSet{"json": 1},
		}
		runner, err := New(opts)
		Expect(err).To(BeNil())
		defer runner.Close()

		// start, check response for initial value
		go runner.Run()
		Expect(<-reloaded).To(BeEquivalentTo(""))
		resp, err := http.Get("http://localhost:8080")
		Expect(err).To(BeNil())
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(body).To(BeEquivalentTo(beforebyte))

		// change config.json, check http response
		ioutil.WriteFile(configFile, afterbyte, 600)
		changedFile := <-reloaded
		Expect(strings.HasSuffix(changedFile, configFile)).To(BeTrue())
		resp, err = http.Get("http://localhost:8080")
		Expect(err).To(BeNil())
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(body).To(BeEquivalentTo(afterbyte))

		close(done)
	}, 2)
})
