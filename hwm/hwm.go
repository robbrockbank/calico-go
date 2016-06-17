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

package hwm

import (
	"gopkg.in/tchap/go-patricia.v2/patricia"
	"fmt"
)

type HighWatermarkTracker struct {
	hwms         *patricia.Trie
	deletionHwms *patricia.Trie
	deletionHwm  uint64
}

func NewHighWatermarkTracker() *HighWatermarkTracker {
	trie := new(HighWatermarkTracker)
	trie.hwms = patricia.NewTrie()
	trie.deletionHwms = nil // No deletion tracking in progress
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
	if oldHwmOrNil != nil {
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

func (trie *HighWatermarkTracker) DeleteOldKeys(hwmLimit uint64) []string {
	if trie.deletionHwms != nil {
		panic("Deletion tracking not compatible with DeleteOldKeys")
	}
	deletedPrefixes := make([]patricia.Prefix, 0, 100)
	trie.hwms.Visit(func(prefix patricia.Prefix, item patricia.Item) error {
		if prefix == nil {
			panic("nil prefix passed to visitor")
		}
		if item.(uint64) < hwmLimit {
			deletedPrefixes = append(deletedPrefixes, prefix)
		}
		return nil
	})
	deletedKeys := make([]string, 0, len(deletedPrefixes))
	for _, childPrefix := range deletedPrefixes {
		fmt.Printf("Prefix: %v\n", childPrefix)
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
