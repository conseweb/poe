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
package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"

	"github.com/conseweb/common/crypto"
	"golang.org/x/crypto/sha3"
)

// DocumentID return doc unique id by using hash algos
func DocumentID(doc []byte) string {
	return fmt.Sprintf("%x", crypto.Hash(sha3.New512(), []byte(fmt.Sprintf("%x%x%d", crypto.Hash(sha256.New(), doc), crypto.Hash(md5.New(), doc), len(doc)))))
}
