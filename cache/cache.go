/*
Copyright Mojing Inc. 2016 All Rights Reserved.
Written by mint.zhao.chiu@gmail.com. github.com: https://www.github.com/mintzhao

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cache

import (
	"time"

	"github.com/conseweb/poe/cache/memory"
	"github.com/conseweb/poe/protos"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	default_cache_name = "memory"
	cacheLogger        = logging.MustGetLogger("cache")
)

// cache request
type CacheInterface interface {
	// Put puts raw data into cache, separated by topic.
	Put(raw []byte, topic string) (*protos.Document, error)

	// Get gets documents which related to `topic`, and the number of documents is `count`, if not enough, returns all
	Get(customer, topic string, count int64) ([]*protos.Document, error)

	// once the customer subscribed the topic, they will get documents which related to the topic
	Subscribe(customer, topic string) bool

	// Topic returns which topic can be used based on proof_wait_period
	Topic(d time.Duration) string

	// DocumentID returns
	DocumentID(rawData []byte) string

	// Close closes cache adapter, if error occurs, return it
	Close() error
}

// NewCache returns a new cache adapter
func NewCache() CacheInterface {
	flogging.LoggingInit("cache")

	// cache
	cacheName := viper.GetString("cache.type")
	if cacheName == "" {
		cacheName = default_cache_name
	}
	cacheLogger.Infof("using %s as cache", cacheName)

	switch cacheName {
	case "kafka":
	case "memory":
		return memory.NewMemoryCache()
	}

	cacheLogger.Fatalf("unsupported cache type %s", cacheName)
	return nil
}
