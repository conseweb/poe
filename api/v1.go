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
	"sort"
	"time"

	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
	"github.com/kataras/iris"
)

// DocumentSubmitRequest
type DocumentSubmitRequest struct {
	ProofWaitPeriod string `json:"proofWaitPeriod" xml:"proofWaitPeriod" form:"proofWaitPeriod"`
	RawDocument     string `json:"rawDocument" xml:"rawDocument" form:"rawDocument"`
}

// DocumentSubmitResponse
type DocumentSubmitResponse struct {
	DocumentID       string `json:"documentId"`
	PerdictProofTime int64  `json:"perdictProofTime"`
}

// submitRaw submits raw file into cache
func (srv *APIServer) submitRaw(ctx *iris.Context) {
	req := new(DocumentSubmitRequest)
	if err := ctx.ReadJSON(req); err != nil {
		apiLogger.Errorf("read submit document return error: %v", err)
		ctx.Error(err.Error(), iris.StatusBadRequest)
		return
	}

	if req.ProofWaitPeriod == "" {
		req.ProofWaitPeriod = utils.GetPeriodLimits()[0].Period.String()
	}
	waitPeriod, err := time.ParseDuration(req.ProofWaitPeriod)
	if err != nil {
		apiLogger.Errorf("parse submit proof wait period return error: %v", err)
		ctx.Error("proofWaitPeriod format is invalid", iris.StatusBadRequest)
		return
	}
	doc, err := srv.cache.Put([]byte(req.RawDocument), waitPeriod)
	if err != nil {
		apiLogger.Errorf("put sumit document into cache return error: %v", err)
		ctx.Error("poe server can't provider service now, sorry", iris.StatusInternalServerError)
		return
	}
	apiLogger.Debugf("document request: %v, document ID: %s", req, doc.Id)

	ctx.JSON(iris.StatusCreated, DocumentSubmitResponse{
		DocumentID:       doc.Id,
		PerdictProofTime: time.Unix(doc.SubmitTime, 0).UTC().Add(time.Duration(doc.WaitDuration)).Unix(),
	})
}

type GetProofStatusResponse struct {
	Status              string `json:"status"` // none/wait/ok
	DocumentID          string `json:"documentId,omitempty"`
	DocumentBlockDigest string `json:"documentBlockDigest,omitempty"`
	PerdictProofTime    int64  `json:"perdictProofTime,omitempty"`
	ProofTime           int64  `json:"proofTime,omitempty"`
}

// getProofStatus returns document's exist proof status
func (srv *APIServer) getProofStatus(ctx *iris.Context) {
	documentID := ctx.Param("id")
	if documentID == "" {
		ctx.Error("document not found", iris.StatusNotFound)
		return
	}

	doc, err := srv.persister.GetDocFromDBByDocID(documentID)
	if err != nil {
		apiLogger.Errorf("get document[%s] return error: %v", documentID, err)
		ctx.JSON(iris.StatusNotFound, &GetProofStatusResponse{
			Status: "none",
		})
		return
	}

	response := &GetProofStatusResponse{
		DocumentID: doc.Id,
	}
	if doc.BlockDigest != "" {
		response.Status = "ok"
		response.ProofTime = doc.ProofTime
	} else {
		response.Status = "wait"
		response.PerdictProofTime = time.Unix(doc.SubmitTime, 0).UTC().Add(time.Duration(doc.WaitDuration)).Unix()
	}
	response.DocumentBlockDigest = doc.BlockDigest

	ctx.JSON(iris.StatusOK, response)
}

type GetProofRequest struct {
	RawDocument string `json:"rawDocument" xml:"rawDocument" form:"rawDocument"`
}

type GetProofResponse struct {
	Status           string `json:"status"` // none/wait/valid/invalid
	DocumentId       string `json:"documentId,omitempty"`
	SubmitTime       int64  `json:"submitTime,omitempty"`
	ProofTime        int64  `json:"proofTime,omitempty"`
	PerdictProofTime int64  `json:"perdictProofTime,omitempty"`
}

// getProof
func (srv *APIServer) getProof(ctx *iris.Context) {
	req := new(GetProofRequest)
	if err := ctx.ReadJSON(req); err != nil {
		apiLogger.Errorf("read get proof document return error: %v", err)
		ctx.Error(err.Error(), iris.StatusBadRequest)
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
		response.PerdictProofTime = time.Unix(doc.SubmitTime, 0).UTC().Add(time.Duration(doc.WaitDuration)).Unix()
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

type GetDocsResponse struct {
	Docs []*protos.Document `json:"docs"`
}

// getRegisteredDocs returns registered in the platform but still not ben proofed documents
func (srv *APIServer) getDocs(ctx *iris.Context) {
	searchType := ctx.URLParam("type")
	count, err := ctx.URLParamInt("count")
	if err != nil || count <= 0 {
		count = 10
	}

	var docs []*protos.Document
	switch searchType {
	case "register":
		docs, err = srv.persister.FindRegisteredDocs(count)
		sort.Sort(RegisteredDocs(docs))
	case "proof":
		docs, err = srv.persister.FindProofedDocs(count)
		sort.Sort(ProofedDocs(docs))
	default:
		ctx.Error("not supported doc status type", iris.StatusBadRequest)
		return
	}
	if err != nil {
		ctx.Error(err.Error(), iris.StatusInternalServerError)
		return
	}

	ctx.JSON(iris.StatusOK, &GetDocsResponse{
		Docs: docs,
	})
}

type RegisteredDocs []*protos.Document

func (docs RegisteredDocs) Len() int           { return len(docs) }
func (docs RegisteredDocs) Less(i, j int) bool { return docs[i].SubmitTime < docs[j].SubmitTime }
func (docs RegisteredDocs) Swap(i, j int)      { docs[i], docs[j] = docs[j], docs[i] }

type ProofedDocs []*protos.Document

func (docs ProofedDocs) Len() int           { return len(docs) }
func (docs ProofedDocs) Less(i, j int) bool { return docs[i].ProofTime < docs[j].ProofTime }
func (docs ProofedDocs) Swap(i, j int)      { docs[i], docs[j] = docs[j], docs[i] }
