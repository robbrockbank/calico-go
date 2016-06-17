// Copyright (c) 2016 Tigera Inc.
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
	"github.com/projectcalico/calico-go/datastore"
	"github.com/projectcalico/calico-go/datastore/etcd"
	"gopkg.in/vmihailenco/msgpack.v2"
	"net"
)

const usage = `etcd driver.

Usage:
  etcd-driver <felix-socket>`

func main() {
	// Parse command-line args.
	arguments, err := docopt.Parse(usage, nil, true, "etcd-driver 0.1", false)
	if err != nil {
		panic(usage)
	}
	felixSckAddr := arguments["<felix-socket>"].(string)

	// Connect to Felix.
	felixConn, err := net.Dial("unix", felixSckAddr)
	if err != nil {
		panic("Failed to connect to felix")
	}
	fmt.Println("Felix connection:", felixConn)
	felixDecoder := msgpack.NewDecoder(felixConn)
	felixEncoder := msgpack.NewEncoder(felixConn)

	// Channel to queue messages to felix.
	toFelix := make(chan map[string]interface{})

	// Get an etcd driver
	felixCbs := felixCallbacks{
		toFelix: toFelix,
	}
	datastore, err := etcd.New(felixCbs, &datastore.DriverConfiguration{})

	// Start background threads to read/write from/to the felix socket.
	go readMessagesFromFelix(felixDecoder, datastore)
	go sendMessagesToFelix(felixEncoder, toFelix)
}

type felixCallbacks struct {
	toFelix chan<- map[string]interface{}
}

func (cbs felixCallbacks) OnConfigLoaded() {
	msg := map[string]interface{}{
		"type": "config_loaded",
		"global": map[string]string{
			"InterfacePrefix": "tap",
		},
		"host": map[string]string{},
	}
	cbs.toFelix <- msg
}

func (cbs felixCallbacks) OnStatusUpdated(status datastore.DriverStatus) {
	statusString := "unknown"
	switch status {
	case datastore.WaitForDatastore: statusString = "wait-for-ready"
	case datastore.InSync: statusString = "in-sync"
	case datastore.ResyncInProgress: statusString = "resync"
	}
	msg := map[string]interface{}{
		"type": "stat",
		"status": statusString,
	}
	cbs.toFelix <- msg
}

func (cbs felixCallbacks) OnKeyUpdated(key string, value string) {
	msg := map[string]interface{}{
		"type": "u",
		"k": key,
		"v": value,
	}
	cbs.toFelix <- msg
}

func (cbs felixCallbacks) OnKeyDeleted(key string) {
	msg := map[string]interface{}{
		"type": "u",
		"k": key,
		"v": nil,
	}
	cbs.toFelix <- msg
}

func readMessagesFromFelix(felixDecoder *msgpack.Decoder, datastore datastore.Driver) {
	for {
		msg, err := felixDecoder.DecodeMap()
		if err != nil {
			panic("Error reading from felix")
		}
		msgType := msg.(map[interface{}]interface{})["type"].(string)
		switch msgType {
		case "init": // Hello message from felix
			datastore.Start()  // Should trigger OnConfigLoaded.
		default:
			fmt.Println("XXXX Unknown message from felix:", msg)
		}
	}
}

func sendMessagesToFelix(felixEncoder *msgpack.Encoder,
	toFelix <-chan map[string]interface{}) {
	for {
		msg := <-toFelix
		//fmt.Println("Writing msg to felix", msg)
		if err := felixEncoder.Encode(msg); err != nil {
			panic("Failed to send message to felix")
		}
		//fmt.Println("Wrote msg to felix", msg)
	}
}
