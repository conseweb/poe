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
package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
)

// MemoryCache
type MemoryCache struct {
	sync.Mutex

	vals           map[string]map[string]*protos.Document
	customerTopics map[string]map[string]bool
	customerReaded map[string]map[string]map[string]bool
}

// NewMemoryCache returns a memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		vals:           make(map[string]map[string]*protos.Document),
		customerTopics: make(map[string]map[string]bool),
		customerReaded: make(map[string]map[string]map[string]bool),
	}
}

// put value into cache
func (m *MemoryCache) Put(raw []byte, topic string) (*protos.Document, error) {
	m.Lock()
	defer m.Unlock()

	_, ok := m.vals[topic]
	if !ok {
		m.vals[topic] = make(map[string]*protos.Document)
	}

	doc := &protos.Document{
		Id:         m.DocumentID(raw),
		Raw:        raw,
		SubmitTime: time.Now().UTC().Unix(),
	}
	m.vals[topic][doc.Id] = doc

	return doc, nil
}

func (m *MemoryCache) Topic(d time.Duration) string {
	return fmt.Sprintf("memory_%s", d.String())
}

func (m *MemoryCache) DocumentID(rawData []byte) string {
	return utils.DocumentID(rawData)
}

func (m *MemoryCache) Get(customer, topic string, count int64) ([]*protos.Document, error) {
	m.Lock()
	defer m.Unlock()

	if !m.customerTopics[customer][topic] {
		return nil, fmt.Errorf("customer %s didn't subscribe topic %s", customer, topic)
	}

	docs := make([]*protos.Document, 0)
	topicDocs, ok := m.vals[topic]
	if !ok {
		return nil, fmt.Errorf("no such topic %s", topic)
	}

	for id, doc := range topicDocs {
		if count > 0 {
			if m.customerReaded[customer][topic][id] {
				continue
			}

			count--
			docs = append(docs, doc)
			delete(m.vals[topic], id)
			if _, ok := m.customerReaded[customer]; !ok {
				m.customerReaded[customer] = make(map[string]map[string]bool)
			}
			if _, ok := m.customerReaded[customer][topic]; !ok {
				m.customerReaded[customer][topic] = make(map[string]bool)
			}
			m.customerReaded[customer][topic][id] = true
		} else {
			break
		}
	}

	return docs, nil
}

func (m *MemoryCache) Subscribe(customer, topic string) bool {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.customerTopics[customer]; !ok {
		m.customerTopics[customer] = make(map[string]bool)
	}
	m.customerTopics[customer][topic] = true
	return true
}

// close cache
func (m *MemoryCache) Close() error {
	m.Lock()
	defer m.Unlock()

	m.vals = make(map[string]map[string]*protos.Document)
	return nil
}
