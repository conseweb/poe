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
package fake

import (
	"fmt"
	"sync"

	"github.com/conseweb/poe/protos"
)

type FakePersister struct {
	sync.RWMutex
	name      string
	documents map[string]*protos.Document
	blockDocs map[string][]string
}

// NewFakePersister return a fake persister, just for development
func NewFakePersister() *FakePersister {
	persister := &FakePersister{
		name:      "fakePersister",
		documents: make(map[string]*protos.Document),
		blockDocs: make(map[string][]string),
	}

	return persister
}

func (p *FakePersister) GetPersisterName() string {
	p.RLock()
	defer p.RUnlock()

	return p.name
}

func (p *FakePersister) PutDocsIntoDB(docs []*protos.Document) error {
	p.Lock()
	defer p.Unlock()

	for _, doc := range docs {
		p.documents[doc.Id] = doc
		fmt.Printf("fake put document[%+v] into DB\n", doc)
	}

	return nil
}

func (p *FakePersister) GetDocFromDBByDocID(docID string) (*protos.Document, error) {
	p.RLock()
	defer p.RUnlock()

	if docID == "" {
		return nil, fmt.Errorf("docID is empty")
	}

	doc, ok := p.documents[docID]
	if !ok {
		return nil, fmt.Errorf("no such document id")
	}

	return doc, nil
}

func (p *FakePersister) SetDocsBlockDigest(docIDs []string, digest string) error {
	p.Lock()
	defer p.Unlock()

	if len(docIDs) == 0 || digest == "" {
		return fmt.Errorf("input params is invalid.")
	}

	p.blockDocs[digest] = docIDs
	for _, id := range docIDs {
		p.documents[id].BlockDigest = digest
	}

	return nil
}

func (p *FakePersister) Close() error {
	return nil
}
