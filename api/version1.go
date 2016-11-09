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
	"net/http"
	"time"

	"github.com/kataras/iris"
)

// DocumentSubmitRequest
type DocumentSubmitRequest struct {
	ProofWaitPeriod string `json:"proof_wait_period" xml:"proof_wait_period" form:"proof_wait_period"`
	RawDocument     string `json:"raw_document" xml:"raw_document" form:"raw_document"`
}

// DocumentSubmitResponse
type DocumentSubmitResponse struct {
	DocumentID string `json:"document_id"`
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
		ctx.EmitError(http.StatusBadRequest)
		return
	}
	apiLogger.Debugf("document: %+v, document ID: %s", doc, docID)

	ctx.JSON(iris.StatusCreated, DocumentSubmitResponse{
		DocumentID: docID,
	})
}

// getProof returns data's exist proof
func (srv *APIServer) getProof(ctx *iris.Context) {

}
