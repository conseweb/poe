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
	doc := new(DocumentSubmitRequest)
	if err := ctx.ReadForm(doc); err != nil {
		apiLogger.Errorf("read submit document return error: %v", err)
		ctx.Panic()
		return
	}

	waitPeriod, err := time.ParseDuration(doc.ProofWaitPeriod)
	if err != nil {
		apiLogger.Errorf("parse submit proof wait period return error: %v", err)
		ctx.Panic()
		return
	}
	docID, err := srv.cache.Put([]byte(doc.RawDocument), srv.cache.Topic(waitPeriod))
	if err != nil {
		apiLogger.Errorf("put sumit document into cache return error: %v", err)
		ctx.EmitError(iris.StatusBadRequest)
		return
	}
	apiLogger.Debugf("document: %+v, document ID: %s", doc, docID)

	ctx.JSON(iris.StatusCreated, DocumentSubmitResponse{
		DocumentID: docID,
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
		apiLogger.Errorf("get document[%s] proof status return error: %v", err)
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
