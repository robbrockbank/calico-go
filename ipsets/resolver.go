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
	"github.com/projectcalico/libcalico/lib"
	"github.com/op/go-logging"
	"github.com/projectcalico/calico-go/labels"
	"github.com/projectcalico/calico-go/labels/selectors"
)

var log = logging.MustGetLogger("ipsets")

type Resolver struct {
	labelIdx labels.LabelInheritanceIndex
}

func NewResolver() *Resolver {
	resolver := Resolver{}
	resolver.labelIdx = labels.NewInheritanceIndex(
		resolver.onMatchStarted, resolver.onMatchStopped)
	return &resolver
}

func (res Resolver) onMatchStarted(selId, labelId interface{}) {
	log.Infof("Labels %v now match selector %v", labelId, selId)
}

func (res Resolver) onMatchStopped(selId, labelId interface{}) {
	log.Infof("Labels %v no longer match selector %v", labelId, selId)
}

func (res Resolver) OnEndpointUpdate(key *libcalico.EndpointKey, endpoint *libcalico.Endpoint) {
	log.Infof("Endpoint %v updated", key)
	res.labelIdx.UpdateLabels(*key, endpoint.Labels, make([]interface{}, 0))
}

func (res Resolver) OnPolicyUpdate(key *libcalico.PolicyKey, policy *libcalico.Policy) {
	log.Infof("Policy %v updated", key)
	sel, err := selector.Parse(policy.Selector)
	if err != nil {
		// FIXME validate selectors earlier
		panic("Invalid selector")
	}
	res.labelIdx.UpdateSelector(*key, sel)
}