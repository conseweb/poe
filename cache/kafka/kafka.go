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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/conseweb/poe/protos"
	"github.com/conseweb/poe/utils"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	kafkaLogger = logging.MustGetLogger("kafka")
)

// kafka cache
type KafkaCache struct {
	sync.RWMutex
	brokers                 []string
	topics                  []string
	config                  *sarama.Config
	producer                sarama.SyncProducer
	consumers               map[string]*cluster.Consumer
	topicConsumersDocsChans map[string]map[string]chan *protos.Document
}

// NewKafkaCache return kafka cache
func NewKafkaCache() *KafkaCache {
	flogging.LoggingInit("kafka")

	cache := new(KafkaCache)

	// kafka config
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = viper.GetInt("cache.kafka.retry")
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	tlsConfig := createTlsConfiguration()
	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}
	cache.config = config

	// kafka brokers
	brokers := viper.GetStringSlice("cache.kafka.brokers")
	kafkaLogger.Debugf("brokers: %v", brokers)

	// kafka producer
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		kafkaLogger.Fatalf("Failed to start Sarama producer: %v", err)
	}
	cache.producer = producer
	cache.brokers = brokers

	// kafka topics
	topicConsumersDocsChans := make(map[string]map[string]chan *protos.Document)
	topics := make([]string, 0)
	for _, period := range utils.GetPeriodLimits() {
		topic := cache.Topic(period.Period)

		topicConsumersDocsChans[topic] = make(map[string]chan *protos.Document)
		topics = append(topics, topic)
	}
	cache.topics = topics
	cache.topicConsumersDocsChans = topicConsumersDocsChans

	// consumers
	cache.consumers = make(map[string]*cluster.Consumer)

	return cache
}

func (k *KafkaCache) Put(raw []byte, topic string) (*protos.Document, error) {
	doc := &protos.Document{
		Id:         k.DocumentID(raw),
		Raw:        raw,
		SubmitTime: time.Now().UTC().Unix(),
	}
	docBytes, err := proto.Marshal(doc)
	if err != nil {
		kafkaLogger.Errorf("proto.Marshal() return error: %v", err)
		return nil, err
	}

	partition, offset, err := k.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(docBytes),
	})
	if err != nil {
		kafkaLogger.Errorf("kafka producer send message return error: %v", err)
		return nil, err
	}
	kafkaLogger.Debugf("proto.Document[%s] has sent to kafka partition: %d, offset: %d", doc.Id, partition, offset)

	return doc, nil
}

func (k *KafkaCache) Get(consumerName, topic string, count int64) ([]*protos.Document, error) {
	if _, ok := k.consumers[consumerName]; !ok {
		if !k.Subscribe(consumerName, topic) {
			return nil, fmt.Errorf("%s unable to subscribe topic %s", consumerName, topic)
		}
	}

	k.constructTopicConsumerDocsChan(topic, consumerName)
	docsChan, ok := k.topicConsumersDocsChans[topic][consumerName]
	if !ok {
		return nil, fmt.Errorf("invalid topic %s", topic)
	}

	docs := make([]*protos.Document, 0)
	//kafkaLogger.Debugf("topic %s docs chan len: %d", topic, len(docsChan))
outer:
	for {
		if count <= 0 {
			break
		}

		select {
		case doc := <-docsChan:
			count--
			docs = append(docs, doc)

		default:
			break outer
		}
	}

	return docs, nil
}

func (k *KafkaCache) Subscribe(consumerName, topic string) bool {
	exist := false
	for _, t := range k.topics {
		if t == topic {
			exist = true
		}
	}
	if !exist {
		kafkaLogger.Errorf("can't subscribe not exist topic %s", topic)
		return false
	}

	if _, ok := k.consumers[consumerName]; !ok {
		config := cluster.NewConfig()
		config.Config = *k.config

		c, err := cluster.NewConsumer(k.brokers, consumerName, k.topics, config)
		if err != nil {
			kafkaLogger.Errorf("setup new consumer return error: %v", err)
			return false
		}
		k.consumers[consumerName] = c
		go k.handleConsumer(consumerName)

		return true
	}

	return false
}

func (k *KafkaCache) handleConsumer(consumerName string) {
	consumer, ok := k.consumers[consumerName]
	if !ok {
		return
	}
	go func() {
		for msg := range consumer.Messages() {
			kafkaLogger.Debugf("consumer %s: %s/%d/%d", consumerName, msg.Topic, msg.Partition, msg.Offset)
			if _, ok := k.topicConsumersDocsChans[msg.Topic]; !ok {
				continue
			}
			k.constructTopicConsumerDocsChan(msg.Topic, consumerName)

			document := &protos.Document{}
			if err := proto.Unmarshal(msg.Value, document); err != nil {
				kafkaLogger.Errorf("proto.Unmarshal() return error: %v", err)
				continue
			}

			k.topicConsumersDocsChans[msg.Topic][consumerName] <- document
			consumer.MarkOffset(msg, "")
		}
	}()

	go func() {
		for note := range consumer.Notifications() {
			kafkaLogger.Infof("Rebalanced: %+v\n", note)
		}
	}()

	go func() {
		for err := range consumer.Errors() {
			kafkaLogger.Errorf("consumer %s return error: %s\n", consumerName, err.Error())
		}
	}()
}

func (k *KafkaCache) constructTopicConsumerDocsChan(topic, consumerName string) {
	if _, ok := k.topicConsumersDocsChans[topic][consumerName]; !ok {
		for _, period := range utils.GetPeriodLimits() {
			if k.Topic(period.Period) == topic {
				k.topicConsumersDocsChans[topic][consumerName] = make(chan *protos.Document, period.Limit)
				return
			}
		}

		kafkaLogger.Fatalf("invalid topic %s", topic)
	}

	return
}

func (k *KafkaCache) Topic(d time.Duration) string {
	return fmt.Sprintf("kafka_%s", d.String())
}

func (k *KafkaCache) DocumentID(rawData []byte) string {
	return utils.DocumentID(rawData)
}

func (k *KafkaCache) Close() error {
	if err := k.producer.Close(); err != nil {
		kafkaLogger.Errorf("failed to shut down producer cleanly: %v", err)
	}

	for name, consumer := range k.consumers {
		if err := consumer.Close(); err != nil {
			kafkaLogger.Errorf("failed to shut dowm consumer %s cleanly: %v", name, consumer)
		}
	}

	for {
		for topic, consumersDocsChans := range k.topicConsumersDocsChans {
			for consumer, docsChan := range consumersDocsChans {
			NEXT:
				lenDocsChan := len(docsChan)
				if lenDocsChan != 0 {
					kafkaLogger.Warningf("topic %s consumer %s documents chan are not empty: %d", topic, consumer, lenDocsChan)
					time.Sleep(time.Millisecond * 50)
					goto NEXT
				}
			}
		}
	}

	return nil
}

// createTlsConfiguration
func createTlsConfiguration() (t *tls.Config) {
	certFile := viper.GetString("cache.kafka.tls.certFile")
	keyFile := viper.GetString("cache.kafka.tls.keyFile")
	caFile := viper.GetString("cache.kafka.tls.caFile")

	if certFile != "" && keyFile != "" && caFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			kafkaLogger.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			kafkaLogger.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: viper.GetBool("cache.kafka.tls.insecureSkipVerify"),
		}
	}
	// will be nil by default if nothing is provided
	return t
}
