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

package ipsets

import (
	"github.com/op/go-logging"
	"github.com/projectcalico/calico-go/labels"
	"github.com/projectcalico/calico-go/labels/selectors"
	"github.com/projectcalico/libcalico/lib"
)

var log = logging.MustGetLogger("ipsets")

type Resolver struct {
	labelIdx labels.LabelInheritanceIndex

	activeSelCalc *ActiveSelectorCalculator
	ipsetCalc     *IpsetCalculator
}

func NewResolver() *Resolver {
	resolver := &Resolver{}
	resolver.activeSelCalc = NewActiveSelectorCalculator()
	resolver.activeSelCalc.OnSelectorActive = resolver.onSelectorActive
	resolver.activeSelCalc.OnSelectorInactive = resolver.onSelectorInactive

	resolver.ipsetCalc = NewIpsetCalculator()
	resolver.ipsetCalc.OnIPAdded = resolver.onIPAdded
	resolver.ipsetCalc.OnIPRemoved = resolver.onIPRemoved

	resolver.labelIdx = labels.NewInheritanceIndex(
		resolver.onMatchStarted, resolver.onMatchStopped)

	return resolver
}

func (res *Resolver) OnEndpointUpdate(key libcalico.EndpointKey, endpoint *libcalico.Endpoint) {
	log.Infof("Endpoint %v updated", key)
	res.ipsetCalc.OnEndpointUpdate(key, endpoint.IPv4Nets)
	res.labelIdx.UpdateLabels(key, endpoint.Labels, make([]interface{}, 0))
}

func (res *Resolver) OnPolicyUpdate(key libcalico.PolicyKey, policy *libcalico.Policy) {
	log.Infof("Policy %v updated", key)
	res.activeSelCalc.UpdatePolicy(key, policy)
}

func (res *Resolver) OnProfileUpdate(key libcalico.ProfileKey, policy *libcalico.Profile) {
	log.Infof("Profile %v updated", key)
	res.activeSelCalc.UpdateProfile(key, policy)
}

func (res *Resolver) onIPAdded(selID, ip string) {
	log.Infof("IP set %v now contains %v", selID, ip)
}

func (res *Resolver) onIPRemoved(selID, ip string) {
	log.Infof("IP set %v no longer contains %v", selID, ip)
}

func (res *Resolver) onMatchStarted(selId, labelId interface{}) {
	log.Infof("Endpoint %v now matches selector %v", labelId, selId)
	res.ipsetCalc.OnMatchStarted(labelId.(libcalico.Key), selId.(string))
}

func (res *Resolver) onMatchStopped(selId, labelId interface{}) {
	log.Infof("Endpoint %v no longer matches selector %v", labelId, selId)
	res.ipsetCalc.OnMatchStopped(labelId.(libcalico.Key), selId.(string))
}

func (res *Resolver) onSelectorActive(sel selector.Selector) {
	log.Infof("Selector %v now active", sel)
	res.labelIdx.UpdateSelector(sel.UniqueId(), sel)
}

func (res *Resolver) onSelectorInactive(sel selector.Selector) {
	log.Infof("Selector %v now inactive", sel)
	res.labelIdx.DeleteSelector(sel.UniqueId())
}

type IpsetCalculator struct {
	keyToIPs            map[libcalico.Key][]string
	keyToMatchingSelIDs map[libcalico.Key]map[string]bool
	selIdToIPToKey      map[string]map[string]map[libcalico.Key]bool

	OnIPAdded   func(selID string, ip string)
	OnIPRemoved func(selID string, ip string)
}

func NewIpsetCalculator() *IpsetCalculator {
	calc := &IpsetCalculator{
		keyToIPs:            make(map[libcalico.Key][]string),
		keyToMatchingSelIDs: make(map[libcalico.Key]map[string]bool),
		selIdToIPToKey:      make(map[string]map[string]map[libcalico.Key]bool),
	}
	return calc
}

func (calc *IpsetCalculator) OnMatchStarted(key libcalico.Key, selId string) {
	matchingIDs, ok := calc.keyToMatchingSelIDs[key]
	if !ok {
		matchingIDs = make(map[string]bool)
		calc.keyToMatchingSelIDs[key] = matchingIDs
	}
	matchingIDs[selId] = true

	ips := calc.keyToIPs[key]
	calc.addMatchToIndex(selId, key, ips)
}

func (calc *IpsetCalculator) addMatchToIndex(selID string, key libcalico.Key, ips []string) {
	log.Debugf("Selector %v now matches %v via %v", selID, ips, key)
	ipToKeys, ok := calc.selIdToIPToKey[selID]
	if !ok {
		ipToKeys = make(map[string]map[libcalico.Key]bool)
		calc.selIdToIPToKey[selID] = ipToKeys
	}

	for _, ip := range ips {
		keys, ok := ipToKeys[ip]
		if !ok {
			keys = make(map[libcalico.Key]bool)
			ipToKeys[ip] = keys
			calc.OnIPAdded(selID, ip)
		}
		keys[key] = true
	}
}

func (calc *IpsetCalculator) OnMatchStopped(key libcalico.Key, selId string) {
	matchingIDs := calc.keyToMatchingSelIDs[key]
	delete(matchingIDs, selId)
	if len(matchingIDs) == 0 {
		delete(calc.keyToMatchingSelIDs, key)
	}

	ips := calc.keyToIPs[key]
	calc.removeMatchFromIndex(selId, key, ips)
}

func (calc *IpsetCalculator) removeMatchFromIndex(selId string, key libcalico.Key, ips []string) {
	ipToKeys := calc.selIdToIPToKey[selId]
	for _, ip := range ips {
		keys := ipToKeys[ip]
		delete(keys, key)
		if len(keys) == 0 {
			calc.OnIPRemoved(selId, ip)
			delete(ipToKeys, ip)
			if len(ipToKeys) == 0 {
				delete(calc.selIdToIPToKey, selId)
			}
		}
	}
}

func (calc *IpsetCalculator) OnEndpointUpdate(endpointKey libcalico.Key, ips []string) {
	oldIPs := calc.keyToIPs[endpointKey]
	if len(ips) == 0 {
		delete(calc.keyToIPs, endpointKey)
	} else {
		calc.keyToIPs[endpointKey] = ips
	}

	oldIPsSet := make(map[string]bool)
	for _, ip := range oldIPs {
		oldIPsSet[ip] = true
	}

	addedIPs := make([]string, 0)
	currentIPs := make(map[string]bool)
	for _, ip := range ips {
		if !oldIPsSet[ip] {
			addedIPs = append(addedIPs, ip)
		}
		currentIPs[ip] = true
	}

	removedIPs := make([]string, 0)
	for _, ip := range oldIPs {
		if !currentIPs[ip] {
			removedIPs = append(removedIPs, ip)
		}
	}

	for selID, _ := range calc.keyToMatchingSelIDs[endpointKey] {
		calc.addMatchToIndex(selID, endpointKey, addedIPs)
		calc.removeMatchFromIndex(selID, endpointKey, removedIPs)
	}
}

func (calc *IpsetCalculator) OnEndpointDelete(endpointKey libcalico.Key) {
	calc.OnEndpointUpdate(endpointKey, []string{})
}

// ActiveSelectorCalculator calculates the active set of selectors from the current set of policies/profiles.
// It generates events for selectors becoming active/inactive.
type ActiveSelectorCalculator struct {
	// selectorsByUid maps from a selector's UID to the selector itself.
	selectorsByUid selByUid
	// activeUidsByResource maps from policy or profile ID to "set" of selector UIDs
	activeUidsByResource map[libcalico.Key]map[string]bool
	// activeResourcesByUid maps from selector UID back to the "set" of resources using it.
	activeResourcesByUid map[string]map[libcalico.Key]bool

	OnSelectorActive   func(selector selector.Selector)
	OnSelectorInactive func(selector selector.Selector)
}

func NewActiveSelectorCalculator() *ActiveSelectorCalculator {
	calc := &ActiveSelectorCalculator{
		selectorsByUid:       make(selByUid),
		activeUidsByResource: make(map[libcalico.Key]map[string]bool),
		activeResourcesByUid: make(map[string]map[libcalico.Key]bool),
	}
	return calc
}

func (calc *ActiveSelectorCalculator) UpdatePolicy(key libcalico.PolicyKey, policy *libcalico.Policy) {
	calc.updateResource(key, policy.Inbound, policy.Outbound)
}

func (calc *ActiveSelectorCalculator) UpdateProfile(key libcalico.ProfileKey, profile *libcalico.Profile) {
	calc.updateResource(key, profile.Rules.Inbound, profile.Rules.Outbound)
}

func (calc *ActiveSelectorCalculator) updateResource(key libcalico.Key, inbound, outbound []libcalico.Rule) {
	// Extract all the new selectors.
	currentSelsByUid := make(selByUid)
	currentSelsByUid.addSelectorsFromRules(inbound)
	currentSelsByUid.addSelectorsFromRules(outbound)

	// Find the set of old selectors.
	knownUids, knownUidsPresent := calc.activeUidsByResource[key]

	// Figure out which selectors are new.
	addedUids := make(map[string]bool)
	for key, _ := range currentSelsByUid {
		if !knownUids[key] {
			addedUids[key] = true
		}
	}

	// Figure out which selectors are no-longer in use.
	removedUids := make(map[string]bool)
	for key, _ := range knownUids {
		if _, ok := currentSelsByUid[key]; !ok {
			removedUids[key] = true
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
				resources = make(map[libcalico.Key]bool)
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

func (sbu selByUid) addSelectorsFromRules(rules []libcalico.Rule) {
	for _, rule := range rules {
		selStrPs := []*string{rule.SrcSelector, rule.DstSelector, rule.NotSrcSelector, rule.NotDstSelector}
		for _, selStrP := range selStrPs {
			if selStrP != nil {
				sel, err := selector.Parse(*selStrP)
				if err != nil {
					panic("FIXME: Handle bad selector")
				}
				sbu[sel.UniqueId()] = sel
			}
		}

	}
}
