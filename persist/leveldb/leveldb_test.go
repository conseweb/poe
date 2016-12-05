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
	"testing"

	"github.com/conseweb/poe/protos"
	"github.com/spf13/viper"
	"gopkg.in/check.v1"
)

func TestALL(t *testing.T) {
	check.TestingT(t)
}

type LeveldbPersisterTest struct {
}

func (t *LeveldbPersisterTest) SetUpSuite(c *check.C) {
	viper.Set("persist.leveldb.folder", c.MkDir())
}

var _ = check.Suite(&LeveldbPersisterTest{})

func (t *LeveldbPersisterTest) Test(c *check.C) {
	ldbp := NewLevelDBPersister()

	docs := []*protos.Document{
		&protos.Document{
			Id:   "id_1",
			Hash: "hash_1",
		},
		&protos.Document{
			Id:   "id_2",
			Hash: "hash_2",
		},
	}

	c.Check(ldbp.PutDocsIntoDB(docs), check.IsNil)
	c.Check(ldbp.SetDocsBlockDigest([]string{"id_1", "id_2"}, "digest", "txid"), check.IsNil)
	doc, err := ldbp.GetDocFromDBByDocID("id_1")
	c.Check(err, check.IsNil)
	c.Check(doc.Hash, check.DeepEquals, "hash_1")
	c.Check(doc.BlockDigest, check.DeepEquals, "digest")

	docsFindByDigest, err := ldbp.FindDocsByBlockDigest("digest")
	c.Check(err, check.IsNil)
	c.Check(len(docsFindByDigest), check.Equals, 2)

	docsFindByHash, err := ldbp.FindDocsByHash("hash_1")
	c.Check(err, check.IsNil)
	c.Check(len(docsFindByHash), check.Equals, 1)

	c.Check(ldbp.Close(), check.IsNil)
}
