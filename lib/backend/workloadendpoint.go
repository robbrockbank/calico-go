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

package backend

import (
	"fmt"

	"regexp"

	"github.com/golang/glog"
	. "github.com/projectcalico/calico-go/lib/common"
	"reflect"
)

var (
	matchWorkloadEndpoint = regexp.MustCompile("^/?calico/v1/host/([^/]+)/workload/([^/]+)/([^/]+)/endpoint/([^/]+)$")
)

type WorkloadEndpointKey struct {
	Hostname       string `json:"-"`
	OrchestratorID string `json:"-"`
	WorkloadID     string `json:"-"`
	EndpointID     string `json:"-"`
}

func (key WorkloadEndpointKey) asEtcdKey() (string, error) {
	if key.Hostname == "" || key.OrchestratorID == "" || key.WorkloadID == "" || key.EndpointID == "" {
		return "", ErrorInsufficientIdentifiers{}
	}
	return fmt.Sprintf("/calico/v1/host/%s/workload/%s/%s/endpoint/%s",
		key.Hostname, key.OrchestratorID, key.WorkloadID, key.EndpointID), nil
}

func (key WorkloadEndpointKey) asEtcdDeleteKey() (string, error) {
	return key.asEtcdKey()
}

func (key WorkloadEndpointKey) valueType() reflect.Type {
	return reflect.TypeOf(WorkloadEndpoint{})
}

type WorkloadEndpointListOptions struct {
	Hostname       string
	OrchestratorID string
	WorkloadID     string
	EndpointID     string
}

func (options WorkloadEndpointListOptions) asEtcdKeyRoot() string {
	k := "/calico/v1/host"
	if options.Hostname == "" {
		return k
	}
	k = k + fmt.Sprintf("/%s/workload", options.Hostname)
	if options.OrchestratorID == "" {
		return k
	}
	k = k + fmt.Sprintf("/%s", options.OrchestratorID)
	if options.WorkloadID == "" {
		return k
	}
	k = k + fmt.Sprintf("/%s/endpoint", options.WorkloadID)
	if options.EndpointID == "" {
		return k
	}
	k = k + fmt.Sprintf("/%s", options.EndpointID)
	return k
}

func (options WorkloadEndpointListOptions) keyFromEtcdResult(ekey string) KeyInterface {
	glog.V(2).Infof("Get WorkloadEndpoint key from %s", ekey)
	r := matchWorkloadEndpoint.FindAllStringSubmatch(ekey, -1)
	if len(r) != 1 {
		glog.V(2).Infof("Didn't match regex")
		return nil
	}
	hostname := r[0][1]
	orch := r[0][2]
	workload := r[0][3]
	endpointID := r[0][4]
	if options.Hostname != "" && hostname != options.Hostname {
		glog.V(2).Infof("Didn't match hostname %s != %s", options.Hostname, hostname)
		return nil
	}
	if options.OrchestratorID != "" && orch != options.OrchestratorID {
		glog.V(2).Infof("Didn't match orchestrator %s != %s", options.OrchestratorID, orch)
		return nil
	}
	if options.WorkloadID != "" && orch != options.WorkloadID {
		glog.V(2).Infof("Didn't match workload %s != %s", options.WorkloadID, workload)
		return nil
	}
	if options.EndpointID != "" && endpointID != options.EndpointID {
		glog.V(2).Infof("Didn't match endpoint ID %s != %s", options.EndpointID, endpointID)
		return nil
	}
	return WorkloadEndpointKey{Hostname: hostname, EndpointID: endpointID}
}

type WorkloadEndpoint struct {
	WorkloadEndpointKey `json:"-"`
	// TODO: Validation for workload endpoint.
	State     string            `json:"state"`
	Name      string            `json:"name"`
	Mac       string            `json:"mac"`
	ProfileID []string          `json:"profile_ids"`
	IPv4Nets  []string          `json:"ipv4_nets"`
	IPv6Nets  []string          `json:"ipv6_nets"`
	Labels    map[string]string `json:"labels,omitempty"`
}
