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
	"fmt"

	"github.com/conseweb/poe/blockchain"
	"github.com/conseweb/poe/cache"
	"github.com/conseweb/poe/persist"
	"github.com/hyperledger/fabric/flogging"
	"github.com/iris-contrib/middleware/cors"
	"github.com/iris-contrib/middleware/logger"
	"github.com/kataras/iris"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	apiLogger        = logging.MustGetLogger("api")
	default_api_addr = ":9694"
)

type APIServer struct {
	irisapi    *iris.Framework
	cache      cache.CacheInterface
	persister  persist.PersistInterface
	blcokchain *blockchain.Blockchain
}

// NewAPIServer returns a api server, not started
func NewAPIServer(cc cache.CacheInterface, persister persist.PersistInterface, bc *blockchain.Blockchain) *APIServer {
	flogging.LoggingInit("api")
	server := new(APIServer)

	// web framework
	irisapi := iris.New()
	irisapi.Use(logger.New())

	// cors
	irisapi.Use(cors.Default())

	// api v1
	{
		irisapi.Post("/poe/v1/documents", server.submitRaw)
		irisapi.Post("/poe/v1/documents/result", server.postProofResult)
		irisapi.Get("/poe/v1/documents/:id/status", server.getProofStatus)
		irisapi.Get("/poe/v1/documents/:id/result", server.getProofResult)
		irisapi.Get("/poe/v1/documents", server.getDocs)
		irisapi.Get("/poe/v1/stat/customize", server.getDocStat)
		irisapi.Get("/poe/v1/stat/normal", server.getNormalDocStat)
	}
	server.irisapi = irisapi

	// cache
	server.cache = cc

	// persister
	server.persister = persister

	// blockchain
	server.blcokchain = bc

	return server
}

// Start start api server
func (srv *APIServer) Start() {
	addr := viper.GetString("api.addr")
	if addr == "" {
		addr = default_api_addr
		apiLogger.Infof("not provider api server listening address, using %s", default_api_addr)
	}

	apiLogger.Infof("api server is listening on %s", addr)
	srv.irisapi.Listen(addr)
}

// Stop stops api server
func (srv *APIServer) Stop() error {
	apiLogger.Info("api server is stopping")
	defer apiLogger.Info("api server is stopped")

	// close irisapi
	if err := srv.irisapi.Close(); err != nil {
		apiLogger.Errorf("close api server irisapi return error: %s", err)
		return err
	}

	return nil
}

func (srv *APIServer) Error(ctx *iris.Context, status int, errs ...interface{}) {
	l := len(errs)
	ret := map[string]interface{}{"error": "unknown"}
	if l > 0 {
		ret["error"] = fmt.Sprint(errs[0])
	}
	if l > 1 {
		ret["message"] = errs[1:]
	}
	ctx.Response.Reset()
	ctx.JSON(status, ret)
	ctx.StopExecution()
}
