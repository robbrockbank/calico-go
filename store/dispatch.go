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

package store

import (
	"github.com/op/go-logging"
	"github.com/projectcalico/libcalico/lib"
)

var log = logging.MustGetLogger("store")

type Dispatcher struct {
	OnEndpointUpdate       func(key libcalico.EndpointKey, endpoint *libcalico.Endpoint)
	OnEndpointDelete       func(key libcalico.EndpointKey)
	OnEndpointParseFailure func(key libcalico.EndpointKey, err error)

	OnHostEndpointUpdate       func(key libcalico.HostEndpointKey, hostEndpoint *libcalico.HostEndpoint)
	OnHostEndpointDelete       func(key libcalico.HostEndpointKey)
	OnHostEndpointParseFailure func(key libcalico.HostEndpointKey, err error)

	OnPolicyUpdate       func(key libcalico.PolicyKey, endpoint *libcalico.Policy)
	OnPolicyDelete       func(key libcalico.PolicyKey)
	OnPolicyParseFailure func(key libcalico.PolicyKey, err error)

	OnTierMetadataUpdate       func(key libcalico.TierMetadataKey, endpoint *libcalico.TierMetadata)
	OnTierMetadataDelete       func(key libcalico.TierMetadataKey)
	OnTierMetadataParseFailure func(key libcalico.TierMetadataKey, err error)
}

// NewDispatcher creates a Dispatcher with all its event handlers set to no-ops.
func NewDispatcher() *Dispatcher {
	d := Dispatcher{
		OnEndpointUpdate:       func(key libcalico.EndpointKey, endpoint *libcalico.Endpoint) {},
		OnEndpointDelete:       func(key libcalico.EndpointKey) {},
		OnEndpointParseFailure: func(key libcalico.EndpointKey, err error) {},

		OnHostEndpointUpdate:       func(key libcalico.HostEndpointKey, hostEndpoint *libcalico.HostEndpoint) {},
		OnHostEndpointDelete:       func(key libcalico.HostEndpointKey) {},
		OnHostEndpointParseFailure: func(key libcalico.HostEndpointKey, err error) {},

		OnPolicyUpdate:       func(key libcalico.PolicyKey, endpoint *libcalico.Policy) {},
		OnPolicyDelete:       func(key libcalico.PolicyKey) {},
		OnPolicyParseFailure: func(key libcalico.PolicyKey, err error) {},

		OnTierMetadataUpdate:       func(key libcalico.TierMetadataKey, endpoint *libcalico.TierMetadata) {},
		OnTierMetadataDelete:       func(key libcalico.TierMetadataKey) {},
		OnTierMetadataParseFailure: func(key libcalico.TierMetadataKey, err error) {},
	}
	return &d
}

func (d Dispatcher) DispatchUpdate(update Update) {
	log.Debug("Dispatching update ", update)
	key := libcalico.ParseKey(update.Key)
	if key == nil {
		// Unknown key.
		log.Debug("Unknown key ", update.Key)
		return
	}
	switch key := key.(type) {
	case libcalico.EndpointKey:
		if update.ValueOrNil == nil {
			d.OnEndpointDelete(key)
		} else {
			endpoint, err := libcalico.ParseEndpoint(key, []byte(*update.ValueOrNil))
			if err != nil {
				d.OnEndpointParseFailure(key, err)
			} else {
				d.OnEndpointUpdate(key, endpoint)
			}
		}
	case libcalico.HostEndpointKey:
		if update.ValueOrNil == nil {
			d.OnHostEndpointDelete(key)
		} else {
			hostEndpoint, err := libcalico.ParseHostEndpoint(key, []byte(*update.ValueOrNil))
			if err != nil {
				d.OnHostEndpointParseFailure(key, err)
			} else {
				d.OnHostEndpointUpdate(key, hostEndpoint)
			}
		}
	case libcalico.PolicyKey:
		if update.ValueOrNil == nil {
			d.OnPolicyDelete(key)
		} else {
			policy, err := libcalico.ParsePolicy(key, []byte(*update.ValueOrNil))
			if err != nil {
				d.OnPolicyParseFailure(key, err)
			} else {
				d.OnPolicyUpdate(key, policy)
			}
		}
	case libcalico.TierMetadataKey:
		if update.ValueOrNil == nil {
			d.OnTierMetadataDelete(key)
		} else {
			endpoint, err := libcalico.ParseTierMetadata(key, []byte(*update.ValueOrNil))
			if err != nil {
				d.OnTierMetadataParseFailure(key, err)
			} else {
				d.OnTierMetadataUpdate(key, endpoint)
			}
		}
	}
}
