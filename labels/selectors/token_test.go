package selector_test

import (
	. "github.com/projectcalico/calico-go/labels/selectors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	It("should tokenize", func() {
		Expect(Tokenize(`label == "value"`)).To(Equal([]Token{
			{TokLabel, "label"},
			{TokEq, nil},
			{TokStringLiteral, "value"},
		}))
		Expect(Tokenize(`a not in "bar" && has(foo) || b in c`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokLabel, "c"},
		}))
	})
})
