package main

import (
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/docopt/docopt-go"
	"golang.org/x/net/context"
	"gopkg.in/vmihailenco/msgpack.v2"
	"gopkg.in/tchap/go-patricia.v2/patricia"
	"net"
)

const usage = `etcd driver.

Usage:
  etcd-driver <felix-socket>`

const (
	actionSet uint8 = iota
	actionDel
	actionSnapFinished
)

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

	// Start background threads to read/write from/to the felix socket.
	go readMessagesFromFelix(felixDecoder, toFelix)
	go sendMessagesToFelix(felixEncoder, toFelix)

	// Start a background thread to read events from etcd.  It will
	// queue events onto the etcdEvents channel.  If it drops out of sync,
	// it will signal on the resyncIndex channel.
	etcdEvents := make(chan event, 20000)
	resyncIndex := make(chan uint64, 5)
	go watchEtcd(etcdEvents, resyncIndex)

	// Start a background thread to read snapshots from etcd.  If will
	// Read a start-of-day snapshot and then wait to be signalled on the
	// resyncIndex channel.
	snapshotUpdates := make(chan event)
	go readSnapshotsFromEtcd(snapshotUpdates, resyncIndex)

	mergeUpdates(snapshotUpdates, etcdEvents, toFelix)
}

func readSnapshotsFromEtcd(snapshotUpdates chan <- event, resyncIndex <-chan uint64) {
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
	for {
		// Wait for the watcher thread to tell us what index it starts
		// from.  We need to load a snapshot with an equal or later
		// index, otherwise we could miss some updates.  (Since we
		// may connect to a follower server, it's possible, if
		// unlikely, for us to read a stale snapshot.)
		minIndex := <-resyncIndex

		// In case we keep resyncing, drain the queue.
		for {
			select {
			case minIndex = <-resyncIndex:
			default:
				break
			}
		}

		for {
			resp, err := kapi.Get(context.Background(), "/", &getOpts)
			if err != nil {
				println("Error getting snapshot, retrying...", err)
				time.Sleep(1 * time.Second)
			} else {
				if resp.Index < minIndex {
					println("Retrieved stale snapshot, rereading...")
					continue
				}

				// If we get here, we should have a good snapshot.
				sendNode(resp.Node, snapshotUpdates, resp)
				snapshotUpdates <- event{
					action:actionSnapFinished,
				}
				break
			}
		}
	}
}

func sendNode(node *client.Node, snapshotUpdates chan <- event, resp *client.Response) {
	if !node.Dir {
		snapshotUpdates <- event{
			key:           node.Key,
			modifiedIndex: resp.Index,
			valueOrNil:    node.Value,
			action:        actionSet,
		}
	} else {
		for _, child := range node.Nodes {
			sendNode(child, snapshotUpdates, resp)
		}
	}
}

func watchEtcd(etcdEvents chan <- event, resyncIndex chan <- uint64) {
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
	watcher := kapi.Watcher("/", &watcherOpts)
	lostSync := true
	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			errCode := err.(client.Error).Code
			if errCode == client.ErrorCodeWatcherCleared ||
			errCode == client.ErrorCodeEventIndexCleared {
				println("Lost sync with etcd, restarting watcher")
				watcher = kapi.Watcher("/", &watcherOpts)
				lostSync = true
			} else {
				fmt.Printf("Error from etcd %v", err)
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
				// new snapshot.
				resyncIndex <- node.ModifiedIndex
				lostSync = false
			}
			etcdEvents <- event{
				action:        actionType,
				modifiedIndex: node.ModifiedIndex,
				key:           resp.Node.Key,
				valueOrNil:    node.Value,
				snapshotStarting: lostSync,
			}
		}
	}
}

func readMessagesFromFelix(felixDecoder *msgpack.Decoder,
toFelix chan <- map[string]interface{}) {
	for {
		msg, err := felixDecoder.DecodeMap()
		if err != nil {
			panic("Error reading from felix")
		}
		msgType := msg.(map[interface{}]interface{})["type"].(string)
		switch msgType {
		case "init": // Hello message from felix
			// Respond with config.
			fmt.Println("Init message from felix, responding " +
			"with config.")
			rsp := map[string]interface{}{
				"type": "config_loaded",
				"global": map[string]string{
					"InterfacePrefix": "tap",
				},
				"host": map[string]string{},
			}
			toFelix <- rsp
		default:
			fmt.Println("XXXX Unknown message from felix:", msg)
		}
	}
}

func sendMessagesToFelix(felixEncoder *msgpack.Encoder,
toFelix <-chan map[string]interface{}) {
	for {
		msg := <-toFelix
		fmt.Println("Writing msg to felix", msg)
		if err := felixEncoder.Encode(msg); err != nil {
			panic("Failed to send message to felix")
		}
		fmt.Println("Wrote msg to felix", msg)
	}
}

type event struct {
	action           uint8
	modifiedIndex    uint64
	key              string
	valueOrNil       string
	snapshotStarting bool
	snapshotFinished bool
}

func mergeUpdates(snapshotUpdates <-chan event, watcherUpdates <-chan event,
toFelix chan <- map[string]interface{}) {
	var e event
	//var minSnapshotIndex uint64
	for {
		select {
		case e = <-snapshotUpdates:
		case e = <-watcherUpdates:
		}
		if e.snapshotStarting {
			// Watcher lost sync, need to track deletions until
			// we get a snapshot from after this index.
			//minSnapshotIndex = e.modifiedIndex
		}
		if e.action == actionSet {
			toFelix <- map[string]interface{}{
				"type": "u",
				"k": e.key,
				"v": e.valueOrNil,
			}
		}
	}
}

type HighWatermarkTracker struct {
	hwms         *patricia.Trie
	deletionHwms *patricia.Trie
	deletionHwm  uint64
}

func NewHighWatermarkTracker() (*HighWatermarkTracker) {
	trie := new(HighWatermarkTracker)
	trie.hwms = patricia.NewTrie()
	trie.deletionHwms = nil  // No deletion tracking in progress
	return trie
}

func (trie *HighWatermarkTracker) StartTrackingDeletions() {
	trie.deletionHwms = patricia.NewTrie()
	trie.deletionHwm = 0
}

func (trie *HighWatermarkTracker) StopTrackingDeletions() {
	trie.deletionHwms = nil
	trie.deletionHwm = 0
}

func (trie *HighWatermarkTracker) StoreUpdate(key string, newModIdx uint64) uint64 {
	if trie.deletionHwms != nil {
		// Optimization: only check if this key is in the deletion
		// trie if we've seen at least one deletion since...
		if newModIdx < trie.deletionHwm {
			_, delHwm := findLongestPrefix(trie.deletionHwms, key)
			if delHwm != nil {
				if newModIdx < delHwm.(uint64) {
					return delHwm.(uint64)
				}
			}
		}
	}

	// Get the old value
	oldHwmOrNil := trie.hwms.Get(patricia.Prefix(key))
	if oldHwmOrNil != nil {
		oldHwm := oldHwmOrNil.(uint64)
		if oldHwm < newModIdx {
			trie.hwms.Set(patricia.Prefix(key), newModIdx)
		}
	} else {
		trie.hwms.Set(patricia.Prefix(key), newModIdx)
	}
	if  oldHwmOrNil != nil {
		return oldHwmOrNil.(uint64)
	} else {
		return 0
	}
}

func (trie *HighWatermarkTracker) StoreDeletion(key string, newModIdx uint64) []string {
	if newModIdx > trie.deletionHwm {
		trie.deletionHwm = newModIdx
	}
	prefix := patricia.Prefix(key)
	if trie.deletionHwms != nil {
		trie.deletionHwms.Set(prefix, newModIdx)
	}
	deletedKeys := make([]string, 1)
	trie.hwms.VisitSubtree(prefix, func(prefix patricia.Prefix, item patricia.Item) error {
		childKey := string(prefix)
		deletedKeys = append(deletedKeys, childKey)
		return nil
	})
	return deletedKeys
}

func (trie *HighWatermarkTracker) DeleteOldKeys(mwnLimit uint64) []string {
	if trie.deletionHwms != nil {
		panic("Deletion tracking not compatible with DeleteOldKeys")
	}
	deletedPrefixes := make([]patricia.Prefix, 100)
	trie.hwms.Visit(func(prefix patricia.Prefix, item patricia.Item) error {
		deletedPrefixes = append(deletedPrefixes, prefix)
		return nil
	})
	deletedKeys := make([]string, len(deletedPrefixes))
	for _, childPrefix := range deletedPrefixes {
		deletedKeys = append(deletedKeys, string(childPrefix))
		trie.hwms.Delete(childPrefix)
	}
	return deletedKeys
}

func findLongestPrefix(trie *patricia.Trie, key string) (patricia.Prefix, patricia.Item) {
	var longestPrefix patricia.Prefix
	var longestItem patricia.Item

	trie.VisitPrefixes(patricia.Prefix(key),
		func(prefix patricia.Prefix, item patricia.Item) error {
			if len(prefix) > len(longestPrefix) {
				longestPrefix = prefix
				longestItem = item
			}
			return nil
		})
	return longestPrefix, longestItem
}



