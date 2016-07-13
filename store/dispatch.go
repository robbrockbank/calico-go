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
	"github.com/projectcalico/calico-go/lib"
	"reflect"
)

var log = logging.MustGetLogger("store")

type ParsedUpdateHandler func(update ParsedUpdate)

type Dispatcher struct {
	listenersByType map[reflect.Kind]ParsedUpdateHandler
}

// NewDispatcher creates a Dispatcher with all its event handlers set to no-ops.
func NewDispatcher() *Dispatcher {
	d := Dispatcher{
		listenersByType: make(map[reflect.Kind]ParsedUpdateHandler),
	}
	return &d
}

type ParsedUpdate struct {
	Key      libcalico.Key
	Value    interface{}
	ParseErr error
	RawJSON  string
}

func (d Dispatcher) DispatchUpdate(update *Update) {
	log.Debug("Dispatching update ", update)
	key := libcalico.ParseKey(update.Key)
	if key == nil {
		// Unknown key.
		log.Debug("Unknown key ", update.Key)
		return
	}

	parsedUpdate := ParsedUpdate{
		Key: key,
	}

	if update.ValueOrNil != nil {
		var data interface{}
		var err error
		parsedUpdate.RawJSON = update.ValueOrNil
		rawData := []byte(*update.ValueOrNil)
		switch key := key.(type) {
		case libcalico.EndpointKey:
			data, err = libcalico.ParseEndpoint(key, rawData)
		case libcalico.HostEndpointKey:
			data, err = libcalico.ParseHostEndpoint(key, rawData)
		case libcalico.PolicyKey:
			data, err = libcalico.ParsePolicy(key, rawData)
		//case libcalico.ProfileKey:
		//	if update.ValueOrNil == nil {
		//		d.OnProfileDelete(key)
		//	} else {
		//		policy, err := libcalico.ParseProfile(key, []byte(*update.ValueOrNil))
		//		if err != nil {
		//			d.OnProfileParseFailure(key, err)
		//		} else {
		//			d.OnProfileUpdate(key, policy)
		//		}
		//		// FIXME: make this generic
		//		json := profile.JSON()
		//		log.Infof("New JSON: %v", json)
		//		update.ValueOrNil = &json
		//	}
		case libcalico.TierMetadataKey:
			data, err = libcalico.ParseTierMetadata(key, rawData)
		}
		parsedUpdate.Value = data
		parsedUpdate.ParseErr = err
	}

	for _, recv := range d.listenersByType[reflect.TypeOf(key)] {
		recv(parsedUpdate)
	}
}
