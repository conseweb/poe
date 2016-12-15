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
package cassandra

import (
	"fmt"
	"math"
	"time"

	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/tsp"
	"github.com/gocql/gocql"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	cassandraLogger = logging.MustGetLogger("cassandra")
)

/*
keyspace: poe
CREATE TABLE documents (
	id text PRIMARY KEY,
	hash text,
	blockDigest text,
	submitTime bigint,
	proofTime bigint,
	waitDuration bigint,
	transactionId text,
	metadata text,
	sign text,
	appName text,
	proofStatus int
);

CREATE INDEX ON poe.documents(hash);
CREATE INDEX ON poe.documents(blockDigest);
CREATE INDEX ON poe.documents(transactionId);
CREATE INDEX ON poe.documents(submitTime);
CREATE INDEX ON poe.documents(appName);
CREATE INDEX ON poe.documents(proofStatus);
*/

const (
	_default_select_sql = "SELECT id, hash, blockDigest, submitTime, proofTime, waitDuration, metadata, transactionId, sign, appName, proofStatus FROM documents"
)

type CassandraPersister struct {
	session *gocql.Session
}

// NewCassandraPersister returns a cassandra persister
func NewCassandraPersister() *CassandraPersister {
	flogging.LoggingInit("cassandra")
	csdra := new(CassandraPersister)

	cluster := gocql.NewCluster(viper.GetStringSlice("persist.cassandra.clusters")...)
	cluster.Keyspace = viper.GetString("persist.cassandra.keyspace")

	session, err := cluster.CreateSession()
	if err != nil {
		cassandraLogger.Fatalf("can't create cassandra session: %v", err)
	}
	csdra.session = session

	return csdra
}

func (c *CassandraPersister) PutDocsIntoDB(docs []*protos.Document) error {
	batch := c.session.NewBatch(gocql.LoggedBatch)
	for _, doc := range docs {
		batch.Query(
			`INSERT INTO documents(id, hash, submitTime, waitDuration, metadata, appName, proofStatus) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			doc.Id,
			doc.Hash,
			doc.SubmitTime,
			doc.WaitDuration,
			doc.Metadata,
			doc.AppName,
			protos.DocProofStatus_NOT_PROOFED,
		)
	}

	if err := c.session.ExecuteBatch(batch); err != nil {
		cassandraLogger.Warningf("put docs into DB return error: %v", err)
	}

	return nil
}

func (c *CassandraPersister) GetDocFromDBByDocID(docID string) (*protos.Document, error) {
	if docID == "" {
		return nil, fmt.Errorf("empty document id")
	}

	doc := &protos.Document{}
	if err := c.session.Query(
		fmt.Sprintf("%s WHERE id = ? LIMIT 1", _default_select_sql),
		docID,
	).Consistency(gocql.One).Scan(
		&doc.Id,
		&doc.Hash,
		&doc.BlockDigest,
		&doc.SubmitTime,
		&doc.ProofTime,
		&doc.WaitDuration,
		&doc.Metadata,
		&doc.Txid,
		&doc.Sign,
		&doc.AppName,
		&doc.ProofStatus,
	); err != nil {
		cassandraLogger.Warningf("get document[%s] from Db return error: %v", docID, err)
		return nil, err
	}
	cassandraLogger.Debugf("doc: %v", doc)

	return doc, nil
}

func (c *CassandraPersister) SetDocsBlockDigest(docIDs []string, digest, txid string) error {
	if len(docIDs) == 0 {
		return nil
	}

	nowTimestamp := tsp.Now().UnixNano()
	batch := c.session.NewBatch(gocql.LoggedBatch)
	for _, docID := range docIDs {
		batch.Query(
			"UPDATE documents SET blockDigest = ?, proofTime = ?, transactionId = ? , proofStatus = ? WHERE id = ?",
			digest,
			nowTimestamp,
			txid,
			protos.DocProofStatus_ALREADY_PROOFED,
			docID,
		)
	}
	if err := c.session.ExecuteBatch(batch); err != nil {
		cassandraLogger.Warningf("set documents blockDigest return error: %v", err)
		return err
	}

	return nil
}

func (c *CassandraPersister) SetDocSignature(docID, sign string) error {
	if docID == "" || sign == "" {
		return fmt.Errorf("invalid params")
	}

	if err := c.session.Query(
		"UPDATE documents SET sign = ? WHERE id = ?",
		sign,
		docID,
	).Exec(); err != nil {
		cassandraLogger.Warningf("set document sign return error: %v", err)
		return err
	}

	return nil
}

func (c *CassandraPersister) FindDocsByBlockDigest(digest string) ([]*protos.Document, error) {
	if digest == "" {
		return nil, fmt.Errorf("invalid digest")
	}

	iter := c.session.Query(
		fmt.Sprintf("%s WHERE blockDigest = ?", _default_select_sql),
		digest,
	).Iter()

	return iterToDocs(iter)
}

func (c *CassandraPersister) FindDocsByHash(hash string) ([]*protos.Document, error) {
	if hash == "" {
		return nil, fmt.Errorf("invalid hash")
	}

	iter := c.session.Query(
		fmt.Sprintf("%s WHERE hash = ?", _default_select_sql),
		hash,
	).Iter()

	return iterToDocs(iter)
}

func (c *CassandraPersister) FindRegisteredDocs(count int) ([]*protos.Document, error) {
	if count <= 0 {
		return nil, fmt.Errorf("invalid param: count %d", count)
	}

	iter := c.session.Query(
		fmt.Sprintf("%s WHERE blockDigest = ? and proofTime = ? LIMIT ? ALLOW FILTERING", _default_select_sql),
		"",
		0,
		count,
	).Iter()

	return iterToDocs(iter)
}

func (c *CassandraPersister) FindProofedDocs(count int) ([]*protos.Document, error) {
	if count <= 0 {
		return nil, fmt.Errorf("invalid param: count: %d", count)
	}

	iter := c.session.Query(
		fmt.Sprintf("%s WHERE blockDigest > ? and proofTime > ? LIMIT ? ALLOW FILTERING", _default_select_sql),
		"",
		0,
		count,
	).Iter()

	return iterToDocs(iter)
}

func (c *CassandraPersister) FindDocs(appName string, proofStatus protos.DocProofStatus, count int) ([]*protos.Document, error) {
	if count <= 0 {
		return nil, fmt.Errorf("invalid param: count: %d", count)
	}

	iter := c.session.Query(
		fmt.Sprintf("%s WHERE appName = ? AND proofStatus = ? LIMIT ? ALLOW FILTERING", _default_select_sql),
		appName,
		proofStatus,
		count,
	).Iter()

	return iterToDocs(iter)
}

func (c *CassandraPersister) DocProofStat(sTime, eTime time.Time) *protos.ProofStat {
	startTime := sTime.UnixNano()
	if startTime <= 0 {
		startTime = 1
	}
	endTime := eTime.UnixNano()
	if endTime <= 0 {
		endTime = math.MaxInt64
	}

	stat := &protos.ProofStat{
		StartTime: startTime,
		EndTime:   endTime,
	}

	if err := c.session.Query(
		"SELECT count(*) FROM documents WHERE submitTime >= ? AND submitTime <= ? ALLOW FILTERING",
		startTime,
		endTime,
	).Consistency(gocql.One).Scan(&stat.TotalDocs); err != nil {
		cassandraLogger.Warningf("count all documents return error: %v", err)
		return stat
	}

	if err := c.session.Query(
		"SELECT count(*) FROM documents WHERE submitTime >= ? AND submitTime <= ? AND proofStatus = ? ALLOW FILTERING",
		startTime,
		endTime,
		protos.DocProofStatus_NOT_PROOFED,
	).Consistency(gocql.One).Scan(&stat.WaitDocs); err != nil {
		cassandraLogger.Warningf("count waitting documents return error: %v", err)
		return stat
	}
	stat.WaitDocs = stat.TotalDocs - stat.ProofedDocs

	return stat
}

func (c *CassandraPersister) Close() error {
	c.session.Close()
	return nil
}

func iterToDocs(iter *gocql.Iter) ([]*protos.Document, error) {
	docs := make([]*protos.Document, 0)
	for {
		doc := &protos.Document{}
		if !iter.Scan(
			&doc.Id,
			&doc.Hash,
			&doc.BlockDigest,
			&doc.SubmitTime,
			&doc.ProofTime,
			&doc.WaitDuration,
			&doc.Metadata,
			&doc.Txid,
			&doc.Sign,
			&doc.AppName,
			&doc.ProofStatus,
		) {
			break
		}
		docs = append(docs, doc)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	return docs, nil
}
