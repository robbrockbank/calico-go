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

package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
	"github.com/projectcalico/calico-go/ipsets"
	"github.com/projectcalico/calico-go/lib"
	"github.com/projectcalico/calico-go/store"
	"github.com/projectcalico/calico-go/store/etcd"
	"gopkg.in/vmihailenco/msgpack.v2"
	"net"
	"strings"
)

const usage = `etcd driver.

Usage:
  etcd-driver <felix-socket>`

var log = logging.MustGetLogger("etcd-driver")

func main() {
	// Parse command-line args.
	arguments, err := docopt.Parse(usage, nil, true, "etcd-driver 0.1", false)
	if err != nil {
		panic(usage)
	}
	felixSckAddr := arguments["<felix-socket>"].(string)

	logging.SetFormatter(logging.GlogFormatter)
	logging.SetLevel(logging.INFO, "")

	// Connect to Felix.
	felixConn, err := net.Dial("unix", felixSckAddr)
	if err != nil {
		panic("Failed to connect to felix")
	}
	fmt.Println("Felix connection:", felixConn)

	// Wrap Felix socket in msgpack encoder/decoder.
	felixDecoder := msgpack.NewDecoder(felixConn)
	felixEncoder := msgpack.NewEncoder(felixConn)

	// Multiple threads need to write to Felix so we use a channel to send
	// messages to the single writer thread.
	toFelix := make(chan map[string]interface{})

	ipsetResolver := ipsets.NewResolver()

	dispatcher := store.NewDispatcher()
	dispatcher.OnEndpointUpdate = ipsetResolver.OnEndpointUpdate
	dispatcher.OnEndpointDelete = ipsetResolver.OnEndpointDelete
	dispatcher.OnEndpointParseFailure = func(key libcalico.EndpointKey, err error) {
		log.Warningf("Failed to parse endpoint %v: %v. "+
			"Treating as deletion.", key, err)
		ipsetResolver.OnEndpointDelete(key)
	}

	dispatcher.OnPolicyUpdate = ipsetResolver.OnPolicyUpdate
	dispatcher.OnPolicyDelete = ipsetResolver.OnPolicyDelete
	dispatcher.OnPolicyParseFailure = func(key libcalico.PolicyKey, err error) {
		log.Warningf("Failed to parse policy %v: %v. "+
			"Treating as deletion.", key, err)
		ipsetResolver.OnPolicyDelete(key)
	}

	// TODO: Profile update

	// Get an etcd driver
	felixCbs := &felixCallbacks{
		toFelix:    toFelix,
		dispatcher: dispatcher,
	}
	datastore, err := etcd.New(felixCbs, &store.DriverConfiguration{})

	// TODO callback functions or callback interface?
	ipsetResolver.OnSelectorAdded = felixCbs.onSelectorAdded
	ipsetResolver.OnSelectorRemoved = felixCbs.onSelectorRemoved
	ipsetResolver.OnIPAdded = felixCbs.onIPAddedToSelector
	ipsetResolver.OnIPRemoved = felixCbs.onIPRemovedFromSelector

	// Start background thread to read messages from Felix.
	go readMessagesFromFelix(felixDecoder, datastore)

	// Use main thread for writing to Felix.
	sendMessagesToFelix(felixEncoder, toFelix)
}

type felixCallbacks struct {
	toFelix    chan<- map[string]interface{}
	dispatcher *store.Dispatcher
}

func (cbs *felixCallbacks) onSelectorAdded(selID string) {
	msg := map[string]interface{}{
		"type":   "sel_added",
		"sel_id": selID,
	}
	cbs.toFelix <- msg
}

func (cbs *felixCallbacks) onSelectorRemoved(selID string) {
	msg := map[string]interface{}{
		"type":   "sel_removed",
		"sel_id": selID,
	}
	cbs.toFelix <- msg
}

func (cbs *felixCallbacks) onIPAddedToSelector(selID string, ip string) {
	msg := map[string]interface{}{
		"type":   "ip_added",
		"sel_id": selID,
		"ip":     ip,
	}
	cbs.toFelix <- msg
}
func (cbs *felixCallbacks) onIPRemovedFromSelector(selID string, ip string) {
	msg := map[string]interface{}{
		"type":   "ip_removed",
		"sel_id": selID,
		"ip":     ip,
	}
	cbs.toFelix <- msg
}

func (cbs *felixCallbacks) OnConfigLoaded(globalConfig map[string]string, hostConfig map[string]string) {
	msg := map[string]interface{}{
		"type":   "config_loaded",
		"global": globalConfig,
		"host":   hostConfig,
	}
	cbs.toFelix <- msg
}

func (cbs *felixCallbacks) OnStatusUpdated(status store.DriverStatus) {
	statusString := "unknown"
	switch status {
	case store.WaitForDatastore:
		statusString = "wait-for-ready"
	case store.InSync:
		statusString = "in-sync"
	case store.ResyncInProgress:
		statusString = "resync"
	}
	log.Infof("Datastore status updated to %v: %v", status, statusString)
	msg := map[string]interface{}{
		"type":   "stat",
		"status": statusString,
	}
	cbs.toFelix <- msg
}

func (cbs *felixCallbacks) OnKeysUpdated(updates []store.Update) {
	for _, update := range updates {
		cbs.dispatcher.DispatchUpdate(&update)
		var msg map[string]interface{}
		if len(update.Key) == 0 {
			panic("Empty key")
		}

		if strings.Index(update.Key, "/calico/v1/host/") == 0 {
			if strings.Contains(update.Key, "/endpoint/") {
				if !strings.Contains(update.Key, "smc-ubuntu") {
					continue
				}
			}
		}

		if update.ValueOrNil == nil {
			msg = map[string]interface{}{
				"type": "u",
				"k":    update.Key,
				"v":    nil,
			}
		} else {
			// Deref the value so that we get better diags if the
			// message is traced out.
			msg = map[string]interface{}{
				"type": "u",
				"k":    update.Key,
				"v":    *update.ValueOrNil,
			}
		}
		cbs.toFelix <- msg
	}
}

func readMessagesFromFelix(felixDecoder *msgpack.Decoder, datastore store.Driver) {
	for {
		msg, err := felixDecoder.DecodeMap()
		if err != nil {
			panic("Error reading from felix")
		}
		msgType := msg.(map[interface{}]interface{})["type"].(string)
		switch msgType {
		case "init": // Hello message from felix
			datastore.Start() // Should trigger OnConfigLoaded.
		default:
			fmt.Println("XXXX Unknown message from felix:", msg)
		}
	}
}

func sendMessagesToFelix(felixEncoder *msgpack.Encoder,
	toFelix <-chan map[string]interface{}) {
	for {
		msg := <-toFelix
		log.Debugf("Writing msg to felix: %#v\n", msg)
		if err := felixEncoder.Encode(msg); err != nil {
			panic("Failed to send message to felix")
		}
	}
}
