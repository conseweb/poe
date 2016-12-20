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
package kafka

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/spf13/viper"
	"gopkg.in/check.v1"
)

type KafkaCacheTest struct {
	broker *sarama.MockBroker
	cache  *KafkaCache
}

var _ = check.Suite(&KafkaCacheTest{})
var testMsg = sarama.StringEncoder("Foo")
var topic = fmt.Sprintf("kafka_%s", time.Minute.String())

func (t *KafkaCacheTest) SetUpSuite(c *check.C) {
	viper.Set("cache.kafka.retry", 10)
	viper.Set("cache.kafka.brokers", []string{"localhost:9092"})
	viper.Set("api.period", map[string]int64{"1m": 1000})

	t.broker = sarama.NewMockBrokerAddr(c, 1, "localhost:9092")
	mockFetchResponse := sarama.NewMockFetchResponse(c, 1)
	for i := 0; i < 10; i++ {
		mockFetchResponse.SetMessage(topic, 0, int64(i+1234), testMsg)
	}

	t.broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(c).
			SetBroker(t.broker.Addr(), t.broker.BrokerID()).
			SetLeader(topic, 0, t.broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(c).
			SetOffset(topic, 0, sarama.OffsetOldest, 0).
			SetOffset(topic, 0, sarama.OffsetNewest, 2345),
		"FetchRequest": mockFetchResponse,
	})
}

func (t *KafkaCacheTest) TearDownSuite(c *check.C) {
	t.broker.Close()
}

func (t *KafkaCacheTest) SetUpTest(c *check.C) {
	t.cache = NewKafkaCache()
}

func (t *KafkaCacheTest) TearDownTest(c *check.C) {
	c.Check(t.cache.Close(), check.IsNil)
}

func (t *KafkaCacheTest) TestPut(c *check.C) {
	_, err1 := t.cache.Put("appName", []byte("1"), "", time.Minute)
	_, err2 := t.cache.Put("appName", []byte("2"), "", time.Minute)

	c.Check(err1, check.IsNil)
	c.Check(err2, check.IsNil)
}
