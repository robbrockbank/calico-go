package selector_test

import (
	. "github.com/projectcalico/calico-go/labels/selectors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	It("should tokenize", func() {
		Expect(Tokenize(`a=="b"`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokEq, nil},
			{TokStringLiteral, "b"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`label == "value"`)).To(Equal([]Token{
			{TokLabel, "label"},
			{TokEq, nil},
			{TokStringLiteral, "value"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`a not in "bar" && !has(foo) || b in c`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokNot, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokLabel, "c"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`a  not  in  "bar"  &&  ! has( foo )  ||  b  in  c `)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokNot, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokLabel, "c"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`a notin"bar"&&!has(foo)||b in"c"`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokNot, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokStringLiteral, "c"},
			{TokEof, nil},
		}))
	})
})
