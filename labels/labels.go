package labels

import (
	"github.com/deckarep/golang-set"
	"github.com/op/go-logging"
	"github.com/projectcalico/calico-go/labels/selectors"
	"github.com/projectcalico/calico-go/multidict"
)

var log = logging.MustGetLogger("labels")

type Index interface {
	UpdateSelector(id string, sel selector.Selector)
	DeleteSelector(id string)
	UpdateLabels(id string, labels map[string]string)
	DeleteLabels(id string)
}

type MatchCallback func(selId, labelId string)

type linearScanIndex struct {
	// All known labels and selectors.
	labelsById    map[string]map[string]string
	selectorsById map[string]selector.Selector

	// Current matches.
	selIdsByLabelId map[string]map[string]bool
	labelIdsBySelId map[string]map[string]bool

	// Callback functions
	OnMatchStarted MatchCallback
	OnMatchStopped MatchCallback
}

func NewIndex(onMatchStarted, onMatchStopped MatchCallback) Index {
	return linearScanIndex{
		labelsById:      make(map[string]map[string]string),
		selectorsById:   make(map[string]selector.Selector),
		selIdsByLabelId: make(map[string]map[string]bool),
		labelIdsBySelId: make(map[string]map[string]bool),
		OnMatchStarted:  onMatchStarted,
		OnMatchStopped:  onMatchStopped,
	}
}

func (idx linearScanIndex) UpdateSelector(id string, sel selector.Selector) {
	log.Debugf("Updating selector %v", id)
	if sel == nil {
		panic("Selector should not be nil")
	}
	idx.scanAllLabels(id, sel)
	idx.selectorsById[id] = sel
}

func (idx linearScanIndex) DeleteSelector(id string) {
	log.Debugf("Deleting selector %v", id)
	matchSet := idx.labelIdsBySelId[id]
	matchSlice := make([]string, 0, len(matchSet))
	for labelId, _ := range matchSet {
		matchSlice = append(matchSlice, labelId)
	}
	for _, labelId := range matchSlice {
		idx.deleteMatch(id, labelId)
	}
	delete(idx.selectorsById, id)
}

func (idx linearScanIndex) UpdateLabels(id string, labels map[string]string) {
	log.Debugf("Updating labels for ID %v", id)
	idx.scanAllSelectors(id, labels)
	idx.labelsById[id] = labels
}

func (idx linearScanIndex) DeleteLabels(id string) {
	log.Debugf("Deleting labels for %v", id)
	matchSet := idx.selIdsByLabelId[id]
	matchSlice := make([]string, 0, len(matchSet))
	for selId, _ := range matchSet {
		matchSlice = append(matchSlice, selId)
	}
	for _, selId := range matchSlice {
		idx.deleteMatch(selId, id)
	}
	delete(idx.labelsById, id)
}

func (idx linearScanIndex) scanAllLabels(selId string, sel selector.Selector) {
	log.Debugf("Scanning all (%v) labels against selector %v",
		len(idx.labelsById), selId)
	for labelId, labels := range idx.labelsById {
		idx.updateMatches(selId, sel, labelId, labels)
	}
}

func (idx linearScanIndex) scanAllSelectors(labelId string, labels map[string]string) {
	log.Debugf("Scanning all (%v) selectors against labels %v",
		len(idx.selectorsById), labelId)
	for selId, sel := range idx.selectorsById {
		idx.updateMatches(selId, sel, labelId, labels)
	}
}

func (idx linearScanIndex) updateMatches(selId string, sel selector.Selector,
	labelId string, labels map[string]string) {
	nowMatches := sel.Evaluate(labels)
	if nowMatches {
		idx.storeMatch(selId, labelId)
	} else {
		idx.deleteMatch(selId, labelId)
	}
}

func (idx linearScanIndex) storeMatch(selId, labelId string) {
	previouslyMatched := idx.labelIdsBySelId[selId][labelId]
	if !previouslyMatched {
		log.Debugf("Selector %v now matches labels %v", selId, labelId)
		if labelIds, ok := idx.labelIdsBySelId[selId]; ok {
			labelIds[labelId] = true
		} else {
			idx.labelIdsBySelId[selId] = map[string]bool{
				labelId: true,
			}
		}
		if selIds, ok := idx.selIdsByLabelId[labelId]; ok {
			selIds[selId] = true
		} else {
			idx.selIdsByLabelId[labelId] = map[string]bool{
				selId: true,
			}
		}
		idx.OnMatchStarted(selId, labelId)
	}
}

func (idx linearScanIndex) deleteMatch(selId, labelId string) {
	previouslyMatched := idx.labelIdsBySelId[selId][labelId]
	if previouslyMatched {
		log.Debugf("Selector %v no longer matches labels %v",
			selId, labelId)
		delete(idx.labelIdsBySelId[selId], labelId)
		if len(idx.labelIdsBySelId[selId]) == 0 {
			delete(idx.labelIdsBySelId, selId)
		}
		delete(idx.selIdsByLabelId[labelId], selId)
		if len(idx.selIdsByLabelId[labelId]) == 0 {
			delete(idx.selIdsByLabelId, labelId)
		}
		idx.OnMatchStopped(selId, labelId)
	}
}

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
