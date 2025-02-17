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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/sign"
	"github.com/conseweb/poe/utils"
	"github.com/jinzhu/now"
	"github.com/kataras/iris"
)

var (
	default_appName = "def_appName"
)

// DocumentSubmitRequest
type DocumentSubmitRequest struct {
	ProofWaitPeriod string `json:"proofWaitPeriod" xml:"proofWaitPeriod" form:"proofWaitPeriod"`
	RawDocument     string `json:"rawDocument" xml:"rawDocument" form:"rawDocument"`
	Metadata        string `json:"metadata" xml:"metadata" form:"metadata"`
	AppName         string `json:"appName" xml:"appName" form:"appName"`
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
	if req.Metadata != "" {
		req.AppName = default_appName
	}

	apiLogger.Debugf("receive<%s, %s>", req.RawDocument, waitPeriod.String())
	doc, err := srv.cache.Put(req.AppName, []byte(req.RawDocument), req.Metadata, waitPeriod)
	if err != nil {
		apiLogger.Errorf("put sumit document into cache return error: %v", err)
		ctx.Error("poe server can't provider service now, sorry", iris.StatusInternalServerError)
		return
	}
	apiLogger.Debugf("return<%s>", doc.Id[:8])

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
	Status           string           `json:"status"` // none/wait/valid/invalid
	Message          string           `json:"message,omitempty"`
	PublicKey        string           `json:"publicKey,omitempty"`
	Doc              *protos.Document `json:"doc,omitempty"`
	PerdictProofTime int64            `json:"perdictProofTime,omitempty"`
}

// getProof
func (srv *APIServer) postProofResult(ctx *iris.Context) {
	req := new(GetProofRequest)
	if err := ctx.ReadJSON(req); err != nil {
		apiLogger.Errorf("read get proof document return error: %v", err)
		ctx.Error(err.Error(), iris.StatusBadRequest)
		return
	}

	response := &GetProofResponse{Doc: &protos.Document{}}
	// document id is made by cache, so always should to get document id though cacher
	docHash := srv.cache.DocumentHash([]byte(req.RawDocument))

	// get document info from persister, if error occurs, means that document has
	docs, err := srv.persister.FindDocsByHash(docHash)
	if err != nil || len(docs) == 0 {
		apiLogger.Errorf("find document[%s] return error: %v", err)
		response.Status = "none"
		response.Message = "not found"
		ctx.JSON(iris.StatusNotFound, response)
		return
	}
	apiLogger.Debugf("find %d documents using same hash[%s]", len(docs), docHash)

	response.Doc = docs[0]
	// if document's blockDigest is blank, means blockchain has not proof exists
	if response.Doc.BlockDigest == "" {
		response.Status = "wait"

		response.PerdictProofTime = time.Unix(response.Doc.SubmitTime, 0).UTC().Add(time.Duration(response.Doc.WaitDuration)).Unix()
		ctx.JSON(iris.StatusAccepted, response)
		return
	}

	// based on document block digest, get all documents in same block
	blockDocs, err := srv.persister.FindDocsByBlockDigest(response.Doc.BlockDigest)
	if err != nil {
		response.Status = "invalid"
		response.Message = err.Error()
		ctx.JSON(iris.StatusNotFound, response)
		return
	}
	apiLogger.Debugf("using same blockdigest docs: %v", blockDocs)

	// verify by blockchain
	if !srv.blcokchain.VerifyDocs(blockDocs) {
		response.Status = "invalid"
		response.Message = "not verified."
		ctx.JSON(iris.StatusInternalServerError, response)
		return
	}

	// sign the document
	if response.Doc.Sign == "" {
		sign, pubKey, err := srv.setDocSign(response.Doc)
		if err != nil {
			response.Status = "invalid"
			response.Message = err.Error()
			ctx.JSON(iris.StatusInternalServerError, response)
			return
		}

		response.Doc.Sign = sign
		response.PublicKey = pubKey
	} else {
		response.PublicKey = hex.EncodeToString(sign.GetPublicKey())
	}
	response.Status = "valid"

	ctx.JSON(iris.StatusOK, response)
}

func (srv *APIServer) getProofResult(ctx *iris.Context) {
	docID := ctx.Param("id")

	response := &GetProofResponse{Doc: &protos.Document{}}
	doc, err := srv.persister.GetDocFromDBByDocID(docID)
	if err != nil {
		srv.Error(ctx, 404, fmt.Errorf("cannot found %v, %s", docID, err.Error()), response)
		return
	}

	response.Doc = doc
	// if document's blockDigest is blank, means blockchain has not proof exists
	if response.Doc.BlockDigest == "" {
		response.Status = "wait"
		response.PerdictProofTime = time.Unix(response.Doc.SubmitTime, 0).UTC().Add(time.Duration(response.Doc.WaitDuration)).Unix()
		srv.Error(ctx, 406, "file is waiting.", response)
		return
	}

	// based on document block digest, get all documents in same block
	blockDocs, err := srv.persister.FindDocsByBlockDigest(response.Doc.BlockDigest)
	if err != nil {
		srv.Error(ctx, 404, fmt.Errorf("cannot found in blockchain, %s", err.Error()), response)
		return
	}
	apiLogger.Debugf("using same blockdigest %v docs.", len(blockDocs))

	// verify by blockchain
	if !srv.blcokchain.VerifyDocs(blockDocs) {
		response.Status = "invalid"
		srv.Error(ctx, 400, "verify failed.", response)
		return
	}

	// sign the document
	if response.Doc.Sign == "" {
		sign, pubKey, err := srv.setDocSign(response.Doc)
		if err != nil {
			response.Status = "invalid"
			srv.Error(ctx, 500, err, response)
			return
		}

		response.Doc.Sign = sign
		response.PublicKey = pubKey
	} else {
		response.PublicKey = hex.EncodeToString(sign.GetPublicKey())
	}
	response.Status = "valid"

	ctx.JSON(iris.StatusOK, response)
}

func (srv *APIServer) setDocSign(doc *protos.Document) (string, string, error) {
	// Serialization structure to []byte
	docRaw, err := json.Marshal(doc)
	if err != nil {
		return "", "", err
	}

	// ecdsa sign
	signRaw, pukRaw, err := sign.ECDSASign(docRaw)
	if err != nil {
		return "", "", err
	}
	sign := hex.EncodeToString(signRaw)
	publicKey := hex.EncodeToString(pukRaw)

	go srv.persister.SetDocSignature(doc.Id, sign)

	return sign, publicKey, nil
}

type GetDocsResponse struct {
	Docs []*protos.Document `json:"docs"`
}

// getRegisteredDocs returns registered in the platform but still not ben proofed documents
func (srv *APIServer) getDocs(ctx *iris.Context) {
	appName := ctx.URLParam("appName")
	if appName == "" {
		appName = default_appName
	}
	searchType := ctx.URLParam("type")
	count, err := ctx.URLParamInt("count")
	if err != nil || count <= 0 {
		count = 10
	}
	if count > 200 {
		count = 200
	}

	var docs []*protos.Document
	switch searchType {
	case "register":
		docs, err = srv.persister.FindDocs(appName, protos.DocProofStatus_NOT_PROOFED, count)
		sort.Sort(RegisteredDocs(docs))
	case "proof":
		docs, err = srv.persister.FindDocs(appName, protos.DocProofStatus_ALREADY_PROOFED, count)
		sort.Sort(ProofedDocs(docs))
	case "":
		docs, err = srv.persister.FindDocs(appName, protos.DocProofStatus_NOT_SPECIFIAL, count)
		sort.Sort(RegisteredDocs(docs))
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

func (srv *APIServer) getDocStat(ctx *iris.Context) {
	startTs, err := ctx.URLParamInt64("startTime")
	if err != nil {
		ctx.Error(err.Error(), iris.StatusBadRequest)
		return
	}

	endTs, err := ctx.URLParamInt64("endTime")
	if err != nil {
		ctx.Error(err.Error(), iris.StatusBadRequest)
		return
	}

	ctx.JSON(iris.StatusOK, srv.persister.DocProofStat(time.Unix(startTs, 0), time.Unix(endTs, 0)))
}

type NormalDocStatResponse struct {
	TotalStat *protos.ProofStat `json:"totalStat,omitempty"`
	TodayStat *protos.ProofStat `json:"todayStat,omitempty"`
	WeekStat  *protos.ProofStat `json:"weekStat,omitempty"`
	MonthStat *protos.ProofStat `json:"monthStat,omitempty"`
}

func (srv *APIServer) getNormalDocStat(ctx *iris.Context) {
	resp := &NormalDocStatResponse{
		TotalStat: srv.persister.DocProofStat(time.Unix(0, 0), time.Unix(0, 0)),
		TodayStat: srv.persister.DocProofStat(now.BeginningOfDay(), now.EndOfDay()),
		WeekStat:  srv.persister.DocProofStat(now.BeginningOfWeek(), now.EndOfWeek()),
		MonthStat: srv.persister.DocProofStat(now.BeginningOfMonth(), now.EndOfMonth()),
	}

	ctx.JSON(iris.StatusOK, resp)
}

func init() {
	now.FirstDayMonday = true
}
