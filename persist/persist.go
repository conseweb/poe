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
package persist

import (
	"time"

	"github.com/conseweb/common/semaphore"
	"github.com/conseweb/poe/cache"
	"github.com/conseweb/poe/persist/cassandra"
	"github.com/conseweb/poe/persist/fake"
	"github.com/conseweb/poe/persist/leveldb"
	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	default_persister_name  = "leveldb"
	default_persistChan_cap = 10000
	persistLogger           = logging.MustGetLogger("persist")
)

// PersistInterface
type PersistInterface interface {

	// PutDocsIntoDB puts documents into database
	PutDocsIntoDB(docs []*protos.Document) error

	// GetDocFromDBByDocID get document info from database based on document id
	GetDocFromDBByDocID(docID string) (*protos.Document, error)

	// SetDocsBlockDigest sets documents's blockDigest, which indicate where documents belongs to
	SetDocsBlockDigest(docIDs []string, digest string) error

	// FindDocsByBlockDigest finds documents belong to a same digest
	FindDocsByBlockDigest(digest string) ([]*protos.Document, error)

	// FindDocsByHash
	FindDocsByHash(hash string) ([]*protos.Document, error)

	// Close closes persister
	Close() error
}

// NewPersister returns a persister based on config
func NewPersister(cc cache.CacheInterface) PersistInterface {
	flogging.LoggingInit("persist")

	persisterName := viper.GetString("persist.type")
	if persisterName == "" {
		persisterName = default_persister_name
	}

	switch persisterName {
	case "fake":
		persister = fake.NewFakePersister()
	case "cassandra":
		persister = cassandra.NewCassandraPersister()
	case "leveldb":
		persister = leveldb.NewLevelDBPersister()
	default:
		persistLogger.Fatalf("unsupported persist type %s", persisterName)
	}
	go continuePersist(cc)

	return persister
}

var (
	docsChan  chan *protos.Document
	docQueue  []*protos.Document
	persister PersistInterface
)

// pull data from cache, push data into database
func continuePersist(cc cache.CacheInterface) {
	persisterName := "persister"
	// cache customer subscribe cache topic
	periodLimits := utils.GetPeriodLimits()
	// semaphore control
	workerCtrl := semaphore.NewSemaphore(viper.GetInt("persist.workers"))

	// get documents from cache
	getDocumentsFromCache := func(dc chan<- *protos.Document) {
		for _, period := range periodLimits {
			go func(period *utils.PeriodLimit, dc chan<- *protos.Document) {
				topic := cc.Topic(period.Period)
				docs, err := cc.Get(persisterName, topic, period.Limit)
				if err != nil {
					persistLogger.Warningf("get topic %s documents from cache return error: %v", topic, err)
					return
				}

				for idx, doc := range docs {
					persistLogger.Debugf("doc[%d] from cache: %v", idx, doc.Id)
					dc <- doc
				}
			}(period, dc)
		}
	}

	// put documents into database
	putDocumentsIntoDB := func(docs []*protos.Document) {
		dstDocs := make([]*protos.Document, len(docs))
		copy(dstDocs, docs)

		workerCtrl.Acquire()
		go func() {
			defer workerCtrl.Release()

			persister.PutDocsIntoDB(dstDocs)
		}()
	}

	// internal documents transfer chan
	chanCap := viper.GetInt("persist.chancap")
	if chanCap <= 0 {
		chanCap = default_persistChan_cap
	}
	docsChan = make(chan *protos.Document, chanCap)
	cacheCheckTicker := time.NewTicker(viper.GetDuration("persist.cacheCheckInterval"))

	// db write batch
	docQueue = make([]*protos.Document, 0)
	queueSize := viper.GetInt("persist.queueSize")
	queueTimeoutTicker := time.NewTicker(viper.GetDuration("persist.queueTimeout"))

	persistLogger.Info("persister is persisting documents into db from cache")
	for {
		select {
		case <-cacheCheckTicker.C:
			workerCtrl.Acquire()
			go func() {
				defer workerCtrl.Release()
				getDocumentsFromCache(docsChan)
			}()
		case doc := <-docsChan:
			docQueue = append(docQueue, doc)
			if len(docQueue) < queueSize {
				continue
			}

			persistLogger.Debugf("documents queue is full, put %d documents into DB", len(docQueue))
			putDocumentsIntoDB(docQueue)
			docQueue = make([]*protos.Document, 0)
		case <-queueTimeoutTicker.C:
			if len(docQueue) == 0 {
				continue
			}

			persistLogger.Debugf("documents queue timeout, put %d documents into DB", len(docQueue))
			putDocumentsIntoDB(docQueue)
			docQueue = make([]*protos.Document, 0)
		}
	}
}

// Close close persister
func Close() error {
	persistLogger.Info("persister is stopping...")
	defer persistLogger.Info("persister is stopped.")

	for {
		if len(docsChan) > 0 || len(docQueue) > 0 {
			time.Sleep(time.Millisecond * 50)
			continue
		}

		break
	}

	return persister.Close()
}
