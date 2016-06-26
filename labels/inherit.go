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
	UpdateParent(id string, labels map[string]string)
	DeleteParent(id string)
}

type labelInheritanceIndex struct {
	index             Index
	labelsByItemId    map[string]map[string]string
	labelsByParentId  map[string]map[string]string
	parentIdsByItemId map[string][]string
	itemIdsByParentId multidict.StringToString
	dirtyItemIds      map[string]bool
}

func NewInheritanceIndex(onMatchStarted, onMatchStopped MatchCallback) LabelInheritanceIndex {
	index := NewIndex(onMatchStarted, onMatchStopped)
	inheritIdx := labelInheritanceIndex{
		index:             index,
		labelsByItemId:    make(map[string]map[string]string),
		labelsByParentId:  make(map[string]map[string]string),
		parentIdsByItemId: make(map[string][]string),
		itemIdsByParentId: multidict.NewStringToString(),
	}
	return &inheritIdx
}

func (idx labelInheritanceIndex) UpdateSelector(id string, sel selector.Selector) {
	idx.index.UpdateSelector(id, sel)
}

func (idx labelInheritanceIndex) DeleteSelector(id string) {
	idx.index.DeleteSelector(id)
}

func (idx labelInheritanceIndex) UpdateLabels(id string, labels map[string]string, parents []string) {
	idx.onItemLabelsUpdate(id, labels)
	idx.onItemParentsUpdate(id, parents)
	idx.flushUpdates()
}

func (idx labelInheritanceIndex) onItemLabelsUpdate(id string, labels map[string]string) {
	idx.labelsByItemId[id] = labels
	idx.dirtyItemIds[id] = true
}

func (idx labelInheritanceIndex) onItemParentsUpdate(id string, parents []string) {
	oldParents := idx.parentIdsByItemId[id]
	for _, parent := range oldParents {
		idx.itemIdsByParentId.Discard(parent, id)
	}
	idx.parentIdsByItemId[id] = parents
	for _, parent := range parents {
		idx.itemIdsByParentId.Put(parent, id)
	}
	idx.dirtyItemIds[id] = true
}

func (idx labelInheritanceIndex) UpdateParent(parentId string, labels map[string]string) {
	idx.labelsByParentId[parentId] = labels
	idx.itemIdsByParentId.Iter(parentId, func(itemId string) {
		idx.dirtyItemIds[itemId] = true
	})
	idx.flushUpdates()
}

func (idx labelInheritanceIndex) flushUpdates() {
	for itemId, _ := range idx.dirtyItemIds {
		itemLabels, ok := idx.labelsByItemId[itemId]
		if !ok {
			// Item deleted.
			idx.index.DeleteLabels(itemId)
		} else {
			// Item updated/created, re-evaluate labels.
			combinedLabels := make(map[string]string)
			parentIds := idx.parentIdsByItemId[itemId]
			for _, parentId := range parentIds {
				parentLabels := idx.labelsByParentId[parentId]
				for k, v := range parentLabels {
					combinedLabels[k] = v
				}
			}
			for k, v := range itemLabels {
				combinedLabels[k] = v
			}
			idx.index.UpdateLabels(itemId, combinedLabels)
		}
	}
	idx.dirtyItemIds = make(map[string]bool)
}