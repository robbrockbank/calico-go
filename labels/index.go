package labels

import (
	"github.com/op/go-logging"
	"github.com/projectcalico/calico-go/labels/selectors"
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

