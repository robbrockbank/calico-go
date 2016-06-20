package selector

import (
	"github.com/projectcalico/calico-go/labels"
	"testing"
)

var selectorTests = []struct {
	sel           string
	expMatches    []labels.Labels
	expNonMatches []labels.Labels
}{
	{`a == "b"`,
		[]labels.Labels{
			{"a": "b"},
			{"a": "b", "c": "d"}},
		[]labels.Labels{
			{},
			{"a": "c"},
			{"c": "d"},
		}},
	{`a == "b" && c == "d"`,
		[]labels.Labels{
			{"a": "b", "c": "d"}},
		[]labels.Labels{
			{},
			{"a": "b", "c": "e"},
			{"a": "c", "c": "d"},
			{"c": "d"},
			{"a": "b"},
		}},
}

func TestParse(t *testing.T) {
	for _, test := range selectorTests {
		sel, err := Parse(test.sel)
		if err != nil {
			t.Errorf("Failed to parse selector %#v", test.sel)
		}
		for _, labels := range test.expMatches {
			if !sel.Evaluate(labels) {
				t.Errorf("Selector %#v should have matched labels %v",
					test.sel, labels)
			}
		}
		for _, labels := range test.expNonMatches {
			if sel.Evaluate(labels) {
				t.Errorf("Selector %#v should not have matched labels %v",
					test.sel, labels)
			}
		}
	}
}
