package blockchain

import (
	"strings"
	"sync"

	"github.com/conseweb/common/crypto"
	poepb "github.com/conseweb/poe/protos"
	"github.com/hyperledger/fabric/events/consumer"
	fabricpb "github.com/hyperledger/fabric/protos"
	"github.com/spf13/viper"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	// grpc client connection
	grpcConn *grpc.ClientConn
	// event client
	eventClis []*consumer.EventsClient
)

type (
	items struct {
		lock *sync.RWMutex
		data map[string]interface{}
	}
	chaincodeWrapper struct {
		Name          string `json:"name,omitempty"`
		Path          string `json:"path,omitempty"`
		Language      string `json:"language,omitempty"`
		SecureContext string `json:"secureContext,omitempty"`
	}
	eventAdapter struct {
		sender *Blockchain
		ecc    chan *fabricpb.Event_ChaincodeEvent
	}
	queryResult struct {
		// 键值
		Key string `json:"key,omitempty"`
		// 键哈希值
		HashKey string `json:"hash_key,omitempty"`
		// 是否存在
		Exist bool `json:"exist,omitempty"`
	}
)

func (this *items) Get(key string) interface{} {
	this.lock.RLock()
	defer this.lock.RUnlock()
	if v, ok := this.data[key]; ok {
		return v
	}
	return nil
}

func (this *items) Set(key string, val interface{}) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	if v, ok := this.data[key]; !ok {
		this.data[key] = val
	} else if v != val {
		this.data[key] = val
	} else {
		return false
	}
	return true
}

func (this *items) Delete(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	delete(this.data, key)
}

func (this *items) Clear() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.data = make(map[string]interface{})
}

func newWrapper(key string) *chaincodeWrapper {
	var (
		wrapper chaincodeWrapper
		e       error
	)
	if e = viper.UnmarshalKey(key, &wrapper); e != nil {
		blockchainLogger.Error(e)
		return nil
	}
	return &wrapper
}

func (this *chaincodeWrapper) ToSpec(function string, args []string) *fabricpb.ChaincodeSpec {
	input := make([][]byte, 1, len(args)+1)
	input[0] = []byte(function)
	for _, v := range args {
		input = append(input, []byte(v))
	}
	spec := &fabricpb.ChaincodeSpec{
		Type: fabricpb.ChaincodeSpec_Type(fabricpb.ChaincodeSpec_Type_value[this.Language]),
		ChaincodeID: &fabricpb.ChaincodeID{
			Path: this.Path,
			Name: this.Name,
		},
		CtorMsg: &fabricpb.ChaincodeInput{
			Args: input,
		},
		SecureContext: this.SecureContext,
	}
	return spec
}

func (this *chaincodeWrapper) Execute(method, function string, args []string) ([]byte, error) {
	var (
		spec      = this.ToSpec(function, args)
		devopsCli = fabricpb.NewDevopsClient(getGrpcConn())
		resp      *fabricpb.Response
		e         error
	)
	switch method {
	case "invoke":
		resp, e = devopsCli.Invoke(context.Background(), &fabricpb.ChaincodeInvocationSpec{ChaincodeSpec: spec})
	case "query":
		resp, e = devopsCli.Query(context.Background(), &fabricpb.ChaincodeInvocationSpec{ChaincodeSpec: spec})
	}
	return resp.Msg, e
}

func (this *eventAdapter) GetInterestedEvents() ([]*fabricpb.Interest, error) {
	return []*fabricpb.Interest{{EventType: fabricpb.EventType_CHAINCODE, RegInfo: &fabricpb.Interest_ChaincodeRegInfo{ChaincodeRegInfo: &fabricpb.ChaincodeReg{ChaincodeID: this.sender.wrapper.Name, EventName: "invoke_completed"}}}}, nil
}

func (this *eventAdapter) Recv(msg *fabricpb.Event) (bool, error) {
	if event, ok := msg.Event.(*fabricpb.Event_ChaincodeEvent); ok && event.ChaincodeEvent.ChaincodeID == this.sender.wrapper.Name {
		if event.ChaincodeEvent.EventName == "invoke_completed" {
			this.ecc <- event
			return true, nil
		}
	}
	return true, nil
}

func (this *eventAdapter) Disconnected(e error) {
	if e != nil {
		blockchainLogger.Error(e)
	}
}

// 启动事件监听
func (this *Blockchain) EventStart() error {
	var (
		adapter               = eventAdapter{sender: this, ecc: make(chan *fabricpb.Event_ChaincodeEvent)}
		addressArray []string = strings.Split(viper.GetString("blockchain.event_address"), ",")
		eventCli     *consumer.EventsClient
		e            error
	)
	for _, addr := range addressArray {
		if eventCli, e = consumer.NewEventsClient(addr, viper.GetDuration("blockchain.reg_timeout"), &adapter); e != nil {
			return e
		}
		if e = eventCli.Start(); e != nil {
			return e
		}
		eventClis = append(eventClis, eventCli)
	}
	for {
		select {
		case ecc := <-adapter.ecc:
			invokeCompleted(adapter.sender, ecc)
		}
	}
	return nil
}

// 释放资源
func (this *Blockchain) Close() error {
	if grpcConn != nil {
		grpcConn.Close()
	}
	if eventClis != nil {
		for _, eventCli := range eventClis {
			eventCli.Stop()
		}
	}
	this.items.Clear()
	return nil
}

// invoke_completed 事件响应处理
func invokeCompleted(sender *Blockchain, e *fabricpb.Event_ChaincodeEvent) error {
	obj := sender.items.Get(e.ChaincodeEvent.TxID)
	if obj != nil {
		if docs, ok := obj.([]*poepb.Document); ok {
			data := strings.Split(string(e.ChaincodeEvent.Payload), ",")
			if len(data) > 0 {
				proofKey := data[0]
				docIds := make([]string, len(docs))
				for idx, doc := range docs {
					docIds[idx] = doc.Id
				}
				blockchainLogger.Infof("<invokeCompleted> proofKey: %s", proofKey)
				blockchainLogger.Infof("<invokeCompleted> docs: %v", docIds)
				go sender.persister.SetDocsBlockDigest(docIds, crypto.Hash(sha3.New512(), []byte(proofKey)))
			}

		}
		sender.items.Delete(e.ChaincodeEvent.TxID)
	}
	return nil
}

// get grpc conn
func getGrpcConn() *grpc.ClientConn {
	var (
		opts []grpc.DialOption = []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTimeout(viper.GetDuration("blockchain.reg_timeout")),
			grpc.WithInsecure(),
		}
		addressArray []string = strings.Split(viper.GetString("blockchain.peer_address"), ",")
	)
	if grpcConn == nil {
		for _, addr := range addressArray {
			grpcConn, _ = grpc.Dial(addr, opts...)
			if grpcConn != nil {
				return grpcConn
			}
		}
	}
	return nil
}
