package main

import (
	"log"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

func main() {
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

	getOpts := client.GetOptions{
		Recursive: true,
		Sort:      false,
		Quorum:    false,
	}
	resp, err := kapi.Get(context.Background(), "/calico/v1/", &getOpts)
	if err != nil {
		log.Fatal("Error getting snapshot")
	} else {
		// print common key info
		log.Printf("Get is done. Metadata is %q\n", resp)
	}
}

type Event struct {
	modifiedIndex int
	key           string
	valueOrNil    string
}

func mergeUpdates() {

}
