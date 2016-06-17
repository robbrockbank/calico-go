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

package etcd

import (
	"fmt"
	"github.com/projectcalico/calico-go/datastore"
	"github.com/projectcalico/calico-go/hwm"
	"time"

	"log"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

func init() {
	datastore.Register("etcd", New)
}

func New(callbacks datastore.Callbacks, config *datastore.DriverConfiguration) (datastore.Driver, error) {
	return &etcdDriver{
		callbacks: callbacks,
		config:    config,
	}, nil
}

type etcdDriver struct {
	callbacks datastore.Callbacks
	config    *datastore.DriverConfiguration
}

func (driver *etcdDriver) Start() {
	// Start a background thread to read events from etcd.  It will
	// queue events onto the etcdEvents channel.  If it drops out of sync,
	// it will signal on the resyncIndex channel.
	etcdEvents := make(chan event, 20000)
	resyncIndex := make(chan uint64, 5)
	go driver.watchEtcd(etcdEvents, resyncIndex)

	// Start a background thread to read snapshots from etcd.  It will
	// read a start-of-day snapshot and then wait to be signalled on the
	// resyncIndex channel.
	snapshotUpdates := make(chan event)
	go driver.readSnapshotsFromEtcd(snapshotUpdates, resyncIndex)

	go driver.mergeUpdates(snapshotUpdates, etcdEvents)

	// TODO actually send some config
	driver.callbacks.OnConfigLoaded()
}

const (
	actionSet uint8 = iota
	actionDel
	actionSnapFinished
)

type event struct {
	action           uint8
	modifiedIndex    uint64
	snapshotIndex    uint64
	key              string
	valueOrNil       string
	snapshotStarting bool
	snapshotFinished bool
}

func (driver *etcdDriver) readSnapshotsFromEtcd(snapshotUpdates chan<- event, resyncIndex <-chan uint64) {
	cfg := client.Config{
		Endpoints:               []string{"http://127.0.0.1:2379"},
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: 10 * time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	kapi := client.NewKeysAPI(c)
	getOpts := client.GetOptions{
		Recursive: true,
		Sort:      false,
		Quorum:    false,
	}
	var highestSnapshotIndex uint64
	var minIndex uint64
	for {
		if highestSnapshotIndex > 0 {
			// Wait for the watcher thread to tell us what index
			// it starts from.  We need to load a snapshot with
			// an equal or later index, otherwise we could miss
			// some updates.  (Since we may connect to a follower
			// server, it's possible, if unlikely, for us to read
			// a stale snapshot.)
			minIndex = <-resyncIndex
			fmt.Printf("Asked for snapshot > %v\n", minIndex)
			if highestSnapshotIndex >= minIndex {
				// We've already read a newer snapshot, no
				// need to re-read.
				continue
			}
		}

	readRetryLoop:
		for {
			resp, err := kapi.Get(context.Background(),
				"/calico/v1", &getOpts)
			if err != nil {
				println("Error getting snapshot, retrying...", err)
				time.Sleep(1 * time.Second)
				continue readRetryLoop
			}

			if resp.Index < minIndex {
				println("Retrieved stale snapshot, rereading...")
				continue readRetryLoop
			}

			// If we get here, we should have a good
			// snapshot.  Send it to the merge thread.
			sendNode(resp.Node, snapshotUpdates, resp)
			snapshotUpdates <- event{
				action:        actionSnapFinished,
				snapshotIndex: resp.Index,
			}
			if resp.Index > highestSnapshotIndex {
				highestSnapshotIndex = resp.Index
			}
			break readRetryLoop

		}
	}
}

func sendNode(node *client.Node, snapshotUpdates chan<- event, resp *client.Response) {
	if !node.Dir {
		snapshotUpdates <- event{
			key:           node.Key,
			modifiedIndex: node.ModifiedIndex,
			snapshotIndex: resp.Index,
			valueOrNil:    node.Value,
			action:        actionSet,
		}
	} else {
		for _, child := range node.Nodes {
			sendNode(child, snapshotUpdates, resp)
		}
	}
}

func (driver *etcdDriver) watchEtcd(etcdEvents chan<- event, resyncIndex chan<- uint64) {
	cfg := client.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	kapi := client.NewKeysAPI(c)

	watcherOpts := client.WatcherOptions{
		AfterIndex: 0, // Start at current index.
		Recursive:  true,
	}
	watcher := kapi.Watcher("/calico/v1", &watcherOpts)
	lostSync := true
	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			errCode := err.(client.Error).Code
			if errCode == client.ErrorCodeWatcherCleared ||
				errCode == client.ErrorCodeEventIndexCleared {
				println("Lost sync with etcd, restarting watcher")
				watcher = kapi.Watcher("/calico/v1",
					&watcherOpts)
				lostSync = true
			} else {
				fmt.Printf("Error from etcd %v\n", err)
				time.Sleep(1 * time.Second)
			}
		} else {
			var actionType uint8
			switch resp.Action {
			case "set", "compareAndSwap", "update", "create":
				actionType = actionSet
			case "delete", "compareAndDelete", "expire":
				actionType = actionDel
			default:
				panic("Unknown action type")
			}

			node := resp.Node
			if node.Dir && actionType == actionSet {
				continue
			}
			if lostSync {
				// Tell the snapshot thread that we need a
				// new snapshot.  The snapshot needs to be
				// from our index or one lower.
				resyncIndex <- node.ModifiedIndex - 1
				lostSync = false
			}
			etcdEvents <- event{
				action:           actionType,
				modifiedIndex:    node.ModifiedIndex,
				key:              resp.Node.Key,
				valueOrNil:       node.Value,
				snapshotStarting: lostSync,
			}
		}
	}
}

func (driver *etcdDriver) mergeUpdates(snapshotUpdates <-chan event, watcherUpdates <-chan event) {
	var e event
	var minSnapshotIndex uint64
	hwms := hwm.NewHighWatermarkTracker()
	for {
		select {
		case e = <-snapshotUpdates:
		//fmt.Printf("Snapshot update %v @ %v\n", e.key, e.modifiedIndex)
		case e = <-watcherUpdates:
			//fmt.Printf("Watcher update %v @ %v\n", e.key, e.modifiedIndex)
		}
		if e.snapshotStarting {
			// Watcher lost sync, need to track deletions until
			// we get a snapshot from after this index.
			minSnapshotIndex = e.modifiedIndex
		}
		switch e.action {
		case actionSet:
			var indexToStore uint64
			if e.snapshotIndex != 0 {
				// Store the snapshot index in the trie so that
				// we can scan the trie later looking for
				// prefixes that are older than the snapshot
				// (and hence must have been deleted while
				// we were out-of-sync).
				indexToStore = e.snapshotIndex
			} else {
				indexToStore = e.modifiedIndex
			}
			oldIdx := hwms.StoreUpdate(e.key, indexToStore)
			//fmt.Printf("%v update %v -> %v\n",
			//	e.key, oldIdx, e.modifiedIndex)
			if oldIdx < e.modifiedIndex {
				// Event is newer than value for that key.
				// Send the update to Felix.
				driver.callbacks.OnKeyUpdated(e.key, e.valueOrNil)
			}
		case actionDel:
			deletedKeys := hwms.StoreDeletion(e.key,
				e.modifiedIndex)
			fmt.Printf("%v deleted; %v keys\n",
				e.key, len(deletedKeys))
			for _, child := range deletedKeys {
				driver.callbacks.OnKeyDeleted(child)
			}
		case actionSnapFinished:
			if e.snapshotIndex >= minSnapshotIndex {
				// Now in sync.
				hwms.StopTrackingDeletions()
				keys := hwms.DeleteOldKeys(e.snapshotIndex)
				fmt.Printf("Snapshot finished at index %v; "+
					"%v keys deleted.\n",
					e.snapshotIndex, len(keys))
				for _, key := range keys {
					driver.callbacks.OnKeyDeleted(key)
				}
			}
		}
	}
}
