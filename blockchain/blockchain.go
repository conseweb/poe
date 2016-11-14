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
	"sort"
	"strings"
	"time"

	"github.com/conseweb/common/crypto"
	"github.com/conseweb/poe/cache"
	"github.com/conseweb/poe/persist"
	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"golang.org/x/crypto/sha3"
)

var (
	blockchainLogger = logging.MustGetLogger("blockchain")
)

// Blockchain
type Blockchain struct {
	name        string
	chaincodeID string
	peers       []string
	cacher      cache.CacheInterface
	persister   persist.PersistInterface
}

// NewBlockchain returns a blockchain handler
func NewBlockchain(cc cache.CacheInterface, persister persist.PersistInterface) *Blockchain {
	bc := &Blockchain{
		name:        "blockchain",
		chaincodeID: viper.GetString("blockchain.chaincodeId"),
		peers:       viper.GetStringSlice("blockchain.peers"),
		cacher:      cc,
		persister:   persister,
	}
	go bc.continueProof()

	return bc
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

	// TODO connect to chaincode peers to verify
	blockchainLogger.Debug("verify documents return true")
	return true
}

// RegisterProof
func (bc *Blockchain) RegisterProof(docs []*protos.Document) {
	formatedDocs := bc.formatDocs(docs)
	if formatedDocs == "" {
		return
	}

	// TODO put formatedDocs into chaincode, and query proof key, hash the key as documents block digest, send to persister
	proofKey := "sdjfoiwejflsjfoiwejflsf"

	docIds := make([]string, len(docs))
	for idx, doc := range docs {
		docIds[idx] = doc.Id
	}
	go bc.persister.SetDocsBlockDigest(docIds, crypto.Hash(sha3.New512(), []byte(proofKey)))
}

func (bc *Blockchain) continueProof() {
	blockchainLogger.Info("blockchain is ready to proof documents")

	// get documents from cache
	getDocsFromCache := func(period *utils.PeriodLimit) []*protos.Document {
		topic := bc.cacher.Topic(period.Period)
		blockchainLogger.Infof("blockchain get topic[%s] documents", topic)
		docs, err := bc.cacher.Get(bc.name, topic, period.Limit)
		if err != nil {
			//blockchainLogger.Warningf("get documents from cache return error: %v", err)
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
