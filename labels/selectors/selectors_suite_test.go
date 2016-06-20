package selector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSelectors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selectors Suite")
}
