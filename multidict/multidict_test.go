package multidict_test

import (
	. "github.com/projectcalico/calico-go/multidict"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StringToString", func() {
	var s2s StringToString
	BeforeEach(func() {
		s2s = NewStringToString()
		s2s.Put("a", "b")
		s2s.Put("a", "c")
		s2s.Put("b", "d")
	})
	It("should contain items that are added", func() {
		Expect(s2s.Contains("a", "b")).To(BeTrue())
		Expect(s2s.Contains("a", "c")).To(BeTrue())
		Expect(s2s.Contains("b", "d")).To(BeTrue())
		Expect(s2s.ContainsKey("a")).To(BeTrue())
		Expect(s2s.ContainsKey("b")).To(BeTrue())
	})
	It("should not contain items with different key", func() {
		Expect(s2s.Contains("b", "b")).To(BeFalse())
		Expect(s2s.ContainsKey("c")).To(BeFalse())
	})
	It("should not contain items with different value", func() {
		Expect(s2s.Contains("a", "a")).To(BeFalse())
	})
	It("should not contain discarded item", func() {
		s2s.Discard("a", "b")
		Expect(s2s.Contains("a", "b")).To(BeFalse())
		Expect(s2s.ContainsKey("a")).To(BeTrue())
		s2s.Discard("a", "c")
		Expect(s2s.ContainsKey("a")).To(BeFalse())
	})
	It("should ignore discard of unknown item", func() {
		s2s.Discard("a", "c")
		s2s.Discard("e", "f")
		Expect(s2s.Contains("a", "b")).To(BeTrue())
	})
	It("should have idempotent insett", func() {
		s2s.Put("a", "b")
		Expect(s2s.Contains("a", "b")).To(BeTrue())
	})
	It("should have idempotent discard", func() {
		s2s.Discard("a", "b")
		s2s.Discard("a", "b")
		Expect(s2s.Contains("a", "b")).To(BeFalse())
	})
})
