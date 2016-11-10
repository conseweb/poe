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
package memory

import (
	"gopkg.in/check.v1"
	"testing"
	"fmt"
)

func TestAll(t *testing.T) {
	check.TestingT(t)
}

type MemoryCacheTest struct {
}

var _ = check.Suite(&MemoryCacheTest{})

func (t *MemoryCacheTest) TestPut(c *check.C) {
	cache := NewMemoryCache()
	defer cache.Close()

	cache.Put([]byte("raw"), "test")
	cache.Put([]byte("raw1"), "test")

	c.Check(len(cache.vals), check.Equals, 1)
	c.Check(len(cache.vals["test"]), check.Equals, 2)
}

func (t *MemoryCacheTest) TestClose(c *check.C) {
	cache := NewMemoryCache()
	cache.Put([]byte("raw"), "test")
	cache.Put([]byte("raw1"), "test")

	cache.Close()

	c.Check(len(cache.vals), check.Equals, 0)
	c.Check(cache.vals["test"], check.IsNil)
}

func (t *MemoryCacheTest) BenchmarkPut(c *check.C) {
	cache := NewMemoryCache()
	defer cache.Close()

	topic := "topic0"
	for i := 0; i < c.N; i++ {
		if i % 10 == 0 {
			topic = fmt.Sprintf("topic%d", i)
		}

		cache.Put([]byte(fmt.Sprintf("raw%d", i)), topic)
	}
}