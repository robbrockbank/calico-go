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

// Resolver processes datastore updates to calculate the current set of active ipsets.
// It generates events for ipsets being added/removed and IPs being added/removed from them.
type Resolver struct {
	// ActiveSelectorCalculator scans the active policies/profiles for
	// selectors...
	activeSelCalc *ActiveSelectorCalculator
	// ...which we pass to the label inheritance index to calculate the
	// endpoints that match...
	labelIdx labels.LabelInheritanceIndex
	// ...which we pass to the ipset calculator to merge the IPs from
	// different endpoints.
	ipsetCalc *IpsetCalculator

	OnSelectorAdded   func(selID string)
	OnIPAdded         func(selID, ip string)
	OnIPRemoved       func(selID, ip string)
	OnSelectorRemoved func(selID string)
}

func NewResolver() *Resolver {
	resolver := &Resolver{
		OnSelectorAdded:   func(selID string) {},
		OnIPAdded:         func(selID, ip string) {},
		OnIPRemoved:       func(selID, ip string) {},
		OnSelectorRemoved: func(selID string) {},
	}
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

// Datastore callbacks:

// OnEndpointUpdate is called when we get an endpoint update from the datastore.
// If fans out the update to the ipset calculator and the label index.
func (res *Resolver) OnEndpointUpdate(key libcalico.EndpointKey, endpoint *libcalico.Endpoint) {
	log.Infof("Endpoint %v updated", key)
	res.ipsetCalc.OnEndpointUpdate(key, endpoint.IPv4Nets)
	res.labelIdx.UpdateLabels(key, endpoint.Labels, make([]interface{}, 0))
}

// OnPolicyUpdate is called when we get a policy update from the datastore.
// It passes through to the ActiveSetCalculator, which extracts the active ipsets from its rules.
func (res *Resolver) OnPolicyUpdate(key libcalico.PolicyKey, policy *libcalico.Policy) {
	log.Infof("Policy %v updated", key)
	res.activeSelCalc.UpdatePolicy(key, policy)
}

// OnProfileUpdate is called when we get a policy update from the datastore.
// It passes through to the ActiveSetCalculator, which extracts the active ipsets from its rules.
func (res *Resolver) OnProfileUpdate(key libcalico.ProfileKey, policy *libcalico.Profile) {
	log.Infof("Profile %v updated", key)
	res.activeSelCalc.UpdateProfile(key, policy)
}

// IpsetCalculator callbacks:

// onIPAdded is called when an IP is now present in an active selector.
func (res *Resolver) onIPAdded(selID, ip string) {
	log.Infof("IP set %v now contains %v", selID, ip)
	res.OnIPAdded(selID, ip)
}

// onIPAdded is called when an IP is no longer present in a selector.
func (res *Resolver) onIPRemoved(selID, ip string) {
	log.Infof("IP set %v no longer contains %v", selID, ip)
	res.OnIPRemoved(selID, ip)
}

// LabelIndex callbacks:

// onMatchStarted is called when an endpoint starts matching an active selector.
func (res *Resolver) onMatchStarted(selId, labelId interface{}) {
	log.Infof("Endpoint %v now matches selector %v", labelId, selId)
	res.ipsetCalc.OnMatchStarted(labelId.(libcalico.Key), selId.(string))
}

// onMatchStopped is called when an endpoint stops matching an active selector.
func (res *Resolver) onMatchStopped(selId, labelId interface{}) {
	log.Infof("Endpoint %v no longer matches selector %v", labelId, selId)
	res.ipsetCalc.OnMatchStopped(labelId.(libcalico.Key), selId.(string))
}

// ActiveSelectorCalculator callbacks:

// onSelectorActive is called when a selector starts being used in a rule.
// It adds the selector to the label index and starts tracking it.
func (res *Resolver) onSelectorActive(sel selector.Selector) {
	log.Infof("Selector %v now active", sel)
	res.OnSelectorAdded(sel.UniqueId())
	res.labelIdx.UpdateSelector(sel.UniqueId(), sel)
}

// onSelectorActive is called when a selector stops being used in a rule.
// It removes the selector to the label index and stops tracking it.
func (res *Resolver) onSelectorInactive(sel selector.Selector) {
	log.Infof("Selector %v now inactive", sel)
	res.labelIdx.DeleteSelector(sel.UniqueId())
	res.OnSelectorRemoved(sel.UniqueId())
}
