package labels

import (
	"testing"
	"github.com/projectcalico/calico-go/labels/selectors"
)

type update struct {
	op string
	labelId string
	selId string
}


func TestLinearScanIndexMainline(t *testing.T) {
	var updates []update
	idx := NewIndex(
		func(selId, labelId string) {
			t.Logf("Match started: %v, %v", selId, labelId)
			updates = append(updates,
				update{op: "start",
					labelId: labelId,
					selId: selId})
		},
	        func(selId, labelId string) {
			t.Logf("Match stopped: %v, %v", selId, labelId)
			updates = append(updates,
				update{op: "stop",
					labelId: labelId,
					selId: selId})
		})

	idx.UpdateLabels("l1", map[string]string{"a": "a1", "b": "b1", "c": "c1"})
	if len(updates) > 0 {
		t.Error("Unexpected update %v after adding just labels", updates[0])
	}
	// Add a non-matching expression
	sel, _ := selector.Parse(`d=="d1"`)
	idx.UpdateSelector("e1", sel)
	if len(updates) > 0 {
		t.Errorf("Unexpected update %v after adding non-matching selector", updates[0])
	}
        // Add a matching expression
	sel, _ = selector.Parse(`a=="a1"`)
	idx.UpdateSelector("e2", sel)
	if len(updates) != 1 {
		t.Errorf("Unexpected/no updates %v after adding non-matching selector", updates)
	} else {
		update := updates[0]
		if !(update.op == "start" && update.labelId == "l1" && update.selId == "e2") {
			t.Errorf("Unexpected update %v after adding matching selector", updates[0])
		}
	}
	updates = updates[:0]

	// Update matching expression, still matches
	sel, _ = selector.Parse(`b=="b1"`)
	idx.UpdateSelector("e2", sel)
	if len(updates) != 0 {
		t.Errorf("Unexpected updates %v after adding updating selector", updates)
	}
        // Update matching expression, no-longer matches
	sel, _ = selector.Parse(`a=="a2"`)
	idx.UpdateSelector("e2", sel)
	if len(updates) != 1 {
		t.Errorf("Unexpected/no updates %v after upating to non-matching selector", updates)
	} else {
		update := updates[0]
		if !(update.op == "stop" && update.labelId == "l1" && update.selId == "e2") {
			t.Errorf("Unexpected update %v after adding matching selector", updates[0])
		}
	}
	updates = updates[:0]

        // Update labels to match.
	idx.UpdateLabels("l1", map[string]string{"a": "a2", "b": "b1", "d": "d1"})
        if len(updates) != 2 {
		t.Errorf("Unexpected/no updates %v after updating labels", updates)
	} else {
		update := updates[0]
		if !(update.op == "start" && update.labelId == "l1") {
			t.Errorf("Unexpected update %v after updating labels to match", updates[0])
		}
		update = updates[1]
		if !(update.op == "start" && update.labelId == "l1") {
			t.Errorf("Unexpected update %v after updating labels to match", updates[0])
		}
	}
	updates = updates[:0]

	//self.index.on_labels_update("l1", {"b": "b2", "d": "d1"})
        //self.assert_add("e1", "l1")
        //self.assert_add("e2", "l1")
        //self.assert_no_updates()
        //self.index.on_labels_update("l1", None)
        //self.assert_remove("e1", "l1")
        //self.assert_remove("e2", "l1")
        //self.index.on_expression_update("e1", None)
        //self.index.on_expression_update("e2", None)
        //self.assert_indexes_empty()
}
