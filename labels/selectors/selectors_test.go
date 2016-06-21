package selector_test

import (
	. "github.com/projectcalico/calico-go/labels/selectors"

	"testing"
)

var selectorTests = []struct {
	sel           string
	expMatches    []map[string]string
	expNonMatches []map[string]string
}{
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
