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
	"github.com/conseweb/common/config"
	"github.com/conseweb/common/exec"
	"github.com/conseweb/poe/api"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/conseweb/poe/cache"
	"github.com/conseweb/poe/persist"
)

var (
	mainLogger = logging.MustGetLogger("main")
)

func init() {
	config.LoadConfig("POE", "poe", "github.com/conseweb/poe")
}

func main() {
	// set logging level
	flogging.LoggingInit("main")

	// cache
	cc := cache.NewCache()

	// persister
	persister := persist.NewPersister(cc)

	// api server
	apisrv := api.NewAPIServer(cc, persister)
	go apisrv.Start()

	// handle signal
	exec.HandleSignal(apisrv.Stop, persister.Close, cc.Close)
}
