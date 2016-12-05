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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NebulousLabs/merkletree"
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
	chainCodeId string          // chaincode 服务唯一标识
	path        string          // chaincode 源码路径
	secureCtx   string          // chaincode 安全上下文，即账户名，默认jim
	balance     string          // 负载均衡算法
	peers       []string        // peer 服务地址列表
	peerBackend backends        // peer 负载均衡器,用于选取一个可用的地址
	events      []string        // peer 事件监听地址列表
	eClis       []*eventsClient // peer 事件集合
	regTimeout  time.Duration   // grpc 超时设置，默认3s
	failOver    int             // grpc 允许链接失败次数
	items       *items          // 存储临时信息，方便异步通讯传参
	cacher      cache.CacheInterface
	persister   persist.PersistInterface
}

// NewBlockchain returns a blockchain handler
func NewBlockchain(cc cache.CacheInterface, persister persist.PersistInterface) (*Blockchain, error) {
	bc := Blockchain{}
	bc.name = "blockchain"
	bc.chainCodeId = viper.GetString("blockchain.chainCodeId")
	if len(strings.TrimSpace(bc.chainCodeId)) == 0 {
		blockchainLogger.Debug("in bc func <NewBlockchain> config item <blockchain.chainCodeId> is not valid,Cannot be empty or contain null characters")
		return nil, errors.New("config item <blockchain.chainCodeId> can not be empty.")
	}
	bc.path = viper.GetString("blockchain.path")
	if len(strings.TrimSpace(bc.path)) == 0 {
		blockchainLogger.Debug("in bc func <NewBlockchain> config item <blockchain.path> is not valid,Cannot be empty or contain null characters")
		return nil, errors.New("config item <blockchain.path> can not be empty.")
	}
	bc.secureCtx = viper.GetString("blockchain.secureCtx")
	if len(strings.TrimSpace(bc.secureCtx)) == 0 {
		blockchainLogger.Debug("in bc func <NewBlockchain> config item <blockchain.secureCtx> is not valid,Cannot be empty or contain null characters")
		return nil, errors.New("config item <blockchain.secureCtx> can not be empty.")
	}
	bc.peers = viper.GetStringSlice("blockchain.peers")
	if len(bc.peers) == 0 {
		blockchainLogger.Debug("in bc func <NewBlockchain> config item <blockchain.peers> is not valid,Cannot be contain null")
		return nil, errors.New("config item <blockchain.peers> can not be empty.")
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
		blockchainLogger.Debug("in bc func <NewBlockchain> config item <blockchain.events> is not valid,Cannot be contain null")
		return nil, errors.New("config item <blockchain.events> can not be empty.")
	}
	bc.items = &items{lock: new(sync.RWMutex), data: make(map[string]interface{})}
	bc.cacher = cc
	bc.persister = persister
	go bc.eventStart()
	go bc.continueProof()
	return &bc, nil
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

	tree := merkletree.New(sha256.New())
	for _, doc := range ds {
		tree.Push([]byte(doc))
	}
	merkletreeRoot := fmt.Sprintf("%x", tree.Root())

	blockchainLogger.Debugf("formated docs: %s", merkletreeRoot)
	return merkletreeRoot
}

// VerifyDocs verify whether documents are unchanged
func (bc *Blockchain) VerifyDocs(docs []*protos.Document) bool {
	formatedDocs := bc.formatDocs(docs)
	if formatedDocs == "" {
		blockchainLogger.Warning("formatedDocs is empty")
		return false
	}
	data, e := bc.execute("query", "existence", []string{"base", formatedDocs})
	if e != nil {
		blockchainLogger.Warningf("chaincode execute query error: %v", e)
		return false
	}

	results := make(map[string]*queryResult)
	if err := json.Unmarshal(data, &results); err != nil {
		blockchainLogger.Warningf("json unmarshal error: %v", err)
		return false
	}
	blockchainLogger.Debugf("verify result: %v", results)
	if result, ok := results[formatedDocs]; ok {
		return result.Exist
	}

	return false
}

// RegisterProof
func (bc *Blockchain) RegisterProof(docs []*protos.Document) {
	formatedDocs := bc.formatDocs(docs)
	if formatedDocs == "" {
		blockchainLogger.Warning("formatedDocs is empty")
		return
	}
	data, e := bc.execute("invoke", "register", []string{"base", formatedDocs})
	if e != nil {
		blockchainLogger.Warningf("chaincode execute invoke error: %v", e)
		return
	}
	bc.items.Set(string(data), docs)
	blockchainLogger.Debugf("chaincode execute txid: %s", string(data))
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
