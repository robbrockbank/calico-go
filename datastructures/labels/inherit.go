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
	"github.com/projectcalico/calico-go/datastructures/multidict"
	"github.com/projectcalico/calico-go/lib/selector"
)

type LabelInheritanceIndex interface {
	UpdateSelector(id interface{}, sel selector.Selector)
	DeleteSelector(id interface{})
	UpdateLabels(id interface{}, labels map[string]string, parents []interface{})
	DeleteLabels(id interface{})
	UpdateParentLabels(id interface{}, labels map[string]string)
	DeleteParentLabels(id interface{})
}

type labelInheritanceIndex struct {
	index             Index
	labelsByItemID    map[interface{}]map[string]string
	labelsByParentID  map[interface{}]map[string]string
	parentIDsByItemID map[interface{}][]interface{}
	itemIDsByParentID multidict.IfaceToIface
	dirtyItemIDs      map[interface{}]bool
}

func NewInheritanceIndex(onMatchStarted, onMatchStopped MatchCallback) LabelInheritanceIndex {
	index := NewIndex(onMatchStarted, onMatchStopped)
	inheritIDx := labelInheritanceIndex{
		index:             index,
		labelsByItemID:    make(map[interface{}]map[string]string),
		labelsByParentID:  make(map[interface{}]map[string]string),
		parentIDsByItemID: make(map[interface{}][]interface{}),
		itemIDsByParentID: multidict.NewIfaceToIface(),
		dirtyItemIDs:      make(map[interface{}]bool),
	}
	return &inheritIDx
}

func (idx *labelInheritanceIndex) UpdateSelector(id interface{}, sel selector.Selector) {
	idx.index.UpdateSelector(id, sel)
}

func (idx *labelInheritanceIndex) DeleteSelector(id interface{}) {
	idx.index.DeleteSelector(id)
}

func (idx *labelInheritanceIndex) UpdateLabels(id interface{}, labels map[string]string, parents []interface{}) {
	log.Debug("Inherit index updating labels for ", id)
	log.Debug("Num dirty items ", len(idx.dirtyItemIDs), " items")
	idx.labelsByItemID[id] = labels
	idx.onItemParentsUpdate(id, parents)
	idx.dirtyItemIDs[id] = true
	idx.flushUpdates()
	log.Debug("Num ending dirty items ", len(idx.dirtyItemIDs), " items")
}

func (idx *labelInheritanceIndex) DeleteLabels(id interface{}) {
	log.Debug("Inherit index deleting labels for ", id)
	delete(idx.labelsByItemID, id)
	var noParents []interface{}
	idx.onItemParentsUpdate(id, noParents)
	idx.dirtyItemIDs[id] = true
	idx.flushUpdates()
}

func (idx *labelInheritanceIndex) onItemParentsUpdate(id interface{}, parents []interface{}) {
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

func (idx *labelInheritanceIndex) UpdateParentLabels(parentID interface{}, labels map[string]string) {
	idx.labelsByParentID[parentID] = labels
	idx.flushChildren(parentID)
}

func (idx *labelInheritanceIndex) DeleteParentLabels(parentID interface{}) {
	delete(idx.labelsByParentID, parentID)
	idx.flushChildren(parentID)
}

func (idx *labelInheritanceIndex) flushChildren(parentID interface{}) {
	idx.itemIDsByParentID.Iter(parentID, func(itemID interface{}) {
		log.Debug("Marking child ", itemID, " dirty")
		idx.dirtyItemIDs[itemID] = true
	})
	idx.flushUpdates()
}

func (idx *labelInheritanceIndex) flushUpdates() {
	for itemID, _ := range idx.dirtyItemIDs {
		log.Debugf("Flushing %#v", itemID)
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
	idx.dirtyItemIDs = make(map[interface{}]bool)
}

var _ LabelInheritanceIndex = (*labelInheritanceIndex)(nil)
