package selector_test

import (
	. "github.com/projectcalico/calico-go/labels/selectors"

	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type selectorTest struct {
	sel           string
	expMatches    []map[string]string
	expNonMatches []map[string]string
}

var selectorTests = []selectorTest{
	{`a == "b"`,
		[]map[string]string{
			{"a": "b"},
			{"a": "b", "c": "d"}},
		[]map[string]string{
			{},
			{"a": "c"},
			{"c": "d"},
		}},
	{`a == "b" && c == "d"`,
		[]map[string]string{
			{"a": "b", "c": "d"}},
		[]map[string]string{
			{},
			{"a": "b", "c": "e"},
			{"a": "c", "c": "d"},
			{"c": "d"},
			{"a": "b"},
		}},
	{`a == "b" || c == "d"`,
		[]map[string]string{
			{"a": "b", "c": "d"},
			{"a": "b"},
			{"c": "d"}},
		[]map[string]string{
			{},
			{"a": "e", "c": "e"},
			{"c": "e"},
			{"a": "e"},
		}},

	// Tests copied from Python version.
	{`a == 'a'`, []map[string]string{{"a": "a"}}, []map[string]string{}},
	{`a == "a"`, []map[string]string{{"a": "a"}}, []map[string]string{}},
	{`a != "b"`, []map[string]string{{"a": "a"}}, []map[string]string{}},
	{`a != "a"`, []map[string]string{{}}, []map[string]string{}},
	// TODO  set literals
	//{`a in {"a"}`, []map[string]string{{"a": "a"}},  []map[string]string{}},
	//{`!a in {"a"}`, []map[string]string{{"a": "b"}},  []map[string]string{}},
	//{`a in {"a", "b"}`, []map[string]string{{"a": "a"}},  []map[string]string{}},
	//{`a in {"a", "b"}`, []map[string]string{{"a": "b"}},  []map[string]string{}},
	//{`a not in {"d", "e"}`, []map[string]string{{"a": "a"}},  []map[string]string{}},
	{`has(a)`, []map[string]string{{"a": "b"}}, []map[string]string{}},
	{`!has(a)`, []map[string]string{{"b": "b"}}, []map[string]string{}},
	{``, []map[string]string{{}}, []map[string]string{}},
	{` `, []map[string]string{{}}, []map[string]string{}},
	{``, []map[string]string{{"a": "b"}}, []map[string]string{}},
	{`all()`, []map[string]string{{}}, []map[string]string{}},
	{` all()`, []map[string]string{{}}, []map[string]string{}},
	{` all()`, []map[string]string{{"a": "b"}}, []map[string]string{}},

	{`a == 'a'`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`a == 'a'`, []map[string]string{}, []map[string]string{{}}},
	{`a != "a"`, []map[string]string{}, []map[string]string{{"a": "a"}}},
	// TODO  set literals
	//{`a in {"a"}`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	//{`a not in {"a"}`, []map[string]string{}, []map[string]string{{"a": "a"}}},
	//{`a in {"a", "b"}`, []map[string]string{}, []map[string]string{{"a": "c"}}},
	{`has(b)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`!!has(b)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`! has(a)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`!has(a)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`!!! has(a)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`!!!has(a)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`!! ! has(a)`, []map[string]string{}, []map[string]string{{"a": "b"}}},
	{`! !!has(a)`, []map[string]string{}, []map[string]string{{"a": "b"}}},

	// Boolean expressions...
	{`a == 'a1' && b == 'b1'`, []map[string]string{{"a": "a1", "b": "b1"}}, []map[string]string{}},
	{`a == 'a1' && b != 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
	{`a != 'a1' && b == 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
	{`a != 'a1' && b != 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
	{`a != 'a1' && !b == 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
	{`!a == 'a1' && b == 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
	{`has(a) && !has(b)`, []map[string]string{{"a": "a"}}, []map[string]string{}},
	{`!has(b) && has(a)`, []map[string]string{{"a": "a"}}, []map[string]string{}},
	{`!(!has(a) || has(b))`, []map[string]string{{"a": "a"}}, []map[string]string{}},
	{`!(has(b) || !has(a))`, []map[string]string{{"a": "a"}}, []map[string]string{}},

	{`a == 'a1' || b == 'b1'`, []map[string]string{{"a": "a1", "b": "b1"}}, []map[string]string{}},
	{`a == 'a1' || b != 'b1'`, []map[string]string{{"a": "a1", "b": "b1"}}, []map[string]string{}},
	{`a != 'a1' || b == 'b1'`, []map[string]string{{"a": "a1", "b": "b1"}}, []map[string]string{}},
	{`a != 'a1' || b != 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
	{`! a == 'a1' || ! b == 'b1'`, []map[string]string{}, []map[string]string{{"a": "a1", "b": "b1"}}},
}

var badSelectors = []string{
	"b == b",         // label == label
	"'b1' == b",      // literal on lhs
	"b",              // bare label
	"a b",            // Garbage
	"!",              // Garbage
	`foo == "bar" &`, // Garbage
	`foo == "bar" |`, // Garbage
	`"FOO`,           // Unterminated string
	`"FOO'`,          // Unterminated string
	`"FOO`,           // Unterminated string
	`'FOO`,           // Unterminated string
	`(`,              // Unterminated paren
	`)`,              // Unterminated paren
	`()`,             // Unterminated paren
	`%`,              // Unexpected char
}

var canonicalisationTests = []struct{
	input string
	expected string
	expectedUid string
}{
	{"", "all()", "yAKsl-CNoToGJvI4pNl6xXkWbnkbEnlK7IRXBA"},
	{" all() ", "all()", "yAKsl-CNoToGJvI4pNl6xXkWbnkbEnlK7IRXBA"},
	{" (all() )", "all()", "yAKsl-CNoToGJvI4pNl6xXkWbnkbEnlK7IRXBA"},
	{`! (has( b)||! has(a ))`, "!(has(b) || !has(a))", "hSyHDjavfOProPgh2ui1yqeCS31caoii1SGzZw"},
	{`! (a == "b"&&! c != "d")`, `!(a == "b" && !c != "d")`, "Vrj0UGjYYduG4mcP4DKl6qrmTxJhacqDcYiWqg"},
}


var _ = Describe("Parser", func() {
	for _, test := range selectorTests {
		Context(fmt.Sprintf("selector %#v", test.sel), func(test selectorTest) func() {
			return func() {
				var sel Selector
				var err error
				BeforeEach(func() {
					sel, err = Parse(test.sel)
					Expect(err).To(BeNil())
				})
				It("should match", func() {
					for _, labels := range test.expMatches {
						By(fmt.Sprintf("%#v matching %v", test.sel, labels))
						Expect(sel.Evaluate(labels)).To(BeTrue())
					}
				})
				It("should not match", func() {
					for _, labels := range test.expNonMatches {
						By(fmt.Sprintf("%#v not matching %v", test.sel, labels))
						Expect(sel.Evaluate(labels)).To(BeFalse())
					}
				})
				It("should match after canonicalising", func() {
					for _, labels := range test.expMatches {
						sel2, err := Parse(sel.String())
						Expect(err).To(BeNil())
						By(fmt.Sprintf("%#v matching %v", test.sel, labels))
						Expect(sel2.Evaluate(labels)).To(BeTrue())
					}
				})
				It("should not match after canonicalising", func() {
					for _, labels := range test.expNonMatches {
						sel2, err := Parse(sel.String())
						Expect(err).To(BeNil())
						By(fmt.Sprintf("%#v not matching %v", test.sel, labels))
						Expect(sel2.Evaluate(labels)).To(BeFalse())
					}
				})
			}
		}(test))
	}

	It("Should reject bad selectors", func() {
		for _, sel := range badSelectors {
			By(fmt.Sprint("Rejecting ", sel))
			_, err := Parse(sel)
			Expect(err).ToNot(BeNil())
		}
	})

	It("should canonicalise properly", func() {
		seenUids := make(map[string]string)
		for _, test := range canonicalisationTests {
			sel, err := Parse(test.input)
			Expect(err).To(BeNil())
			canon := sel.String()
			Expect(canon).To(Equal(test.expected))

			roundTripped, err := Parse(canon)
			Expect(err).To(BeNil())
			Expect(roundTripped.String()).To(Equal(canon))
			uid := sel.UniqueId()
			Expect(roundTripped.UniqueId()).To(Equal(uid))

			if otherCanon := seenUids[uid]; otherCanon != "" {
				Expect(otherCanon).To(Equal(canon))
			} else {
				seenUids[uid] = canon
			}
		}
	})

	It("should calculate the correct UID", func() {
		for _, test := range canonicalisationTests {
			sel, err := Parse(test.input)
			Expect(err).To(BeNil())
			Expect(sel.UniqueId()).To(Equal(test.expectedUid),
				"incorrect UID for " + test.input)
		}
	})
})
