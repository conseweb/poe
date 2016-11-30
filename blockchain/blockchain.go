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
package blockchain

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/conseweb/poe/cache"
	"github.com/conseweb/poe/persist"
	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	blockchainLogger = logging.MustGetLogger("blockchain")
)

// Blockchain
type Blockchain struct {
	name        string
	chainCodeId string
	path        string
	secureCtx   string
	balance     string
	peers       []string
	peerBackend backends
	events      []string
	eClis       []*eventsClient
	regTimeout  time.Duration
	failOver    int
	items       *items
	cacher      cache.CacheInterface
	persister   persist.PersistInterface
}

// NewBlockchain returns a blockchain handler
func NewBlockchain(cc cache.CacheInterface, persister persist.PersistInterface) *Blockchain {
	bc := Blockchain{}
	bc.name = "blockchain"
	bc.chainCodeId = viper.GetString("blockchain.chainCodeId")
	if len(strings.TrimSpace(bc.chainCodeId)) == 0 {
		blockchainLogger.Error("in bc func <NewBlockchain> config item <blockchain.chainCodeId> is not valid,Cannot be empty or contain null characters")
		return nil
	}
	bc.path = viper.GetString("blockchain.path")
	if len(strings.TrimSpace(bc.path)) == 0 {
		blockchainLogger.Error("in bc func <NewBlockchain> config item <blockchain.path> is not valid,Cannot be empty or contain null characters")
		return nil
	}
	bc.secureCtx = viper.GetString("blockchain.secureCtx")
	if len(strings.TrimSpace(bc.secureCtx)) == 0 {
		blockchainLogger.Error("in bc func <NewBlockchain> config item <blockchain.secureCtx> is not valid,Cannot be empty or contain null characters")
		return nil
	}
	bc.peers = viper.GetStringSlice("blockchain.peers")
	if len(bc.peers) == 0 {
		blockchainLogger.Error("in bc func <NewBlockchain> config item <blockchain.peers> is not valid,Cannot be contain null")
		return nil
	}
	bc.balance = viper.GetString("blockchain.balance")
	if len(strings.TrimSpace(bc.balance)) == 0 {
		bc.balance = "round-robin"
	}
	bc.peerBackend = build(bc.balance, bc.peers)
	bc.regTimeout = viper.GetDuration("blockchain.regTimeout")
	if bc.regTimeout.String() == "0s" {
		bc.regTimeout = 3
	}
	bc.failOver = viper.GetInt("blockchain.failover")
	if bc.failOver == 0 {
		bc.failOver = 3
	}
	bc.events = viper.GetStringSlice("blockchain.events")
	if len(bc.events) == 0 {
		blockchainLogger.Error("in bc func <NewBlockchain> config item <blockchain.events> is not valid,Cannot be contain null")
		return nil
	}
	bc.items = &items{lock: new(sync.RWMutex), data: make(map[string]interface{})}
	bc.cacher = cc
	bc.persister = persister
	go bc.eventStart()
	go bc.continueProof()
	return &bc

}

func (bc *Blockchain) formatDocs(docs []*protos.Document) string {
	if len(docs) == 0 {
		return ""
	}

	ds := make([]string, len(docs))
	for idx, doc := range docs {
		ds[idx] = doc.Id
	}
	sort.Strings(ds)

	return strings.Join(ds, "")
}

// VerifyDocs verify whether documents are unchanged
func (bc *Blockchain) VerifyDocs(docs []*protos.Document) bool {
	formatedDocs := bc.formatDocs(docs)
	if formatedDocs == "" {
		return false
	}
	data, e := bc.execute("query", "existence", []string{"base", formatedDocs})
	if e != nil {
		blockchainLogger.Error(e)
		return false
	}
	var result []queryResult
	if e = json.Unmarshal(data, &result); e != nil {
		blockchainLogger.Error(e)
		return false
	}
	if len(result) > 0 {
		return result[0].Exist
	}
	return false
	// TODO connect to chaincode peers to verify
	//blockchainLogger.Debug("verify documents return true")
	//return true
}

// RegisterProof
func (bc *Blockchain) RegisterProof(docs []*protos.Document) {
	formatedDocs := bc.formatDocs(docs)
	if formatedDocs == "" {
		return
	}
	data, e := bc.execute("invoke", "register", []string{"base", formatedDocs})
	if e != nil {
		blockchainLogger.Error(e)
	}
	bc.items.Set(string(data), docs)
	// TODO put formatedDocs into chaincode, and query proof key, hash the key as documents block digest, send to persister
	//proofKey := "sdjfoiwejflsjfoiwejflsf"

	//docIds := make([]string, len(docs))
	//for idx, doc := range docs {
	//	docIds[idx] = doc.Id
	//}
	//go bc.persister.SetDocsBlockDigest(docIds, crypto.Hash(sha3.New512(), []byte(proofKey)))
}

func (bc *Blockchain) continueProof() {
	blockchainLogger.Info("blockchain is ready to proof documents")

	// get documents from cache
	getDocsFromCache := func(period *utils.PeriodLimit) []*protos.Document {
		topic := bc.cacher.Topic(period.Period)
		blockchainLogger.Infof("blockchain get topic[%s] documents", topic)
		docs, err := bc.cacher.Get(bc.name, topic, period.Limit)
		if err != nil {
			blockchainLogger.Warningf("get documents from cache return error: %v", err)
			return nil
		}

		return docs
	}

	// cache customer subscribe cache topic
	periodLimits := utils.GetPeriodLimits()
	for _, period := range periodLimits {
		bc.cacher.Subscribe(bc.name, bc.cacher.Topic(period.Period))
		go func(period *utils.PeriodLimit) {
			ticker := time.NewTicker(period.Period)
			for {
				select {
				case <-ticker.C:
					bc.RegisterProof(getDocsFromCache(period))
				}
			}
		}(period)
	}
}
