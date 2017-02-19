package treeloader_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTreeloader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Treeloader Suite")
}
