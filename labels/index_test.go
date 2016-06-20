package labels_test

import (
	. "github.com/projectcalico/calico-go/labels"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/calico-go/labels/selectors"
)

type update struct {
	op      string
	labelId string
	selId   string
}

var _ = Describe("Index", func() {
	var (
		updates []update
		idx     Index
	)

	onMatchStart := func(selId, labelId string) {
		updates = append(updates,
			update{op: "start",
				labelId: labelId,
				selId:   selId})
	}
	onMatchStop := func(selId, labelId string) {
		updates = append(updates,
			update{op: "stop",
				labelId: labelId,
				selId:   selId})
	}

	BeforeEach(func() {
		updates = make([]update, 0)
		idx = NewIndex(onMatchStart, onMatchStop)
	})

	a_eq_a1, _ := selector.Parse(`a=="a1"`)
	a_eq_b, _ := selector.Parse(`a=="b"`)
	c_eq_d, _ := selector.Parse(`c=="d"`)

	Context("with empty index", func() {
		It("should do nothing when adding labels", func() {
			idx.UpdateLabels("foo", map[string]string{"a": "b"})
			idx.UpdateLabels("bar", map[string]string{})
			Expect(updates).To(BeEmpty())
		})
		It("should do nothing when adding selectors", func() {
			idx.UpdateSelector("foo", a_eq_a1)
			idx.UpdateSelector("bar", a_eq_a1)
			Expect(updates).To(BeEmpty())
		})
	})

	Context("with one set of labels added", func() {
		BeforeEach(func() {
			idx.UpdateLabels("l1",
				map[string]string{"a": "b", "c": "d"})
		})

		It("should ignore non-matching selectors", func() {
			By("ignoring selector add")
			idx.UpdateSelector("e1", a_eq_a1)
			By("ignoring selector delete")
			idx.DeleteSelector("e1")
			Expect(updates).To(BeEmpty())
		})

		It("should fire correct events for matching selector", func() {
			By("firing start event on addition")
			idx.UpdateSelector("e1", a_eq_b)
			Expect(updates).To(Equal([]update{update{
				"start", "l1", "e1",
			}}))
			updates = updates[:0]
			By("ignoring idempotent update")
			idx.UpdateSelector("e1", a_eq_b)
			Expect(updates).To(BeEmpty())
			By("ignoring update to also-matching selector")
			idx.UpdateSelector("e1", c_eq_d)
			Expect(updates).To(BeEmpty())
			By("firing stop event on deletion")
			idx.DeleteSelector("e1")
			Expect(updates).To(Equal([]update{update{
				"stop", "l1", "e1",
			}}))
		})

		It("should handle multiple matches", func() {
			By("firing events for both")
			idx.UpdateSelector("e1", a_eq_b)
			idx.UpdateSelector("e2", c_eq_d)
			Expect(updates).To(Equal([]update{
				update{"start", "l1", "e1"},
				update{"start", "l1", "e2"},
			}))
			updates = updates[:0]

			By("firing stop for update to non-matching selector")
			idx.UpdateSelector("e2", a_eq_a1)
			Expect(updates).To(Equal([]update{
				update{"stop", "l1", "e2"},
			}))
			updates = updates[:0]

			By("firing stop when selector deleted")
			idx.DeleteSelector("e1")
			Expect(updates).To(Equal([]update{
				update{"stop", "l1", "e1"},
			}))
		})
	})

	Context("with one selector added", func() {
		BeforeEach(func() {
			idx.UpdateSelector("e1", a_eq_a1)
		})

		It("should ignore non-matching labels", func() {
			idx.UpdateLabels("l1", map[string]string{"a": "b"})
			Expect(updates).To(BeEmpty())
		})
		It("should fire correct events for match", func() {
			By("firing for add")
			idx.UpdateLabels("l1", map[string]string{"a": "a1"})
			Expect(updates).To(Equal([]update{update{
				"start", "l1", "e1",
			}}))
			updates = updates[:0]
			By("ignoring idempotent add")
			idx.UpdateLabels("l1", map[string]string{"a": "a1"})
			Expect(updates).To(BeEmpty())
			By("ignoring update to also-matching labels")
			idx.UpdateLabels("l1",
				map[string]string{"a": "a1", "b": "c"})
			Expect(updates).To(BeEmpty())
			By("firing stop on delete")
			idx.DeleteLabels("l1")
			Expect(updates).To(Equal([]update{update{
				"stop", "l1", "e1",
			}}))
		})
		It("should handle multiple matches", func() {
			By("firing events for both")
			idx.UpdateLabels("l1", map[string]string{"a": "a1"})
			idx.UpdateLabels("l2",
				map[string]string{"a": "a1", "b": "b1"})
			Expect(updates).To(Equal([]update{
				update{"start", "l1", "e1"},
				update{"start", "l2", "e1"},
			}))
			updates = updates[:0]

			By("handling updates to non-matching labels")
			idx.UpdateLabels("l1", map[string]string{"a": "a2"})
			Expect(updates).To(Equal([]update{
				update{"stop", "l1", "e1"},
			}))
			updates = updates[:0]

			By("handling removal of selector")
			idx.DeleteSelector("e1")
			Expect(updates).To(Equal([]update{
				update{"stop", "l2", "e1"},
			}))
		})
	})
})