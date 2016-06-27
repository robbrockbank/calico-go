// Copyright (c) 2016 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package labels

import (
	"github.com/projectcalico/calico-go/labels/selectors"
	"github.com/projectcalico/calico-go/multidict"
)

type LabelInheritanceIndex interface {
	UpdateSelector(id string, sel selector.Selector)
	DeleteSelector(id string)
	UpdateLabels(id string, labels map[string]string, parents []string)
	DeleteLabels(id string)
	UpdateParentLabels(id string, labels map[string]string)
	DeleteParentLabels(id string)
}

type labelInheritanceIndex struct {
	index             Index
	labelsByItemID    map[string]map[string]string
	labelsByParentID  map[string]map[string]string
	parentIDsByItemID map[string][]string
	itemIDsByParentID multidict.StringToString
	dirtyItemIDs      map[string]bool
}

func NewInheritanceIndex(onMatchStarted, onMatchStopped MatchCallback) LabelInheritanceIndex {
	index := NewIndex(onMatchStarted, onMatchStopped)
	inheritIDx := labelInheritanceIndex{
		index:             index,
		labelsByItemID:    make(map[string]map[string]string),
		labelsByParentID:  make(map[string]map[string]string),
		parentIDsByItemID: make(map[string][]string),
		itemIDsByParentID: multidict.NewStringToString(),
	}
	return &inheritIDx
}

func (idx labelInheritanceIndex) UpdateSelector(id string, sel selector.Selector) {
	idx.index.UpdateSelector(id, sel)
}

func (idx labelInheritanceIndex) DeleteSelector(id string) {
	idx.index.DeleteSelector(id)
}

func (idx labelInheritanceIndex) UpdateLabels(id string, labels map[string]string, parents []string) {
	idx.labelsByItemID[id] = labels
	idx.onItemParentsUpdate(id, parents)
	idx.dirtyItemIDs[id] = true
	idx.flushUpdates()
}

func (idx labelInheritanceIndex) DeleteLabels(id string) {
	delete(idx.labelsByItemID, id)
	var noParents []string
	idx.onItemParentsUpdate(id, noParents)
	idx.dirtyItemIDs[id] = true
	idx.flushUpdates()
}

func (idx labelInheritanceIndex) onItemParentsUpdate(id string, parents []string) {
	oldParents := idx.parentIDsByItemID[id]
	for _, parent := range oldParents {
		idx.itemIDsByParentID.Discard(parent, id)
	}
	if len(parents) > 0 {
		idx.parentIDsByItemID[id] = parents
	} else {
		delete(idx.parentIDsByItemID, id)
	}
	for _, parent := range parents {
		idx.itemIDsByParentID.Put(parent, id)
	}
}

func (idx labelInheritanceIndex) UpdateParentLabels(parentID string, labels map[string]string) {
	idx.labelsByParentID[parentID] = labels
	idx.flushChildren(parentID)
}

func (idx labelInheritanceIndex) DeleteParentLabels(parentID string) {
	delete(idx.labelsByParentID, parentID)
	idx.flushChildren(parentID)
}

func (idx labelInheritanceIndex) flushChildren(parentID string) {
	idx.itemIDsByParentID.Iter(parentID, func(itemID string) {
		idx.dirtyItemIDs[itemID] = true
	})
	idx.flushUpdates()
}

func (idx labelInheritanceIndex) flushUpdates() {
	for itemID, _ := range idx.dirtyItemIDs {
		itemLabels, ok := idx.labelsByItemID[itemID]
		if !ok {
			// Item deleted.
			idx.index.DeleteLabels(itemID)
		} else {
			// Item updated/created, re-evaluate labels.
			combinedLabels := make(map[string]string)
			parentIDs := idx.parentIDsByItemID[itemID]
			for _, parentID := range parentIDs {
				parentLabels := idx.labelsByParentID[parentID]
				for k, v := range parentLabels {
					combinedLabels[k] = v
				}
			}
			for k, v := range itemLabels {
				combinedLabels[k] = v
			}
			idx.index.UpdateLabels(itemID, combinedLabels)
		}
	}
	idx.dirtyItemIDs = make(map[string]bool)
}

var _ LabelInheritanceIndex = (*labelInheritanceIndex)(nil)
