package labels_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLabels(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Labels Suite")
}
