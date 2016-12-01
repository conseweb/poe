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
	"fmt"
	"gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type DocumentTest struct {
}

var _ = check.Suite(&DocumentTest{})

func (t *DocumentTest) TestDocumentHash(c *check.C) {
	str := "dflsjfoiwefjlasfffdfjjfggjhjhggdsdfadjfoiewffjkhkjalsdjfoiewasjdfjewoiosdjff"
	id := "e568851e15a09b5e80f0caa11dda549ce0f5e56f0ddca530857ba94e483283423d403e7110c1eff6716d6db01d71813b469a0fd870d7d524a131cd6c40a30d9b"
	c.Check(DocumentHash([]byte(str)), check.Equals, id)
}

func (t *DocumentTest) BenchmarkDocumentID(c *check.C) {
	for i := 0; i < c.N; i++ {
		DocumentID([]byte(fmt.Sprintf("sdfjiwjeflsajdfi9jsdfijisdf_%d", i)))
	}
}

func (t *DocumentTest) BenchmarkDocumentHash(c *check.C) {
	for i := 0; i < c.N; i++ {
		DocumentHash([]byte(fmt.Sprintf("sdfjiwjeflsajdfi9jsdfijisdf_%d", i)))
	}
}
