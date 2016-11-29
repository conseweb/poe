package blockchain

import (
	"errors"
	"strings"
	"sync"

	"github.com/conseweb/common/crypto"
	poepb "github.com/conseweb/poe/protos"
	"github.com/hyperledger/fabric/events/consumer"
	fabricpb "github.com/hyperledger/fabric/protos"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var eventClis []*consumer.EventsClient

type (
	items struct {
		lock *sync.RWMutex
		data map[string]interface{}
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

func (self *items) Get(key string) interface{} {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if v, ok := self.data[key]; ok {
		return v
	}
	return nil
}

func (self *items) Set(key string, val interface{}) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	if v, ok := self.data[key]; !ok {
		self.data[key] = val
	} else if v != val {
		self.data[key] = val
	} else {
		return false
	}
	return true
}

func (self *items) Delete(key string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.data, key)
}

func (self *items) Clear() {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.data = make(map[string]interface{})
}

func (adapter *eventAdapter) GetInterestedEvents() ([]*fabricpb.Interest, error) {
	return []*fabricpb.Interest{{EventType: fabricpb.EventType_CHAINCODE, RegInfo: &fabricpb.Interest_ChaincodeRegInfo{ChaincodeRegInfo: &fabricpb.ChaincodeReg{ChaincodeID: adapter.sender.name, EventName: "invoke_completed"}}}}, nil
}

func (adapter *eventAdapter) Recv(msg *fabricpb.Event) (bool, error) {
	if event, ok := msg.Event.(*fabricpb.Event_ChaincodeEvent); ok && event.ChaincodeEvent.ChaincodeID == adapter.sender.name {
		if event.ChaincodeEvent.EventName == "invoke_completed" {
			adapter.ecc <- event
			return true, nil
		}
	}
	return true, nil
}

func (adapter *eventAdapter) Disconnected(e error) {
	if e != nil {
		blockchainLogger.Error(e)
	}
}

func (bc *Blockchain) toSpec(function string, args []string) *fabricpb.ChaincodeSpec {
	input := make([][]byte, 1, len(args)+1)
	input[0] = []byte(function)
	for _, v := range args {
		input = append(input, []byte(v))
	}
	spec := &fabricpb.ChaincodeSpec{
		Type: fabricpb.ChaincodeSpec_Type(fabricpb.ChaincodeSpec_Type_value["GOLANG"]),
		ChaincodeID: &fabricpb.ChaincodeID{
			Path: bc.path,
			Name: bc.name,
		},
		CtorMsg: &fabricpb.ChaincodeInput{
			Args: input,
		},
		SecureContext: bc.secureCtx,
	}
	return spec
}

func (bc *Blockchain) execute(method, function string, args []string) ([]byte, error) {
	var (
		opts []grpc.DialOption = []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTimeout(bc.regTimeout),
			grpc.WithInsecure(),
		}
		conn      *grpc.ClientConn
		devopsCli fabricpb.DevopsClient
		spec      = bc.toSpec(function, args)
		peerAddr  string
		resp      *fabricpb.Response
		e         error
	)
	defer conn.Close()
	for i := 0; i < bc.peerBackend.Len(); i++ {
		peerAddr = bc.peerBackend.Choose().String()
		for j := 0; j < bc.failOver; j++ {
			conn, _ = grpc.Dial(peerAddr, opts...)
			if conn != nil {
				break
			}
			blockchainLogger.Infof("in bc func <execute> Could not create client conn to %s", peerAddr)
		}
	}
	if conn == nil {
		return nil, errors.New("in bc func <execute> create grpc client conn valid")
	}
	devopsCli = fabricpb.NewDevopsClient(conn)
	switch method {
	case "invoke":
		resp, e = devopsCli.Invoke(context.Background(), &fabricpb.ChaincodeInvocationSpec{ChaincodeSpec: spec})
	case "query":
		resp, e = devopsCli.Query(context.Background(), &fabricpb.ChaincodeInvocationSpec{ChaincodeSpec: spec})
	}
	return resp.Msg, e
}

// 启动事件监听
func (bc *Blockchain) eventStart() {
	var (
		adapter  = eventAdapter{sender: bc, ecc: make(chan *fabricpb.Event_ChaincodeEvent)}
		eventCli *consumer.EventsClient
		e        error
	)
	for _, addr := range bc.events {
		if eventCli, e = consumer.NewEventsClient(addr, bc.regTimeout, &adapter); e != nil {
			blockchainLogger.Errorf("in bc func <eventStart> error: %v", e)
			continue
		}
		if e = eventCli.Start(); e != nil {
			blockchainLogger.Errorf("in bc func <eventStart> error: %v", e)
			continue
		}
		eventClis = append(eventClis, eventCli)
	}
	if eventCli == nil {
		blockchainLogger.Error("in bc func <eventStart> create grpc client conn valid")
		return
	}
	for {
		select {
		case ecc := <-adapter.ecc:
			invokeCompleted(adapter.sender, ecc)
		}
	}
}

func (bc *Blockchain) Close() error {
	if len(eventClis) > 0 {
		for _, cli := range eventClis {
			cli.Stop()
		}
	}
	bc.items.Clear()
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
