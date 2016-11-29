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
	"sync"
	"time"
	"log"

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
	host        = kingpin.Flag("host", "host of poe").Short('h').Default("http://0.0.0.0:9694").String()
	concurrency = kingpin.Flag("concurrency", "request concurrency").Short('c').Default("100").Int()
	tn          = kingpin.Flag("tn", "number of tests").Short('n').Default("1000").Int()
	wp          = kingpin.Flag("wp", "wait period, using duration string, default '1m'").Short('p').Default("1m").String()
)

func main() {
	kingpin.Parse()

	wg := &sync.WaitGroup{}
	concurrencyCtrl = semaphore.NewSemaphore(*concurrency)
	wpDuration, err := time.ParseDuration(*wp)
	if err != nil {
		panic(err)
	}

	wg.Add(*tn)
	for i := 0; i < *tn; i++ {
		concurrencyCtrl.Acquire()
		go func() {
			docData, err := randutil.String(100, randutil.Ascii)
			if err != nil {
				panic(err)
			}

			docId := docRegister(*wp, docData)
			log.Printf(docId)
			time.AfterFunc(wpDuration, func() {
				status := docProofStatus(docId)
				if status.Status == "wait" {
					panic(fmt.Errorf("document[%s] hasn't been proofed", docId))
				}

				proofResult := docProofResult(docData)
				if proofResult.Status == "wait" {
					panic(fmt.Errorf("document[%s] proof too slow", docId))
				}

				concurrencyCtrl.Release()
				wg.Done()
			})
		}()
	}
	wg.Wait()
}

var (
	// concurrency control
	concurrencyCtrl *semaphore.Semaphore
)

func docRegister(wp, data string) string {
	submitResp := new(api.DocumentSubmitResponse)
	_, _, errs := gorequest.New().Post(fmt.Sprintf("%s%s", *host, api_doc_reg)).Type("json").Send(map[string]string{
		"proofWaitPeriod": wp,
		"rawDocument":     data,
	}).EndStruct(submitResp)
	if len(errs) != 0 {
		panic(errs[0])
	}

	//docmaps[submitResp.DocumentID] = data
	return submitResp.DocumentID
}

func docProofStatus(docId string) *api.GetProofStatusResponse {
	statusResp := new(api.GetProofStatusResponse)
	_, _, errs := gorequest.New().Get(fmt.Sprintf("%s%s", *host, fmt.Sprintf(api_doc_proof_status, docId))).EndStruct(statusResp)
	if len(errs) != 0 {
		panic(errs[0])
	}

	if statusResp.Status == "none" {
		panic(fmt.Errorf("document[%s] has not been accepted by poe", docId))
	}

	return statusResp
}

func docProofResult(data string) *api.GetProofResponse {
	proofResp := new(api.GetProofResponse)
	_, _, errs := gorequest.New().Post(fmt.Sprintf("%s%s", *host, api_doc_proof_result)).Type("json").Send(map[string]string{
		"rawDocument": data,
	}).EndStruct(proofResp)
	if len(errs) != 0 {
		panic(errs[0])
	}

	if proofResp.Status == "none" || proofResp.Status == "invalid" {
		panic(fmt.Errorf("document proof failure."))
	}

	return proofResp
}
