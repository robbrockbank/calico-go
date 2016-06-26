package multidict_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMultidict(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multidict Suite")
}
