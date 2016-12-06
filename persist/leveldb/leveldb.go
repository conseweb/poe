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
package leveldb

import (
	"fmt"
	"strings"

	"github.com/conseweb/poe/protos"
	"github.com/golang/protobuf/proto"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	leveldbLogger = logging.MustGetLogger("leveldb")
)

// levelDB persister
type LevelDBPersister struct {
	db *leveldb.DB
}

// NewLevelDBPersister
func NewLevelDBPersister() *LevelDBPersister {
	ldbp := new(LevelDBPersister)

	db, err := leveldb.OpenFile(viper.GetString("persist.leveldb.folder"), nil)
	if err != nil {
		leveldbLogger.Fatalf("leveldb open file return error: %v", err)
	}
	ldbp.db = db

	return ldbp
}

func (l *LevelDBPersister) PutDocsIntoDB(docs []*protos.Document) error {
	if len(docs) == 0 {
		return fmt.Errorf("invalid params")
	}

	batch := new(leveldb.Batch)
	for _, doc := range docs {
		if doc.Id == "" || doc.Hash == "" {
			return fmt.Errorf("document is lack of identity")
		}

		docBytes, err := proto.Marshal(doc)
		if err != nil {
			return err
		}

		batch.Put(l.docIdKey(doc.Id), docBytes)

		// dochash index
		hashIdsBytes, err := l.db.Get(l.docHashKey(doc.Hash), nil)
		if err != nil || len(hashIdsBytes) == 0 {
			batch.Put(l.docHashKey(doc.Hash), []byte(doc.Id))
		} else {
			hashIds := strings.Split(string(hashIdsBytes), "/")
			alreadyExist := false
			for _, hashId := range hashIds {
				if hashId == doc.Id {
					alreadyExist = true
					break
				}
			}
			if !alreadyExist {
				hashIds = append(hashIds, doc.Id)
			}
			batch.Put(l.docHashKey(doc.Hash), []byte(strings.Join(hashIds, "/")))
		}
	}

	return l.db.Write(batch, nil)
}

func (l *LevelDBPersister) docIdKey(docId string) []byte {
	return []byte(fmt.Sprintf("docId_%s", docId))
}

func (l *LevelDBPersister) docHashKey(hash string) []byte {
	return []byte(fmt.Sprintf("docHash_%s", hash))
}

func (l *LevelDBPersister) GetDocFromDBByDocID(docID string) (*protos.Document, error) {
	if docID == "" {
		return nil, fmt.Errorf("invalid document id")
	}

	docBytes, err := l.db.Get(l.docIdKey(docID), nil)
	if err != nil {
		return nil, err
	}

	doc := &protos.Document{}
	if err := proto.Unmarshal(docBytes, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (l *LevelDBPersister) SetDocsBlockDigest(docIDs []string, digest, txid string) error {
	if len(docIDs) == 0 || digest == "" {
		return fmt.Errorf("invalid params")
	}

	docs, err := l.getDocsByDocIds(docIDs)
	if err != nil {
		return err
	}
	for idx, _ := range docs {
		docs[idx].BlockDigest = digest
	}
	if err := l.PutDocsIntoDB(docs); err != nil {
		return err
	}

	// set digest index
	return l.db.Put(l.digestDocIDsKey(digest), []byte(strings.Join(docIDs, "/")), nil)
}

func (l *LevelDBPersister) getDocsByDocIds(docIDs []string) ([]*protos.Document, error) {
	docs := make([]*protos.Document, len(docIDs))
	for idx, docID := range docIDs {
		doc, err := l.GetDocFromDBByDocID(docID)
		if err != nil {
			return nil, err
		}
		docs[idx] = doc
	}

	return docs, nil
}

func (l *LevelDBPersister) digestDocIDsKey(digest string) []byte {
	return []byte(fmt.Sprintf("blockDigest_%s", digest))
}

func (l *LevelDBPersister) FindDocsByBlockDigest(digest string) ([]*protos.Document, error) {
	docIdsBytes, err := l.db.Get(l.digestDocIDsKey(digest), nil)
	if err != nil || len(docIdsBytes) == 0 {
		return nil, err
	}

	docIds := strings.Split(string(docIdsBytes), "/")
	return l.getDocsByDocIds(docIds)
}

func (l *LevelDBPersister) FindDocsByHash(hash string) ([]*protos.Document, error) {
	docIdsBytes, err := l.db.Get(l.docHashKey(hash), nil)
	if err != nil || len(docIdsBytes) == 0 {
		return nil, err
	}

	docIds := strings.Split(string(docIdsBytes), "/")
	return l.getDocsByDocIds(docIds)
}

func (l *LevelDBPersister) FindRegisteredDocs(count int) ([]*protos.Document, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (l *LevelDBPersister) FindProofedDocs(count int) ([]*protos.Document, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (l *LevelDBPersister) Close() error {
	return l.db.Close()
}
