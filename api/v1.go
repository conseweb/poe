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
package api

import (
	"time"

	"github.com/kataras/iris"
)

// DocumentSubmitRequest
type DocumentSubmitRequest struct {
	ProofWaitPeriod string `json:"proofWaitPeriod" xml:"proofWaitPeriod" form:"proofWaitPeriod"`
	RawDocument     string `json:"rawDocument" xml:"rawDocument" form:"rawDocument"`
}

// DocumentSubmitResponse
type DocumentSubmitResponse struct {
	DocumentID string `json:"documentId"`
}

// submitRaw submits raw file into cache
func (srv *APIServer) submitRaw(ctx *iris.Context) {
	req := new(DocumentSubmitRequest)
	if err := ctx.ReadForm(req); err != nil {
		apiLogger.Errorf("read submit document return error: %v", err)
		ctx.EmitError(iris.StatusBadRequest)
		return
	}

	waitPeriod, err := time.ParseDuration(req.ProofWaitPeriod)
	if err != nil {
		apiLogger.Errorf("parse submit proof wait period return error: %v", err)
		ctx.EmitError(iris.StatusBadRequest)
		return
	}
	doc, err := srv.cache.Put([]byte(req.RawDocument), srv.cache.Topic(waitPeriod))
	if err != nil {
		apiLogger.Errorf("put sumit document into cache return error: %v", err)
		ctx.EmitError(iris.StatusBadRequest)
		return
	}
	apiLogger.Debugf("document request: %v, document ID: %s", req, doc.Id)

	ctx.JSON(iris.StatusCreated, DocumentSubmitResponse{
		DocumentID: doc.Id,
	})
}

type GetProofStatusResponse struct {
	Status              string `json:"status"` // none/wait/ok
	DocumentID          string `json:"documentId,omitempty"`
	DocumentBlockDigest string `json:"documentBlockDigest,omitempty"`
}

// getProofStatus returns document's exist proof status
func (srv *APIServer) getProofStatus(ctx *iris.Context) {
	documentID := ctx.Param("id")
	if documentID == "" {
		ctx.NotFound()
		return
	}

	document, err := srv.persister.GetDocFromDBByDocID(documentID)
	if err != nil {
		apiLogger.Errorf("get document[%s] return error: %v", documentID, err)
		ctx.JSON(iris.StatusNotFound, &GetProofStatusResponse{
			Status: "none",
		})
		return
	}

	response := &GetProofStatusResponse{
		DocumentID: document.Id,
	}
	if document.BlockDigest != "" {
		response.Status = "ok"
	} else {
		response.Status = "wait"
	}
	response.DocumentBlockDigest = document.BlockDigest

	ctx.JSON(iris.StatusOK, response)
}

type GetProofRequest struct {
	RawDocument string `json:"rawDocument" xml:"rawDocument" form:"rawDocument"`
}

type GetProofResponse struct {
	Status     string `json:"status"` // none/wait/valid/invalid
	DocumentId string `json:"documentId,omitempty"`
	SubmitTime int64  `json:"submitTime,omitempty"`
	ProofTime  int64  `json:"proofTime,omitempty"`
}

// getProof
func (srv *APIServer) getProof(ctx *iris.Context) {
	req := new(GetProofRequest)
	if err := ctx.ReadForm(req); err != nil {
		apiLogger.Errorf("read get proof document return error: %v", err)
		ctx.Panic()
		return
	}

	response := &GetProofResponse{}
	// document id is made by cache, so always should to get document id though cacher
	docHash := srv.cache.DocumentHash([]byte(req.RawDocument))

	// get document info from persister, if error occurs, means that document has
	docs, err := srv.persister.FindDocsByHash(docHash)
	if err != nil || len(docs) == 0 {
		apiLogger.Errorf("find document[%s] return error: %v", err)
		response.Status = "none"
		ctx.JSON(iris.StatusNotFound, response)
		return
	}
	apiLogger.Debugf("find %d documents using same hash[%s]", len(docs), docHash)

	doc := docs[0]
	// if document's blockDigest is blank, means blockchain has not proof exists
	if doc.BlockDigest == "" {
		response.Status = "wait"
		response.DocumentId = doc.Id
		response.SubmitTime = doc.SubmitTime
		ctx.JSON(iris.StatusOK, response)
		return
	}

	response.DocumentId = doc.Id
	response.SubmitTime = doc.SubmitTime
	// based on document block digest, get all documents in same block
	blockDocs, err := srv.persister.FindDocsByBlockDigest(doc.BlockDigest)
	if err != nil {
		response.Status = "invalid"
		ctx.JSON(iris.StatusOK, response)
		return
	}

	// verify by blockchain
	if srv.blcokchain.VerifyDocs(blockDocs) {
		response.Status = "valid"
		response.ProofTime = doc.ProofTime
	} else {
		response.Status = "invalid"
	}

	ctx.JSON(iris.StatusOK, response)
}
