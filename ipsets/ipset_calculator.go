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

import "github.com/projectcalico/calico-go/lib"

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
	log.Debugf("Selector %v now matches IPs %v via %v", selID, ips, key)
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

func (calc *IpsetCalculator) removeMatchFromIndex(selID string, key libcalico.Key, ips []string) {
	log.Debugf("Selector %v no longer matches IPs %v via %v", selID, ips, key)
	ipToKeys := calc.selIdToIPToKey[selID]
	for _, ip := range ips {
		keys := ipToKeys[ip]
		delete(keys, key)
		if len(keys) == 0 {
			calc.OnIPRemoved(selID, ip)
			delete(ipToKeys, ip)
			if len(ipToKeys) == 0 {
				delete(calc.selIdToIPToKey, selID)
			}
		}
	}
}

func (calc *IpsetCalculator) OnEndpointUpdate(endpointKey libcalico.Key, ips []string) {
	log.Debugf("Endpoint %v IPs updated to %v", endpointKey, ips)
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

	matchingSels := calc.keyToMatchingSelIDs[endpointKey]
	log.Debugf("Updating IPs in matching selectors: %v", matchingSels)
	for selID, _ := range matchingSels {
		calc.addMatchToIndex(selID, endpointKey, addedIPs)
		calc.removeMatchFromIndex(selID, endpointKey, removedIPs)
	}
}

func (calc *IpsetCalculator) OnEndpointDelete(endpointKey libcalico.Key) {
	calc.OnEndpointUpdate(endpointKey, []string{})
}
