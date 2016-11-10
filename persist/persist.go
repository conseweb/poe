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
	"github.com/conseweb/poe/persist/fake"
	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	default_persister_name  = ""
	default_persistChan_cap = 10000
	persistLogger           = logging.MustGetLogger("persist")
)

// PersistInterface
type PersistInterface interface {

	// GetPersisterName get persister name
	GetPersisterName() string

	// PutDocsIntoDB puts documents into database
	PutDocsIntoDB(docs []*protos.Document) error

	// GetDocFromDBByDocID get document info from database based on document id
	GetDocFromDBByDocID(docID string) (*protos.Document, error)

	// SetDocsBlockDigest sets documents's blockDigest, which indicate where documents belongs to
	SetDocsBlockDigest(docIDs []string, digest string) error

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
	default:
		persistLogger.Fatalf("unsupported persist type %s", persisterName)
	}
	go continuePersist(cc)

	return persister
}

// ClosePersister close persister
func ClosePersister() error {
	persistLogger.Info("persister is stopping...")
	defer persistLogger.Info("persister is stopped.")

	closeChan <- 1
	for {
		if len(docsChan) > 0 || len(docQueue) > 0 {
			time.Sleep(time.Second)
			continue
		}

		break
	}

	return persister.Close()
}

var (
	docsChan  chan *protos.Document
	docQueue  []*protos.Document
	persister PersistInterface
	closeChan = make(chan int)
)

// pull data from cache, push data into database
func continuePersist(cc cache.CacheInterface) {
	persisterName := persister.GetPersisterName()

	// cache customer subscribe cache topic
	periodLimits := utils.GetPeriodLimits()
	for _, period := range periodLimits {
		cc.Subscribe(persisterName, cc.Topic(period.Period))
	}

	// get documents from cache
	getDocumentsFromCache := func(dc chan<- *protos.Document) {
		for _, period := range periodLimits {
			docs, err := cc.Get(persisterName, cc.Topic(period.Period), period.Limit)
			if err != nil {
				persistLogger.Warningf("get documents from cache return error: %v", err)
				continue
			}

			for _, doc := range docs {
				dc <- doc
			}
		}
	}

	// put documents into database
	putDocumentsIntoDB := func(docs []*protos.Document) {
		persister.PutDocsIntoDB(docs)
	}

	// semaphore control
	workerCtrl := semaphore.NewSemaphore(viper.GetInt("persist.workers"))

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

	persistLogger.Info("the persister is persisting documents into db...")
	for {
		select {
		case <-cacheCheckTicker.C:
			persistLogger.Info("get documents from cacher...")

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
			dstDocs := make([]*protos.Document, len(docQueue))
			copy(dstDocs, docQueue)
			workerCtrl.Acquire()
			go func() {
				defer workerCtrl.Release()
				putDocumentsIntoDB(dstDocs)
			}()
			docQueue = make([]*protos.Document, 0)
		case <-queueTimeoutTicker.C:
			if len(docQueue) == 0 {
				continue
			}

			persistLogger.Debugf("documents queue timeout, put %d documents into DB", len(docQueue))
			dstDocs := make([]*protos.Document, len(docQueue))
			copy(dstDocs, docQueue)
			workerCtrl.Acquire()
			go func() {
				defer workerCtrl.Release()
				putDocumentsIntoDB(dstDocs)
			}()
			docQueue = make([]*protos.Document, 0)
		case <-closeChan:
			close(docsChan)
		}
	}
}
