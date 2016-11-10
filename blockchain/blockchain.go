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
package blockchain

import (
	"github.com/conseweb/poe/cache"
	"github.com/conseweb/poe/persist"
)

// Blockchain
type Blockchain struct {
	chaincodeID string
	peers       []string
	cacher      cache.CacheInterface
	persister   persist.PersistInterface
}

func NewBlockchain() *Blockchain {
	return &Blockchain{}
}
