// Copyright (c) 2016 Tigera, Inc. All rights reserved.
//
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
package ipsets

import (
	"github.com/projectcalico/calico-go/lib/backend"
	"github.com/projectcalico/calico-go/lib/selector"
)

// ActiveSelectorCalculator calculates the active set of selectors from the current set of policies/profiles.
// It generates events for selectors becoming active/inactive.
type ActiveSelectorCalculator struct {
	// selectorsByUid maps from a selector's UID to the selector itself.
	selectorsByUid selByUid
	// activeUidsByResource maps from policy or profile ID to "set" of selector UIDs
	activeUidsByResource map[backend.KeyInterface]map[string]bool
	// activeResourcesByUid maps from selector UID back to the "set" of resources using it.
	activeResourcesByUid map[string]map[backend.KeyInterface]bool

	OnSelectorActive   func(selector selector.Selector)
	OnSelectorInactive func(selector selector.Selector)
}

func NewActiveSelectorCalculator() *ActiveSelectorCalculator {
	calc := &ActiveSelectorCalculator{
		selectorsByUid:       make(selByUid),
		activeUidsByResource: make(map[backend.KeyInterface]map[string]bool),
		activeResourcesByUid: make(map[string]map[backend.KeyInterface]bool),
	}
	return calc
}

func (calc *ActiveSelectorCalculator) UpdatePolicy(key backend.PolicyKey, policy *backend.Policy) {
	calc.updateResource(key, policy.InboundRules, policy.OutboundRules)
}

func (calc *ActiveSelectorCalculator) DeletePolicy(key backend.PolicyKey) {
	calc.updateResource(key, []backend.Rule{}, []backend.Rule{})
}

func (calc *ActiveSelectorCalculator) UpdateProfile(key backend.ProfileKey, profile *backend.Profile) {
	calc.updateResource(key, profile.Rules.InboundRules, profile.Rules.OutboundRules)
}

func (calc *ActiveSelectorCalculator) updateResource(key backend.KeyInterface, inbound, outbound []backend.Rule) {
	// Extract all the new selectors.
	currentSelsByUid := make(selByUid)
	currentSelsByUid.addSelectorsFromRules(inbound)
	currentSelsByUid.addSelectorsFromRules(outbound)

	// Find the set of old selectors.
	knownUids, knownUidsPresent := calc.activeUidsByResource[key]
	log.Debugf("Known UIDs for %v: %v", key, knownUids)

	// Figure out which selectors are new.
	addedUids := make(map[string]bool)
	for uid, _ := range currentSelsByUid {
		if !knownUids[uid] {
			log.Debugf("Added UID: %v", uid)
			addedUids[uid] = true
		}
	}

	// Figure out which selectors are no-longer in use.
	removedUids := make(map[string]bool)
	for uid, _ := range knownUids {
		if _, ok := currentSelsByUid[uid]; !ok {
			log.Debugf("Removed UID: %v", uid)
			removedUids[uid] = true
		}
	}

	// Add the new into the index, triggering events as we discover
	// newly-active selectors.
	if len(addedUids) > 0 {
		if !knownUidsPresent {
			knownUids = make(map[string]bool)
			calc.activeUidsByResource[key] = knownUids
		}
		for uid, _ := range addedUids {
			knownUids[uid] = true
			resources, ok := calc.activeResourcesByUid[uid]
			if !ok {
				log.Debugf("Selector became active: %v", uid)
				resources = make(map[backend.KeyInterface]bool)
				calc.activeResourcesByUid[uid] = resources
				sel := currentSelsByUid[uid]
				calc.selectorsByUid[uid] = sel
				// This selector just became active, trigger event.
				calc.OnSelectorActive(sel)
			}
			resources[key] = true
		}
	}

	// And remove the old, trigerring events as we clean up unused
	// selectors.
	for uid, _ := range removedUids {
		delete(knownUids, uid)
		resources := calc.activeResourcesByUid[uid]
		delete(resources, key)
		if len(resources) == 0 {
			log.Debugf("Selector became inactive: %v", uid)
			delete(calc.activeResourcesByUid, uid)
			sel := calc.selectorsByUid[uid]
			delete(calc.selectorsByUid, uid)
			// This selector just became inactive, trigger event.
			calc.OnSelectorInactive(sel)
		}
	}
}

// selByUid is an augmented map with methods to assist in extracting rules from policies.
type selByUid map[string]selector.Selector

func (sbu selByUid) addSelectorsFromRules(rules []backend.Rule) {
	for i, rule := range rules {
		selStrPs := []*string{&rule.SrcSelector,
			&rule.DstSelector,
			&rule.NotSrcSelector,
			&rule.NotDstSelector}
		for _, selStrP := range selStrPs {
			if *selStrP != "" {
				sel, err := selector.Parse(*selStrP)
				if err != nil {
					panic("FIXME: Handle bad selector")
				}
				uid := sel.UniqueId()
				sbu[uid] = sel
				// FIXME: Remove this horrible hack where we update the policy rule
				*selStrP = uid
			}
		}
		rules[i] = rule
	}
}
