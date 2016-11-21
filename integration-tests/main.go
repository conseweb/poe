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
package main

import (
	"fmt"
	"time"

	"github.com/conseweb/common/semaphore"
	"github.com/conseweb/poe/api"
	"github.com/jmcvetta/randutil"
	"github.com/parnurzeal/gorequest"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Test flow:
// 1. send documents requests, 100 - 2000 reqs per second
// 2. check requests proof status
// 3. verify every request's proof result
const (
	// api url for document register
	api_doc_reg = "/api/v1/documents"
	// api url for document proof status
	api_doc_proof_status = "/api/v1/documents/%s/status"
	// api url for proof result check
	api_doc_proof_result = "/api/v1/documents/result"
)

var (
	host        = kingpin.Arg("host", "host of poe").Default("http://localhost:9694").String()
	concurrency = kingpin.Flag("concurrency", "request concurrency").Short('c').Default("100").Int()
	tn          = kingpin.Arg("tn", "number of tests").Default("1000").Int()
)

func main() {
	kingpin.Parse()

	concurrencyCtrl = semaphore.NewSemaphore(*concurrency)
	docmaps = make(map[string]string)
	for i := 0; i < *tn; i++ {
		concurrencyCtrl.Acquire()
		go func() {
			defer concurrencyCtrl.Release()

			docData, err := randutil.String(100, randutil.Ascii)
			if err != nil {
				panic(err)
			}

			docId := docRegister(docData)
			time.AfterFunc(time.Minute*2, func() {
				status := docProofStatus(docId)
				if status.Status == "wait" {
					panic(fmt.Errorf("document[%s] hasn't been proofed", docId))
				}

				proofResult := docProofResult(docData)
				if proofResult.Status == "wait" {
					panic(fmt.Errorf("document[%s] proof too slow", docId))
				}
			})
		}()
	}
}

var (
	// document id map
	docmaps map[string]string
	// concurrency control
	concurrencyCtrl *semaphore.Semaphore
)

func docRegister(data string) string {
	submitResp := new(api.DocumentSubmitResponse)
	_, _, errs := gorequest.New().Post(fmt.Sprintf("%s%s", *host, api_doc_reg)).Send(map[string]string{
		"proofWaitPeriod": "1m",
		"rawDocument":     data,
	}).EndStruct(submitResp)
	if len(errs) != 0 {
		panic(errs)
	}

	docmaps[submitResp.DocumentID] = data
	return submitResp.DocumentID
}

func docProofStatus(docId string) *api.GetProofStatusResponse {
	statusResp := new(api.GetProofStatusResponse)
	_, _, errs := gorequest.New().Get(fmt.Sprintf("%s%s", *host, fmt.Sprintf(api_doc_proof_status, docId))).EndStruct(statusResp)
	if len(errs) != 0 {
		panic(errs)
	}

	if statusResp.Status == "none" {
		panic(fmt.Errorf("document[%s] has not been accepted by poe", docId))
	}

	return statusResp
}

func docProofResult(data string) *api.GetProofResponse {
	proofResp := new(api.GetProofResponse)
	_, _, errs := gorequest.New().Post(fmt.Sprintf("%s%s", *host, api_doc_proof_result)).Send(map[string]string{
		"rawDocument": data,
	}).EndStruct(proofResp)
	if len(errs) != 0 {
		panic(errs)
	}

	if proofResp.Status == "none" || proofResp.Status == "invalid" {
		panic(fmt.Errorf("document proof failure."))
	}

	return proofResp
}
