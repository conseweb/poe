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
package fake

import (
	"testing"

	"github.com/conseweb/poe/protos"
	"gopkg.in/check.v1"
)

func TestALL(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&FakePersisterTest{})

type FakePersisterTest struct {
}

func (t *FakePersisterTest) SetUpSuite(c *check.C) {
}

func (t *FakePersisterTest) SetUpTest(c *check.C) {
}

func (t *FakePersisterTest) TearDownTest(c *check.C) {
}

func (t *FakePersisterTest) TestGetPersisterName(c *check.C) {
	p := NewFakePersister()
	c.Check(p.GetPersisterName(), check.Equals, "fakePersister")
	p.Close()
}

func (t *FakePersisterTest) TestPutDocumentsIntoDB(c *check.C) {
	p := NewFakePersister()
	docs := []*protos.Document{
		&protos.Document{
			Id:  "3de292f2deda3a152270d54f08748d4bd8a106c6c45e8314cecd0769302785d54afe92efe560be7e2ffafc2c0f7c0b902b4b502e870a461090843432a9641ec0",
			Raw: []byte("dflsjfoiwefjlasfffdfjjfggjhjhggadjfoiewffjkhkjalsdjfoiewasjdfjewoiosdjff"),
		},
		&protos.Document{
			Id:  "e568851e15a09b5e80f0caa11dda549ce0f5e56f0ddca530857ba94e483283423d403e7110c1eff6716d6db01d71813b469a0fd870d7d524a131cd6c40a30d9b",
			Raw: []byte("dflsjfoiwefjlasfffdfjjfggjhjhggdsdfadjfoiewffjkhkjalsdjfoiewasjdfjewoiosdjff"),
		},
	}

	c.Check(p.PutDocumentsIntoDB(docs...), check.IsNil)
	c.Check(len(p.documents), check.Equals, 2)

	p.Close()
}
